package loader

import (
	"fmt"
	"strconv"
	"strings"
	"timetabling-UDP/internal/data"
	"timetabling-UDP/internal/models"
)

type UniversityState struct {
	Courses         map[int]models.Course
	Sections        map[int]models.Section
	Teachers        map[int]models.Teacher
	Rooms           map[int]models.Room
	RawEvents       []models.LogicalEvent
	RoomConstraints *models.RoomConstraints // ✅ NUEVO: Restricciones de salas
}

type SectionLookUp map[string]map[int]int
type ProcessedEvents map[string]bool

func LoadFicData(basePath string) (*UniversityState, error) {

	// 1. CARGA DE SALAS
	rooms, err := LoadCSV(fmt.Sprintf("%s/rooms.csv", basePath))
	if err != nil {
		return nil, err
	}

	universityState := &UniversityState{
		Courses:   make(map[int]models.Course),
		Sections:  make(map[int]models.Section),
		Teachers:  make(map[int]models.Teacher),
		Rooms:     make(map[int]models.Room),
		RawEvents: []models.LogicalEvent{},
	}

	for i, row := range rooms[1:] {
		capacity, _ := strconv.Atoi(row[1])
		code := row[0]
		universityState.Rooms[i+1] = models.Room{
			ID: i + 1, Code: code, Capacity: capacity, RoomType: getRoomType(code),
		}
	}

	// 2. CARGA DE PROFESORES Y CURSOS
	teachersJson, err := loadJSON[JSONTeacher](fmt.Sprintf("%s/profesores.json", basePath))
	if err != nil {
		return nil, err
	}
	for _, t := range teachersJson {
		universityState.Teachers[t.ID] = ConvertJSONTeacherToModel(t)
	}

	coursesJson, err := loadJSON[JSONCourse](fmt.Sprintf("%s/oferta_academica.json", basePath))
	if err != nil {
		return nil, err
	}

	reqMap := data.RequirementsMap()
	globalSectionID := 1
	sectionLookUp := make(map[string]map[int]int)

	// === BUCLE 1: POBLAR CURSOS Y SECCIONES ===
	for i, c := range coursesJson {
		courseID := i + 1
		courseModel, sectionModel := ConvertJSONCourseToModel(c, courseID, reqMap)

		if len(courseModel.Requirements) == 0 {
			courseModel.Requirements = inferRequirementsFromCode(courseModel.Code)
		}
		if len(courseModel.Requirements) == 0 {
			continue
		}

		universityState.Courses[courseID] = courseModel

		cleanCode := strings.ToUpper(strings.TrimSpace(courseModel.Code))
		if _, ok := sectionLookUp[cleanCode]; !ok {
			sectionLookUp[cleanCode] = make(map[int]int)
		}

		for _, section := range sectionModel {
			section.ID = globalSectionID
			universityState.Sections[globalSectionID] = section
			sectionLookUp[cleanCode][section.SectionNumber] = globalSectionID
			globalSectionID++
		}
	}

	// === PREPARACIÓN PARA FUSIÓN INTELIGENTE ===
	globalEventID := 1
	processedEvents := make(map[string]bool)
	sectionFusionMap := make(map[int][]int)

	//
	for _, tJSON := range teachersJson {
		if len(tJSON.TeachingLoad) == 0 {
			continue
		}

		for _, load := range tJSON.TeachingLoad {
			targetCode := strings.ToUpper(strings.TrimSpace(load.CourseCode))
			loadType := strings.ToUpper(strings.TrimSpace(load.EventType))

			var courseID int
			var courseDist models.Distribution
			found := false
			for cid, c := range universityState.Courses {
				if strings.ToUpper(strings.TrimSpace(c.Code)) == targetCode {
					courseID = cid
					courseDist = c.Distribution
					found = true
					break
				}
			}
			if !found {
				continue
			}

			var eType models.EventType
			var duration int
			var rType models.RoomType
			var freq int

			switch loadType {
			case "CATEDRA":
				eType = models.CAT
				duration = courseDist.DurationLectures
				rType = models.CR
				freq = courseDist.NumLectures
			case "LABORATORIO":
				eType = models.LAB
				duration = courseDist.DurationLabs
				rType = models.LR
				freq = courseDist.NumLabs
			case "AYUDANTIA":
				eType = models.AY
				duration = courseDist.DurationAssistants
				rType = models.CR
				freq = courseDist.NumAssistants
			default:
				continue
			}
			if duration == 0 {
				continue
			}

			var parentSectionIDs []int
			totalSize := 0

			for _, secNum := range load.RelatedSections {
				if gID, ok := sectionLookUp[targetCode][secNum]; ok {
					parentSectionIDs = append(parentSectionIDs, gID)
					totalSize += universityState.Sections[gID].StudentsNumber
					// Marcamos como procesado
					processedEvents[fmt.Sprintf("%d-%s", gID, loadType)] = true
				}
			}

			if len(parentSectionIDs) == 0 {
				continue
			}

			// Registro de patrón de fusión para Cátedras
			if eType == models.CAT && len(parentSectionIDs) > 1 {
				for _, id := range parentSectionIDs {
					sectionFusionMap[id] = parentSectionIDs
				}
			}

			lEvent := models.LogicalEvent{
				ID:               globalEventID,
				CourseID:         courseID,
				Type:             eType,
				ParentSectionIDs: parentSectionIDs,
				EventNumber:      load.EventNumber, // <--- Correcto desde archivo de profes
				EventSize:        totalSize,
				DurationBlocks:   duration,
				Frequency:        freq,
				TeachersIDs:      []int{tJSON.ID},
				RoomType:         rType,
			}
			universityState.RawEvents = append(universityState.RawEvents, lEvent)
			globalEventID++
		}
	}

	// === BUCLE 3: BARRIDO Y FUSIÓN AUTOMÁTICA POR EVENT NUMBER ===
	// Este bucle maneja eventos que no fueron procesados por el archivo de profesores (ej. Sin Profesor)
	// Implementa una fusión inteligente basada puramente en el EventNumber del JSON de Oferta.

	// Mapa para rastrear fusiones ya creadas en este paso
	// Key: "CourseCode-Type-EventNumber" -> LogicalEventID
	// Mapa para rastrear fusiones ya creadas en este paso
	// Key: "CourseCode-Type-EventNumber" -> LogicalEventID
	// createdAutoEvents := make(map[string]int) // No used anymore with pendingGroups approach

	for _, cJSON := range coursesJson {
		targetCode := strings.ToUpper(strings.TrimSpace(cJSON.Code))
		if _, ok := sectionLookUp[targetCode]; !ok {
			continue
		}

		// Obtener info del curso
		var courseDist models.Distribution
		var courseID int
		// Asumimos que todas las secciones del mismo curso tienen el mismo ID de curso
		// Tomamos la primera sección válida para sacar el ID
		for _, s := range cJSON.Sections {
			if gID, ok := sectionLookUp[targetCode][s.SectionNumber]; ok {
				sec := universityState.Sections[gID]
				courseID = sec.CourseID
				courseDist = universityState.Courses[courseID].Distribution
				break
			}
		}
		if courseID == 0 {
			continue
		}

		// Recolectar todos los eventos pendientes por (Type, EventNum) -> [List of Sections]
		// Esto nos permite agrupar secciones que comparten evento ANTES de crearlos
		type PendingEventGroup struct {
			Type         string
			EventNumber  int
			SectionIDs   []int
			TotalSize    int
			TeachersJSON TeacherField
		}
		// Key: "Type-EventNum"
		pendingGroups := make(map[string]*PendingEventGroup)

		for _, sJSON := range cJSON.Sections {
			globalID, ok := sectionLookUp[targetCode][sJSON.SectionNumber]
			if !ok {
				continue
			}

			for rawType, eventsList := range sJSON.AssignedEvents {
				typeStr := strings.ToUpper(strings.TrimSpace(rawType))

				// Si ya fue procesado por el archivo de profesores, lo ignoramos
				if processedEvents[fmt.Sprintf("%d-%s", globalID, typeStr)] {
					continue
				}

				for _, evtDetail := range eventsList {
					// Clave de agrupación: Tipo y Número de Evento
					groupKey := fmt.Sprintf("%s-%d", typeStr, evtDetail.EventNumber)

					if _, exists := pendingGroups[groupKey]; !exists {
						pendingGroups[groupKey] = &PendingEventGroup{
							Type:         typeStr,
							EventNumber:  evtDetail.EventNumber,
							SectionIDs:   []int{},
							TotalSize:    0,
							TeachersJSON: evtDetail.Teacher, // Guardamos teacher info (aunque sea null)
						}
					}

					// Agregar esta sección al grupo
					group := pendingGroups[groupKey]
					group.SectionIDs = append(group.SectionIDs, globalID)
					group.TotalSize += universityState.Sections[globalID].StudentsNumber
				}
			}
		}

		// Procesar los grupos recolectados y crear LogicalEvents
		for _, group := range pendingGroups {

			// Determinar parámetros del modelo
			var eType models.EventType
			var duration int
			var rType models.RoomType
			var freq int

			switch group.Type {
			case "CATEDRA":
				eType = models.CAT
				duration = courseDist.DurationLectures
				rType = models.CR
				freq = courseDist.NumLectures
			case "LABORATORIO":
				eType = models.LAB
				duration = courseDist.DurationLabs
				rType = models.LR
				freq = courseDist.NumLabs
				// Excepción: Los laboratorios NO se fusionan, son por sección.
				// A menos que explícitamente se quiera (pero el usuario pidió Cátedra).
				// El modelo de dominio soporta Labs compartidos?
				// domain/lab.go tiene "Section *Section" (singular).
				// Por lo tanto, Labs NO se pueden fusionar en el modelo actual.
				// Debemos crear uno por sección.
			case "AYUDANTIA":
				eType = models.AY
				duration = courseDist.DurationAssistants
				rType = models.CR
				freq = courseDist.NumAssistants
			default:
				continue
			}

			if duration == 0 {
				continue
			}

			// Caso especial LABORATORIO: Desagrupar
			if eType == models.LAB {
				for _, secID := range group.SectionIDs {
					lEvent := models.LogicalEvent{
						ID:               globalEventID,
						CourseID:         courseID,
						Type:             eType,
						ParentSectionIDs: []int{secID},
						EventNumber:      group.EventNumber,
						EventSize:        universityState.Sections[secID].StudentsNumber,
						DurationBlocks:   duration,
						Frequency:        freq,
						TeachersIDs:      []int{},
						RoomType:         rType,
					}
					universityState.RawEvents = append(universityState.RawEvents, lEvent)
					globalEventID++

					// Marcar como procesado
					// (Aunque ya no es necesario porque pendingGroups reemplaza el bucle directo)
				}
				continue
			}

			// Caso CATEDRA y AYUDANTIA: Crear UN evento fusionado
			lEvent := models.LogicalEvent{
				ID:               globalEventID,
				CourseID:         courseID,
				Type:             eType,
				ParentSectionIDs: group.SectionIDs, // Todas las secciones juntas
				EventNumber:      group.EventNumber,
				EventSize:        group.TotalSize,
				DurationBlocks:   duration,
				Frequency:        freq,
				TeachersIDs:      []int{}, // TODO: Podríamos intentar resolver nombres de TeachersJSON
				RoomType:         rType,
			}
			universityState.RawEvents = append(universityState.RawEvents, lEvent)
			globalEventID++
		}
	}

	// ✅ NUEVO: Cargar restricciones de salas
	constraintsPath := fmt.Sprintf("%s/rooms_constraints.json", basePath)
	constraints, err := LoadRoomConstraints(constraintsPath)
	if err != nil {
		return nil, fmt.Errorf("error cargando room constraints: %w", err)
	}
	universityState.RoomConstraints = constraints

	if err := ValidateState(universityState); err != nil {
		return nil, err
	}
	return universityState, nil
}

// Funciones auxiliares (se mantienen igual)
func getRoomType(code string) models.RoomType {
	if strings.Contains(strings.ToUpper(code), "LAB") {
		return models.LR
	}
	return models.CR
}

func inferRequirementsFromCode(code string) []models.Requirement {
	const ElectiveSemester = 9
	prefix := ""
	if len(code) >= 3 {
		prefix = code[:3]
	}
	var major models.Major
	switch prefix {
	case "CIT":
		major = models.CIT
	case "CII":
		major = models.CII
	case "COC":
		major = models.COC
	default:
		return nil
	}
	return []models.Requirement{{Major: major, Semester: ElectiveSemester}}
}
