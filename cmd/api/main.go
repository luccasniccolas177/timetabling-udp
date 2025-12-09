package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/solver"
)

func main() {
	// 1. CARGA DE DATOS
	fmt.Println("⏳ [PASO 1] Cargando datos de la universidad...")
	state, err := loader.LoadFicData("data/input")
	if err != nil {
		log.Fatalf("❌ Error fatal cargando datos: %v", err)
	}
	fmt.Printf("✅ Datos cargados. Eventos Lógicos: %d\n", len(state.RawEvents))

	// 2. CONSTRUCCIÓN DEL GRAFO (Modelado)
	fmt.Println("\n🔗 [PASO 2] Construyendo Grafo de Conflictos (Nodos y Aristas)...")
	conflictGraph := graph.BuildConflictGraph(state)

	// Mostrar estadísticas del grafo para entender la complejidad
	conflictGraph.PrintStats()

	// 3. EJECUCIÓN DEL SOLVER (Coloreo)
	fmt.Println("\n🎨 [PASO 3] Ejecutando Algoritmo de Coloreo (Heurística: Largest Degree)...")
	solution := solver.ColorGraph(conflictGraph)

	// 4. REPORTE DE RESULTADOS
	printSolutionReport(solution)
}

func printSolutionReport(sol *solver.Solution) {
	fmt.Println("\n================================================================================")
	fmt.Println("📊 REPORTE FINAL DE TIMETABLING (SOLO TIEMPO)")
	fmt.Println("================================================================================")

	// Validar si el horario cabe en la semana (35 bloques aprox en la UDP)
	limit := 35
	status := "✅ FACTIBLE"
	if sol.TotalColors > limit {
		status = "❌ INFACTIBLE (Se necesitan más bloques que los disponibles en la semana)"
	}

	fmt.Printf("Bloques Temporales Necesarios (Colores): %d\n", sol.TotalColors)
	fmt.Printf("Estado del Horario: %s\n", status)
	fmt.Println("--------------------------------------------------------------------------------")

	// Imprimir detalle de los primeros 5 bloques para verificar
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Bloque (Color)\t| Cantidad Eventos\t| Ejemplos de Eventos Asignados")
	fmt.Fprintln(w, "--------------\t| ----------------\t| -----------------------------")

	// Ordenar los colores para imprimir en orden
	var keys []int
	for k := range sol.Schedule {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		events := sol.Schedule[k]

		// Generar un string de ejemplo con los primeros 2-3 eventos del bloque
		exampleStr := ""
		count := 0
		for _, e := range events {
			// Recuperar nombre del curso desde el Data original
			// Nota: e.Data es *models.LogicalEvent. Para el nombre necesitamos buscar en state,
			// pero aquí solo tenemos el evento. Usaremos el UUID como referencia rápida.
			exampleStr += fmt.Sprintf("[%s] ", e.UUID)
			count++
			if count >= 2 {
				exampleStr += "..."
				break
			}
		}

		fmt.Fprintf(w, "Bloque %d\t| %d eventos\t| %s\n", k, len(events), exampleStr)
	}
	w.Flush()
	fmt.Println("================================================================================")
}
