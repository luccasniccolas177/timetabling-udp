package solver

import (
	"sort"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
)

// ColorSet representa un conjunto de actividades que pueden ocurrir en el mismo periodo, no hay conflictos entre actividades
type ColorSet struct {
	Color      int                // Número del color/periodo
	Activities []*domain.Activity // Actividades asignadas a este periodo
}

// GreedyColoring implementa el algoritmo de Dutton-Brigham para colorear el grafo.
// Retorna una lista de ColorSets donde cada set es un periodo sin conflictos.
func GreedyColoring(g *graph.ConflictGraph) []ColorSet {
	// Crear copia de trabajo del grafo
	H := cloneGraph(g)

	var colorSets []ColorSet
	color := 0

	// Mientras queden vértices sin colorear
	for H.NumVertices() > 0 {
		// Encontrar conjunto independiente máximo
		colorSet := findMaxIndependentSet(H)

		if len(colorSet) == 0 {
			break
		}

		// Crear ColorSet con las actividades
		cs := ColorSet{
			Color:      color,
			Activities: make([]*domain.Activity, len(colorSet)),
		}
		for i, id := range colorSet {
			cs.Activities[i] = g.Vertices[id]
		}
		colorSets = append(colorSets, cs)

		// Eliminar vértices coloreados del grafo de trabajo
		for _, id := range colorSet {
			removeVertex(H, id)
		}

		color++
	}

	return colorSets
}

// findMaxIndependentSet encuentra un conjunto independiente máximo
func findMaxIndependentSet(H *graph.ConflictGraph) []int {
	if H.NumVertices() == 0 {
		return nil
	}

	seed := maxDegreeVertex(H)
	if seed == -1 {
		return nil
	}

	independentSet := []int{seed}
	merged := map[int]bool{seed: true}

	for {
		// Buscar mejor candidato para fusionar (máximos vecinos comunes, no adyacente)
		candidate := findBestMergeCandidate(H, independentSet, merged)

		if candidate == -1 {
			// No hay más candidatos, el conjunto está completo
			break
		}

		// Agregar al conjunto independiente
		independentSet = append(independentSet, candidate)
		merged[candidate] = true
	}

	return independentSet
}

// findBestMergeCandidate encuentra el vértice no adyacente con más vecinos comunes.
func findBestMergeCandidate(H *graph.ConflictGraph, currentSet []int, merged map[int]bool) int {
	bestCandidate := -1
	maxCommonNeighbors := -1

	// Para cada candidato
	for candidateID := range H.Vertices {
		// continuar si ya está en el conjunto
		if merged[candidateID] {
			continue
		}

		// Verificar que NO sea adyacente a ningún vértice del conjunto actual
		isAdjacent := false
		for _, setID := range currentSet {
			if H.HasEdge(candidateID, setID) {
				isAdjacent = true
				break
			}
		}
		if isAdjacent {
			continue
		}

		// Contar vecinos comunes con el conjunto
		commonNeighbors := countCommonNeighbors(H, currentSet, candidateID)

		// Elegir el candidato con más vecinos comunes
		if commonNeighbors > maxCommonNeighbors {
			maxCommonNeighbors = commonNeighbors
			bestCandidate = candidateID
		} else if commonNeighbors == maxCommonNeighbors && bestCandidate != -1 {
			// si hay empate, elegir el de mayor grado
			if H.Degree(candidateID) > H.Degree(bestCandidate) {
				bestCandidate = candidateID
			}
		}
	}

	// Si no hay candidatos con vecinos comunes, buscar cualquier vertice no adyacente con max grado
	if bestCandidate == -1 {
		bestCandidate = findMaxDegreeNonAdjacent(H, currentSet, merged)
	}

	return bestCandidate
}

// countCommonNeighbors cuenta cuántos vecinos del conjunto son también vecinos del candidato.
func countCommonNeighbors(H *graph.ConflictGraph, currentSet []int, candidateID int) int {
	candidateNeighbors := make(map[int]bool)
	for _, n := range H.Neighbors(candidateID) {
		candidateNeighbors[n] = true
	}

	count := 0
	for _, setID := range currentSet {
		for _, neighbor := range H.Neighbors(setID) {
			if candidateNeighbors[neighbor] {
				count++
			}
		}
	}
	return count
}

// findMaxDegreeNonAdjacent encuentra el vértice de mayor grado no adyacente al conjunto.
func findMaxDegreeNonAdjacent(H *graph.ConflictGraph, currentSet []int, merged map[int]bool) int {
	bestID := -1
	maxDegree := -1

	for id := range H.Vertices {
		if merged[id] {
			continue
		}

		// Verificar no-adyacencia
		isAdjacent := false
		for _, setID := range currentSet {
			if H.HasEdge(id, setID) {
				isAdjacent = true
				break
			}
		}
		if isAdjacent {
			continue
		}

		if H.Degree(id) > maxDegree {
			maxDegree = H.Degree(id)
			bestID = id
		}
	}
	return bestID
}

// maxDegreeVertex retorna el ID del vértice con mayor grado.
func maxDegreeVertex(H *graph.ConflictGraph) int {
	maxID := -1
	maxDeg := -1
	for id := range H.Vertices {
		if H.Degree(id) > maxDeg {
			maxDeg = H.Degree(id)
			maxID = id
		}
	}
	return maxID
}

// cloneGraph crea una copia del grafo para trabajar sin modificar el original.
func cloneGraph(g *graph.ConflictGraph) *graph.ConflictGraph {
	clone := graph.New()

	// Copiar vértices
	for id, a := range g.Vertices {
		clone.Vertices[id] = a
		clone.Adjacency[id] = make(map[int]bool)
	}

	// Copiar aristas
	for id, neighbors := range g.Adjacency {
		for n := range neighbors {
			clone.Adjacency[id][n] = true
		}
	}

	return clone
}

// removeVertex elimina un vértice y todas sus aristas del grafo.
func removeVertex(H *graph.ConflictGraph, id int) {
	// Eliminar aristas hacia este vértice
	for neighborID := range H.Adjacency[id] {
		delete(H.Adjacency[neighborID], id)
	}
	// Eliminar el vértice
	delete(H.Adjacency, id)
	delete(H.Vertices, id)
}

// AssignBlocksToColorSets asigna bloques temporales (0-34) a cada ColorSet.
// Cada color se mapea a un bloque diferente, saltando el bloque protegido del miércoles.
func AssignBlocksToColorSets(colorSets []ColorSet) {
	block := 0
	for i := range colorSets {
		// Saltar el bloque protegido del miércoles (11:30-12:50)
		for domain.IsProtectedBlock(block) && block < domain.TotalBlocks {
			block++
		}
		if block >= domain.TotalBlocks {
			block = 0
			// Volver a verificar
			for domain.IsProtectedBlock(block) && block < domain.TotalBlocks {
				block++
			}
		}
		for _, activity := range colorSets[i].Activities {
			activity.Block = block
		}
		block++
	}
}

// SortColorSetsBySize ordena los ColorSets por tamaño.
func SortColorSetsBySize(colorSets []ColorSet) {
	sort.Slice(colorSets, func(i, j int) bool {
		return len(colorSets[i].Activities) > len(colorSets[j].Activities)
	})
}
