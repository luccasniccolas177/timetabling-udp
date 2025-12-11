package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/solver"
)

func main() {
	// 1. CARGA DE DATOS
	fmt.Println("⏳ [PASO 1] Cargando datos de la universidad...")
	university, err := loader.LoadUniversity("data/input")
	if err != nil {
		log.Fatalf("❌ Error fatal cargando datos: %v", err)
	}

	// Mostrar estadísticas de carga
	printLoadStats(university)

	// 2. CONSTRUCCIÓN DEL GRAFO
	fmt.Println("\n🔗 [PASO 2] Construyendo Grafo de Conflictos...")
	conflictGraph := graph.BuildConflictGraph(university)
	conflictGraph.PrintStats()

	// 3. VALIDACIÓN DE RESTRICCIONES DE SALAS
	fmt.Println("\n🏢 [VALIDACIÓN] Ejemplo de Restricciones de Salas:")
	printRoomConstraintsExample(university)

	// 🎨 [PASO 3] Ejecutar Algoritmo de Coloreo
	fmt.Println("\n🎨 [PASO 3] Ejecutando Algoritmo de Coloreo (Greedy)...")
	solution := solver.ColorGraph(conflictGraph)

	// 🔥 [PASO 3.1] Optimización con Simulated Annealing
	fmt.Println("\n🔥 [PASO 3.1] Optimizando Horario con Simulated Annealing...")
	saConfig := solver.SAConfig{
		InitialTemp: 5000.0,
		CoolingRate: 0.995, // Enfriamiento lento para evitar mínimos locales
		Iterations:  50000,
		MaxColors:   35,
	}
	solution = solver.OptimizeSchedule(solution, conflictGraph, saConfig)

	// 🏢 [PASO 3.5] Asignar Salas Físicas con manejo de DUD list
	// Implementa Burke et al. sección 2.5: iteración con re-coloreo
	// Y Sección 3.1: Algoritmo de desplazamiento
	fmt.Println("\n🏢 [PASO 3.5] Asignando Salas Físicas (Burke Algo 3.1 + Re-coloreo)...")

	maxIterations := 10
	iteration := 0
	var dudList []*domain.ClassSession

	// Usamos 35 bloques como máximo (semana completa)
	maxBlocks := 35

	for iteration < maxIterations {
		// Usar el nuevo algoritmo de desplazamiento (Burke 3.1)
		dudList = solver.AssignRoomsBurke(solution, university)

		totalSessions := solution.GetTotalSessions()
		assignedCount := totalSessions - len(dudList)

		fmt.Printf("   👉 Iteración %d: %d/%d asignadas (%.1f%%) | %d DUDs\n",
			iteration, assignedCount, totalSessions,
			float64(assignedCount)/float64(totalSessions)*100, len(dudList))

		if len(dudList) == 0 {
			fmt.Println("✅ Todas las sesiones tienen sala asignada")
			break
		}

		iteration++
		fmt.Printf("\n🔄 Iteración %d: %d sesiones en DUD list, intentando re-coloreo...\n",
			iteration, len(dudList))

		// Re-coloreo inteligente
		recolored := solver.RecolorDUDs(solution, conflictGraph, dudList, maxBlocks)

		if recolored == 0 {
			fmt.Println("⚠️  No se pudieron re-colorear más sesiones. Deteniendo iteraciones.")
			break
		}
	}

	if len(dudList) > 0 {
		fmt.Printf("⚠️  Resultado Final: %d sesiones quedaron sin sala.\n", len(dudList))
	} else {
		fmt.Println("🎉 ¡Éxito Completo! Todas las sesiones asignadas.")
	}

	// 🔍 [PASO 4] Validar Balance de Secciones
	fmt.Println("\n🔍 [PASO 4] Validando Balance de Secciones...")
	balanceIssues := solver.ValidateSectionBalance(solution, university)
	solver.PrintSectionBalanceReport(balanceIssues)

	// 6. REPORTE DE RESULTADOS
	printSolutionReport(solution)

	// 7. EXPORTAR JSON DETALLADO
	fmt.Println("\n💾 [PASO FINAL] Exportando reporte detallado a 'horario_detalle.json'...")
	if err := exportScheduleJSON(solution, "horario_detalle.json"); err != nil {
		fmt.Printf("❌ Error exportando JSON: %v\n", err)
	} else {
		fmt.Println("✅ Reporte guardado exitosamente.")
	}
}

// exportScheduleJSON genera un archivo JSON con el detalle de asignaciones agrupado por curso y sección
func exportScheduleJSON(sol *solver.Solution, filename string) error {
	type TimeSlotDetail struct {
		Day       string `json:"day"`
		Block     int    `json:"block_number"`
		TimeRange string `json:"time_range"`
	}

	type ClassEvent struct {
		Type         string         `json:"type"` // Cátedra, Lab, Ayudantía
		Room         string         `json:"room"`
		RoomCapacity int            `json:"room_capacity"`
		Time         TimeSlotDetail `json:"time"`
	}

	type SectionSchedule struct {
		SectionNumber int          `json:"section_number"`
		Events        []ClassEvent `json:"events"`
	}

	type CourseSchedule struct {
		CourseCode string            `json:"course_code"`
		CourseName string            `json:"course_name"`
		Sections   []SectionSchedule `json:"sections"`
	}

	type FullReport struct {
		Courses []CourseSchedule `json:"courses"`
		Stats   struct {
			TotalCourses  int `json:"total_courses"`
			TotalSections int `json:"total_sections"`
		} `json:"stats"`
	}

	// Mapa auxiliar para agrupar: CourseCode -> SectionNumber -> Events
	// Usamos un mapa anidado para facilitar la agrupación
	groupedData := make(map[string]map[int][]ClassEvent)

	// Mapa para recuperar info extra del curso (Nombre)
	courseInfo := make(map[string]string)

	// Iterar sobre todos los bloques del horario (1 a 35)
	for blockID, sessions := range sol.Schedule {
		day, timeRange := getBlockTimeInfo(blockID)

		for _, session := range sessions {
			if session.AssignedRoom == nil {
				continue // Ignorar sesiones sin sala asignada (DUDs)
			}

			course := session.Class.GetCourse()
			courseCode := course.Code
			courseInfo[courseCode] = course.Name

			// Construir el evento base
			eventType := ""
			switch session.GetType() {
			case domain.ClassTypeLecture:
				eventType = "Cátedra"
			case domain.ClassTypeTutorial:
				eventType = "Ayudantía"
			case domain.ClassTypeLab:
				eventType = "Laboratorio"
			}

			event := ClassEvent{
				Type:         eventType,
				Room:         session.AssignedRoom.Code,
				RoomCapacity: session.AssignedRoom.Capacity,
				Time: TimeSlotDetail{
					Day:       day,
					Block:     blockID,
					TimeRange: timeRange,
				},
			}

			// Una sesión puede pertenecer a MULTIPLES secciones (ej. Cátedra compartida)
			// Debemos agregar este evento a cada una de las secciones correspondientes
			sections := session.Class.GetSections()
			for _, sec := range sections {
				sectionNum := sec.Number

				if _, ok := groupedData[courseCode]; !ok {
					groupedData[courseCode] = make(map[int][]ClassEvent)
				}
				groupedData[courseCode][sectionNum] = append(groupedData[courseCode][sectionNum], event)
			}
		}
	}

	// Convertir el mapa a la estructura final ordenada
	var report FullReport
	var sectionCount int

	// Obtener códigos de curso y ordenar
	var courseCodes []string
	for code := range groupedData {
		courseCodes = append(courseCodes, code)
	}
	sort.Strings(courseCodes)

	for _, code := range courseCodes {
		sectionsMap := groupedData[code]

		var sectionSchedules []SectionSchedule

		// Obtener números de sección y ordenar
		var sectionNums []int
		for num := range sectionsMap {
			sectionNums = append(sectionNums, num)
		}
		sort.Ints(sectionNums)

		for _, num := range sectionNums {
			events := sectionsMap[num]
			// Ordenar eventos por bloque
			sort.Slice(events, func(i, j int) bool {
				return events[i].Time.Block < events[j].Time.Block
			})

			sectionSchedules = append(sectionSchedules, SectionSchedule{
				SectionNumber: num,
				Events:        events,
			})
			sectionCount++
		}

		report.Courses = append(report.Courses, CourseSchedule{
			CourseCode: code,
			CourseName: courseInfo[code],
			Sections:   sectionSchedules,
		})
	}

	report.Stats.TotalCourses = len(report.Courses)
	report.Stats.TotalSections = sectionCount

	// Escribir archivo
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// getBlockTimeInfo retorna el día y rango horario aproximado para un bloque dado (1-35)
func getBlockTimeInfo(blockID int) (string, string) {
	// Asumimos 7 bloques por día
	/*
		Día 1 (Lun): 1-7
		Día 2 (Mar): 8-14
		Día 3 (Mie): 15-21
		Día 4 (Jue): 22-28
		Día 5 (Vie): 29-35
	*/

	dayIndex := (blockID - 1) / 7
	blockIndex := (blockID - 1) % 7 // 0 a 6

	days := []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
	dayName := "Desconocido"
	if dayIndex >= 0 && dayIndex < len(days) {
		dayName = days[dayIndex]
	}

	// Horarios UDP Actualizados
	times := []string{
		"08:30 - 09:50", // 1
		"10:00 - 11:20", // 2
		"11:30 - 12:50", // 3
		"13:00 - 14:20", // 4 (Almuerzo/Clase)
		"14:30 - 15:50", // 5
		"16:00 - 17:20", // 6
		"17:25 - 18:45", // 7 (Vespertino 1)
	}

	timeRange := "??:?? - ??:??"
	if blockIndex >= 0 && blockIndex < len(times) {
		timeRange = times[blockIndex]
	}

	return dayName, timeRange
}

// printLoadStats muestra estadísticas de los datos cargados
func printLoadStats(university *domain.University) {
	totalLectures := len(university.Lectures)
	totalTutorials := len(university.Tutorials)
	totalLabs := len(university.Labs)
	totalClasses := totalLectures + totalTutorials + totalLabs

	fmt.Printf("✅ Datos cargados:\n")
	fmt.Printf("   - Cursos: %d\n", len(university.Courses))
	fmt.Printf("   - Secciones: %d\n", len(university.Sections))
	fmt.Printf("   - Clases: %d (Cátedras: %d, Ayudantías: %d, Labs: %d)\n",
		totalClasses, totalLectures, totalTutorials, totalLabs)
	fmt.Printf("   - Profesores: %d\n", len(university.Teachers))
	fmt.Printf("   - Salas: %d\n", len(university.Rooms))
}

// printRoomConstraintsExample muestra ejemplos de validación de restricciones
func printRoomConstraintsExample(university *domain.University) {
	if university.RoomConstraints == nil {
		fmt.Println("  ⚠️  No hay restricciones de salas cargadas")
		return
	}

	// Ejemplo 1: CIT1000 LABORATORIO puede usar LAB D
	isValid := university.RoomConstraints.IsValidRoomForClass("CIT1000", domain.ClassTypeLab, "LAB D")
	fmt.Printf("  ✓ CIT1000 LABORATORIO puede usar LAB D: %v\n", isValid)

	// Ejemplo 2: CBF1000 LABORATORIO solo puede usar LAB MECANICA
	isValid = university.RoomConstraints.IsValidRoomForClass("CBF1000", domain.ClassTypeLab, "LAB MECANICA")
	fmt.Printf("  ✓ CBF1000 LABORATORIO puede usar LAB MECANICA: %v\n", isValid)

	isValid = university.RoomConstraints.IsValidRoomForClass("CBF1000", domain.ClassTypeLab, "LAB D")
	fmt.Printf("  ✗ CBF1000 LABORATORIO puede usar LAB D: %v\n", isValid)

	// Ejemplo 3: Curso genérico CATEDRA puede usar sala normal (DEFAULTS)
	isValid = university.RoomConstraints.IsValidRoomForClass("CURSO_GENERICO", domain.ClassTypeLecture, "101")
	fmt.Printf("  ✓ Curso genérico CATEDRA puede usar sala 101: %v (DEFAULTS)\n", isValid)

	// Ejemplo 4: Contar salas válidas para CIT1000 LAB
	validRooms := university.RoomConstraints.GetValidRoomsForClass("CIT1000", domain.ClassTypeLab, university.Rooms)
	fmt.Printf("  📊 CIT1000 LABORATORIO tiene %d salas válidas disponibles\n", len(validRooms))

	fmt.Printf("  📋 Total de restricciones cargadas: %d cursos específicos + DEFAULTS\n",
		len(university.RoomConstraints.CourseConstraints))
}

// printSolutionReport imprime el reporte final de la solución
func printSolutionReport(sol *solver.Solution) {
	fmt.Println("\n================================================================================")
	fmt.Println("📊 REPORTE FINAL DE TIMETABLING")
	fmt.Println("================================================================================")

	// Validar factibilidad
	status := "✅ FACTIBLE"
	if !sol.IsFeasible() {
		status = "❌ INFACTIBLE (Se necesitan más bloques que los disponibles en la semana)"
	}

	fmt.Printf("Bloques Temporales Necesarios (Colores): %d\n", sol.TotalColors)
	fmt.Printf("Total de Sesiones Asignadas: %d\n", sol.GetTotalSessions())
	fmt.Printf("Estado del Horario: %s\n", status)
	fmt.Println("--------------------------------------------------------------------------------")

	// Imprimir detalle de bloques
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Bloque (Color)\t| Cantidad Sesiones\t| Ejemplos de Sesiones Asignadas")
	fmt.Fprintln(w, "--------------\t| ----------------\t| -----------------------------")

	// Ordenar colores
	var keys []int
	for k := range sol.Schedule {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Imprimir cada bloque
	for _, k := range keys {
		sessions := sol.Schedule[k]

		// Generar ejemplos
		exampleStr := ""
		count := 0
		for _, session := range sessions {
			exampleStr += fmt.Sprintf("[%s] ", session.ID)
			count++
			if count >= 2 {
				exampleStr += "..."
				break
			}
		}

		fmt.Fprintf(w, "Bloque %d\t| %d sesiones\t| %s\n", k, len(sessions), exampleStr)
	}

	w.Flush()
	fmt.Println("================================================================================")
}
