package graph

import (
	"fmt"
	"timetabling-UDP/internal/domain"
)

// addSingleLectureConflicts agrega aristas entre clases únicas del mismo semestre
// Si un semestre tiene cursos con una sola clase (cátedra, ayudantía o lab),
// todos los estudiantes deben tomarla, por lo tanto no pueden solaparse
func addSingleLectureConflicts(g *ConflictGraph, sessions []*domain.ClassSession, university *domain.University) {
	fmt.Println("🔍 Detectando clases únicas por semestre...")

	// Agrupar por semestre
	type SemesterKey struct {
		Major    domain.Major
		Semester int
	}

	// Contar cuántas clases de cada tipo tiene cada curso
	courseLectureCount := make(map[int]int)
	courseTutorialCount := make(map[int]int)
	courseLabCount := make(map[int]int)

	for _, lecture := range university.Lectures {
		courseLectureCount[lecture.Course.ID]++
	}
	for _, tutorial := range university.Tutorials {
		courseTutorialCount[tutorial.Course.ID]++
	}
	for _, lab := range university.Labs {
		courseLabCount[lab.Course.ID]++
	}

	// Agrupar sesiones de clases únicas por semestre
	semesterSingleClasses := make(map[SemesterKey][]*domain.ClassSession)

	for _, session := range sessions {
		course := session.GetCourse()
		classType := session.GetType()

		// Verificar si este curso tiene solo 1 clase de este tipo
		isUnique := false
		switch classType {
		case domain.ClassTypeLecture:
			isUnique = courseLectureCount[course.ID] == 1
		case domain.ClassTypeTutorial:
			isUnique = courseTutorialCount[course.ID] == 1
		case domain.ClassTypeLab:
			isUnique = courseLabCount[course.ID] == 1
		}

		if !isUnique {
			continue // Este curso tiene múltiples clases de este tipo
		}

		// Agrupar por semestre
		for _, entry := range course.Curriculum {
			// Excluir electivos
			if isElective(course.Code) {
				continue
			}

			key := SemesterKey{entry.Major, entry.Semester}
			semesterSingleClasses[key] = append(semesterSingleClasses[key], session)
		}
	}

	// Para cada semestre, conectar todas las clases únicas
	// FILTRO: Solo aplicar a semestres con ≤6 cursos (evitar explosión en sem 9-10)
	totalEdges := 0
	semestersWithConstraints := 0

	for key, sessionGroup := range semesterSingleClasses {
		if len(sessionGroup) < 2 {
			continue // No hay conflictos si solo hay 1 clase
		}

		// Contar cursos únicos
		uniqueCourses := make(map[int]bool)
		for _, session := range sessionGroup {
			uniqueCourses[session.GetCourse().ID] = true
		}

		// FILTRO: Solo aplicar a semestres con ≤6 cursos con clases únicas
		// Semestres 9-10 tienen muchos electivos con clases únicas
		if len(uniqueCourses) > 6 {
			fmt.Printf("  ⏭️  Saltando %s Sem %d: %d cursos (demasiados)\n",
				key.Major, key.Semester, len(uniqueCourses))
			continue
		}

		// Conectar todas las sesiones de clases únicas
		edges := connectAllInClique(g, sessionGroup)
		totalEdges += edges
		semestersWithConstraints++

		fmt.Printf("  📌 %s Sem %d: %d cursos con clase única → %d aristas\n",
			key.Major, key.Semester, len(uniqueCourses), edges)
	}

	fmt.Printf("  ✅ Agregadas %d aristas por clases únicas en %d semestres (≤6 cursos)\n",
		totalEdges, semestersWithConstraints)
}
