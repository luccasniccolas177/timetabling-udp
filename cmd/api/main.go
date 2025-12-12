package main

import (
	"fmt"
	"log"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/exporter"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/solver"
)

func main() {
	// Cargar actividades según Distribution
	activities, err := loader.LoadActivitiesWithExpansion(
		"data/input/oferta_academica.json",
		"data/input/courses.json",
	)
	if err != nil {
		log.Fatalf("Error cargando actividades: %v", err)
	}

	// Cargar salas desde CSV
	rooms, err := loader.LoadRooms("data/input/rooms.csv")
	if err != nil {
		log.Fatalf("Error cargando salas: %v", err)
	}

	// Cargar profesores desde JSON
	teachers, err := loader.LoadTeachers("data/input/profesores.json")
	if err != nil {
		log.Fatalf("Error cargando profesores: %v", err)
	}

	// Cargar restricciones de salas
	roomConstraints, err := loader.LoadRoomConstraints("data/input/rooms_constraints.json")
	if err != nil {
		log.Fatalf("Error cargando restricciones de salas: %v", err)
	}

	// Cargar PlanLocations para cliques de semestre
	planLocations, err := loader.LoadCoursePlanLocations("data/input/courses.json")
	if err != nil {
		log.Fatalf("Error cargando plan locations: %v", err)
	}

	// Cargar cursos electivos
	electives, err := loader.LoadElectives("data/input/courses.json")
	if err != nil {
		log.Fatalf("Error cargando electivos: %v", err)
	}

	// Construir grafo de conflictos con cliques de semestre (sin electivos)
	conflictGraph := graph.BuildFromActivitiesWithCliques(activities, planLocations, electives)

	// Estadísticas generales
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("           UDP TIMETABLING")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Total de actividades: %d\n", len(activities))
	fmt.Printf("Total de salas:       %d\n", len(rooms))
	fmt.Printf("Total de profesores:  %d\n", len(teachers))
	fmt.Printf("Cursos con restricción de sala: %d\n\n", len(roomConstraints))

	// Contar por tipo de actividad
	counts := map[domain.EventCategory]int{}
	for _, a := range activities {
		counts[a.Type]++
	}
	fmt.Println("Actividades por tipo:")
	fmt.Printf("   CÁTEDRAS:     %d\n", counts[domain.CAT])
	fmt.Printf("   AYUDANTÍAS:   %d\n", counts[domain.AY])
	fmt.Printf("   LABORATORIOS: %d\n", counts[domain.LAB])

	// Estadísticas del grafo
	fmt.Println("\nGrafo de Conflictos:")
	fmt.Printf("   Vértices (actividades): %d\n", conflictGraph.NumVertices())
	fmt.Printf("   Aristas (conflictos):   %d\n", conflictGraph.NumEdges())

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("           EJECUTANDO SCHEDULER CON RESTRICCIONES")
	fmt.Println("═══════════════════════════════════════════════════════════")

	result := solver.IntegratedSchedulerWithConstraints(activities, conflictGraph, rooms, roomConstraints)

	fmt.Printf("\nResultado del Scheduling:\n")
	fmt.Printf("   Periodos utilizados:     %d\n", result.TotalPeriods)
	fmt.Printf("   Bloques disponibles:     %d\n", domain.TotalBlocks)

	// Contar actividades programadas
	totalScheduled := 0
	for _, p := range result.Periods {
		for _, ra := range p.Assignments {
			totalScheduled += len(ra.Activities)
		}
	}

	fmt.Printf("   Actividades programadas: %d/%d\n", totalScheduled, len(activities))
	fmt.Printf("   Sin programar (DUD):     %d\n", len(result.FinalDUD))

	if len(result.FinalDUD) == 0 {
		fmt.Println("   ÉXITO: Todas las actividades programadas")
	} else if result.TotalPeriods > domain.TotalBlocks {
		fmt.Println("   INFACTIBLE: Se excedieron los bloques disponibles")
	} else {
		fmt.Printf("   PARCIAL: %d actividades sin sala\n", len(result.FinalDUD))
	}

	// Mostrar distribución por periodo
	fmt.Println("\nDistribución por periodo:")
	fmt.Println("   Periodo | Bloque | Programadas | Salas Usadas")
	fmt.Println("   --------|--------|-------------|-------------")

	limit := 10
	if len(result.Periods) < limit {
		limit = len(result.Periods)
	}
	for i := 0; i < limit; i++ {
		p := result.Periods[i]
		count := 0
		for _, ra := range p.Assignments {
			count += len(ra.Activities)
		}
		fmt.Printf("   %7d | %6d | %11d | %d\n", p.Number, p.Block, count, len(p.Assignments))
	}
	if len(result.Periods) > limit {
		fmt.Printf("   ... y %d periodos más\n", len(result.Periods)-limit)
	}

	// Estadísticas de uso de salas
	roomUsage := make(map[string]int)
	for _, p := range result.Periods {
		for _, ra := range p.Assignments {
			roomUsage[ra.RoomCode]++
		}
	}
	fmt.Printf("\nSalas únicas utilizadas: %d de %d\n", len(roomUsage), len(rooms))

	// Mostrar ejemplos de asignación del primer periodo
	if len(result.Periods) > 0 {
		p := result.Periods[0]
		fmt.Println("\n   Ejemplo (Periodo 0):")
		shown := 0
		for _, ra := range p.Assignments {
			if shown >= 5 {
				break
			}
			for _, a := range ra.Activities {
				fmt.Printf("   - %-25s → Sala: %-12s (%d est.)\n", a.Code, a.Room, a.Students)
				shown++
				if shown >= 5 {
					break
				}
			}
		}
	}

	// Mostrar todas las actividades sin sala
	if len(result.FinalDUD) > 0 {
		fmt.Printf("\n  TODAS las actividades sin sala (%d):\n", len(result.FinalDUD))
		for _, a := range result.FinalDUD {
			fmt.Printf("   - %-30s | %-10s | Curso: %-25s | %d est.\n", a.Code, a.Type, a.CourseName, a.Students)
		}
	}
	if len(result.FinalDUD) == 0 {
		fmt.Println("\n═══════════════════════════════════════════════════════════")
		fmt.Println("           SIMULATED ANNEALING - OPTIMIZACIÓN")
		fmt.Println("═══════════════════════════════════════════════════════════")

		// Cargar prerrequisitos
		prerequisites, err := loader.LoadPrerequisites("data/input/courses.json")
		if err != nil {
			log.Fatalf("Error cargando prerrequisitos: %v", err)
		}

		config := solver.DefaultSAConfig()
		fmt.Printf("\n  Parámetros SA:\n")
		fmt.Printf("   Temp. inicial:  %.0f\n", config.InitialTemp)
		fmt.Printf("   Tasa enfriamiento: %.4f\n", config.CoolingRate)
		fmt.Printf("   Iteraciones/T: %d\n", config.IterationsPerT)

		fmt.Println("\n Ejecutando optimización (bloques + salas)...")
		saResult := solver.SimulatedAnnealing(activities, rooms, config, prerequisites, planLocations, electives, roomConstraints)

		fmt.Printf("\n Resultado SA:\n")
		fmt.Printf("   Costo inicial:      %.0f\n", saResult.InitialCost)
		fmt.Printf("   Costo final:        %.0f\n", saResult.FinalCost)
		fmt.Printf("   Mejora:             %.1f%%\n", (1-saResult.FinalCost/saResult.InitialCost)*100)
		fmt.Printf("   Iteraciones:        %d\n", saResult.Iterations)
		fmt.Printf("   Mejoras aceptadas:  %d\n", saResult.Improvements)
		fmt.Printf("\n Métricas de calidad:\n")
		fmt.Printf("   Penalidad espejo:   %.0f\n", saResult.MirrorPenalty)
		fmt.Printf("   AY en miércoles:    %.1f%%\n", saResult.WednesdayBonus)
		fmt.Printf("   Prereq en mismo bloque: %.1f%%\n", saResult.PrereqBonus)
		fmt.Printf("   Hermanos misma sala: %.1f%%\n", saResult.RoomConsistency)
		fmt.Printf("   Sep. ideal (3 días): %.1f%%\n", saResult.DaySeparation)

		// Exportar a JSON
		outputFile := "data/output/schedule.json"
		if err := exporter.ExportScheduleToJSON(activities, outputFile); err != nil {
			fmt.Printf("\n Error exportando JSON: %v\n", err)
		} else {
			fmt.Printf("\n Horario exportado a: %s\n", outputFile)
		}
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════")
}
