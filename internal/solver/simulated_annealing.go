package solver

import (
	"math"
	"math/rand"
	"strconv"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/utils"
)

// SAConfig contiene los parámetros del Simulated Annealing.
type SAConfig struct {
	InitialTemp    float64 // Temperatura inicial
	CoolingRate    float64 // Tasa de enfriamiento (0.99 típico)
	MinTemp        float64 // Temperatura mínima para parar
	IterationsPerT int     // Iteraciones por nivel de temperatura
}

// DefaultSAConfig retorna configuración por defecto con más iteraciones.
func DefaultSAConfig() SAConfig {
	return SAConfig{
		InitialTemp:    1000.0,
		CoolingRate:    0.999, // Más lento = más iteraciones
		MinTemp:        0.01,  // Temperatura mínima más baja
		IterationsPerT: 5000,  // Reducido para prueba
	}
}

// SAResult contiene el resultado de la optimización.
type SAResult struct {
	InitialCost     float64
	FinalCost       float64
	Iterations      int
	Improvements    int
	MirrorPenalty   float64
	WednesdayBonus  float64
	PrereqBonus     float64 // Porcentaje de pares prereq en mismo bloque
	RoomConsistency float64 // Porcentaje de hermanos en misma sala
	DaySeparation   float64 // Porcentaje de CAT con separación ideal (3 días)
}

// SimulatedAnnealing optimiza el horario usando SA.
// puede mover bloques y salas.
// Hard constraints validados:
// - RC1: Conflicto de profesor
// - RC2: Conflicto de sección
// - RC3: Sala duplicada en bloque
// - RC4: Capacidad de sala
// - RC5: Tipo de sala (LAB/CLASSROOM)
// - RC6: Restricciones específicas de sala
// - RC7: Cliques de semestre
// Soft constraints:
// - Cátedras hermanas en mismo slot horario
// - Cátedras hermanas en MISMA SALA (nuevo)
// - Ayudantías en miércoles
// - Prerrequisitos en mismo bloque
func SimulatedAnnealing(activities []domain.Activity, rooms []domain.Room, config SAConfig, prerequisites map[string][]string, planLocations map[string]map[string]int, electives map[string]bool, constraints loader.RoomConstraints) SAResult {

	// Construir índices útiles
	siblingGroups := buildSiblingIndex(activities)
	courseActivities := buildCourseIndex(activities)
	prereqPairs := buildPrereqPairs(prerequisites, courseActivities)

	// Construir mapa de cliques de semestre para validación rápida (sin electivos)
	cliqueConflicts := buildCliqueMap(activities, planLocations, electives)

	// Índice de salas por código para validación rápida
	roomMap := buildRoomMap(rooms)

	// Calcular costo inicial (ahora incluye room consistency)
	initialCost := calculateTotalCostWithRooms(activities, siblingGroups, prereqPairs)

	// SA loop
	temperature := config.InitialTemp
	currentCost := initialCost
	iterations := 0
	improvements := 0

	// Índice de actividades por bloque y sala
	blockOccupancy := buildBlockOccupancy(activities)
	roomBlockOccupancy := buildRoomBlockOccupancy(activities) // room+block -> activity

	for temperature > config.MinTemp {
		for i := 0; i < config.IterationsPerT; i++ {
			iterations++

			// Seleccionar actividad aleatoria
			idx := rand.Intn(len(activities))
			activity := &activities[idx]

			// 50% probabilidad de mover bloque, 50% de mover sala
			moveType := rand.Intn(2)

			if moveType == 0 {
				// === MOVIMIENTO DE BLOQUE ===
				newBlock := rand.Intn(domain.TotalBlocks)
				oldBlock := activity.Block

				if newBlock == oldBlock {
					continue
				}

				// Verificar hard constraints para nuevo bloque
				if hasConflictInBlockWithRoom(activity, newBlock, activity.Room, blockOccupancy, roomBlockOccupancy, cliqueConflicts) {
					continue
				}

				// Calcular delta de costo
				oldCost := activityCostForBlockAndRoom(activity, oldBlock, activity.Room, siblingGroups)
				newCostVal := activityCostForBlockAndRoom(activity, newBlock, activity.Room, siblingGroups)
				delta := newCostVal - oldCost

				// Aceptar o rechazar
				if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
					removeFromOccupancy(activity, oldBlock, activity.Room, blockOccupancy, roomBlockOccupancy)
					activity.Block = newBlock
					addToOccupancy(activity, newBlock, activity.Room, blockOccupancy, roomBlockOccupancy)

					currentCost += delta
					if delta < 0 {
						improvements++
					}
				}
			} else {
				// === MOVIMIENTO DE SALA ===
				newRoom := selectValidRoom(activity, activity.Block, rooms, roomMap, constraints, roomBlockOccupancy)
				if newRoom == "" || newRoom == activity.Room {
					continue
				}

				// La sala ya fue validada (RC4, RC5, RC6, RC3)
				oldRoom := activity.Room

				// Calcular delta de costo (room consistency)
				oldCost := activityCostForBlockAndRoom(activity, activity.Block, oldRoom, siblingGroups)
				newCostVal := activityCostForBlockAndRoom(activity, activity.Block, newRoom, siblingGroups)
				delta := newCostVal - oldCost

				// Aceptar o rechazar
				if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
					removeFromOccupancy(activity, activity.Block, oldRoom, blockOccupancy, roomBlockOccupancy)
					activity.Room = newRoom
					addToOccupancy(activity, activity.Block, newRoom, blockOccupancy, roomBlockOccupancy)

					currentCost += delta
					if delta < 0 {
						improvements++
					}
				}
			}
		}

		temperature *= config.CoolingRate
	}

	// Calcular costos finales
	finalCost := calculateTotalCostWithRooms(activities, siblingGroups, prereqPairs)
	mirrorPenalty := calculateMirrorPenalty(activities, siblingGroups)
	wednesdayBonus := calculateWednesdayBonus(activities)
	prereqBonus := calculatePrereqBonus(activities, prereqPairs)
	roomConsistency := calculateRoomConsistency(activities, siblingGroups)
	daySeparation := calculateDaySeparationMetric(activities, siblingGroups)

	return SAResult{
		InitialCost:     initialCost,
		FinalCost:       finalCost,
		Iterations:      iterations,
		Improvements:    improvements,
		MirrorPenalty:   mirrorPenalty,
		WednesdayBonus:  wednesdayBonus,
		PrereqBonus:     prereqBonus,
		RoomConsistency: roomConsistency,
		DaySeparation:   daySeparation,
	}
}

// PrereqPair representa un par de actividades que son prerrequisito/dependiente
type PrereqPair struct {
	PrereqActivity *domain.Activity
	DepActivity    *domain.Activity
}

// buildCourseIndex crea índice de actividades por curso
func buildCourseIndex(activities []domain.Activity) map[string][]*domain.Activity {
	index := make(map[string][]*domain.Activity)
	for i := range activities {
		a := &activities[i]
		index[a.CourseCode] = append(index[a.CourseCode], a)
	}
	return index
}

// buildPrereqPairs crea lista de pares prerrequisito/dependiente
func buildPrereqPairs(prerequisites map[string][]string, courseActivities map[string][]*domain.Activity) []PrereqPair {
	var pairs []PrereqPair

	for depCourse, prereqCodes := range prerequisites {
		depActivities := courseActivities[depCourse]
		for _, prereqCode := range prereqCodes {
			prereqActivities := courseActivities[prereqCode]
			// Crear par por cada combinación de actividades
			for _, prereq := range prereqActivities {
				for _, dep := range depActivities {
					pairs = append(pairs, PrereqPair{PrereqActivity: prereq, DepActivity: dep})
				}
			}
		}
	}
	return pairs
}

// calculatePrereqBonus calcula porcentaje de pares prereq en mismo bloque
func calculatePrereqBonus(activities []domain.Activity, prereqPairs []PrereqPair) float64 {
	if len(prereqPairs) == 0 {
		return 0.0
	}

	sameBlock := 0
	for _, pair := range prereqPairs {
		if pair.PrereqActivity.Block == pair.DepActivity.Block {
			sameBlock++
		}
	}

	return float64(sameBlock) / float64(len(prereqPairs)) * 100.0
}

// buildBlockOccupancy crea índice de actividades por bloque (considerando duración)
func buildBlockOccupancy(activities []domain.Activity) map[int][]*domain.Activity {
	occ := make(map[int][]*domain.Activity)
	for i := range activities {
		a := &activities[i]
		duration := a.Duration
		if duration < 1 {
			duration = 1
		}
		// Registrar en cada bloque que ocupa
		for d := 0; d < duration; d++ {
			b := a.Block + d
			occ[b] = append(occ[b], a)
		}
	}
	return occ
}

// buildCliqueMap construye un mapa de conflictos por clique de semestre.
func buildCliqueMap(activities []domain.Activity, planLocations map[string]map[string]int, electives map[string]bool) map[string]map[string]bool {
	// identificar cursos con 1 sola sección (o fusionadas)
	courseSectionGroups := make(map[string]map[string]bool)
	for i := range activities {
		a := &activities[i]
		if courseSectionGroups[a.CourseCode] == nil {
			courseSectionGroups[a.CourseCode] = make(map[string]bool)
		}
		key := utils.SectionGroupKey(a.Sections)
		courseSectionGroups[a.CourseCode][key] = true
	}

	// Solo cursos con 1 sección Y NO electivos
	singleSectionCourses := make(map[string]bool)
	for code, groups := range courseSectionGroups {
		if len(groups) == 1 && !electives[code] {
			singleSectionCourses[code] = true
		}
	}

	// agrupar por carrera y semestre
	semesterCourses := make(map[string]map[int][]string)
	for code := range singleSectionCourses {
		if locs, ok := planLocations[code]; ok {
			for major, sem := range locs {
				if semesterCourses[major] == nil {
					semesterCourses[major] = make(map[int][]string)
				}
				semesterCourses[major][sem] = append(semesterCourses[major][sem], code)
			}
		}
	}

	// crear mapa de conflictos
	conflicts := make(map[string]map[string]bool)
	for _, semesters := range semesterCourses {
		for _, courses := range semesters {
			if len(courses) < 2 {
				continue
			}
			// Todos contra todos en este semestre
			for _, c1 := range courses {
				if conflicts[c1] == nil {
					conflicts[c1] = make(map[string]bool)
				}
				for _, c2 := range courses {
					if c1 != c2 {
						conflicts[c1][c2] = true
					}
				}
			}
		}
	}
	return conflicts
}

// buildSiblingIndex crea un índice de actividades por SiblingGroupID.
func buildSiblingIndex(activities []domain.Activity) map[string][]*domain.Activity {
	index := make(map[string][]*domain.Activity)
	for i := range activities {
		if activities[i].SiblingGroupID != "" {
			index[activities[i].SiblingGroupID] = append(
				index[activities[i].SiblingGroupID],
				&activities[i],
			)
		}
	}
	return index
}

// blockToDaySlot convierte bloque (0-34) a día y slot.
func blockToDaySlot(block int) (day, slot int) {
	day = block / domain.BlocksPerDay
	slot = block % domain.BlocksPerDay
	return
}

// calculateMirrorPenalty calcula la penalidad de espejo.
func calculateMirrorPenalty(activities []domain.Activity, siblings map[string][]*domain.Activity) float64 {
	penalty := 0.0
	counted := make(map[string]bool)

	for i := range activities {
		a := &activities[i]
		if a.SiblingGroupID == "" || counted[a.SiblingGroupID] {
			continue
		}
		counted[a.SiblingGroupID] = true

		sibs := siblings[a.SiblingGroupID]
		if len(sibs) < 2 {
			continue
		}

		// Verificar si todos están en espejo
		_, baseSlot := blockToDaySlot(sibs[0].Block)
		baseRoom := sibs[0].Room

		for j := 1; j < len(sibs); j++ {
			_, slot := blockToDaySlot(sibs[j].Block)
			if slot != baseSlot {
				penalty += 50.0
			}
			if sibs[j].Room != baseRoom {
				penalty += 20.0
			}
		}
	}
	return penalty
}

// calculateWednesdayBonus calcula cuántas AY están en miércoles.
func calculateWednesdayBonus(activities []domain.Activity) float64 {
	ayOnWednesday := 0
	totalAY := 0

	for i := range activities {
		if activities[i].Type == domain.AY {
			totalAY++
			day, _ := blockToDaySlot(activities[i].Block)
			if day == 2 {
				ayOnWednesday++
			}
		}
	}

	if totalAY == 0 {
		return 0
	}
	return float64(ayOnWednesday) / float64(totalAY) * 100.0
}

// buildRoomMap crea un mapa de código de sala a Room para búsqueda rápida
func buildRoomMap(rooms []domain.Room) map[string]domain.Room {
	m := make(map[string]domain.Room)
	for _, r := range rooms {
		m[r.Code] = r
	}
	return m
}

// buildRoomBlockOccupancy crea índice (room, block) -> actividad ocupante
func buildRoomBlockOccupancy(activities []domain.Activity) map[string]*domain.Activity {
	occ := make(map[string]*domain.Activity)
	for i := range activities {
		a := &activities[i]
		duration := a.Duration
		if duration < 1 {
			duration = 1
		}
		// registrar en cada bloque que ocupa
		for d := 0; d < duration; d++ {
			b := a.Block + d
			key := a.Room + ":" + strconv.Itoa(b)
			occ[key] = a
		}
	}
	return occ
}

// selectValidRoom selecciona una sala válida aleatoria para la actividad en el bloque dado, valida: RC3, RC4, RC5 y RC6
func selectValidRoom(activity *domain.Activity, block int, rooms []domain.Room, roomMap map[string]domain.Room, constraints loader.RoomConstraints, roomBlockOcc map[string]*domain.Activity) string {
	// obtener salas permitidas por restricción específica
	eventType := eventTypeToString(activity.Type)
	allowedCodes := constraints.GetAllowedRooms(activity.CourseCode, eventType)

	duration := activity.Duration
	if duration < 1 {
		duration = 1
	}

	var validRooms []string

roomLoop:
	for _, room := range rooms {
		// RC3
		for d := 0; d < duration; d++ {
			b := block + d
			key := room.Code + ":" + strconv.Itoa(b)
			if existing := roomBlockOcc[key]; existing != nil && existing.ID != activity.ID {
				continue roomLoop
			}
		}

		// RC6
		if allowedCodes != nil {
			if !contains(allowedCodes, room.Code) {
				continue
			}
		} else {
			// RC5
			if activity.Type == domain.LAB && room.Type != domain.RoomLab {
				continue
			}
			if activity.Type != domain.LAB && room.Type != domain.RoomClassroom {
				continue
			}
		}

		// RC4
		if activity.Students > room.Capacity {
			continue
		}

		validRooms = append(validRooms, room.Code)
	}

	if len(validRooms) == 0 {
		return ""
	}

	// seleccionar aleatoriamente entre las válidas
	return validRooms[rand.Intn(len(validRooms))]
}

// hasConflictInBlockWithRoom verifica conflictos considerando la sala propuesta y la duración de la actividad (puede ocupar múltiples bloques consecutivos).
func hasConflictInBlockWithRoom(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity, cliqueConflicts map[string]map[string]bool) bool {
	duration := activity.Duration
	if duration < 1 {
		duration = 1
	}

	// validar que no cruce días: todos los bloques deben estar en el mismo día
	startDay := block / domain.BlocksPerDay
	endBlock := block + duration - 1
	endDay := endBlock / domain.BlocksPerDay
	if startDay != endDay {
		return true // cruzaría días - inválido
	}

	// validar que no exceda el último bloque del día
	slotInDay := block % domain.BlocksPerDay
	if slotInDay+duration > domain.BlocksPerDay {
		return true // No cabe en el día
	}

	// validar que no ocupe el bloque protegido del miércoles (11:30-12:50)
	// esto aplica tanto al bloque directo como a actividades multi-bloque que lo atraviesen
	if domain.OccupiesProtectedBlock(block, duration) {
		return true // ocuparía el bloque protegido
	}

	// verificar cada bloque que ocuparía la actividad
	for i := 0; i < duration; i++ {
		b := block + i

		// verificar ocupación de sala en este bloque
		key := room + ":" + strconv.Itoa(b)
		if existing := roomBlockOcc[key]; existing != nil && existing.ID != activity.ID {
			return true // sala ocupada en este bloque
		}

		// verificar conflictos con otras actividades en este bloque
		for _, other := range blockOcc[b] {
			if other.ID == activity.ID {
				continue
			}
			if activity.SharesTeacher(other) {
				return true
			}
			if activity.SharesSection(other) {
				return true
			}
			// Clique de semestre
			if cliqueConflicts[activity.CourseCode] != nil && cliqueConflicts[activity.CourseCode][other.CourseCode] {
				return true
			}
		}
	}
	return false
}

// removeFromOccupancy limpia los indices de ocupación actual, se usa cuando SA mueve una actividad de un bloque a otro
func removeFromOccupancy(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity) {
	duration := activity.Duration
	if duration < 1 {
		duration = 1
	}

	for i := 0; i < duration; i++ {
		b := block + i

		// remover de blockOcc
		list := blockOcc[b]
		for j, a := range list {
			if a.ID == activity.ID {
				blockOcc[b] = append(list[:j], list[j+1:]...)
				break
			}
		}

		// remover de roomBlockOcc
		key := room + ":" + strconv.Itoa(b)
		delete(roomBlockOcc, key)
	}
}

// addToOccupancy agrega actividad a los índices considerando su duración, se usa cuando SA mueve una actividad de un bloque a otro
func addToOccupancy(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity) {
	duration := activity.Duration
	if duration < 1 {
		duration = 1
	}

	for i := 0; i < duration; i++ {
		b := block + i
		blockOcc[b] = append(blockOcc[b], activity)
		key := room + ":" + strconv.Itoa(b)
		roomBlockOcc[key] = activity
	}
}

// activityCostForBlockAndRoom calcula costo de actividad en bloque + sala
// Incluye penalidades de espejo (horario Y sala) Y separación de días
func activityCostForBlockAndRoom(activity *domain.Activity, block int, room string, siblings map[string][]*domain.Activity) float64 {
	cost := 0.0

	// solo evaluar hermanos para cátedras
	if activity.SiblingGroupID != "" && activity.Type == domain.CAT {
		sibs := siblings[activity.SiblingGroupID]
		myDay, mySlot := blockToDaySlot(block)

		// filtrar solo hermanos CAT (no AY)
		var catSibs []*domain.Activity
		for _, sib := range sibs {
			if sib.Type == domain.CAT {
				catSibs = append(catSibs, sib)
			}
		}

		for _, sib := range catSibs {
			if sib.ID == activity.ID {
				continue
			}
			sibDay, sibSlot := blockToDaySlot(sib.Block)

			// penalidad por NO estar en espejo (mismo slot horario)
			if sibSlot != mySlot {
				cost += 50.0
			}

			// penalidad por hermano en distinta sala
			if sib.Room != room {
				cost += 30.0
			}

			daySeparation := abs(myDay - sibDay)

			if len(catSibs) == 2 {
				// para 2 cátedras: ideal 3 días (Lun-Jue, Mar-Vie)
				switch daySeparation {
				case 3: // ideal: Lun-Jue o Mar-Vie
					cost -= 20.0 // Bonus
				case 2: // aceptable: Lun-Mie, Mar-Jue, Mie-Vie
					cost += 0.0 // Neutro
				case 1: // malo: días consecutivos
					cost += 25.0
				case 0: // muy malo: mismo día
					cost += 60.0
				default: // 4 días (Lun-Vie)
					cost += 10.0 // menos ideal que 3
				}
			} else if len(catSibs) >= 3 {
				// para 3+ cátedras: deben estar en días diferentes
				if daySeparation == 0 {
					cost += 80.0 // muy malo: dos CAT el mismo día
				} else if daySeparation == 1 {
					cost += 15.0 // aceptable pero no ideal
				}
			}
		}

		// Verificar que CAT no esté el mismo día que su AY
		for _, sib := range sibs {
			if sib.Type == domain.AY {
				ayDay, _ := blockToDaySlot(sib.Block)
				if myDay == ayDay {
					cost += 35.0 // Penalizar CAT mismo día que AY
				}
			}
		}
	}

	// Bonus por AY en miércoles
	if activity.Type == domain.AY {
		day, _ := blockToDaySlot(block)
		if day != 2 {
			cost += 10.0 // Penalidad si AY NO está en miércoles
		}
	}

	return cost
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// calculateTotalCostWithRooms calcula costo total incluyendo la consistencia de salas y separación de días
func calculateTotalCostWithRooms(activities []domain.Activity, siblings map[string][]*domain.Activity, prereqPairs []PrereqPair) float64 {
	cost := 0.0

	// costo de espejo, sala y separación de días (catedras)
	counted := make(map[string]bool)
	for i := range activities {
		a := &activities[i]
		if a.SiblingGroupID == "" || a.Type != domain.CAT || counted[a.SiblingGroupID] {
			continue
		}
		counted[a.SiblingGroupID] = true

		sibs := siblings[a.SiblingGroupID]

		// filtrar solo catedra
		var catSibs []*domain.Activity
		for _, sib := range sibs {
			if sib.Type == domain.CAT {
				catSibs = append(catSibs, sib)
			}
		}

		if len(catSibs) < 2 {
			continue
		}

		baseDay, baseSlot := blockToDaySlot(catSibs[0].Block)
		baseRoom := catSibs[0].Room

		for j := 1; j < len(catSibs); j++ {
			day, slot := blockToDaySlot(catSibs[j].Block)

			// espejo
			if slot != baseSlot {
				cost += 50.0
			}
			// sala
			if catSibs[j].Room != baseRoom {
				cost += 30.0
			}

			// separación de días
			daySeparation := abs(day - baseDay)
			if len(catSibs) == 2 {
				switch daySeparation {
				case 3:
					cost -= 20.0
				case 2:
					cost += 0.0
				case 1:
					cost += 25.0
				case 0:
					cost += 60.0
				default:
					cost += 10.0
				}
			} else if len(catSibs) >= 3 {
				if daySeparation == 0 {
					cost += 80.0
				} else if daySeparation == 1 {
					cost += 15.0
				}
			}
		}

		// verificar CAT vs AY mismo día
		for _, cat := range catSibs {
			catDay, _ := blockToDaySlot(cat.Block)
			for _, sib := range sibs {
				if sib.Type == domain.AY {
					ayDay, _ := blockToDaySlot(sib.Block)
					if catDay == ayDay {
						cost += 35.0
					}
				}
			}
		}
	}

	// Costo de AY no en miércoles
	for i := range activities {
		if activities[i].Type == domain.AY {
			day, _ := blockToDaySlot(activities[i].Block)
			if day != 2 {
				cost += 10.0
			}
		}
	}

	// Bonus por prereqs en mismo bloque
	for _, pair := range prereqPairs {
		if pair.PrereqActivity.Block == pair.DepActivity.Block {
			cost -= 15.0
		}
	}

	return cost
}

// calculateRoomConsistency calcula % de grupos de hermanos que comparten sala
func calculateRoomConsistency(activities []domain.Activity, siblings map[string][]*domain.Activity) float64 {
	totalGroups := 0
	consistentGroups := 0

	counted := make(map[string]bool)
	for i := range activities {
		a := &activities[i]
		if a.SiblingGroupID == "" || counted[a.SiblingGroupID] {
			continue
		}
		counted[a.SiblingGroupID] = true

		sibs := siblings[a.SiblingGroupID]
		if len(sibs) < 2 {
			continue
		}

		totalGroups++
		baseRoom := sibs[0].Room
		allSame := true
		for j := 1; j < len(sibs); j++ {
			if sibs[j].Room != baseRoom {
				allSame = false
				break
			}
		}
		if allSame {
			consistentGroups++
		}
	}

	if totalGroups == 0 {
		return 100.0
	}
	return float64(consistentGroups) / float64(totalGroups) * 100.0
}

// calculateDaySeparationMetric calcula % de grupos CAT con separación ideal de días
func calculateDaySeparationMetric(activities []domain.Activity, siblings map[string][]*domain.Activity) float64 {
	totalGroups := 0
	idealGroups := 0

	counted := make(map[string]bool)
	for i := range activities {
		a := &activities[i]
		if a.SiblingGroupID == "" || a.Type != domain.CAT || counted[a.SiblingGroupID] {
			continue
		}
		counted[a.SiblingGroupID] = true

		sibs := siblings[a.SiblingGroupID]

		// Filtrar solo CAT
		var catSibs []*domain.Activity
		for _, sib := range sibs {
			if sib.Type == domain.CAT {
				catSibs = append(catSibs, sib)
			}
		}

		if len(catSibs) != 2 {
			continue // Solo medimos grupos de 2 CAT
		}

		totalGroups++
		day0, _ := blockToDaySlot(catSibs[0].Block)
		day1, _ := blockToDaySlot(catSibs[1].Block)
		separation := abs(day0 - day1)

		if separation == 3 { // Lun-Jue o Mar-Vie
			idealGroups++
		}
	}

	if totalGroups == 0 {
		return 100.0
	}
	return float64(idealGroups) / float64(totalGroups) * 100.0
}
