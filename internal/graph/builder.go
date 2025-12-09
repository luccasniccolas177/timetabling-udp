package graph

import (
	"fmt"
	"timetabling-UDP/internal/loader"
	"timetabling-UDP/internal/models"
)

// BuildConflictGraph transforma los datos de la universidad en un grafo matemático G(V, E)
func BuildConflictGraph(state *loader.UniversityState) *ConflictGraph {
	// 1. Inicializar grafo vacío
	g := NewConflictGraph()

	// 2. FASE DE NODOS
	var allNodes []*models.EventInstance

	for i := range state.RawEvents {
		lEvent := &state.RawEvents[i]

		// Validación de frecuencia
		freq := lEvent.Frequency
		if freq < 1 {
			freq = 1
		}

		// Prefijo para UUID
		var ePrefix string
		switch lEvent.Type {
		case models.CAT:
			ePrefix = "C"
		case models.LAB:
			ePrefix = "L"
		case models.AY:
			ePrefix = "A"
		default:
			ePrefix = "E"
		}

		// Expansión por frecuencia
		for k := 1; k <= freq; k++ {
			// Recuperar código del curso para el UUID
			courseCode := state.Courses[lEvent.CourseID].Code
			uuid := fmt.Sprintf("%s-%s%d-I%d", courseCode, ePrefix, lEvent.EventNumber, k)

			instance := &models.EventInstance{
				UUID:           uuid,
				LogicalEventID: lEvent.ID,
				Index:          k,
				Data:           lEvent,
				AssignedSlot:   -1,
				Color:          0,
			}

			g.AddNode(instance)
			allNodes = append(allNodes, instance)
		}
	}

	// 3. FASE DE ARISTAS: Detectar Conflictos (Hard Constraints)

	// A. Conflictos de Profesor (El recurso humano es indivisible)
	teacherBuckets := make(map[int][]*models.EventInstance)

	// B. Conflictos de Mismo Evento (Coherencia interna del curso)
	// La Clase 1 y Clase 2 de la misma sección no pueden ser simultáneas.
	sameEventBuckets := make(map[int][]*models.EventInstance)

	// [ELIMINADO] C. Conflictos de Malla (Semestre) - Desactivado a petición
	// curriculumBuckets := make(map[string][]*models.EventInstance)

	// Llenado de Buckets
	for _, node := range allNodes {
		// Bucket Profesor
		for _, teacherID := range node.Data.TeachersIDs {
			teacherBuckets[teacherID] = append(teacherBuckets[teacherID], node)
		}

		// Bucket Mismo Evento (Por ID lógico)
		sameEventBuckets[node.LogicalEventID] = append(sameEventBuckets[node.LogicalEventID], node)

		/* // [DESACTIVADO] Bucket Malla
		course := state.Courses[node.Data.CourseID]
		for _, req := range course.Requirements {
			key := fmt.Sprintf("%s-%d", req.Major, req.Semester)
			curriculumBuckets[key] = append(curriculumBuckets[key], node)
		}
		*/
	}

	// 4. GENERAR ARISTAS (Conectar todos contra todos en cada bucket)

	// A. Profesores
	for _, nodes := range teacherBuckets {
		connectAllInClique(g, nodes)
	}

	// B. Mismo Evento
	for _, nodes := range sameEventBuckets {
		connectAllInClique(g, nodes)
	}

	/*

		for _, nodes := range curriculumBuckets {
			connectAllInClique(g, nodes)
		}
	*/

	return g
}

// connectAllInClique agrega aristas entre todos los pares de nodos en la lista
func connectAllInClique(g *ConflictGraph, nodes []*models.EventInstance) {
	if len(nodes) < 2 {
		return
	}
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			g.AddEdge(nodes[i].UUID, nodes[j].UUID)
		}
	}
}
