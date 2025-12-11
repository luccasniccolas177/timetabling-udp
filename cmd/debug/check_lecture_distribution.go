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

	// Construir grafo
	g := graph.BuildConflictGraph(university)

	// Colorear
	solution := solver.ColorGraph(g)

	// Analizar distribución de instancias de cátedras
	fmt.Println("🔍 Analizando distribución de instancias de cátedras...")
	fmt.Println()

	// Agrupar sesiones por cátedra
	lectureInstances := make(map[int]map[int][]*domain.ClassSession)

	for slot, sessions := range solution.Schedule {
		for _, session := range sessions {
			if session.GetType() == domain.ClassTypeLecture {
				classID := session.Class.GetID()
				if lectureInstances[classID] == nil {
					lectureInstances[classID] = make(map[int][]*domain.ClassSession)
				}
				lectureInstances[classID][int(slot)] = append(lectureInstances[classID][int(slot)], session)
			}
		}
	}

	// Verificar si todas las instancias están en el mismo bloque
	sameSlot := 0
	differentSlots := 0
	examples := 0

	for classID, slots := range lectureInstances {
		if len(slots) > 1 {
			differentSlots++
			if examples < 5 {
				fmt.Printf("❌ Cátedra ID %d tiene instancias en %d bloques diferentes:\n", classID, len(slots))
				for slot, sessions := range slots {
					fmt.Printf("   Bloque %d: %d sesiones\n", slot, len(sessions))
				}
				fmt.Println()
				examples++
			}
		} else {
			sameSlot++
		}
	}

	fmt.Println("📊 RESUMEN:")
	fmt.Printf("✅ Cátedras con todas las instancias en el mismo bloque: %d\n", sameSlot)
	fmt.Printf("❌ Cátedras con instancias en bloques diferentes: %d\n", differentSlots)
	fmt.Println()

	if differentSlots > 0 {
		fmt.Println("⚠️  PROBLEMA: Algunas cátedras tienen instancias en bloques diferentes")
		fmt.Println("   Esto impide asignar la misma sala a todas las instancias")
		fmt.Println("   SOLUCIÓN: Agregar restricción para forzar mismo bloque")
	} else {
		fmt.Println("✅ PERFECTO: Todas las cátedras tienen instancias en el mismo bloque")
		fmt.Println("   Se puede proceder con la asignación de salas")
	}
}
