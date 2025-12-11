package graph

import (
	"timetabling-UDP/internal/domain"
)

// ConflictGraph representa el grafo de conflictos G = (V, E).
// Los vértices son Activities y las aristas representan conflictos.
type ConflictGraph struct {
	Vertices  map[int]*domain.Activity // ID -> Activity
	Adjacency map[int]map[int]bool     // ID -> Set de IDs adyacentes
}

// New crea un grafo de conflictos vacío.
func New() *ConflictGraph {
	return &ConflictGraph{
		Vertices:  make(map[int]*domain.Activity),
		Adjacency: make(map[int]map[int]bool),
	}
}

// AddVertex agrega una actividad como vértice.
func (g *ConflictGraph) AddVertex(a *domain.Activity) {
	g.Vertices[a.ID] = a
	if g.Adjacency[a.ID] == nil {
		g.Adjacency[a.ID] = make(map[int]bool)
	}
}

// AddEdge agrega una arista (conflicto) entre dos actividades.
func (g *ConflictGraph) AddEdge(id1, id2 int) {
	if g.Adjacency[id1] == nil {
		g.Adjacency[id1] = make(map[int]bool)
	}
	if g.Adjacency[id2] == nil {
		g.Adjacency[id2] = make(map[int]bool)
	}
	g.Adjacency[id1][id2] = true
	g.Adjacency[id2][id1] = true
}

// HasEdge verifica si existe una arista entre dos vértices.
func (g *ConflictGraph) HasEdge(id1, id2 int) bool {
	if adj, ok := g.Adjacency[id1]; ok {
		return adj[id2]
	}
	return false
}

// Degree retorna el grado (número de conflictos) de un vértice.
func (g *ConflictGraph) Degree(id int) int {
	return len(g.Adjacency[id])
}

// Neighbors retorna los IDs de los vecinos (conflictos) de un vértice.
func (g *ConflictGraph) Neighbors(id int) []int {
	var neighbors []int
	for n := range g.Adjacency[id] {
		neighbors = append(neighbors, n)
	}
	return neighbors
}

// NumVertices retorna el número de vértices.
func (g *ConflictGraph) NumVertices() int {
	return len(g.Vertices)
}

// NumEdges retorna el número de aristas (dividido por 2 porque es no dirigido).
func (g *ConflictGraph) NumEdges() int {
	total := 0
	for _, adj := range g.Adjacency {
		total += len(adj)
	}
	return total / 2
}

// BuildFromActivities construye el grafo a partir de una lista de actividades.
// Detecta conflictos por: mismo profesor o mismas secciones.
func BuildFromActivities(activities []domain.Activity) *ConflictGraph {
	g := New()

	// Agregar todos los vértices
	for i := range activities {
		g.AddVertex(&activities[i])
	}

	// Detectar conflictos (O(n²) pero necesario)
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			a1 := &activities[i]
			a2 := &activities[j]

			if areConflicting(a1, a2) {
				g.AddEdge(a1.ID, a2.ID)
			}
		}
	}

	return g
}

// areConflicting determina si dos actividades tienen conflicto hard.
func areConflicting(a1, a2 *domain.Activity) bool {
	// Conflicto 1: Comparten profesor (no puede estar en dos lugares)
	if a1.SharesTeacher(a2) {
		return true
	}

	// Conflicto 2: Comparten sección (mismos estudiantes)
	if a1.SharesSection(a2) {
		return true
	}

	return false
}
