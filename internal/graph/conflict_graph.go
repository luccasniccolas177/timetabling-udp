package graph

import (
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/utils"
)

// ConflictGraph representa el grafo de conflictos G = (V, E). donde los nodos/vertices son las actividades y las aristas representan
// conflictos entre las actividades
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

// Degree retorna el grado de un vértice, numero de vecinos
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

	// Detectar conflictos de profesores y secciones
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

// areConflicting determina si dos actividades tienen rompen una restricción dura.
func areConflicting(a1, a2 *domain.Activity) bool {
	// Comparten profesor
	if a1.SharesTeacher(a2) {
		return true
	}

	// Comparten sección (mismos estudiantes)
	if a1.SharesSection(a2) {
		return true
	}

	return false
}

// BuildFromActivitiesWithCliques construye el grafo incluyendo cliques por semestre. este clique que es un subgrafo completo se crea cuando
// en un solo semestre hay N cursos con una sola sección (o secciones fusionadas) y todos son de la misma carrera. no se toma en cuenta electivos
func BuildFromActivitiesWithCliques(
	activities []domain.Activity,
	planLocations map[string]map[string]int, // CourseCode -> Major -> Semester
	electives map[string]bool, // Set de cursos electivos (excluir de cliques)
) *ConflictGraph {
	g := New()

	// Agregar todos los vértices
	for i := range activities {
		g.AddVertex(&activities[i])
	}

	// Detectar conflictos
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			a1 := &activities[i]
			a2 := &activities[j]

			if areConflicting(a1, a2) {
				g.AddEdge(a1.ID, a2.ID)
			}
		}
	}

	// Contar secciones únicas por curso (secciones fusionadas cuentan como 1)
	courseSectionGroups := make(map[string]map[string]bool)
	courseActivities := make(map[string][]*domain.Activity)

	for i := range activities {
		a := &activities[i]
		if courseSectionGroups[a.CourseCode] == nil {
			courseSectionGroups[a.CourseCode] = make(map[string]bool)
		}

		// Crear id único para el grupo de secciones
		sectionGroupID := utils.SectionGroupKey(a.Sections)
		courseSectionGroups[a.CourseCode][sectionGroupID] = true
		courseActivities[a.CourseCode] = append(courseActivities[a.CourseCode], a)
	}

	// Identificar cursos con una sola sección/grupo
	singleSectionCourses := make(map[string]bool)
	for courseCode, groups := range courseSectionGroups {
		if len(groups) == 1 && !electives[courseCode] {
			singleSectionCourses[courseCode] = true
		}
	}

	// se agrupan los cursos por carrera y semestre
	semesterCourses := make(map[string]map[int][]string)

	for courseCode := range singleSectionCourses {
		if planLoc, ok := planLocations[courseCode]; ok {
			for major, semester := range planLoc {
				if semesterCourses[major] == nil {
					semesterCourses[major] = make(map[int][]string)
				}
				semesterCourses[major][semester] = append(
					semesterCourses[major][semester],
					courseCode,
				)
			}
		}
	}

	// Crear cliques para cada carrera y semestre
	for _, semesters := range semesterCourses {
		for _, courseCodes := range semesters {
			if len(courseCodes) < 2 {
				continue
			}

			// Recolectar todas las actividades de estos cursos
			var cliqueCourses []*domain.Activity
			for _, cc := range courseCodes {
				cliqueCourses = append(cliqueCourses, courseActivities[cc]...)
			}

			// Crear clique
			for i := 0; i < len(cliqueCourses); i++ {
				for j := i + 1; j < len(cliqueCourses); j++ {
					a1 := cliqueCourses[i]
					a2 := cliqueCourses[j]
					// Solo agregar si no existe ya (evitar duplicados)
					if !g.HasEdge(a1.ID, a2.ID) {
						g.AddEdge(a1.ID, a2.ID)
					}
				}
			}
		}
	}

	return g
}
