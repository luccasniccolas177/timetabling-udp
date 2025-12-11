package models

import "strings"

// RoomConstraints almacena las restricciones de salas por curso
type RoomConstraints struct {
	// CourseConstraints: CourseCode → EventType → []AllowedRooms
	CourseConstraints map[string]map[EventType][]string

	// Defaults para cursos sin restricciones específicas
	Defaults map[EventType][]string
}

// IsValidRoomForEvent verifica si una sala es válida para un evento dado
func (rc *RoomConstraints) IsValidRoomForEvent(courseCode string, eventType EventType, roomCode string) bool {
	// 1. Buscar restricciones específicas del curso
	if courseRestrictions, exists := rc.CourseConstraints[courseCode]; exists {
		if allowedRooms, hasType := courseRestrictions[eventType]; hasType {
			return rc.isRoomInWhitelist(roomCode, allowedRooms)
		}
	}

	// 2. Usar DEFAULTS si no hay restricciones específicas
	if defaultRooms, exists := rc.Defaults[eventType]; exists {
		return rc.isRoomInWhitelist(roomCode, defaultRooms)
	}

	// 3. Si no hay defaults, permitir cualquier sala
	return true
}

// isRoomInWhitelist verifica si una sala está en la whitelist
// Maneja tokens especiales: ANY_CLASSROOM, ANY_LAB
func (rc *RoomConstraints) isRoomInWhitelist(roomCode string, whitelist []string) bool {
	for _, allowed := range whitelist {
		// Manejar tokens especiales
		if allowed == "ANY_CLASSROOM" {
			// Cualquier sala que NO sea laboratorio
			if !strings.HasPrefix(roomCode, "LAB") && roomCode != "AUDITORIO -2" && roomCode != "AUDITORIO 3" {
				return true
			}
		} else if allowed == "ANY_LAB" {
			// Cualquier laboratorio
			if strings.HasPrefix(roomCode, "LAB") {
				return true
			}
		} else if allowed == roomCode {
			// Match exacto
			return true
		}
	}

	return false
}

// GetValidRoomsForEvent retorna todas las salas válidas para un evento
func (rc *RoomConstraints) GetValidRoomsForEvent(courseCode string, eventType EventType, allRooms map[int]Room) []Room {
	validRooms := make([]Room, 0)

	for _, room := range allRooms {
		if rc.IsValidRoomForEvent(courseCode, eventType, room.Code) {
			validRooms = append(validRooms, room)
		}
	}

	return validRooms
}
