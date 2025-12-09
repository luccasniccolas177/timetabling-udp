package graph

import (
	"fmt"
	"timetabling-UDP/internal/models"
)

// ConflictGraph representa un grafo G(V, E), donde V son los eventos (catedras, ayudantias y labs) y E representan los conflictos o restricciones
type ConflictGraph struct {
	// mapeamos los identifcadores del evento con la instancia
	Nodes map[string]*models.EventInstance

	// Lista de Adyacencia: Mapa de UUID -> Set de UUIDs vecinos
	// Usamos map[string]bool para búsquedas y eliminaciones O(1)
	AdjacencyList map[string]map[string]bool

	// MergeHistory: Mapa para rastrear fusiones.
	// Key: UUID del nodo "sobreviviente" (u) -> Value: Lista de UUIDs absorbidos (v)
	// Esto es vital para el paso final del paper: "colour all vertices merged into v_j colour i"
	MergeHistory map[string][]string
}

// NewConflictGraph inicializa el grafo
func NewConflictGraph() *ConflictGraph {
	return &ConflictGraph{
		Nodes:         make(map[string]*models.EventInstance),
		AdjacencyList: make(map[string]map[string]bool),
		MergeHistory:  make(map[string][]string),
	}
}

func (g *ConflictGraph) AddNode(e *models.EventInstance) {
	// si el identificador no se encuentra dentro de los nodos actuales se agrega al grafo y se inicializa la lista de adjacencia
	if _, ok := g.Nodes[e.UUID]; !ok {
		g.Nodes[e.UUID] = e
		g.AdjacencyList[e.UUID] = make(map[string]bool)
	}
}

func (g *ConflictGraph) AddEdge(uUUID, vUUID string) {
	// verificamos que los identificadores no sea iguales y que existan como nodos registrados
	if uUUID == vUUID {
		return
	}
	if _, ok := g.Nodes[uUUID]; !ok {
		return
	}
	if _, ok := g.Nodes[vUUID]; !ok {
		return
	}

	// registramos la arista(conflicto) entre ambos eventos
	g.AdjacencyList[uUUID][vUUID] = true
	g.AdjacencyList[vUUID][uUUID] = true
}

func (g *ConflictGraph) GetDegree(UUID string) int {
	// verificamos que existe el identificador y retornamos el numero de eventos con conflictos
	if neighbors, ok := g.AdjacencyList[UUID]; ok {
		return len(neighbors)
	}
	return 0
}

// GetNeighbors retorna una lista con los UUIDs de lo vecinos
func (g *ConflictGraph) GetNeighbors(UUID string) []string {
	neighbors := make([]string, 0, len(g.AdjacencyList[UUID]))
	for neighborUUID := range g.AdjacencyList[UUID] {
		neighbors = append(neighbors, neighborUUID)
	}
	return neighbors
}

// RemoveNode elimina un nodo y limpia todas sus referencias en los vecinos.
func (g *ConflictGraph) RemoveNode(targetUUID string) {
	if _, exists := g.Nodes[targetUUID]; !exists {
		return
	}

	// eliminar las referencias de v en todos sus vecinos
	for neighborUUID := range g.AdjacencyList[targetUUID] {
		delete(g.AdjacencyList[neighborUUID], targetUUID)
	}

	// eliminar la entrada del nodo en la lista de adyacencia
	delete(g.AdjacencyList, targetUUID)

	// eliminar v
	delete(g.Nodes, targetUUID)
}

// MergeNodes fusiona el nodo v dentro de u
// Implementa la lógica: "Merge v_j and y_i into v_j"[cite: 131].
func (g *ConflictGraph) MergeNodes(uUUID, vUUID string) {
	if uUUID == vUUID {
		return
	}
	if _, ok := g.Nodes[uUUID]; !ok {
		return
	}
	if _, ok := g.Nodes[vUUID]; !ok {
		return
	}

	// agregamos a u todos los vecinos de v, se heredan
	for neighborOfV := range g.AdjacencyList[vUUID] {
		// si ya estan conectados, se omite
		if neighborOfV != uUUID {
			g.AddEdge(uUUID, neighborOfV)
		}
	}

	// 2. Guardar Historial (Crucial para colorear al final):
	// Registramos que 'v' ahora es parte de 'u'.
	g.MergeHistory[uUUID] = append(g.MergeHistory[uUUID], vUUID)

	// Si 'v' ya había absorbido a otros antes, esos también pasan a ser parte de 'u'
	if absorbedByV, ok := g.MergeHistory[vUUID]; ok {
		g.MergeHistory[uUUID] = append(g.MergeHistory[uUUID], absorbedByV...)
		delete(g.MergeHistory, vUUID) // Limpiamos el historial del nodo borrado
	}

	// 3. Eliminar el nodo absorbido del grafo activo
	g.RemoveNode(vUUID)
}

func (g *ConflictGraph) IsNull() bool {
	return len(g.Nodes) == 0
}

// Copy crea una copia profunda (Deep Copy) del grafo.
// Es fundamental para el algoritmo recursivo que reduce el grafo (H = G).
func (g *ConflictGraph) Copy() *ConflictGraph {
	newGraph := NewConflictGraph()

	// copiar Nodos
	for id, node := range g.Nodes {
		newGraph.Nodes[id] = node
		// Inicializar mapa de adyacencia vacío para el nuevo nodo
		newGraph.AdjacencyList[id] = make(map[string]bool)
	}

	// 2. copiar aristas
	for u, neighbors := range g.AdjacencyList {
		for v := range neighbors {
			newGraph.AdjacencyList[u][v] = true
		}
	}

	// 3. Copiar Historial de Fusiones (Si lo estás usando)
	for survivor, absorbed := range g.MergeHistory {
		// Necesitamos copiar el slice, no solo referenciarlo
		newSlice := make([]string, len(absorbed))
		copy(newSlice, absorbed)
		newGraph.MergeHistory[survivor] = newSlice
	}

	return newGraph
}

func (g *ConflictGraph) GetCommonNeighbors(uUUID, vUUID string) []string {
	// Validaciones básicas
	if uUUID == vUUID {
		return []string{}
	}
	if _, ok := g.Nodes[uUUID]; !ok {
		return []string{}
	}
	if _, ok := g.Nodes[vUUID]; !ok {
		return []string{}
	}

	var common []string

	// Iteramos sobre los vecinos de u y verificamos existencia en v usando el mapa.
	// Acceder a g.AdjacencyList[vUUID][vecino] es O(1), muy rápido.

	// Obtenemos el mapa de vecinos de V directamente para búsquedas rápidas
	vNeighborsMap := g.AdjacencyList[vUUID]

	for uNeighbor := range g.AdjacencyList[uUUID] {
		// Verificamos si este vecino de U también está en el mapa de V
		if _, exists := vNeighborsMap[uNeighbor]; exists {
			common = append(common, uNeighbor)
		}
	}

	return common
}

// PrintStats imprime un resumen para verificar la construcción
func (g *ConflictGraph) PrintStats() {
	v := len(g.Nodes)
	e := 0
	for _, neighbors := range g.AdjacencyList {
		e += len(neighbors)
	}
	e = e / 2 // Dividir por 2 porque es no dirigido (A->B y B->A cuentan como 1 arista)

	density := 0.0
	if v > 1 {
		// Fórmula de densidad: 2|E| / (|V| * (|V|-1))
		density = float64(2*e) / float64(v*(v-1))
	}

	fmt.Printf("📊 ESTADÍSTICAS DEL GRAFO G(V, E):\n")
	fmt.Printf("   - Vértices (|V|): %d\n", v)
	fmt.Printf("   - Aristas  (|E|): %d\n", e)
	fmt.Printf("   - Densidad:       %.4f\n", density)
}
