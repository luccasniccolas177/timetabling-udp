package solver

import (
	"fmt"
	"sort"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
)

// RecolorDUDs intenta reasignar colores (bloques) a las sesiones en la lista DUD
// Busca bloques válidos (sin conflictos de grafo) que estén menos saturados
// Retorna el número de sesiones re-coloreadas exitosamente
func RecolorDUDs(solution *Solution, g *graph.ConflictGraph, dudList []*domain.ClassSession, maxColors int) int {
	fmt.Println("  🎨 Re-coloreando sesiones DUD...")

	recolored := 0

	// 1. Analizar uso de bloques para priorizar los menos usados
	blockUsage := make(map[int]int)
	for color := 1; color <= maxColors; color++ {
		blockUsage[color] = len(solution.Schedule[color])
	}

	// Ordenar colores por uso (ascendente)
	type colorUsage struct {
		color int
		count int
	}
	var sortedColors []colorUsage
	for color := 1; color <= maxColors; color++ {
		sortedColors = append(sortedColors, colorUsage{color, blockUsage[color]})
	}
	sort.Slice(sortedColors, func(i, j int) bool {
		return sortedColors[i].count < sortedColors[j].count
	})

	// 2. Intentar mover cada sesión DUD
	for _, session := range dudList {
		originalColor := session.Color
		moved := false

		// Probar colores desde el menos usado
		for _, usage := range sortedColors {
			newColor := usage.color

			// No mover al mismo color (ya sabemos que falla por falta de sala)
			if newColor == originalColor {
				continue
			}

			// Verificar validez del nuevo color (sin conflictos en el grafo)
			if isValidColor(session, newColor, g) {
				// Mover sesión
				moveSession(solution, session, originalColor, newColor)
				recolored++
				moved = true

				// Actualizar uso (simple heurística, no reordenamos todo)
				blockUsage[newColor]++
				blockUsage[originalColor]--
				break
			}
		}

		if !moved {
			// No se encontró color alternativo simple
			// Aquí se podría implementar estrategias más agresivas (swaps, kick-out)
			// Por ahora, se deja en el mismo color (fallará de nuevo o quizás tenga suerte si otros se movieron)
		}
	}

	fmt.Printf("  ✅ %d/%d sesiones re-coloreadas exitosamente\n", recolored, len(dudList))
	return recolored
}

// isValidColor verifica si una sesión puede ser asignada a un color sin violar restricciones duras del grafo
func isValidColor(session *domain.ClassSession, color int, g *graph.ConflictGraph) bool {
	// Verificar vecinos en el grafo de conflictos
	// Si algún vecino ya tiene este color asignado, es un conflicto

	nodeID := session.ID

	// Verificar si el nodo existe en el grafo
	if _, ok := g.Nodes[nodeID]; !ok {
		return false
	}

	// Verificar vecinos
	neighbors := g.AdjacencyList[nodeID]
	for neighborID := range neighbors {
		neighborSession := g.Nodes[neighborID]
		// Si el vecino tiene el color que queremos asignar, hay conflicto
		if neighborSession.Color == color {
			return false // Conflicto: vecino tiene el mismo color
		}
	}

	return true
}

// moveSession mueve una sesión de un color a otro en la solución
func moveSession(solution *Solution, session *domain.ClassSession, oldColor, newColor int) {
	// 1. Remover de la lista del color anterior
	oldList := solution.Schedule[oldColor]
	for i, s := range oldList {
		if s == session {
			// Eliminar preservando orden (o no importa)
			solution.Schedule[oldColor] = append(oldList[:i], oldList[i+1:]...)
			break
		}
	}

	// 2. Agregar a la lista del nuevo color
	solution.Schedule[newColor] = append(solution.Schedule[newColor], session)

	// 3. Actualizar la sesión
	session.Color = newColor
	session.AssignedSlot = domain.TimeSlot(newColor)
	session.AssignedRoom = nil // Resetear sala, ya que se movió y necesita nueva asignación
}
