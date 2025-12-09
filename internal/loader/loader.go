package loader

import (
	"fmt"
	"strconv"
	"strings"
	"timetabling-UDP/internal/data"
	"timetabling-UDP/internal/models"
)

type UniversityState struct {
	Courses   map[int]models.Course
	Sections  map[int]models.Section
	Teachers  map[int]models.Teacher
	Rooms     map[int]models.Room
	RawEvents []models.LogicalEvent
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

	// === BUCLE 3: BARRIDO Y FUSIÓN DE ESPEJO (AYUDANTÍAS HUÉRFANAS) ===
	for _, cJSON := range coursesJson {
		targetCode := strings.ToUpper(strings.TrimSpace(cJSON.Code))
		if _, ok := sectionLookUp[targetCode]; !ok {
			continue
		}

		var courseDist models.Distribution
		var courseID int
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

		for _, sJSON := range cJSON.Sections {
			globalID, ok := sectionLookUp[targetCode][sJSON.SectionNumber]
			if !ok {
				continue
			}

			for rawType, eventsList := range sJSON.AssignedEvents { // Iteramos KEY y VALOR (lista)
				typeStr := strings.ToUpper(strings.TrimSpace(rawType))

				// Si este tipo ya fue procesado para esta sección, saltamos TODOS los eventos de ese tipo
				if processedEvents[fmt.Sprintf("%d-%s", globalID, typeStr)] {
					continue
				}

				// Iteramos la lista de eventos dentro de este tipo (para capturar el numero)
				for _, evtDetail := range eventsList {

					var eType models.EventType
					var duration int
					var rType models.RoomType
					var freq int

					switch typeStr {
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

					// --- LÓGICA DE FUSIÓN POR ESPEJO ---
					finalParentIDs := []int{globalID}
					finalSize := universityState.Sections[globalID].StudentsNumber

					if typeStr == "AYUDANTIA" || typeStr == "CATEDRA" {
						if group, exists := sectionFusionMap[globalID]; exists {

							// Verificamos si ya procesamos este grupo en Bucle 3 para no duplicar
							groupKey := fmt.Sprintf("GROUP-%d-%s-%d", group[0], typeStr, evtDetail.EventNumber)
							if processedEvents[groupKey] {
								continue
							}

							finalParentIDs = group
							finalSize = 0
							for _, gID := range group {
								finalSize += universityState.Sections[gID].StudentsNumber
								processedEvents[fmt.Sprintf("%d-%s", gID, typeStr)] = true
							}
							processedEvents[groupKey] = true
						}
					}
					// -----------------------------------

					lEvent := models.LogicalEvent{
						ID:               globalEventID,
						CourseID:         courseID,
						Type:             eType,
						ParentSectionIDs: finalParentIDs,
						EventNumber:      evtDetail.EventNumber, // <--- AQUÍ CAPTURAMOS EL NÚMERO
						EventSize:        finalSize,
						DurationBlocks:   duration,
						Frequency:        freq,
						TeachersIDs:      []int{}, // Sin profesor
						RoomType:         rType,
					}
					universityState.RawEvents = append(universityState.RawEvents, lEvent)
					globalEventID++
				}
			}
		}
	}

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
