package solver

import (
	"fmt"
	"math"
	"math/rand"
	"time"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
)

type SAConfig struct {
	InitialTemp float64
	CoolingRate float64
	Iterations  int
	MaxColors   int
}

// OptimizeSchedule aplica Simulated Annealing para mejorar la solución inicial
func OptimizeSchedule(initialSolution *Solution, g *graph.ConflictGraph, config SAConfig) *Solution {
	fmt.Printf("\n🔥 [Simulated Annealing] Iniciando optimización...\n")
	fmt.Printf("   Config: Temp=%.1f, Cooling=%.4f, Iters=%d\n",
		config.InitialTemp, config.CoolingRate, config.Iterations)

	// Semilla aleatoria
	rand.Seed(time.Now().UnixNano())

	// Empezar con la solución actual
	currentSolution := initialSolution
	currentCost := calculateCost(currentSolution)

	// bestSolution := currentSolution.Copy() // Necesitamos implementar Copy deep
	bestCost := currentCost

	temperature := config.InitialTemp

	fmt.Printf("   Costo Inicial: %.2f\n", currentCost)

	// Lista de todas las sesiones para elegir aleatoriamente
	var allSessions []*domain.ClassSession
	for _, session := range g.Nodes {
		allSessions = append(allSessions, session)
	}

	// Construir mapa auxiliar eficiente
	type GroupKey struct {
		Code    string
		Section int
	}
	groups := make(map[GroupKey][]*domain.ClassSession)

	for _, s := range allSessions {
		if s.GetType() == domain.ClassTypeLecture {
			k := GroupKey{
				Code:    s.Class.GetCourse().Code,
				Section: s.Class.GetSections()[0].Number,
			}
			groups[k] = append(groups[k], s)
		}
	}

	acceptedMoves := 0
	improvedMoves := 0

	for i := 0; i < config.Iterations; i++ {
		// 1. Generar vecino (movimiento)

		// Estrategia: 70% Movimiento Inteligente (Si es cátedra), 30% Aleatorio
		useSmartMove := rand.Float64() < 0.7

		sessionIdx := rand.Intn(len(allSessions))
		candidateSession := allSessions[sessionIdx]

		oldColor := candidateSession.Color
		newColor := -1

		// Intentar movimiento inteligente si es Cátedra y tiene hermanas
		if useSmartMove && candidateSession.GetType() == domain.ClassTypeLecture {
			k := GroupKey{
				Code:    candidateSession.Class.GetCourse().Code,
				Section: candidateSession.Class.GetSections()[0].Number,
			}
			siblings := groups[k]

			// Si tiene hermanas, intentar moverse al espejo de una de ellas
			if len(siblings) > 1 {
				// Elegir una hermana al azar (que no sea ella misma)
				// Como siblings incluye a self, elegimos random.
				sibling := siblings[rand.Intn(len(siblings))]
				if sibling.ID != candidateSession.ID {
					// Calcular espejo del bloque de la hermana
					siblingBlock := sibling.Color
					mirrors := getMirrorBlocks(siblingBlock)

					if len(mirrors) > 0 {
						// Elegir un espejo al azar
						targetMirror := mirrors[rand.Intn(len(mirrors))]

						// Validar si es posible
						if targetMirror >= 1 && targetMirror <= config.MaxColors {
							// Preferencia de Gap: Intentar elegir el espejo que de Gap DE 3 DIAS
							// Si hay múltiples espejos, elegir el que maximice la separación ideal (3)
							// Calcular Gap actual
							currentDay := (siblingBlock - 1) / 7
							targetDay := (targetMirror - 1) / 7

							gap := int(math.Abs(float64(targetDay - currentDay)))
							if gap == 3 {
								// Ideal!
								newColor = targetMirror
								break // Encontramos el mejor, salir
							} else {
								// Si no es ideal, guardamos como backup y seguimos buscando
								newColor = targetMirror
							}
						}
					}
				}
			}
		}

		// Si no se eligió color inteligente (o falló), usar aleatorio
		if newColor == -1 {
			newColor = rand.Intn(config.MaxColors) + 1
		}

		if newColor == oldColor {
			continue // No hay cambio
		}

		// Verificar validez dura (Grafo de conflictos)
		if !isValidMove(currentSolution, candidateSession, newColor, g) {
			continue // Movimiento inválido, saltar
		}

		// Aplicar movimiento tentativo
		moveSessionSA(currentSolution, candidateSession, oldColor, newColor)

		// 2. Calcular nuevo costo
		newCost := calculateCost(currentSolution)
		delta := newCost - currentCost

		// 3. Criterio de Aceptación (Metropolis)
		shouldAccept := false
		if delta < 0 {
			// Mejora: Aceptar siempre
			shouldAccept = true
			improvedMoves++
		} else {
			// Empeora: Aceptar con probabilidad e^(-delta/T)
			probability := math.Exp(-delta / temperature)

			// Truco: Si el movimiento inteligente generó un espejo,
			// es probable que bajara mucho el costo.
			// Si estamos subiendo el costo, probablemente rompimos otro espejo o sobrecargamos.

			if rand.Float64() < probability {
				shouldAccept = true
			}
		}

		if shouldAccept {
			currentCost = newCost
			acceptedMoves++

			// Actualizar mejor global
			if currentCost < bestCost {
				bestCost = currentCost
				// Guardar copia de la mejor (esto es costoso, optimización: guardar solo ids o diffs)
				// Por simplicidad en este prototipo, asumimos que 'currentSolution'
				// eventualmente convergerá a algo bueno, o podríamos guardar snapshot.
				// Para evitar overhead de DeepCopy en cada mejora, confiamos en el estado final
				// o hacemos DeepCopy solo si la mejora es significativa.
			}
		} else {
			// Revertir movimiento
			moveSessionSA(currentSolution, candidateSession, newColor, oldColor)
		}

		// Enfriamiento
		temperature *= config.CoolingRate

		// Log periódico
		if i%1000 == 0 {
			// fmt.Printf("Iter %d: Temp=%.2f Cost=%.2f\n", i, temperature, currentCost)
		}
	}

	fmt.Printf("   Costo Final: %.2f (Mejor: %.2f)\n", currentCost, bestCost)
	fmt.Printf("   Movimientos Aceptados: %d, Mejoras: %d\n", acceptedMoves, improvedMoves)

	return currentSolution
}

// calculateCost evalúa la calidad de la solución basándose en soft constraints
func calculateCost(sol *Solution) float64 {
	cost := 0.0

	// Mapa auxiliar para búscar espejos rápidamente
	// CourseCode -> SectionNum -> []Blocks
	courseSchedule := make(map[string]map[int][]int)

	// Reconstruir mapa de sesiones
	// Iterar por bloques es más rápido que por sesiones sueltas si ya está en schedule
	for blockID, sessions := range sol.Schedule {
		// Penalty por balance de carga (suave)
		// Ideal: ~34 sesiones por bloque (1200 / 35)
		// deviation := float64(len(sessions) - 34)
		// cost += deviation * deviation * 0.1 // Peso bajo

		for _, s := range sessions {
			// Construir estructura para espejos
			if s.GetType() == domain.ClassTypeLecture {
				code := s.Class.GetCourse().Code
				secNum := s.Class.GetSections()[0].Number // Asumiendo sección principal

				if _, ok := courseSchedule[code]; !ok {
					courseSchedule[code] = make(map[int][]int)
				}
				courseSchedule[code][secNum] = append(courseSchedule[code][secNum], blockID)
			}

			// Soft Constraint: Ayudantías en Miércoles (15-21)
			// Penalty fuerte si NO es miércoles
			if s.GetType() == domain.ClassTypeTutorial {
				if !isWednesdayBlock(blockID) {
					cost += 500 // Castigo REDUCIDO (antes 2000)
				}
			}
		}
	}

	// Soft Constraint: Horarios Espejo con Separación de Días
	for _, secMap := range courseSchedule {
		for _, blocks := range secMap {
			if len(blocks) < 2 {
				continue // Nada que espejar
			}

			// Verificar coherencia espejo y Gap
			firstMod := (blocks[0] - 1) % 7
			firstDay := (blocks[0] - 1) / 7

			for _, b := range blocks[1:] {
				mod := (b - 1) % 7
				currentDay := (b - 1) / 7

				// 1. Espejo (Mismo bloque)
				if mod != firstMod {
					cost += 5000 // Castigo AUMENTADO: Debe ser espejo
				} else {
					// 2. Gap de días (Solo si es espejo, evaluamos el gap)
					gap := int(math.Abs(float64(firstDay - currentDay)))

					if gap == 3 {
						// Ideal (Lunes-Jueves, Martes-Viernes)
						cost += 0
					} else if gap == 2 {
						// Aceptable (Lunes-Miércoles, Miércoles-Viernes)
						cost += 200 // Penalización leve
					} else if gap == 1 {
						// Muy juntos (Lunes-Martes)
						cost += 1000 // Penalización media-alta
					} else if gap == 0 {
						// Mismo día (Imposible físicamente para el mismo curso usualmente, pero por si acaso)
						cost += 5000
					} else {
						// Gap 4+ (Lunes-Viernes)
						cost += 500 // Aceptable pero no ideal
					}
				}
			}
		}
	}

	return cost
}

// isValidMove verifica si mover una sesión a targetColor viola restricciones duras
func isValidMove(sol *Solution, session *domain.ClassSession, targetColor int, g *graph.ConflictGraph) bool {
	// Verificar conflictos con sesiones ya presentes en el targetColor
	// Usamos el grafo original para chequear adyacencia

	// Nota: sol.Schedule[targetColor] tiene las sesiones en ese color
	// Pero el método más rápido es chequear vecinos en el grafo

	neighbors := g.AdjacencyList[session.ID]
	for neighborID := range neighbors {
		neighborNode := g.Nodes[neighborID]
		if neighborNode.Color == targetColor {
			return false // Conflicto directo
		}
	}
	return true
}

// moveSessionSA ejecuta el cambio de color en la estructura de datos
func moveSessionSA(sol *Solution, session *domain.ClassSession, oldColor, newColor int) {
	// 1. Actualizar objeto sesión
	session.Color = newColor
	session.AssignedSlot = domain.TimeSlot(newColor)

	// 2. Actualizar mapa Schedule (Costoso, optimizar si es lento)
	// Remover de lista vieja
	oldList := sol.Schedule[oldColor]
	for i, s := range oldList {
		if s.ID == session.ID {
			// Swap remove
			sol.Schedule[oldColor] = append(oldList[:i], oldList[i+1:]...)
			break
		}
	}

	// Agregar a lista nueva
	sol.Schedule[newColor] = append(sol.Schedule[newColor], session)
}

// Copy crea una copia profunda de la solución (Limitada a estructura, punteros a objetos de dominio se mantienen)
func (s *Solution) Copy() *Solution {
	newSol := NewSolution()
	newSol.TotalColors = s.TotalColors

	for k, v := range s.Schedule {
		// Copiar slice
		newSlice := make([]*domain.ClassSession, len(v))
		copy(newSlice, v)
		newSol.Schedule[k] = newSlice
	}
	return newSol
}
