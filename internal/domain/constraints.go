package domain

// RoomConstraints almacena las restricciones de salas por curso
// Indica qué salas son válidas para cada tipo de clase de cada curso
type RoomConstraints struct {
	// CourseConstraints: CourseCode → ClassType → []AllowedRoomCodes
	// Ejemplo: "CIT1000" → "LABORATORIO" → ["LAB D", "LAB O", "LAB U"]
	CourseConstraints map[string]map[ClassType][]string

	// Defaults para cursos sin restricciones específicas
	// Ejemplo: "CATEDRA" → ["ANY_CLASSROOM"]
	Defaults map[ClassType][]string
}

// IsValidRoomForClass verifica si una sala es válida para una clase dada
func (rc *RoomConstraints) IsValidRoomForClass(courseCode string, classType ClassType, roomCode string) bool {
	// 1. Buscar restricciones específicas del curso
	if courseRestrictions, exists := rc.CourseConstraints[courseCode]; exists {
		if allowedRooms, hasType := courseRestrictions[classType]; hasType {
			return rc.isRoomInWhitelist(roomCode, allowedRooms)
		}
	}

	// 2. Usar DEFAULTS si no hay restricciones específicas
	if defaultRooms, exists := rc.Defaults[classType]; exists {
		return rc.isRoomInWhitelist(roomCode, defaultRooms)
	}

	// 3. Si no hay defaults, permitir cualquier sala
	return true
}

// isRoomInWhitelist verifica si una sala está en la whitelist
// Maneja tokens especiales: ANY_CLASSROOM, ANY_LAB
func (rc *RoomConstraints) isRoomInWhitelist(roomCode string, whitelist []string) bool {
	for _, allowed := range whitelist {
		// Tokens especiales
		if allowed == "ANY_CLASSROOM" {
			// Cualquier sala que NO sea laboratorio
			if !isLaboratoryRoom(roomCode) {
				return true
			}
		} else if allowed == "ANY_LAB" {
			// Cualquier laboratorio
			if isLaboratoryRoom(roomCode) {
				return true
			}
		} else if allowed == roomCode {
			// Match exacto
			return true
		}
	}

	return false
}

// isLaboratoryRoom determina si un código de sala corresponde a un laboratorio
func isLaboratoryRoom(roomCode string) bool {
	// Los laboratorios empiezan con "LAB"
	return len(roomCode) >= 3 && roomCode[:3] == "LAB"
}

// GetValidRoomsForClass retorna todas las salas válidas para una clase
func (rc *RoomConstraints) GetValidRoomsForClass(courseCode string, classType ClassType, allRooms map[int]*Room) []*Room {
	validRooms := make([]*Room, 0)

	for _, room := range allRooms {
		if rc.IsValidRoomForClass(courseCode, classType, room.Code) {
			validRooms = append(validRooms, room)
		}
	}

	return validRooms
}
