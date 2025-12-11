package solver

import (
	"fmt"
	"sort"
	"timetabling-UDP/internal/domain"
)

// AssignRoomsBurke implementa el algoritmo de asignación de salas descrito en la Sección 3.1 del paper
// "Finding Rooms for Each Exam" adaptado para Course Scheduling.
// Utiliza lógica de desplazamiento para optimizar el uso de salas.
func AssignRoomsBurke(solution *Solution, university *domain.University) []*domain.ClassSession {
	fmt.Println("\n🏢 [ALGORITMO BURKE 3.1] Asignando Salas con Desplazamiento...")

	var totalDudList []*domain.ClassSession

	// Ordenar salas por capacidad (R1...Rm) ascendente
	rooms := make([]*domain.Room, 0, len(university.Rooms))
	for _, room := range university.Rooms {
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})

	// El algoritmo se aplica POR PERIODO (Bloque)
	// Iteramos sobre todos los bloques usados en la solución
	sortedBlocks := getSortedBlocks(solution)

	// Mapa para rastrear "Sala Familiar" (La sala usada por la primera sesión de un curso/sección)
	// Clave: Code + SectionNumber -> RoomID
	familyRooms := make(map[string]int)

	for _, block := range sortedBlocks {
		sessions := solution.Schedule[block]
		if len(sessions) == 0 {
			continue
		}

		// 1. DUDs de este bloque (pasamos familyRooms para priorizar)
		blockDuds := assignRoomsForBlock(sessions, rooms, university, familyRooms)
		totalDudList = append(totalDudList, blockDuds...)

		// Actualizar familyRooms con las nuevas asignaciones
		for _, s := range sessions {
			if s.AssignedRoom != nil {
				key := fmt.Sprintf("%s-%d", s.Class.GetCourse().Code, s.Class.GetSections()[0].Number)
				// Si no tiene sala familiar asignada, guardar esta
				if _, exists := familyRooms[key]; !exists {
					familyRooms[key] = s.AssignedRoom.ID
				}
			}
		}
	}

	return totalDudList
}

// assignRoomsForBlock aplica el algoritmo de la Sección 3.1 para un bloque específico
func assignRoomsForBlock(sessions []*domain.ClassSession, sortedRooms []*domain.Room, university *domain.University, familyRooms map[string]int) []*domain.ClassSession {
	// 1. Ordenar sesiones por tamaño (c1...cn) ascendente (smallest first)
	sortedSessions := make([]*domain.ClassSession, len(sessions))
	copy(sortedSessions, sessions)
	sort.Slice(sortedSessions, func(i, j int) bool {
		return sortedSessions[i].Class.GetStudentCount() < sortedSessions[j].Class.GetStudentCount()
	})

	// Estructura temporal: Sala -> Lista de Sesiones asignadas provisionalmente
	roomAssignments := make(map[int][]*domain.ClassSession)

	// Lista de sesiones que no caben en ninguna sala (ni la más grande)
	var initialDuds []*domain.ClassSession

	// PASO 1-3 del Paper adaptado a Course Scheduling (Sección 3.1)
	// + Lógica de "Misma Sala" (Preferencia)
	for _, session := range sortedSessions {
		assigned := false

		// PREFERENCIA: Misma Sala
		// Verificar si tiene una sala "familiar"
		key := fmt.Sprintf("%s-%d", session.Class.GetCourse().Code, session.Class.GetSections()[0].Number)
		preferredRoomID, hasFamilyRoom := familyRooms[key]

		if hasFamilyRoom {
			// Intentar asignar a la sala preferida PRIMERO
			// Buscar el objeto Room
			var targetRoom *domain.Room
			for _, r := range sortedRooms {
				if r.ID == preferredRoomID {
					targetRoom = r
					break
				}
			}

			if targetRoom != nil {
				// Verificar si la sala preferida está libre (en este bloque temporal) y válida
				// Nota: roomAssignments[id] puede tener sesiones provisionales.
				// Si está vacía o podemos convivir (no, rooms son exclusivas por bloque? Paper dice 'more than one... move').
				// Aquí estamos asignando provisionalmente. Si ponemos 2, luego se resuelve el conflicto.
				// PERO, si ponemos 2 sesiones en la misma sala, una se moverá.
				// Si ponemos nuestra sesión PREFERIDA aquí, competirá con quien ya esté.
				// Si somos más pequeños, nos quedamos. Si somos más grandes, nos mueven.
				// Intentémoslo si es válida.

				if session.Class.GetStudentCount() <= targetRoom.Capacity &&
					university.RoomConstraints.IsValidRoomForClass(session.GetCourse().Code, session.GetType(), targetRoom.Code) {

					roomAssignments[targetRoom.ID] = append(roomAssignments[targetRoom.ID], session)
					assigned = true
				}
			}
		}

		if !assigned {
			// Si no se pudo asignar a la preferida (o no tenía), usar greedy normal
			// Buscar la sala más pequeña donde quepa Y sea válida por constraints
			for _, room := range sortedRooms {
				// Verificar capacidad y restricciones duras de tipo de sala
				if session.Class.GetStudentCount() <= room.Capacity &&
					university.RoomConstraints.IsValidRoomForClass(session.GetCourse().Code, session.GetType(), room.Code) {

					// Asignación provisional
					roomAssignments[room.ID] = append(roomAssignments[room.ID], session)
					assigned = true
					break
				}
			}
		}

		if !assigned {
			initialDuds = append(initialDuds, session)
		}
	}

	// PASO 4-6: Resolver conflictos (Desplazamiento)
	var displacedSessions []*domain.ClassSession
	displacedSessions = append(displacedSessions, initialDuds...)

	for i := 0; i < len(sortedRooms); i++ {
		room := sortedRooms[i]
		assigned := roomAssignments[room.ID]

		if len(assigned) == 0 {
			// Si tenemos sesiones desplazadas pendientes, intentar poner una aquí
			if len(displacedSessions) > 0 {
				var keptDisplaced []*domain.ClassSession
				var placedHere *domain.ClassSession

				// Ordenar desplazados por tamaño
				sort.Slice(displacedSessions, func(k, l int) bool {
					return displacedSessions[k].Class.GetStudentCount() < displacedSessions[l].Class.GetStudentCount()
				})

				for _, dSession := range displacedSessions {
					if placedHere == nil &&
						dSession.Class.GetStudentCount() <= room.Capacity &&
						university.RoomConstraints.IsValidRoomForClass(dSession.GetCourse().Code, dSession.GetType(), room.Code) {
						placedHere = dSession
					} else {
						keptDisplaced = append(keptDisplaced, dSession)
					}
				}

				if placedHere != nil {
					roomAssignments[room.ID] = []*domain.ClassSession{placedHere}
					displacedSessions = keptDisplaced
					assigned = roomAssignments[room.ID]
				} else {
					displacedSessions = keptDisplaced
				}
			}
		}

		// Si hay más de 1 curso en la sala (conflicto)
		if len(assigned) > 1 {
			// Dejar el más pequeño, desplazar el resto
			sort.Slice(assigned, func(k, l int) bool {
				return assigned[k].Class.GetStudentCount() < assigned[l].Class.GetStudentCount()
			})

			keep := assigned[0]
			move := assigned[1:]

			roomAssignments[room.ID] = []*domain.ClassSession{keep}
			displacedSessions = append(displacedSessions, move...)
		}
	}

	// Al final, los que quedan en displacedSessions son DUDs
	for roomID, sessions := range roomAssignments {
		if len(sessions) > 0 {
			session := sessions[0]
			room := getRoomByID(sortedRooms, roomID)
			session.AssignedRoom = room
		}
	}
	for _, dud := range displacedSessions {
		dud.AssignedRoom = nil
	}

	return displacedSessions
}

func getSortedBlocks(solution *Solution) []int {
	var keys []int
	for k := range solution.Schedule {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func getRoomByID(rooms []*domain.Room, id int) *domain.Room {
	for _, r := range rooms {
		if r.ID == id {
			return r
		}
	}
	return nil
}
