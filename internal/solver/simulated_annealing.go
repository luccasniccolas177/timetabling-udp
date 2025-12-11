package solver

import (
	"math"
	"math/rand"
	"time"

	"timetabling-UDP/internal/domain"
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
		IterationsPerT: 200,   // Más iteraciones por temperatura
	}
}

// SAResult contiene el resultado de la optimización.
type SAResult struct {
	InitialCost    float64
	FinalCost      float64
	Iterations     int
	Improvements   int
	MirrorPenalty  float64
	WednesdayBonus float64
}

// SimulatedAnnealing optimiza el horario usando SA.
// IMPORTANTE: Solo swapea BLOQUES, no salas. Las salas se mantienen fijas
// para respetar restricciones de capacidad y tipo de sala.
// Soft constraints:
// - Cátedras hermanas (SiblingGroupID) deben estar en mismo slot horario
// - Ayudantías idealmente en miércoles
func SimulatedAnnealing(activities []domain.Activity, rooms []domain.Room, config SAConfig) SAResult {
	rand.Seed(time.Now().UnixNano())

	// Construir índices útiles
	siblingGroups := buildSiblingIndex(activities)

	// Calcular costo inicial
	initialCost := calculateTotalCost(activities, siblingGroups)

	// SA loop
	temperature := config.InitialTemp
	currentCost := initialCost
	iterations := 0
	improvements := 0

	// Índice de actividades por bloque (para verificar conflictos rápidamente)
	blockOccupancy := buildBlockOccupancy(activities)

	for temperature > config.MinTemp {
		for i := 0; i < config.IterationsPerT; i++ {
			iterations++

			// Seleccionar actividad aleatoria
			idx := rand.Intn(len(activities))
			activity := &activities[idx]

			// Seleccionar nuevo bloque aleatorio
			newBlock := rand.Intn(domain.TotalBlocks)
			oldBlock := activity.Block

			if newBlock == oldBlock {
				continue
			}

			// Verificar si el nuevo bloque causa conflictos
			if hasConflictInBlock(activity, newBlock, blockOccupancy) {
				continue
			}

			// Calcular delta de costo
			oldCost := activityCostForBlock(activity, oldBlock, siblingGroups)
			newCostVal := activityCostForBlock(activity, newBlock, siblingGroups)
			delta := newCostVal - oldCost

			// Aceptar o rechazar
			if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
				// Mover actividad al nuevo bloque
				removeFromBlockOccupancy(activity, oldBlock, blockOccupancy)
				activity.Block = newBlock
				addToBlockOccupancy(activity, newBlock, blockOccupancy)

				currentCost += delta
				if delta < 0 {
					improvements++
				}
			}
		}

		temperature *= config.CoolingRate
	}

	// Calcular costos finales
	finalCost := calculateTotalCost(activities, siblingGroups)
	mirrorPenalty := calculateMirrorPenalty(activities, siblingGroups)
	wednesdayBonus := calculateWednesdayBonus(activities)

	return SAResult{
		InitialCost:    initialCost,
		FinalCost:      finalCost,
		Iterations:     iterations,
		Improvements:   improvements,
		MirrorPenalty:  mirrorPenalty,
		WednesdayBonus: wednesdayBonus,
	}
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
func hasConflictInBlock(activity *domain.Activity, block int, occupancy map[int][]*domain.Activity) bool {
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
	}
	return false
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
