package solver

import "timetabling-UDP/internal/models"

// Solution representa el resultado de la fase de coloreo.
type Solution struct {
	// Schedule: Mapa de Bloque Horario (Color) -> Lista de Eventos asignados
	// Key: Color Index (1, 2, 3...)
	// Value: Slice de eventos que ocurren simultáneamente
	Schedule map[int][]*models.EventInstance

	// TotalColors: Cantidad de bloques necesarios según el algoritmo
	TotalColors int
}

func NewSolution() *Solution {
	return &Solution{
		Schedule:    make(map[int][]*models.EventInstance),
		TotalColors: 0,
	}
}
