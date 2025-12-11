package solver

import (
	"fmt"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
)

// ColorGraph ejecuta el algoritmo de coloración de grafos
// Implementa la Sección 2.1.2 del paper:
// "An algorithm to give a Reasonable Colouring of any given graph"
func ColorGraph(originalGraph *graph.ConflictGraph) *Solution {

	// PASO 1: "Let j:=1, H:=G" [cite: 124]
	// Crear copia profunda para destruir/fusionar nodos sin afectar el original
	H := originalGraph.Copy()

	solution := NewSolution()

	fmt.Println("🚀 Iniciando algoritmo de coloreo...")

	// Bucle Principal: Mientras queden vértices en H
	for !H.IsNull() {

		// PASO 2: "Let vj be a vertex of maximal degree in H" [cite: 126]
		pivotID := findMaxDegreeNode(H)

		// PASO 3 y 4 (Bucle de Fusión):
		// "Merge vj and x_hat into vj... until no choice is possible" [cite: 131]
		for {
			candidateID := findBestMergeCandidate(H, pivotID)

			if candidateID == "" {
				// No hay más candidatos compatibles
				break
			}

			// Fusionar candidato dentro del pivot
			H.MergeNodes(pivotID, candidateID)
		}

		// PASO 5: Asignar color [cite: 132]
		// ESTRATEGIA: Balanced Coloring (Least Loaded Valid Color) + Heurísticas de Negocio

		mergedGroup := H.MergeHistory[pivotID]
		mergedGroup = append(mergedGroup, pivotID) // Agregar el líder

		// Identificar tipo de sesión líder para aplicar heurísticas
		isTutorial := false
		isLecture := false
		var pivotSession *domain.ClassSession
		if session, ok := originalGraph.Nodes[pivotID]; ok {
			pivotSession = session
			if session.GetType() == domain.ClassTypeTutorial {
				isTutorial = true
			} else if session.GetType() == domain.ClassTypeLecture {
				isLecture = true
			}
		}

		// Pre-calcular bloques espejo deseados si es Cátedra
		// Buscamos otras sesiones DEL MISMO CURSO Y SECCIÓN ya asignadas
		desiredMirrorBlocks := make(map[int]bool)
		if isLecture && pivotSession != nil {
			// Accedemos a las sesiones a través de la clase -> sección -> sesiones
			// Nota: Esto asume que pivotSession está vinculado correctamente
			if course := pivotSession.GetCourse(); course != nil {
				// Buscar sesiones ya asignadas de este curso/sección en la solución actual
				// Recorremos el schedule actual para encontrar "hermanos"
				for bID, assignedSessions := range solution.Schedule {
					for _, s := range assignedSessions {
						// Verificar si es "hermano": Mismo Curso, Misma Sección, Tipo Cátedra
						if s.Class.GetCourse().Code == course.Code &&
							s.GetType() == domain.ClassTypeLecture &&
							s.Class.GetSections()[0].Number == pivotSession.Class.GetSections()[0].Number { // Simplificación: asumiendo 1 sección principal

							// Es un hermano ya asignado en bloque bID
							// Calcular sus espejos y marcarlos como deseables
							mirrors := getMirrorBlocks(bID)
							for _, m := range mirrors {
								desiredMirrorBlocks[m] = true
							}
						}
					}
				}
			}
		}

		bestColor := -1
		minLoad := 999999
		maxBlocks := 35 // Límite de la semana

		// Buscar el mejor color (bloque) entre los disponibles
		// Intentamos reutilizar colores existentes para mantenernos dentro de los 35 bloques
		// Si no es posible, expandiremos.
		searchLimit := maxBlocks
		if solution.TotalColors > maxBlocks {
			searchLimit = solution.TotalColors
		}

		for c := 1; c <= searchLimit; c++ {
			// Verificar validez (Conflictos)
			canUseBlock := true
			for _, sessionID := range mergedGroup {
				if session, ok := originalGraph.Nodes[sessionID]; ok {
					if solution.HasConflictInBlock(c, session, originalGraph) {
						canUseBlock = false
						break
					}
				}
			}

			if canUseBlock {
				// Calcular "Carga" (Load)
				load := len(solution.Schedule[c])

				// --- HEURÍSTICAS ---

				// 1. Preferencia Ayudantías -> Miércoles
				if isTutorial && isWednesdayBlock(c) {
					load -= 10000 // Gran preferencia (Forzada si es posible)
				}

				// 2. Preferencia Cátedras -> Horarios Espejo
				if isLecture && desiredMirrorBlocks[c] {
					load -= 5000 // Preferencia muy alta para alinear horarios
				}

				// -------------------

				if load < minLoad {
					minLoad = load
					bestColor = c
				}
			}
		}

		// Si no encontramos color válido en los existentes, crear uno nuevo
		if bestColor == -1 {
			// Buscar el primer color válido hacia arriba
			c := searchLimit + 1
			for {
				canUseBlock := true
				for _, sessionID := range mergedGroup {
					if session, ok := originalGraph.Nodes[sessionID]; ok {
						if solution.HasConflictInBlock(c, session, originalGraph) {
							canUseBlock = false
							break
						}
					}
				}
				if canUseBlock {
					bestColor = c
					break
				}
				c++
			}
		}

		actualColor := bestColor

		// Asignar color a todas las sesiones del grupo
		for _, sessionID := range mergedGroup {
			if session, ok := originalGraph.Nodes[sessionID]; ok {
				session.Color = actualColor
				session.AssignedSlot = domain.TimeSlot(actualColor)
				solution.Schedule[actualColor] = append(solution.Schedule[actualColor], session)
			}
		}

		// "Start again from 2 with H := H-{vj}" [cite: 131]
		H.RemoveNode(pivotID)

		// No incrementamos colorIndex linealmente porque reutilizamos colores
	}

	solution.TotalColors = findMaxUsedColor(solution)
	return solution
}

func isWednesdayBlock(block int) bool {
	return block >= 14 && block <= 20
}

// findMaxDegreeNode busca el nodo con más conflictos en el grafo
// ... (resto del archivo)
func findMaxDegreeNode(g *graph.ConflictGraph) string {
	maxDegree := -1
	var bestNodeID string

	for id := range g.Nodes {
		deg := g.GetDegree(id)
		if deg > maxDegree {
			maxDegree = deg
			bestNodeID = id
		}
	}
	return bestNodeID
}

// getMirrorBlocks retorna los bloques equivalentes en otros días (mismo horario)
// Asumiendo 7 bloques por día
func getMirrorBlocks(blockID int) []int {
	// Normalizar a índice 0-6 (bloque del día)
	blockIndex := (blockID - 1) % 7

	// Generar los 5 días (o 6)
	// Lunes: 1 + index
	// Martes: 8 + index
	// Miércoles: 15 + index
	// Jueves: 22 + index
	// Viernes: 29 + index
	// Sábado: 36 + index (opcional, por ahora hasta viernes 35)

	mirrors := []int{}
	for d := 0; d < 5; d++ { // 5 días laborables estándar
		m := (d * 7) + 1 + blockIndex
		if m != blockID { // No incluirse a sí mismo (opcional, pero útil para lógica de conjuntos)
			mirrors = append(mirrors, m)
		}
	}
	return mirrors
}

// findBestMergeCandidate busca el mejor candidato para fusionar con targetID
// Implementa la lógica de "Triples" y "Common Neighbors" del paper
func findBestMergeCandidate(g *graph.ConflictGraph, targetID string) string {
	var bestCandidate string
	maxCommon := -1
	maxDegree := -1

	// Obtener vecinos del target para verificar adyacencia
	targetAdjacency := g.AdjacencyList[targetID]

	for candidateID := range g.Nodes {
		// 1. Validaciones básicas
		if candidateID == targetID {
			continue
		}

		// 2. REGLA DE ORO: No pueden ser adyacentes
		// Si son vecinos, tienen conflicto y no se pueden fusionar
		if targetAdjacency[candidateID] {
			continue
		}

		// 3. Calcular métricas
		commonNeighbors := g.GetCommonNeighbors(targetID, candidateID)
		commonCount := len(commonNeighbors)
		degree := g.GetDegree(candidateID)

		// 4. Lógica de Selección
		// "find y_i such that m_i = max(...)"
		if commonCount > maxCommon {
			maxCommon = commonCount
			maxDegree = degree
			bestCandidate = candidateID
		} else if commonCount == maxCommon {
			// Empate: elegir el de mayor grado
			// "choose a vertex of maximal degree non-adjacent"
			if degree > maxDegree {
				maxDegree = degree
				bestCandidate = candidateID
			}
		}
	}

	return bestCandidate
}
