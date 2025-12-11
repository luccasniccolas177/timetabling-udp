package graph

import (
	"fmt"
	"timetabling-UDP/internal/domain"
)

// BuildConflictGraph construye el grafo de conflictos desde el modelo de dominio
// Reemplaza la versión anterior que usaba UniversityState
func BuildConflictGraph(university *domain.University) *ConflictGraph {
	// 1. Inicializar grafo vacío
	g := NewConflictGraph()

	// 2. GENERAR TODAS LAS SESIONES DE CLASE
	// Cada clase (Lecture, Tutorial, Lab) genera 1 o más sesiones según su frecuencia
	allSessions := generateAllSessions(university)

	fmt.Printf("📊 Generadas %d sesiones de clase\n", len(allSessions))

	// 3. AGREGAR NODOS AL GRAFO
	for _, session := range allSessions {
		g.AddNode(session)
	}

	// 4. AGREGAR ARISTAS (CONFLICTOS)

	// A. Conflictos de Profesor
	// Dos sesiones no pueden estar en el mismo slot si comparten profesor
	addTeacherConflicts(g, allSessions)

	// B. Conflictos de Misma Clase
	// Las múltiples sesiones de una misma clase no pueden estar en el mismo slot
	// Ejemplo: Cátedra 1 con 3 sesiones → las 3 deben estar en slots diferentes
	addSameClassConflicts(g, allSessions)

	// B2. Conflictos de Misma Sección
	// Las diferentes clases de una misma sección no pueden estar en el mismo slot
	// Ejemplo: Cátedra, Ayudantía y Lab de Sección 1 → deben estar en slots diferentes
	addSameSectionConflicts(g, allSessions)

	// B3. Conflictos de Cátedra Única por Semestre
	// Si en un semestre hay cursos con una sola cátedra, esas cátedras no pueden solaparse
	// Ejemplo: Semestre 8 COC tiene 5 cursos con 1 cátedra cada uno → no pueden solaparse
	addSingleLectureConflicts(g, allSessions, university)

	// C. Conflictos de Escasez de Salas
	// Dos sesiones que comparten exactamente 1 sala válida no pueden estar juntas
	addRoomScarcityConflicts(g, allSessions, university)

	// D. Conflictos de Semestre (DESACTIVADO)
	// NOTA: Desactivado porque causa que el horario necesite 51 bloques (infactible)
	// Con esta restricción desactivada, el horario usa 27 bloques (factible)
	// Los estudiantes tienen suficiente flexibilidad con 20+ secciones por curso
	// para evitar conflictos manualmente
	// addSelectiveCurriculumConflicts(g, allSessions, university)

	return g
}

// generateAllSessions genera todas las sesiones de clase del semestre
func generateAllSessions(university *domain.University) []*domain.ClassSession {
	sessions := make([]*domain.ClassSession, 0)

	// Generar sesiones de Lectures
	for _, lecture := range university.Lectures {
		lectureSessions := domain.GenerateSessions(lecture)
		sessions = append(sessions, lectureSessions...)
	}

	// Generar sesiones de Tutorials
	for _, tutorial := range university.Tutorials {
		tutorialSessions := domain.GenerateSessions(tutorial)
		sessions = append(sessions, tutorialSessions...)
	}

	// Generar sesiones de Labs
	for _, lab := range university.Labs {
		labSessions := domain.GenerateSessions(lab)
		sessions = append(sessions, labSessions...)
	}

	return sessions
}

// addTeacherConflicts agrega aristas entre sesiones que comparten profesor
func addTeacherConflicts(g *ConflictGraph, sessions []*domain.ClassSession) {
	// Agrupar sesiones por profesor
	teacherBuckets := make(map[int][]*domain.ClassSession)

	for _, session := range sessions {
		teachers := session.Class.GetTeachers()
		for _, teacher := range teachers {
			teacherBuckets[teacher.ID] = append(teacherBuckets[teacher.ID], session)
		}
	}

	// Conectar todas las sesiones de cada profesor
	addedEdges := 0
	for _, sessionGroup := range teacherBuckets {
		addedEdges += connectAllInClique(g, sessionGroup)
	}

	fmt.Printf("  ✅ Agregadas %d aristas por conflictos de profesor\n", addedEdges)
}

// addSameClassConflicts agrega aristas entre sesiones de la misma clase
func addSameClassConflicts(g *ConflictGraph, sessions []*domain.ClassSession) {
	// Agrupar sesiones por clase (mismo ID de Lecture/Tutorial/Lab)
	classBuckets := make(map[int][]*domain.ClassSession)

	for _, session := range sessions {
		classID := session.Class.GetID()
		classBuckets[classID] = append(classBuckets[classID], session)
	}

	// Conectar todas las sesiones de cada clase
	addedEdges := 0
	for _, sessionGroup := range classBuckets {
		addedEdges += connectAllInClique(g, sessionGroup)
	}

	fmt.Printf("  ✅ Agregadas %d aristas por conflictos de misma clase\n", addedEdges)
}

// addRoomScarcityConflicts agrega aristas entre sesiones con escasez crítica de salas
func addRoomScarcityConflicts(g *ConflictGraph, sessions []*domain.ClassSession, university *domain.University) {
	if university.RoomConstraints == nil {
		fmt.Println("  ⚠️  No hay restricciones de salas cargadas")
		return
	}

	fmt.Println("🔍 Detectando incompatibilidades de salas...")

	addedEdges := 0
	totalComparisons := 0

	// Para cada par de sesiones
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			totalComparisons++

			session1 := sessions[i]
			session2 := sessions[j]

			// Verificar si ya tienen una arista
			if g.HasEdge(session1.ID, session2.ID) {
				continue
			}

			// Contar salas compartidas
			sharedRooms := countSharedValidRooms(session1, session2, university)

			// Si comparten EXACTAMENTE 1 sala → Conflicto crítico
			if sharedRooms == 1 {
				g.AddEdge(session1.ID, session2.ID)
				addedEdges++
			}
		}
	}

	fmt.Printf("  ✅ Agregadas %d aristas por escasez de salas (de %d comparaciones)\n", addedEdges, totalComparisons)
}

// addSelectiveCurriculumConflicts agrega restricciones de semestre SOLO para cursos obligatorios
// Excluye electivos (códigos que empiezan con "ELE-") ya que los estudiantes solo toman algunos
func addSelectiveCurriculumConflicts(g *ConflictGraph, sessions []*domain.ClassSession, university *domain.University) {
	fmt.Println("🔍 Analizando restricciones de semestre (excluyendo electivos)...")

	// Agrupar sesiones por semestre
	type SemesterKey struct {
		Major    domain.Major
		Semester int
	}

	semesterSessions := make(map[SemesterKey][]*domain.ClassSession)

	for _, session := range sessions {
		course := session.GetCourse()

		// EXCLUIR ELECTIVOS: Los electivos tienen códigos que empiezan con "ELE-"
		if isElective(course.Code) {
			continue // No agregar electivos a restricciones de semestre
		}

		for _, entry := range course.Curriculum {
			key := SemesterKey{entry.Major, entry.Semester}
			semesterSessions[key] = append(semesterSessions[key], session)
		}
	}

	// Para cada semestre, agregar restricciones solo entre cursos obligatorios
	// FILTROS:
	// 1. Solo semestres ≥4 (semestres 1-3 tienen demasiadas secciones)
	// 2. Solo semestres con promedio ≤3 secciones por curso
	totalEdges := 0
	semestersWithConstraints := 0

	for key, sessionGroup := range semesterSessions {
		if len(sessionGroup) < 2 {
			continue // No hay conflictos si solo hay 1 curso
		}

		// Contar cursos únicos
		uniqueCourses := make(map[int]bool)
		for _, session := range sessionGroup {
			uniqueCourses[session.GetCourse().ID] = true
		}

		// FILTRO 1: Excluir semestres 1-3 (demasiadas secciones)
		if key.Semester <= 3 {
			fmt.Printf("  ⏭️  Saltando %s Sem %d: semestre inicial (muchas secciones)\n",
				key.Major, key.Semester)
			continue
		}

		// FILTRO 2: Calcular promedio de secciones por curso
		// Contar secciones reales por curso
		courseSectionCount := make(map[int]int)
		for _, section := range university.Sections {
			for courseID := range uniqueCourses {
				if section.Course.ID == courseID {
					courseSectionCount[courseID]++
				}
			}
		}

		// Calcular promedio
		totalSections := 0
		for _, count := range courseSectionCount {
			totalSections += count
		}
		avgSections := float64(totalSections) / float64(len(courseSectionCount))

		// Solo aplicar si promedio ≤3 secciones por curso
		if avgSections > 3.0 {
			fmt.Printf("  ⏭️  Saltando %s Sem %d: %.1f sec/curso promedio (demasiadas)\n",
				key.Major, key.Semester, avgSections)
			continue
		}

		// Agregar restricciones entre todos los cursos obligatorios del semestre
		edges := connectAllInClique(g, sessionGroup)
		totalEdges += edges
		semestersWithConstraints++

		fmt.Printf("  📌 %s Sem %d: %d cursos, %.1f sec/curso → %d aristas\n",
			key.Major, key.Semester, len(uniqueCourses), avgSections, edges)
	}

	fmt.Printf("  ✅ Agregadas %d aristas de semestre para %d semestres (sem ≥4, ≤3 sec/curso promedio, sin electivos)\n",
		totalEdges, semestersWithConstraints)
}

// isElective determina si un curso es electivo basándose en su código
// Los electivos tienen códigos que empiezan con "ELE-"
func isElective(courseCode string) bool {
	return len(courseCode) >= 4 && courseCode[:4] == "ELE-"
}

// connectAllInClique conecta todos los pares de sesiones en un grupo
// Retorna el número de aristas agregadas
func connectAllInClique(g *ConflictGraph, sessions []*domain.ClassSession) int {
	if len(sessions) < 2 {
		return 0
	}

	addedEdges := 0
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			g.AddEdge(sessions[i].ID, sessions[j].ID)
			addedEdges++
		}
	}

	return addedEdges
}

// countSharedValidRooms cuenta cuántas salas válidas comparten dos sesiones
func countSharedValidRooms(s1, s2 *domain.ClassSession, university *domain.University) int {
	course1 := s1.GetCourse()
	course2 := s2.GetCourse()

	classType1 := s1.GetType()
	classType2 := s2.GetType()

	// Obtener salas válidas para cada sesión
	validRooms1 := university.RoomConstraints.GetValidRoomsForClass(
		course1.Code,
		classType1,
		university.Rooms,
	)

	validRooms2 := university.RoomConstraints.GetValidRoomsForClass(
		course2.Code,
		classType2,
		university.Rooms,
	)

	// Crear set de IDs de salas para sesión 1
	rooms1Set := make(map[int]bool)
	for _, room := range validRooms1 {
		rooms1Set[room.ID] = true
	}

	// Contar intersección
	sharedCount := 0
	for _, room := range validRooms2 {
		if rooms1Set[room.ID] {
			sharedCount++
		}
	}

	return sharedCount
}

// addSameSectionConflicts agrega aristas entre diferentes clases de la misma sección
// Ejemplo: Cátedra, Ayudantía y Lab de Sección 1 no pueden estar en el mismo slot
func addSameSectionConflicts(g *ConflictGraph, sessions []*domain.ClassSession) {
	// Agrupar sesiones por sección
	sectionBuckets := make(map[int][]*domain.ClassSession)

	for _, session := range sessions {
		// Una sesión puede pertenecer a múltiples secciones (clases compartidas)
		sections := session.GetSections()
		for _, section := range sections {
			sectionBuckets[section.ID] = append(sectionBuckets[section.ID], session)
		}
	}

	// Para cada sección, conectar TODAS sus sesiones
	// Esto incluye: Cátedra (todas sus instancias) + Ayudantía + Lab
	addedEdges := 0
	for _, sessionGroup := range sectionBuckets {
		addedEdges += connectAllInClique(g, sessionGroup)
	}

	fmt.Printf("  ✅ Agregadas %d aristas por conflictos de misma sección\n", addedEdges)
}
