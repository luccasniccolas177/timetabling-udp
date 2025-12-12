package solver

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/loader"
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
		IterationsPerT: 2000,  // Reducido para prueba
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
}

// SimulatedAnnealing optimiza el horario usando SA.
// Ahora puede mover BLOQUES y SALAS.
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
	rand.Seed(time.Now().UnixNano())

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

	return SAResult{
		InitialCost:     initialCost,
		FinalCost:       finalCost,
		Iterations:      iterations,
		Improvements:    improvements,
		MirrorPenalty:   mirrorPenalty,
		WednesdayBonus:  wednesdayBonus,
		PrereqBonus:     prereqBonus,
		RoomConsistency: roomConsistency,
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

// calculateTotalCostWithPrereq calcula costo total incluyendo bonus de prereqs
func calculateTotalCostWithPrereq(activities []domain.Activity, siblings map[string][]*domain.Activity, prereqPairs []PrereqPair) float64 {
	cost := 0.0
	for i := range activities {
		cost += activityCost(&activities[i], siblings)
	}

	// Bonus por prereqs en mismo bloque (costo negativo = bueno)
	for _, pair := range prereqPairs {
		if pair.PrereqActivity.Block == pair.DepActivity.Block {
			cost -= 15.0 // Bonus balanceado
		}
	}

	return cost
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

// buildBlockOccupancy crea índice de actividades por bloque
func buildBlockOccupancy(activities []domain.Activity) map[int][]*domain.Activity {
	occ := make(map[int][]*domain.Activity)
	for i := range activities {
		b := activities[i].Block
		occ[b] = append(occ[b], &activities[i])
	}
	return occ
}

// hasConflictInBlock verifica si mover actividad a bloque causa conflicto
func hasConflictInBlock(activity *domain.Activity, block int, occupancy map[int][]*domain.Activity, cliqueConflicts map[string]map[string]bool) bool {
	for _, other := range occupancy[block] {
		if other.ID == activity.ID {
			continue
		}
		// Conflicto si comparten profesor
		if activity.SharesTeacher(other) {
			return true
		}
		// Conflicto si comparten sección (mismo curso)
		if activity.SharesSection(other) {
			return true
		}
		// Conflicto si usan la misma sala
		if activity.Room == other.Room {
			return true
		}
		// Conflicto de CLIQUE DE SEMESTRE (hard constraint)
		// Si el otro curso está en mi lista de conflictos de clique
		if cliqueConflicts[activity.CourseCode] != nil && cliqueConflicts[activity.CourseCode][other.CourseCode] {
			return true
		}
	}
	return false
}

// buildCliqueMap construye un mapa de conflictos por clique de semestre.
// Retorna: CourseCode -> Map[ConflictingCourseCode]bool
// Los cursos electivos NO forman parte de los cliques.
func buildCliqueMap(activities []domain.Activity, planLocations map[string]map[string]int, electives map[string]bool) map[string]map[string]bool {
	// 1. Identificar cursos con 1 sola sección (o fusionadas)
	courseSectionGroups := make(map[string]map[string]bool)
	for i := range activities {
		a := &activities[i]
		if courseSectionGroups[a.CourseCode] == nil {
			courseSectionGroups[a.CourseCode] = make(map[string]bool)
		}
		key := sectionGroupKey(a.Sections)
		courseSectionGroups[a.CourseCode][key] = true
	}

	// Solo cursos con 1 sección Y NO electivos
	singleSectionCourses := make(map[string]bool)
	for code, groups := range courseSectionGroups {
		if len(groups) == 1 && !electives[code] {
			singleSectionCourses[code] = true
		}
	}

	// 2. Agrupar por (Major, Semester)
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

	// 3. Crear mapa de conflictos
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

// sectionGroupKey crea una clave única para un grupo de secciones (helper duplicado de graph, mover a utils idealmente)
func sectionGroupKey(sections []int) string {
	if len(sections) == 0 {
		return "empty"
	}
	sorted := make([]int, len(sections))
	copy(sorted, sections)
	sort.Ints(sorted) // Usando sort.Ints

	key := ""
	for _, s := range sorted {
		if key != "" {
			key += "-"
		}
		key += strconv.Itoa(s) // Usando strconv
	}
	return key
}

// removeFromBlockOccupancy quita actividad del índice
func removeFromBlockOccupancy(activity *domain.Activity, block int, occupancy map[int][]*domain.Activity) {
	list := occupancy[block]
	for i, a := range list {
		if a.ID == activity.ID {
			occupancy[block] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

// addToBlockOccupancy agrega actividad al índice
func addToBlockOccupancy(activity *domain.Activity, block int, occupancy map[int][]*domain.Activity) {
	occupancy[block] = append(occupancy[block], activity)
}

// activityCostForBlock calcula costo considerando un bloque hipotético
func activityCostForBlock(a *domain.Activity, block int, siblings map[string][]*domain.Activity) float64 {
	cost := 0.0

	// Penalidad por hermanos NO en espejo
	if a.SiblingGroupID != "" {
		sibs := siblings[a.SiblingGroupID]
		for _, sib := range sibs {
			if sib.ID == a.ID {
				continue
			}
			day1, slot1 := blockToDaySlot(block)
			day2, slot2 := blockToDaySlot(sib.Block)

			if slot1 != slot2 {
				cost += 50.0 // Diferente hora
			} else if day1 == day2 {
				cost += 100.0 // Mismo día (no es espejo válido)
			}
		}
	}

	// Penalidad para AY no en miércoles
	if a.Type == domain.AY {
		day, _ := blockToDaySlot(block)
		if day != 2 {
			cost += 10.0
		}
	}

	return cost
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

// buildRoomIndex crea un índice de salas por código.
func buildRoomIndex(rooms []domain.Room) map[string]domain.Room {
	index := make(map[string]domain.Room)
	for _, r := range rooms {
		index[r.Code] = r
	}
	return index
}

// calculateTotalCost calcula el costo total del horario.
func calculateTotalCost(activities []domain.Activity, siblings map[string][]*domain.Activity) float64 {
	cost := 0.0
	for i := range activities {
		cost += activityCost(&activities[i], siblings)
	}
	return cost
}

// activityCost calcula el costo de una actividad.
func activityCost(a *domain.Activity, siblings map[string][]*domain.Activity) float64 {
	cost := 0.0

	// Penalidad por hermanos NO en espejo
	if a.SiblingGroupID != "" {
		sibs := siblings[a.SiblingGroupID]
		for _, sib := range sibs {
			if sib.ID == a.ID {
				continue
			}
			// Calcular si están en espejo (mismo slot horario, día diferente)
			day1, slot1 := blockToDaySlot(a.Block)
			day2, slot2 := blockToDaySlot(sib.Block)

			// Penalizar si NO están en mismo slot pero diferente día
			if slot1 != slot2 {
				cost += 50.0 // Penalidad alta: diferente hora
			} else if day1 == day2 {
				cost += 100.0 // Penalidad muy alta: mismo día (conflicto)
			}

			// Penalizar si NO están en la misma sala
			if a.Room != sib.Room {
				cost += 20.0 // Penalidad media: diferente sala
			}
		}
	}

	// Bonus para AY en miércoles (día 2, bloques 14-20)
	if a.Type == domain.AY {
		day, _ := blockToDaySlot(a.Block)
		if day != 2 { // No es miércoles
			cost += 10.0 // Penalidad leve
		}
	}

	return cost
}

// blockToDaySlot convierte bloque (0-34) a día y slot.
// Día: 0=Lun, 1=Mar, 2=Mié, 3=Jue, 4=Vie
// Slot: 0-6 (7 bloques por día)
func blockToDaySlot(block int) (day, slot int) {
	day = block / domain.BlocksPerDay
	slot = block % domain.BlocksPerDay
	return
}

// selectRandomSwap selecciona dos índices aleatorios para swap.
func selectRandomSwap(activities []domain.Activity) (int, int) {
	n := len(activities)
	return rand.Intn(n), rand.Intn(n)
}

// isValidSwap verifica si un swap es válido (misma sección = conflicto).
func isValidSwap(a1, a2 *domain.Activity) bool {
	// No permitir swap si comparten profesor
	if a1.SharesTeacher(a2) {
		return false
	}
	// No permitir swap si comparten sección
	if a1.SharesSection(a2) {
		return false
	}
	return true
}

// calculateMirrorPenalty calcula solo la penalidad de espejo.
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

// ═══════════════════════════════════════════════════════════════════════════
// NUEVAS FUNCIONES PARA ROOM SWAP
// ═══════════════════════════════════════════════════════════════════════════

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
		key := a.Room + ":" + strconv.Itoa(a.Block)
		occ[key] = a
	}
	return occ
}

// selectValidRoom selecciona una sala válida aleatoria para la actividad en el bloque dado
// Valida: RC3 (no ocupada), RC4 (capacidad), RC5 (tipo), RC6 (restricción específica)
func selectValidRoom(activity *domain.Activity, block int, rooms []domain.Room, roomMap map[string]domain.Room, constraints loader.RoomConstraints, roomBlockOcc map[string]*domain.Activity) string {
	// Obtener salas permitidas por restricción específica (RC6)
	eventType := eventTypeToString(activity.Type)
	allowedCodes := constraints.GetAllowedRooms(activity.CourseCode, eventType)

	var validRooms []string

	for _, room := range rooms {
		// RC3: No ocupada en este bloque
		key := room.Code + ":" + strconv.Itoa(block)
		if existing := roomBlockOcc[key]; existing != nil && existing.ID != activity.ID {
			continue
		}

		// RC6: Restricción específica
		if allowedCodes != nil {
			if !contains(allowedCodes, room.Code) {
				continue
			}
		} else {
			// RC5: Tipo de sala
			if activity.Type == domain.LAB && room.Type != domain.RoomLab {
				continue
			}
			if activity.Type != domain.LAB && room.Type != domain.RoomClassroom {
				continue
			}
		}

		// RC4: Capacidad
		if activity.Students > room.Capacity {
			continue
		}

		validRooms = append(validRooms, room.Code)
	}

	if len(validRooms) == 0 {
		return ""
	}

	// Seleccionar aleatoriamente entre las válidas
	return validRooms[rand.Intn(len(validRooms))]
}

// hasConflictInBlockWithRoom verifica conflictos considerando la sala propuesta
func hasConflictInBlockWithRoom(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity, cliqueConflicts map[string]map[string]bool) bool {
	// Verificar ocupación de sala
	key := room + ":" + strconv.Itoa(block)
	if existing := roomBlockOcc[key]; existing != nil && existing.ID != activity.ID {
		return true // Sala ocupada
	}

	// Verificar otros conflictos en el bloque
	for _, other := range blockOcc[block] {
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
	return false
}

// removeFromOccupancy remueve actividad de ambos índices
func removeFromOccupancy(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity) {
	// Remover de blockOcc
	list := blockOcc[block]
	for i, a := range list {
		if a.ID == activity.ID {
			blockOcc[block] = append(list[:i], list[i+1:]...)
			break
		}
	}
	// Remover de roomBlockOcc
	key := room + ":" + strconv.Itoa(block)
	delete(roomBlockOcc, key)
}

// addToOccupancy agrega actividad a ambos índices
func addToOccupancy(activity *domain.Activity, block int, room string, blockOcc map[int][]*domain.Activity, roomBlockOcc map[string]*domain.Activity) {
	blockOcc[block] = append(blockOcc[block], activity)
	key := room + ":" + strconv.Itoa(block)
	roomBlockOcc[key] = activity
}

// activityCostForBlockAndRoom calcula costo de actividad en bloque+sala dados
// Incluye penalidades de espejo (horario Y sala)
func activityCostForBlockAndRoom(activity *domain.Activity, block int, room string, siblings map[string][]*domain.Activity) float64 {
	cost := 0.0

	// Penalidad por hermanos en distinto bloque horario (espejo)
	if activity.SiblingGroupID != "" {
		sibs := siblings[activity.SiblingGroupID]
		_, mySlot := blockToDaySlot(block)

		for _, sib := range sibs {
			if sib.ID == activity.ID {
				continue
			}
			_, sibSlot := blockToDaySlot(sib.Block)
			if sibSlot != mySlot {
				cost += 50.0 // Penalidad por NO estar en espejo
			}
			// Penalidad por hermano en distinta sala
			if sib.Room != room {
				cost += 30.0 // Penalidad por NO estar en misma sala
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

// calculateTotalCostWithRooms calcula costo total incluyendo room consistency
func calculateTotalCostWithRooms(activities []domain.Activity, siblings map[string][]*domain.Activity, prereqPairs []PrereqPair) float64 {
	cost := 0.0

	// Costo de espejo (horario + sala)
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

		_, baseSlot := blockToDaySlot(sibs[0].Block)
		baseRoom := sibs[0].Room

		for j := 1; j < len(sibs); j++ {
			_, slot := blockToDaySlot(sibs[j].Block)
			if slot != baseSlot {
				cost += 50.0
			}
			if sibs[j].Room != baseRoom {
				cost += 30.0
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
