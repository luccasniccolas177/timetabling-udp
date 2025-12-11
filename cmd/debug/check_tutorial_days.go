package main

import (
	"fmt"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/solver"
)

func main() {
	// Cargar datos
	university, err := loader.LoadUniversity("data/input")
	if err != nil {
		panic(err)
	}

	// Construir grafo y colorear
	g := graph.BuildConflictGraph(university)
	solution := solver.ColorGraph(g)

	// Analizar ayudantías por día
	fmt.Println("🔍 Analizando distribución de ayudantías por día...")
	fmt.Println()

	tutorialsByDay := make(map[string]int)
	totalTutorials := 0

	for _, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeTutorial {
				totalTutorials++
				slot := int(session.AssignedSlot)

				var day string
				if slot >= 0 && slot <= 6 {
					day = "Lunes"
				} else if slot >= 7 && slot <= 13 {
					day = "Martes"
				} else if slot >= 14 && slot <= 20 {
					day = "Miércoles"
				} else if slot >= 21 && slot <= 27 {
					day = "Jueves"
				} else if slot >= 28 && slot <= 34 {
					day = "Viernes"
				} else {
					day = "Fuera de rango"
				}

				tutorialsByDay[day]++
			}
		}
	}

	fmt.Println("📊 Distribución de ayudantías por día:")
	for day, count := range tutorialsByDay {
		percentage := float64(count) / float64(totalTutorials) * 100
		fmt.Printf("  %s: %d (%.1f%%)\n", day, count, percentage)
	}

	fmt.Printf("\nTotal ayudantías: %d\n", totalTutorials)
}
