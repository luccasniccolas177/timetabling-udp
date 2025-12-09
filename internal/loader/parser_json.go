package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"timetabling-UDP/internal/models"
)

type JSONAssignedEvent struct {
	EventNumber int          `json:"event_number"`
	Teacher     TeacherField `json:"teacher"`
}

type JSONSection struct {
	SectionNumber  int                            `json:"section_number"`
	TotalEnrolled  int                            `json:"total_enrolled"`
	AssignedEvents map[string][]JSONAssignedEvent `json:"assigned_events"`
}

type JSONCourse struct {
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	Distribution models.Distribution  `json:"distribution"`
	Sections     []JSONSection        `json:"sections"`
	Requirements []models.Requirement `json:"requirements"`
}

type JSONTeachingLoad struct {
	CourseCode      string `json:"course_code"`
	CourseName      string `json:"course_name"`
	EventType       string `json:"event_type"`
	EventNumber     int    `json:"event_number"`
	RelatedSections []int  `json:"related_sections"`
}

type JSONTeacher struct {
	ID                int                `json:"id"`
	Name              string             `json:"name"`
	UnavailableBlocks map[string]any     `json:"unavailable_blocks"` // lo definiremos después
	TeachingLoad      []JSONTeachingLoad `json:"teaching_load"`
}

func loadJSON[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var result []T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}

	return result, nil
}

func ConvertJSONCourseToModel(c JSONCourse, courseID int, reqMap map[string][]models.Requirement) (models.Course, []models.Section) {
	course := models.Course{
		ID:           courseID,
		Name:         c.Name,
		Code:         c.Code,
		Distribution: c.Distribution,
		Requirements: reqMap[c.Code],
	}

	// convertir secciones
	var sections []models.Section
	for i, s := range c.Sections {
		sections = append(sections, models.Section{
			ID:             i + 1,
			CourseID:       courseID,
			SectionNumber:  s.SectionNumber,
			StudentsNumber: s.TotalEnrolled,
		})
	}

	return course, sections
}

func ConvertJSONTeacherToModel(t JSONTeacher) models.Teacher {
	return models.Teacher{
		ID:   t.ID,
		Name: t.Name,
		// TODO: unavailable blocks
		// TODO: teaching load si lo vas a usar a futuro
	}
}

type TeacherField struct {
	Names []string
}

func (t *TeacherField) UnmarshalJSON(data []byte) error {

	// Caso 1: null
	if string(data) == "null" {
		t.Names = nil
		return nil
	}

	// Caso 2: string
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single != "" {
			t.Names = []string{single}
		}
		return nil
	}

	// Caso 3: array de strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		t.Names = arr
		return nil
	}

	return fmt.Errorf("teacher tiene un formato inválido: %s", string(data))
}
