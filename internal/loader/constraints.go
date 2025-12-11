package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"timetabling-UDP/internal/models"
)

// LoadRoomConstraints carga las restricciones de salas desde JSON
func LoadRoomConstraints(filepath string) (*models.RoomConstraints, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo constraints: %w", err)
	}
	defer file.Close()

	// Estructura temporal para parsear JSON
	var rawData map[string]map[string][]string

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&rawData); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %w", err)
	}

	constraints := &models.RoomConstraints{
		CourseConstraints: make(map[string]map[models.EventType][]string),
		Defaults:          make(map[models.EventType][]string),
	}

	// Procesar cada curso
	for courseCode, eventTypes := range rawData {
		if courseCode == "DEFAULTS" {
			// Procesar defaults
			for typeStr, rooms := range eventTypes {
				eventType := parseEventType(typeStr)
				constraints.Defaults[eventType] = rooms
			}
		} else {
			// Procesar restricciones específicas del curso
			constraints.CourseConstraints[courseCode] = make(map[models.EventType][]string)

			for typeStr, rooms := range eventTypes {
				eventType := parseEventType(typeStr)
				constraints.CourseConstraints[courseCode][eventType] = rooms
			}
		}
	}

	return constraints, nil
}

// parseEventType convierte string a EventType
func parseEventType(typeStr string) models.EventType {
	switch typeStr {
	case "CATEDRA":
		return models.CAT
	case "AYUDANTIA":
		return models.AY
	case "LABORATORIO":
		return models.LAB
	default:
		return models.CAT // Default
	}
}
