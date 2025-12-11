package main

import (
	"fmt"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/models"
	"timetabling-UDP/internal/solver"
)

func main() {
	// Cargar datos
	state, err := loader.LoadFicData("data/input")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Analizar Semestre 1 de CIT
	fmt.Println("🔍 Análisis Detallado: CIT Semestre 1")
	fmt.Println("================================================================================")

	// Obtener cursos del semestre 1
	var semester1Courses []models.Course
	for _, course := range state.Courses {
		for _, req := range course.Requirements {
			if req.Major == models.CIT && req.Semester == 1 {
				semester1Courses = append(semester1Courses, course)
				break
			}
		}
	}

	fmt.Printf("\n📚 Cursos del Semestre 1 (CIT): %d\n", len(semester1Courses))
	for i, course := range semester1Courses {
		sections := getSectionsForCourse(course.ID, state)
		fmt.Printf("%d. %s (%s) - %d secciones\n", i+1, course.Code, course.Name, len(sections))
	}

	// Crear una solución de prueba simple
	fmt.Println("\n🎨 Creando solución de prueba...")
	testSolution := createTestSolution(semester1Courses, state)

	// Analizar eventos por curso
	fmt.Println("\n📊 Eventos por curso en la solución:")
	for _, course := range semester1Courses {
		events := getEventsForCourse(course.ID, testSolution, state)
		fmt.Printf("\n%s (%d eventos):\n", course.Code, len(events))

		// Agrupar por slot
		slotGroups := make(map[models.TimeSlot]int)
		for _, event := range events {
			slotGroups[event.AssignedSlot]++
		}

		fmt.Printf("  Distribución por slot: %d slots diferentes\n", len(slotGroups))
		for slot, count := range slotGroups {
			if count > 0 {
				fmt.Printf("    Slot %d: %d eventos\n", slot, count)
			}
		}
	}

	// Intentar encontrar combinación factible
	fmt.Println("\n🔍 Buscando combinación factible...")
	sections := make([][]models.Section, len(semester1Courses))
	for i, course := range semester1Courses {
		sections[i] = getSectionsForCourse(course.ID, state)
	}

	found := findAndPrintFeasibleCombination(sections, testSolution, state)
	if found {
		fmt.Println("✅ Se encontró combinación factible")
	} else {
		fmt.Println("❌ NO se encontró combinación factible")
	}
}

func getSectionsForCourse(courseID int, state *loader.UniversityState) []models.Section {
	sections := []models.Section{}
	for _, section := range state.Sections {
		if section.CourseID == courseID {
			sections = append(sections, section)
		}
	}
	return sections
}

func getEventsForCourse(courseID int, solution *solver.Solution, state *loader.UniversityState) []*models.EventInstance {
	events := []*models.EventInstance{}
	for _, slotEvents := range solution.Schedule {
		for _, event := range slotEvents {
			if event.Data.CourseID == courseID {
				events = append(events, event)
			}
		}
	}
	return events
}

func createTestSolution(courses []models.Course, state *loader.UniversityState) *solver.Solution {
	// Crear una solución simple para pruebas
	// En realidad deberíamos cargar la solución real del coloreo
	solution := solver.NewSolution()

	// Por ahora, solo crear estructura vacía
	// La solución real vendría del ColorGraph
	return solution
}

func findAndPrintFeasibleCombination(courseSections [][]models.Section, solution *solver.Solution, state *loader.UniversityState) bool {
	attempts := 0
	maxAttempts := 1000

	var findCombination func(int, []models.Section) bool
	findCombination = func(index int, current []models.Section) bool {
		attempts++
		if attempts > maxAttempts {
			fmt.Printf("⚠️  Límite de intentos alcanzado (%d)\n", maxAttempts)
			return false
		}

		if index == len(courseSections) {
			// Verificar conflictos
			if !hasConflicts(current, solution, state) {
				fmt.Println("\n✅ Combinación factible encontrada:")
				for i, section := range current {
					fmt.Printf("  Curso %d: Sección %d\n", i+1, section.SectionNumber)
				}
				return true
			}
			return false
		}

		for _, section := range courseSections[index] {
			current = append(current, section)
			if !hasConflicts(current, solution, state) {
				if findCombination(index+1, current) {
					return true
				}
			}
			current = current[:len(current)-1]
		}

		return false
	}

	result := findCombination(0, []models.Section{})
	fmt.Printf("\n📊 Total de intentos: %d\n", attempts)
	return result
}

func hasConflicts(sections []models.Section, solution *solver.Solution, state *loader.UniversityState) bool {
	events := getEventsForSections(sections, solution, state)

	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i].AssignedSlot == events[j].AssignedSlot {
				return true
			}
		}
	}

	return false
}

func getEventsForSections(sections []models.Section, solution *solver.Solution, state *loader.UniversityState) []*models.EventInstance {
	events := []*models.EventInstance{}

	for _, section := range sections {
		for _, slotEvents := range solution.Schedule {
			for _, event := range slotEvents {
				for _, parentSectionID := range event.Data.ParentSectionIDs {
					if parentSectionID == section.ID {
						events = append(events, event)
						break
					}
				}
			}
		}
	}

	return events
}
