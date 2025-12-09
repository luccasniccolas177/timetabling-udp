package solver

import (
	"fmt"
	"timetabling-UDP/internal/graph"
)

// ColorGraph ejecuta el algoritmo de la Sección 2.1.2 del paper.
// "An algorithm to give a Reasonable Colouring of any given graph"
func ColorGraph(originalGraph *graph.ConflictGraph) *Solution {

	// PASO 1: "Let j:=1, H:=G" [cite: 124]
	// Creamos una copia profunda (H) para destruir/fusionar nodos sin afectar el original.
	// El grafo original se mantiene intacto para consultar los datos al final.
	H := originalGraph.Copy()

	solution := NewSolution()
	colorIndex := 1 // j en el paper

	fmt.Println("🚀 Iniciando algoritmo de coloreo...")

	// Bucle Principal: Mientras queden vértices en H
	for !H.IsNull() {

		// PASO 2: "Let vj be a vertex of maximal degree in H" [cite: 126]
		pivotID := findMaxDegreeNode(H)

		// Este nodo (pivot) será el "líder" del grupo de color actual.
		// Inicialmente, el color group es solo él.

		// PASO 3 y 4 (Bucle de Fusión):
		// "Merge vj and x_hat into vj... until no choice is possible" [cite: 131]
		for {
			// Usamos la heurística implementada anteriormente para buscar candidato
			candidateID := findBestMergeCandidate(H, pivotID)

			if candidateID == "" {
				// No hay más candidatos compatibles (o el grafo se vació)
				break
			}

			// Fusionar candidato dentro del pivot
			// Esto actualiza las aristas en H y guarda el rastro en MergeHistory
			H.MergeNodes(pivotID, candidateID)
		}

		// PASO 5: Asignar color y limpiar [cite: 132]
		// "Colour all vertices merged into vj colour i"
		// 1. Recuperamos quiénes fueron absorbidos por el pivot
		mergedGroup := H.MergeHistory[pivotID]
		mergedGroup = append(mergedGroup, pivotID) // Agregamos al líder también

		// 2. Asignamos el color en la solución final (usando los datos del grafo original)
		for _, uuid := range mergedGroup {
			// Buscamos el nodo en el grafo ORIGINAL (porque en H ya está mutado/borrado)
			if node, ok := originalGraph.Nodes[uuid]; ok {
				node.Color = colorIndex
				// node.AssignedSlot se mapeará luego a día/hora real
				solution.Schedule[colorIndex] = append(solution.Schedule[colorIndex], node)
			}
		}

		// "Start again from 2 with H := H-{vj}" [cite: 131]
		// Como ya fusionamos a todos dentro de pivotID, al borrar pivotID borramos el grupo entero.
		H.RemoveNode(pivotID)

		// Avanzamos al siguiente color
		colorIndex++
	}

	solution.TotalColors = colorIndex - 1
	return solution
}

// findMaxDegreeNode busca el nodo con más conflictos en el grafo actual H.
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

// findBestMergeCandidate busca el mejor candidato para fusionar con targetID.
// Implementa la lógica de "Triples" y "Common Neighbors" del paper
func findBestMergeCandidate(g *graph.ConflictGraph, targetID string) string {
	var bestCandidate string

	// Variables para el tracking de la mejor opción
	maxCommon := -1
	maxDegree := -1

	// Obtenemos los vecinos del target para verificar adyacencia rápidamente
	// (Recordemos: No podemos fusionar si ya son adyacentes/conflictivos)
	targetAdjacency := g.AdjacencyList[targetID]

	for candidateID := range g.Nodes {
		// 1. Validaciones básicas
		if candidateID == targetID {
			continue
		}

		// 2. REGLA DE ORO: No pueden ser adyacentes
		// Si candidateID está en la lista de vecinos de target, chocan. No se pueden fusionar.
		if targetAdjacency[candidateID] {
			continue
		}

		// 3. Calcular métricas
		// common: Cantidad de vecinos compartidos (m_i en el paper)
		commonNeighbors := g.GetCommonNeighbors(targetID, candidateID)
		commonCount := len(commonNeighbors)

		// degree: Grado del candidato (para desempate o fallback)
		degree := g.GetDegree(candidateID)

		// Lógica de Selección

		// Caso A: Encontramos uno con MÁS vecinos comunes que el actual mejor.
		// "find y_i such that m_i = max(...)"
		if commonCount > maxCommon {
			maxCommon = commonCount
			maxDegree = degree
			bestCandidate = candidateID

		} else if commonCount == maxCommon {
			// Caso B: Empate en vecinos comunes (o ambos son 0).
			// "choose a vertex of maximal degree non-adjacent"
			// Si m_i es 0 para todos, esto automáticamente selecciona el de mayor grado.
			if degree > maxDegree {
				maxDegree = degree
				bestCandidate = candidateID
			}
		}
	}

	return bestCandidate
}
