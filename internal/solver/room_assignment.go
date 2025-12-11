package solver

import (
	"fmt"
	"sort"
	"timetabling-UDP/internal/domain"
)

// RoomAssignment representa la asignación de una sala a una sesión
type RoomAssignment struct {
	Session *domain.ClassSession
	Room    *domain.Room
	Score   float64
}

// AssignRooms asigna salas físicas a todas las sesiones ya coloreadas
// Retorna lista DUD de sesiones sin sala asignada (para re-coloreo)
// Implementa algoritmo de Burke et al. sección 2.3 y 2.5
func AssignRooms(solution *Solution, university *domain.University) []*domain.ClassSession {
	fmt.Println("\n🏢 [FASE 2] Asignando Salas Físicas...")

	// Tracking de salas ocupadas por bloque
	roomOccupancy := make(map[int]map[int]bool) // [slot][roomID] = occupied

	// DUD list: sesiones que no pudieron ser asignadas
	var dudList []*domain.ClassSession

	// Paso 1: Agrupar sesiones por tipo
	lectures, tutorials, labs := groupSessionsByType(solution)

	fmt.Printf("  📊 Sesiones a asignar: %d cátedras, %d ayudantías, %d labs\n",
		len(lectures), len(tutorials), len(labs))

	// Paso 2: Asignar cátedras (prioridad alta - misma sala para todas las instancias)
	lectureDuds := assignLectures(lectures, university, roomOccupancy)
	dudList = append(dudList, lectureDuds...)

	// Paso 3: Asignar ayudantías (preferencia miércoles)
	tutorialDuds := assignTutorials(tutorials, university, roomOccupancy)
	dudList = append(dudList, tutorialDuds...)

	// Paso 4: Asignar labs (restricciones específicas de sala)
	labDuds := assignLabs(labs, university, roomOccupancy)
	dudList = append(dudList, labDuds...)

	// Paso 5: Reporte de estadísticas
	printRoomAssignmentStats(solution, university)

	if len(dudList) > 0 {
		fmt.Printf("\n⚠️  DUD List: %d sesiones sin sala (serán re-coloreadas)\n", len(dudList))
	}

	return dudList
}

// groupSessionsByType agrupa sesiones por tipo de clase
func groupSessionsByType(solution *Solution) ([]*domain.ClassSession, []*domain.ClassSession, []*domain.ClassSession) {
	var lectures, tutorials, labs []*domain.ClassSession

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			switch session.GetType() {
			case domain.ClassTypeLecture:
				lectures = append(lectures, session)
			case domain.ClassTypeTutorial:
				tutorials = append(tutorials, session)
			case domain.ClassTypeLab:
				labs = append(labs, session)
			}
		}
	}

	return lectures, tutorials, labs
}

// assignLectures asigna salas a cátedras, retorna DUD list
func assignLectures(lectures []*domain.ClassSession, university *domain.University, roomOccupancy map[int]map[int]bool) []*domain.ClassSession {
	fmt.Println("  🎓 Asignando cátedras...")

	var dudList []*domain.ClassSession

	// Agrupar por cátedra (mismo Class.ID)
	lectureGroups := make(map[int][]*domain.ClassSession)
	for _, lecture := range lectures {
		classID := lecture.Class.GetID()
		lectureGroups[classID] = append(lectureGroups[classID], lecture)
	}

	assigned := 0
	failed := 0

	for _, instances := range lectureGroups {
		// Encontrar sala que esté disponible en TODOS los bloques de las instancias
		room := findBestRoomForLectureGroup(instances, university, roomOccupancy)

		if room == nil {
			failed++
			// Agregar todas las instancias a DUD list
			dudList = append(dudList, instances...)
			continue
		}

		// Asignar MISMA sala a todas las instancias
		for _, instance := range instances {
			instance.AssignedRoom = room
			markRoomOccupied(int(instance.AssignedSlot), room.ID, roomOccupancy)
			assigned++
		}
	}

	fmt.Printf("    ✅ Asignadas: %d/%d cátedras (%d instancias)\n",
		len(lectureGroups)-failed, len(lectureGroups), assigned)

	if failed > 0 {
		fmt.Printf("    ⚠️  %d cátedras sin sala → DUD list\n", failed)
	}

	return dudList
}

// findBestRoomForLectureGroup encuentra la mejor sala para un grupo de instancias de cátedra
func findBestRoomForLectureGroup(instances []*domain.ClassSession, university *domain.University, roomOccupancy map[int]map[int]bool) *domain.Room {
	if len(instances) == 0 {
		return nil
	}

	// Obtener restricciones del curso
	course := instances[0].GetCourse()
	classType := instances[0].GetType()

	// Obtener salas válidas según restricciones
	validRooms := university.RoomConstraints.GetValidRoomsForClass(course.Code, classType, university.Rooms)

	// Filtrar salas que estén disponibles en TODOS los bloques
	availableRooms := filterAvailableRoomsForAll(instances, validRooms, roomOccupancy)

	if len(availableRooms) == 0 {
		return nil
	}

	// Calcular score para cada sala y elegir la mejor
	bestRoom := availableRooms[0]
	bestScore := scoreRoomForLecture(instances[0], bestRoom)

	for _, room := range availableRooms[1:] {
		score := scoreRoomForLecture(instances[0], room)
		if score > bestScore {
			bestScore = score
			bestRoom = room
		}
	}

	return bestRoom
}

// filterAvailableRoomsForAll filtra salas disponibles en todos los bloques
func filterAvailableRoomsForAll(instances []*domain.ClassSession, rooms []*domain.Room, roomOccupancy map[int]map[int]bool) []*domain.Room {
	var available []*domain.Room

	for _, room := range rooms {
		isAvailable := true
		for _, instance := range instances {
			slot := int(instance.AssignedSlot)
			if roomOccupancy[slot] != nil && roomOccupancy[slot][room.ID] {
				isAvailable = false
				break
			}
		}
		if isAvailable {
			available = append(available, room)
		}
	}

	return available
}

// scoreRoomForLecture calcula el score de una sala para una cátedra
func scoreRoomForLecture(session *domain.ClassSession, room *domain.Room) float64 {
	score := 0.0

	// Restricción blanda 1: Maximizar ocupación (+0 a +50)
	studentCount := session.Class.GetStudentCount()
	occupancy := float64(studentCount) / float64(room.Capacity)

	if occupancy >= 0.8 && occupancy <= 1.0 {
		score += 50 // Ocupación óptima (80-100%)
	} else if occupancy >= 0.6 && occupancy < 0.8 {
		score += 35 // Aceptable (60-80%)
	} else if occupancy >= 0.4 && occupancy < 0.6 {
		score += 20 // Subóptimo (40-60%)
	} else if occupancy < 0.4 {
		score += 5 // Muy subóptimo (<40%)
	} else {
		// Sobrecapacidad (>100%)
		score -= 100 // Penalización fuerte
	}

	// Bonus por sala más pequeña que aún cabe (evitar desperdiciar salas grandes)
	if occupancy >= 0.8 && occupancy <= 1.0 {
		wasteScore := 50 * (1.0 - (float64(room.Capacity-studentCount) / float64(room.Capacity)))
		score += wasteScore
	}

	return score
}

// assignTutorials asigna salas a ayudantías, retorna DUD list
func assignTutorials(tutorials []*domain.ClassSession, university *domain.University, roomOccupancy map[int]map[int]bool) []*domain.ClassSession {
	fmt.Println("  📝 Asignando ayudantías...")

	var dudList []*domain.ClassSession

	// Ordenar: miércoles primero
	sort.Slice(tutorials, func(i, j int) bool {
		return isMiercoles(tutorials[i].AssignedSlot) && !isMiercoles(tutorials[j].AssignedSlot)
	})

	assigned := 0

	for _, tutorial := range tutorials {
		room := findBestRoomForSession(tutorial, university, roomOccupancy)

		if room == nil {
			dudList = append(dudList, tutorial)
			continue
		}

		tutorial.AssignedRoom = room
		markRoomOccupied(int(tutorial.AssignedSlot), room.ID, roomOccupancy)
		assigned++
	}

	fmt.Printf("    ✅ Asignadas: %d/%d ayudantías\n", assigned, len(tutorials))

	if len(dudList) > 0 {
		fmt.Printf("    ⚠️  %d ayudantías sin sala → DUD list\n", len(dudList))
	}

	return dudList
}

// assignLabs asigna salas a labs, retorna DUD list
func assignLabs(labs []*domain.ClassSession, university *domain.University, roomOccupancy map[int]map[int]bool) []*domain.ClassSession {
	fmt.Println("  🔬 Asignando laboratorios...")

	var dudList []*domain.ClassSession
	assigned := 0

	for _, lab := range labs {
		room := findBestRoomForSession(lab, university, roomOccupancy)

		if room == nil {
			dudList = append(dudList, lab)
			continue
		}

		lab.AssignedRoom = room
		markRoomOccupied(int(lab.AssignedSlot), room.ID, roomOccupancy)
		assigned++
	}

	fmt.Printf("    ✅ Asignados: %d/%d laboratorios\n", assigned, len(labs))

	if len(dudList) > 0 {
		fmt.Printf("    ⚠️  %d labs sin sala → DUD list\n", len(dudList))
	}

	return dudList
}

// findBestRoomForSession encuentra la mejor sala para una sesión individual
func findBestRoomForSession(session *domain.ClassSession, university *domain.University, roomOccupancy map[int]map[int]bool) *domain.Room {
	course := session.GetCourse()
	classType := session.GetType()

	// Obtener salas válidas
	validRooms := university.RoomConstraints.GetValidRoomsForClass(course.Code, classType, university.Rooms)

	// Filtrar salas disponibles en este bloque
	slot := int(session.AssignedSlot)
	var availableRooms []*domain.Room

	for _, room := range validRooms {
		if roomOccupancy[slot] == nil || !roomOccupancy[slot][room.ID] {
			availableRooms = append(availableRooms, room)
		}
	}

	if len(availableRooms) == 0 {
		return nil
	}

	// Calcular score y elegir mejor
	bestRoom := availableRooms[0]
	bestScore := scoreRoomForSession(session, bestRoom)

	for _, room := range availableRooms[1:] {
		score := scoreRoomForSession(session, room)
		if score > bestScore {
			bestScore = score
			bestRoom = room
		}
	}

	return bestRoom
}

// scoreRoomForSession calcula el score de una sala para una sesión
func scoreRoomForSession(session *domain.ClassSession, room *domain.Room) float64 {
	score := 0.0

	// Maximizar ocupación
	studentCount := session.Class.GetStudentCount()
	occupancy := float64(studentCount) / float64(room.Capacity)

	if occupancy >= 0.8 && occupancy <= 1.0 {
		score += 50
	} else if occupancy >= 0.6 {
		score += 30
	} else {
		score += 10
	}

	// Bonus por miércoles para ayudantías
	if session.GetType() == domain.ClassTypeTutorial && isMiercoles(session.AssignedSlot) {
		score += 20
	}

	return score
}

// isMiercoles verifica si un slot es miércoles
func isMiercoles(slot domain.TimeSlot) bool {
	// Estructura: 7 bloques por día, 5 días (Lunes a Viernes)
	// Bloque 0-6: Lunes
	// Bloque 7-13: Martes
	// Bloque 14-20: Miércoles ← Objetivo
	// Bloque 21-27: Jueves
	// Bloque 28-34: Viernes

	slotInt := int(slot)

	// Miércoles es el día 2 (0-indexed)
	// Bloques 14-20 son miércoles
	return slotInt >= 14 && slotInt <= 20
}

// markRoomOccupied marca una sala como ocupada en un bloque
func markRoomOccupied(slot int, roomID int, roomOccupancy map[int]map[int]bool) {
	if roomOccupancy[slot] == nil {
		roomOccupancy[slot] = make(map[int]bool)
	}
	roomOccupancy[slot][roomID] = true
}

// validateRoomAssignments valida que todas las sesiones tengan sala asignada
func validateRoomAssignments(solution *Solution) error {
	unassigned := 0

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.AssignedRoom == nil {
				unassigned++
			}
		}
	}

	if unassigned > 0 {
		return fmt.Errorf("%d sesiones sin sala asignada", unassigned)
	}

	return nil
}

// printRoomAssignmentStats imprime estadísticas de asignación
func printRoomAssignmentStats(solution *Solution, university *domain.University) {
	fmt.Println("\n📊 ESTADÍSTICAS DE ASIGNACIÓN DE SALAS")
	fmt.Println("================================================================================")

	// Calcular ocupación promedio
	totalOccupancy := 0.0
	count := 0

	roomUsage := make(map[int]int)

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.AssignedRoom != nil {
				// Ocupación
				studentCount := session.Class.GetStudentCount()
				occupancy := float64(studentCount) / float64(session.AssignedRoom.Capacity)
				totalOccupancy += occupancy
				count++

				// Uso de sala
				roomUsage[session.AssignedRoom.ID]++
			}
		}
	}

	avgOccupancy := totalOccupancy / float64(count) * 100
	fmt.Printf("Ocupación promedio de salas: %.1f%%\n", avgOccupancy)

	// Top 5 salas más usadas
	type roomCount struct {
		room  *domain.Room
		count int
	}

	var roomCounts []roomCount
	for roomID, count := range roomUsage {
		room := university.Rooms[roomID]
		roomCounts = append(roomCounts, roomCount{room, count})
	}

	sort.Slice(roomCounts, func(i, j int) bool {
		return roomCounts[i].count > roomCounts[j].count
	})

	fmt.Println("\nTop 5 salas más usadas:")
	for i := 0; i < 5 && i < len(roomCounts); i++ {
		rc := roomCounts[i]
		fmt.Printf("%d. Sala %d (cap: %d): %d sesiones\n",
			i+1, rc.room.ID, rc.room.Capacity, rc.count)
	}

	// Verificar restricciones blandas cumplidas
	lecturesWithSameRoom := countLecturesWithSameRoom(solution)
	totalLectures := countTotalLectures(solution)

	fmt.Printf("\nCátedras con misma sala para todas las instancias: %d/%d (%.1f%%)\n",
		lecturesWithSameRoom, totalLectures,
		float64(lecturesWithSameRoom)/float64(totalLectures)*100)

	tutorialsOnWednesday := countTutorialsOnWednesday(solution)
	totalTutorials := countTotalTutorials(solution)

	fmt.Printf("Ayudantías en miércoles: %d/%d (%.1f%%)\n",
		tutorialsOnWednesday, totalTutorials,
		float64(tutorialsOnWednesday)/float64(totalTutorials)*100)

	fmt.Println("================================================================================")
}

// Helper functions for stats
func countLecturesWithSameRoom(solution *Solution) int {
	lectureGroups := make(map[int][]*domain.ClassSession)

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeLecture {
				classID := session.Class.GetID()
				lectureGroups[classID] = append(lectureGroups[classID], session)
			}
		}
	}

	count := 0
	for _, instances := range lectureGroups {
		if len(instances) < 2 {
			count++
			continue
		}

		sameRoom := true
		firstRoom := instances[0].AssignedRoom
		for _, instance := range instances[1:] {
			if instance.AssignedRoom == nil || instance.AssignedRoom.ID != firstRoom.ID {
				sameRoom = false
				break
			}
		}

		if sameRoom {
			count++
		}
	}

	return count
}

func countTotalLectures(solution *Solution) int {
	lectureGroups := make(map[int]bool)

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeLecture {
				lectureGroups[session.Class.GetID()] = true
			}
		}
	}

	return len(lectureGroups)
}

func countTutorialsOnWednesday(solution *Solution) int {
	count := 0

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeTutorial && isMiercoles(session.AssignedSlot) {
				count++
			}
		}
	}

	return count
}

func countTotalTutorials(solution *Solution) int {
	count := 0

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeTutorial {
				count++
			}
		}
	}

	return count
}
