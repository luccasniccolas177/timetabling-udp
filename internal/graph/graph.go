package graph

import (
	"fmt"
	"timetabling-UDP/internal/domain"
)

// ConflictGraph representa un grafo G(V, E)
// V = Sesiones de clase (ClassSession)
// E = Conflictos (dos sesiones no pueden estar en el mismo slot)
type ConflictGraph struct {
	// Nodos: ID de sesión → Sesión de clase
	Nodes map[string]*domain.ClassSession

	// Lista de Adyacencia: ID → Set de IDs vecinos
	// Usamos map[string]bool para búsquedas O(1)
	AdjacencyList map[string]map[string]bool

	// MergeHistory: Rastrea fusiones de nodos
	// Key: ID del nodo sobreviviente → Value: IDs absorbidos
	// Vital para colorear nodos fusionados con el mismo color
	MergeHistory map[string][]string
}

// NewConflictGraph inicializa un grafo vacío
func NewConflictGraph() *ConflictGraph {
	return &ConflictGraph{
		Nodes:         make(map[string]*domain.ClassSession),
		AdjacencyList: make(map[string]map[string]bool),
		MergeHistory:  make(map[string][]string),
	}
}

// AddNode agrega una sesión al grafo
func (g *ConflictGraph) AddNode(session *domain.ClassSession) {
	if _, ok := g.Nodes[session.ID]; !ok {
		g.Nodes[session.ID] = session
		g.AdjacencyList[session.ID] = make(map[string]bool)
	}
}

// AddEdge agrega una arista (conflicto) entre dos sesiones
func (g *ConflictGraph) AddEdge(sessionID1, sessionID2 string) {
	// Validaciones
	if sessionID1 == sessionID2 {
		return
	}
	if _, ok := g.Nodes[sessionID1]; !ok {
		return
	}
	if _, ok := g.Nodes[sessionID2]; !ok {
		return
	}

	// Agregar arista bidireccional
	g.AdjacencyList[sessionID1][sessionID2] = true
	g.AdjacencyList[sessionID2][sessionID1] = true
}

// HasEdge verifica si existe una arista entre dos sesiones
func (g *ConflictGraph) HasEdge(sessionID1, sessionID2 string) bool {
	if neighbors, ok := g.AdjacencyList[sessionID1]; ok {
		return neighbors[sessionID2]
	}
	return false
}

// GetDegree retorna el número de vecinos de una sesión
func (g *ConflictGraph) GetDegree(sessionID string) int {
	if neighbors, ok := g.AdjacencyList[sessionID]; ok {
		return len(neighbors)
	}
	return 0
}

// GetNeighbors retorna los IDs de los vecinos de una sesión
func (g *ConflictGraph) GetNeighbors(sessionID string) []string {
	neighbors := make([]string, 0, len(g.AdjacencyList[sessionID]))
	for neighborID := range g.AdjacencyList[sessionID] {
		neighbors = append(neighbors, neighborID)
	}
	return neighbors
}

// RemoveNode elimina una sesión del grafo
func (g *ConflictGraph) RemoveNode(sessionID string) {
	if _, exists := g.Nodes[sessionID]; !exists {
		return
	}

	// Eliminar referencias en vecinos
	for neighborID := range g.AdjacencyList[sessionID] {
		delete(g.AdjacencyList[neighborID], sessionID)
	}

	// Eliminar nodo
	delete(g.AdjacencyList, sessionID)
	delete(g.Nodes, sessionID)
}

// MergeNodes fusiona dos nodos (u absorbe a v)
func (g *ConflictGraph) MergeNodes(uID, vID string) {
	if uID == vID {
		return
	}
	if _, ok := g.Nodes[uID]; !ok {
		return
	}
	if _, ok := g.Nodes[vID]; !ok {
		return
	}

	// u hereda todos los vecinos de v
	for neighborID := range g.AdjacencyList[vID] {
		if neighborID != uID {
			g.AddEdge(uID, neighborID)
		}
	}

	// Guardar historial de fusión
	g.MergeHistory[uID] = append(g.MergeHistory[uID], vID)

	// Si v ya había absorbido otros, u los hereda
	if absorbedByV, ok := g.MergeHistory[vID]; ok {
		g.MergeHistory[uID] = append(g.MergeHistory[uID], absorbedByV...)
		delete(g.MergeHistory, vID)
	}

	// Eliminar v
	g.RemoveNode(vID)
}

// IsNull verifica si el grafo está vacío
func (g *ConflictGraph) IsNull() bool {
	return len(g.Nodes) == 0
}

// Copy crea una copia profunda del grafo
func (g *ConflictGraph) Copy() *ConflictGraph {
	newGraph := NewConflictGraph()

	// Copiar nodos
	for id, session := range g.Nodes {
		newGraph.Nodes[id] = session
		newGraph.AdjacencyList[id] = make(map[string]bool)
	}

	// Copiar aristas
	for u, neighbors := range g.AdjacencyList {
		for v := range neighbors {
			newGraph.AdjacencyList[u][v] = true
		}
	}

	// Copiar historial de fusiones
	for survivor, absorbed := range g.MergeHistory {
		newSlice := make([]string, len(absorbed))
		copy(newSlice, absorbed)
		newGraph.MergeHistory[survivor] = newSlice
	}

	return newGraph
}

// GetCommonNeighbors retorna los vecinos comunes de dos sesiones
func (g *ConflictGraph) GetCommonNeighbors(uID, vID string) []string {
	if uID == vID {
		return []string{}
	}
	if _, ok := g.Nodes[uID]; !ok {
		return []string{}
	}
	if _, ok := g.Nodes[vID]; !ok {
		return []string{}
	}

	var common []string

	// Obtener vecinos de v como mapa para búsqueda O(1)
	vNeighborsMap := g.AdjacencyList[vID]

	// Buscar vecinos comunes
	for uNeighbor := range g.AdjacencyList[uID] {
		if _, exists := vNeighborsMap[uNeighbor]; exists {
			common = append(common, uNeighbor)
		}
	}

	return common
}

// PrintStats imprime estadísticas del grafo
func (g *ConflictGraph) PrintStats() {
	v := len(g.Nodes)
	e := 0
	for _, neighbors := range g.AdjacencyList {
		e += len(neighbors)
	}
	e = e / 2 // Dividir por 2 (grafo no dirigido)

	density := 0.0
	if v > 1 {
		// Densidad: 2|E| / (|V| * (|V|-1))
		density = float64(2*e) / float64(v*(v-1))
	}

	fmt.Printf("📊 ESTADÍSTICAS DEL GRAFO G(V, E):\n")
	fmt.Printf("   - Vértices (|V|): %d\n", v)
	fmt.Printf("   - Aristas  (|E|): %d\n", e)
	fmt.Printf("   - Densidad:       %.4f\n", density)
}
