package loader

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"timetabling-UDP/internal/domain"
)

// CourseOfertaJSON representa un curso en oferta_academica.json
type CourseOfertaJSON struct {
	CourseCode string         `json:"course_code"`
	CourseName string         `json:"course_name"`
	Activities []ActivityJSON `json:"activities"`
}

// ActivityJSON representa una actividad del json
type ActivityJSON struct {
	ID             int      `json:"id"`
	ActivityCode   string   `json:"activity_code"`
	Type           string   `json:"type"`
	EventNumber    int      `json:"event_number"`
	LinkedSections []int    `json:"linked_sections"`
	TotalStudents  int      `json:"total_students"`
	Teachers       []string `json:"teachers"`
	Comment        string   `json:"comment"`
}

// CourseDistributionJSON representa un curso en courses.json
type CourseDistributionJSON struct {
	ID           int              `json:"ID"`
	Code         string           `json:"Code"`
	Name         string           `json:"Name"`
	Distribution DistributionJSON `json:"Distribution"`
}

// DistributionJSON representa la carga semanal
type DistributionJSON struct {
	NumCAT      int `json:"NumCAT"`
	NumAY       int `json:"NumAY"`
	NumLAB      int `json:"NumLAB"`
	DurationCAT int `json:"DurationCAT"`
	DurationAY  int `json:"DurationAY"`
	DurationLAB int `json:"DurationLAB"`
}

// LoadCourseDistributions lee courses.json y retorna el mapa de distribuciones donde la llave es el codigo del curso
func LoadCourseDistributions(path string) (map[string]DistributionJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var courses []CourseDistributionJSON
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	result := make(map[string]DistributionJSON)
	for _, c := range courses {
		result[c.Code] = c.Distribution
	}
	return result, nil
}

// CourseFullJSON representa un curso completo en courses.json con PlanLocation
type CourseFullJSON struct {
	ID           int              `json:"ID"`
	Code         string           `json:"Code"`
	Name         string           `json:"Name"`
	PlanLocation map[string]int   `json:"PlanLocation"` // Major -> Semester
	Distribution DistributionJSON `json:"Distribution"`
	IsElective   bool             `json:"IsElective"` // Si es electivo
}

// LoadCoursePlanLocations lee courses.json y retorna el mapa con la carrera y semestre en que se imparte el curso
func LoadCoursePlanLocations(path string) (map[string]map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var courses []CourseFullJSON
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, c := range courses {
		result[c.Code] = c.PlanLocation
	}
	return result, nil
}

// LoadElectives lee courses.json y retorna los codigos de los cursos electivos, para diferenciarlos de los de la malla
func LoadElectives(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var courses []CourseFullJSON
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, c := range courses {
		if c.IsElective {
			result[c.Code] = true
		}
	}
	return result, nil
}

// LoadPrerequisites lee courses.json y retorna mapa con los los prerequisitos de cada curso
func LoadPrerequisites(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Estructura con Prerequisites como IDs
	type CourseWithPrereq struct {
		ID            int    `json:"ID"`
		Code          string `json:"Code"`
		Prerequisites []int  `json:"Prerequisites"`
	}

	var courses []CourseWithPrereq
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	// Crear mapa ID -> Code
	idToCode := make(map[int]string)
	for _, c := range courses {
		idToCode[c.ID] = c.Code
	}

	// Crear mapa de prerequisitos
	result := make(map[string][]string)
	for _, c := range courses {
		if len(c.Prerequisites) > 0 {
			var prereqCodes []string
			for _, prereqID := range c.Prerequisites {
				if code, ok := idToCode[prereqID]; ok {
					prereqCodes = append(prereqCodes, code)
				}
			}
			result[c.Code] = prereqCodes
		}
	}
	return result, nil
}

// LoadActivitiesWithExpansion lee oferta_academica.json y crea una el conjutno de actividades por cada sección siguiendo la distribución del curso
func LoadActivitiesWithExpansion(ofertaPath, coursesPath string) ([]domain.Activity, error) {
	// Cargar distribuciones de cursos
	distributions, err := LoadCourseDistributions(coursesPath)
	if err != nil {
		return nil, fmt.Errorf("error cargando courses.json: %w", err)
	}

	// Cargar oferta académica
	data, err := os.ReadFile(ofertaPath)
	if err != nil {
		return nil, err
	}

	var courses []CourseOfertaJSON
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	var activities []domain.Activity
	activityID := 1 // Contador global de IDs

	for _, c := range courses {
		dist := distributions[c.CourseCode]

		for _, a := range c.Activities {
			eventType := parseEventCategory(a.Type)

			// Determinar cuántas sesiones semanales y duración según tipo
			numSessions := 1
			duration := 1
			switch eventType {
			case domain.CAT:
				numSessions = dist.NumCAT
				duration = dist.DurationCAT
			case domain.AY:
				numSessions = dist.NumAY
				duration = dist.DurationAY
			case domain.LAB:
				numSessions = dist.NumLAB
				duration = dist.DurationLAB
			}

			// Si no hay distribución definida, usar valores por defecto
			if numSessions == 0 {
				numSessions = 1
			}
			if duration == 0 {
				duration = 1
			}

			// SiblingGroupID para agrupar actividades espejo (catedras)
			siblingGroup := ""
			if eventType == domain.CAT {
				siblingGroup = buildSiblingGroupID(c.CourseCode, a.LinkedSections)
			}

			// Crear N actividades (sesiones) para este evento
			for session := 1; session <= numSessions; session++ {
				sessionCode := fmt.Sprintf("%s-S%d", a.ActivityCode, session)

				activity := domain.NewActivity(
					activityID,
					sessionCode,
					c.CourseCode,
					c.CourseName,
					eventType,
					a.EventNumber,
					a.LinkedSections,
					a.TotalStudents,
					a.Teachers,
					siblingGroup,
					duration, // Duración en bloques
				)
				activities = append(activities, activity)
				activityID++
			}
		}
	}
	return activities, nil
}

// buildSiblingGroupID genera un ID único para agrupar cátedras hermanas.
func buildSiblingGroupID(courseCode string, sections []int) string {
	if len(sections) == 0 {
		return ""
	}
	secs := make([]string, len(sections))
	for i, s := range sections {
		secs[i] = strconv.Itoa(s)
	}
	return fmt.Sprintf("%s-CAT-%s", courseCode, strings.Join(secs, ","))
}

func parseEventCategory(s string) domain.EventCategory {
	switch s {
	case "CATEDRA":
		return domain.CAT
	case "AYUDANTIA":
		return domain.AY
	case "LABORATORIO":
		return domain.LAB
	default:
		return domain.CAT
	}
}

// LoadRooms lee rooms.csv y retorna las salas utilizadas
func LoadRooms(path string) ([]domain.Room, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var rooms []domain.Room
	for i, record := range records {
		if i == 0 { // Skip header
			continue
		}
		if len(record) < 2 {
			continue
		}
		capacity, _ := strconv.Atoi(record[1])
		roomType := domain.RoomClassroom
		if strings.HasPrefix(record[0], "LAB") {
			roomType = domain.RoomLab
		}
		rooms = append(rooms, domain.Room{
			ID:       i,
			Code:     record[0],
			Capacity: capacity,
			Type:     roomType,
		})
	}
	return rooms, nil
}

// TeacherJSON representa un profesor de profesores.json
type TeacherJSON struct {
	ID                int                `json:"id"`
	Name              string             `json:"name"`
	UnavailableBlocks map[string][]int   `json:"unavailable_blocks"`
	TeachingLoad      []TeachingLoadJSON `json:"teaching_load"`
}

// TeachingLoadJSON representa la carga docente
type TeachingLoadJSON struct {
	CourseCode      string `json:"course_code"`
	CourseName      string `json:"course_name"`
	EventType       string `json:"event_type"`
	EventNumber     int    `json:"event_number"`
	RelatedSections []int  `json:"related_sections"`
}

// LoadTeachers lee profesores.json y retorna los profesores de la FIC
func LoadTeachers(path string) ([]domain.Teacher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var teachersJSON []TeacherJSON
	if err := json.Unmarshal(data, &teachersJSON); err != nil {
		return nil, err
	}

	var teachers []domain.Teacher
	for _, t := range teachersJSON {
		// Aplanar bloques no disponibles de todos los días
		var busyBlocks []int
		for _, blocks := range t.UnavailableBlocks {
			busyBlocks = append(busyBlocks, blocks...)
		}
		teachers = append(teachers, domain.Teacher{
			ID:         t.ID,
			Name:       t.Name,
			BusyBlocks: busyBlocks,
		})
	}
	return teachers, nil
}

// RoomConstraints mapea CourseCode -> EventType -> []AllowedRooms
type RoomConstraints map[string]map[string][]string

// LoadRoomConstraints lee las restricciones de salas de rooms_constraints.json.
func LoadRoomConstraints(path string) (RoomConstraints, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var constraints RoomConstraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		return nil, err
	}

	return constraints, nil
}

// GetAllowedRooms retorna las salas permitidas para una actividad.
// Si no hay restricción, retorna nil (significa cualquier sala).
func (rc RoomConstraints) GetAllowedRooms(courseCode string, eventType string) []string {
	if courseConstraints, ok := rc[courseCode]; ok {
		if allowed, ok := courseConstraints[eventType]; ok {
			return allowed
		}
	}
	return nil // cualquier sala
}

// FilterRoomsByConstraint filtra las salas disponibles según las restricciones
func FilterRoomsByConstraint(rooms []domain.Room, allowedCodes []string) []domain.Room {
	if allowedCodes == nil {
		return rooms // Sin restricción, todas disponibles
	}

	allowedSet := make(map[string]bool)
	for _, code := range allowedCodes {
		allowedSet[code] = true
	}

	var filtered []domain.Room
	for _, r := range rooms {
		if allowedSet[r.Code] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
