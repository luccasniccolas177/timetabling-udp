package solver

import (
	"fmt"
	"timetabling-UDP/internal/domain"
)

// SemesterIssue representa un problema de balance de secciones en un semestre
type SemesterIssue struct {
	Major         domain.Major
	Semester      int
	Courses       []*domain.Course
	TotalSections int
	Message       string
}

// SemesterKey identifica un semestre único
type SemesterKey struct {
	Major    domain.Major
	Semester int
}

// ValidateSectionBalance verifica que cada semestre tenga al menos una combinación factible de secciones
// Para cada semestre, verifica que existe al menos una combinación de secciones donde un estudiante
// puede tomar todos los cursos requeridos sin conflictos de horario
func ValidateSectionBalance(solution *Solution, university *domain.University) []SemesterIssue {
	issues := []SemesterIssue{}

	// Agrupar cursos por (Major, Semester)
	semesterCourses := groupCoursesBySemester(university)

	fmt.Printf("  📊 Analizando %d semestres...\n", len(semesterCourses))

	// Para cada semestre, verificar balance
	for key, courses := range semesterCourses {
		if len(courses) < 2 {
			continue // No hay problema si solo hay 1 curso
		}

		// Optimización: Limitar validación a semestres con ≤8 cursos
		if len(courses) > 8 {
			fmt.Printf("  ⚠️  Saltando %s Semestre %d (%d cursos - demasiados para validar)\n",
				key.Major, key.Semester, len(courses))
			continue
		}

		// Verificar si existe combinación factible
		if !hasFeasibleCombination(courses, solution, university) {
			issue := SemesterIssue{
				Major:         key.Major,
				Semester:      key.Semester,
				Courses:       courses,
				TotalSections: countTotalSections(courses, university),
				Message: fmt.Sprintf("No existe combinación de secciones sin conflictos para %d cursos",
					len(courses)),
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// groupCoursesBySemester agrupa cursos por (Major, Semester)
func groupCoursesBySemester(university *domain.University) map[SemesterKey][]*domain.Course {
	result := make(map[SemesterKey][]*domain.Course)

	for _, course := range university.Courses {
		// Un curso puede pertenecer a múltiples semestres/carreras
		for _, entry := range course.Curriculum {
			key := SemesterKey{
				Major:    entry.Major,
				Semester: entry.Semester,
			}

			// Evitar duplicados
			found := false
			for _, c := range result[key] {
				if c.ID == course.ID {
					found = true
					break
				}
			}

			if !found {
				result[key] = append(result[key], course)
			}
		}
	}

	return result
}

// hasFeasibleCombination verifica si existe al menos una combinación de secciones sin conflictos
func hasFeasibleCombination(courses []*domain.Course, solution *Solution, university *domain.University) bool {
	// Obtener todas las secciones de cada curso
	courseSections := make([][]*domain.Section, len(courses))

	for i, course := range courses {
		sections := getSectionsForCourse(course.ID, university)
		if len(sections) == 0 {
			// Si un curso no tiene secciones, no hay problema
			return true
		}
		courseSections[i] = sections
	}

	// Usar backtracking para encontrar combinación factible
	return findFeasibleCombination(courseSections, 0, []*domain.Section{}, solution)
}

// getSectionsForCourse obtiene todas las secciones de un curso
func getSectionsForCourse(courseID int, university *domain.University) []*domain.Section {
	sections := []*domain.Section{}

	for _, section := range university.Sections {
		if section.Course.ID == courseID {
			sections = append(sections, section)
		}
	}

	return sections
}

// findFeasibleCombination usa backtracking para encontrar una combinación sin conflictos
func findFeasibleCombination(courseSections [][]*domain.Section, index int, current []*domain.Section, solution *Solution) bool {
	// Caso base: hemos seleccionado una sección de cada curso
	if index == len(courseSections) {
		// Verificar si la combinación actual no tiene conflictos
		return !hasConflicts(current, solution)
	}

	// Probar cada sección del curso actual
	for _, section := range courseSections[index] {
		// Agregar sección a la combinación actual
		current = append(current, section)

		// Optimización: Poda temprana - verificar conflictos antes de continuar
		if !hasConflicts(current, solution) {
			// Recursión: intentar completar la combinación
			if findFeasibleCombination(courseSections, index+1, current, solution) {
				return true // Encontramos una combinación factible
			}
		}

		// Backtrack: quitar la sección
		current = current[:len(current)-1]
	}

	return false // No se encontró combinación factible
}

// hasConflicts verifica si un conjunto de secciones tiene conflictos de horario
func hasConflicts(sections []*domain.Section, solution *Solution) bool {
	// Obtener todas las sesiones de estas secciones
	sessions := getSessionsForSections(sections, solution)

	// Verificar si algún par de sesiones está en el mismo slot
	// IMPORTANTE: Solo verificar conflictos entre sesiones de DIFERENTES secciones
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			// Verificar si están en el mismo slot
			if sessions[i].AssignedSlot == sessions[j].AssignedSlot {
				// Verificar si son de diferentes secciones
				if !areSameSection(sessions[i], sessions[j], sections) {
					return true // Conflicto real: diferentes secciones, mismo slot
				}
			}
		}
	}

	return false // No hay conflictos
}

// areSameSection verifica si dos sesiones pertenecen a la misma sección
func areSameSection(s1, s2 *domain.ClassSession, sections []*domain.Section) bool {
	// Obtener las secciones de cada sesión
	s1Sections := s1.GetSections()
	s2Sections := s2.GetSections()

	// Dos sesiones son de la misma sección si comparten al menos una sección
	for _, sec1 := range s1Sections {
		for _, sec2 := range s2Sections {
			if sec1.ID == sec2.ID {
				return true
			}
		}
	}

	return false
}

// getSessionsForSections obtiene todas las sesiones asignadas a un conjunto de secciones
func getSessionsForSections(sections []*domain.Section, solution *Solution) []*domain.ClassSession {
	sessions := []*domain.ClassSession{}

	// Para cada sección, obtener todas sus clases y luego todas sus sesiones
	for _, section := range sections {
		// Obtener todas las clases de la sección
		classes := section.GetAllClasses()

		// Para cada clase, buscar sus sesiones en la solución
		for _, class := range classes {
			// Buscar sesiones de esta clase en todos los bloques
			for _, slotSessions := range solution.Schedule {
				for _, session := range slotSessions {
					// Verificar si esta sesión pertenece a la clase
					if session.Class.GetID() == class.GetID() {
						sessions = append(sessions, session)
					}
				}
			}
		}
	}

	return sessions
}

// countTotalSections cuenta el total de secciones en un conjunto de cursos
func countTotalSections(courses []*domain.Course, university *domain.University) int {
	total := 0
	for _, course := range courses {
		sections := getSectionsForCourse(course.ID, university)
		total += len(sections)
	}
	return total
}

// PrintSectionBalanceReport imprime un reporte de problemas de balance
func PrintSectionBalanceReport(issues []SemesterIssue) {
	if len(issues) == 0 {
		fmt.Println("✅ Todos los semestres tienen balance de secciones correcto")
		return
	}

	fmt.Printf("\n⚠️  ADVERTENCIA: %d semestres con problemas de balance de secciones\n", len(issues))
	fmt.Println("================================================================================")

	for i, issue := range issues {
		fmt.Printf("\n%d. %s - Semestre %d\n", i+1, issue.Major, issue.Semester)
		fmt.Printf("   Cursos afectados: %d\n", len(issue.Courses))
		fmt.Printf("   Total de secciones: %d\n", issue.TotalSections)
		fmt.Printf("   Problema: %s\n", issue.Message)

		fmt.Printf("   Cursos:\n")
		for _, course := range issue.Courses {
			fmt.Printf("     - %s (%s)\n", course.Code, course.Name)
		}
	}

	fmt.Println("================================================================================")
	fmt.Println("💡 Sugerencia: Algunos cursos necesitan re-coloración para permitir combinaciones factibles")
}
