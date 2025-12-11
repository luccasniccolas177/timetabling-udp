package solver

import (
	"timetabling-UDP/internal/domain"
)

// Solution representa una solución al problema de timetabling
// Contiene el horario generado (asignación de colores/slots a sesiones)
type Solution struct {
	// Schedule: Color → Lista de sesiones asignadas a ese color
	// Cada color representa un bloque horario diferente
	Schedule map[int][]*domain.ClassSession

	// TotalColors: Número total de colores (bloques) necesarios
	TotalColors int
}

// NewSolution crea una nueva solución vacía
func NewSolution() *Solution {
	return &Solution{
		Schedule:    make(map[int][]*domain.ClassSession),
		TotalColors: 0,
	}
}

// IsBlockUsed verifica si un bloque ya tiene sesiones asignadas
func (s *Solution) IsBlockUsed(block int) bool {
	_, exists := s.Schedule[block]
	return exists && len(s.Schedule[block]) > 0
}

// HasConflictInBlock verifica si una sesión tiene conflictos con alguna sesión ya en el bloque
func (s *Solution) HasConflictInBlock(block int, session *domain.ClassSession, graph interface{}) bool {
	sessionsInBlock, exists := s.Schedule[block]
	if !exists || len(sessionsInBlock) == 0 {
		return false // Bloque vacío, no hay conflictos
	}

	// Necesitamos acceso al grafo para verificar aristas
	// Por ahora, asumimos que si hay sesiones en el bloque, verificamos conflictos básicos
	// Esto es una simplificación - idealmente deberíamos verificar aristas del grafo

	// Verificar conflictos básicos: mismo profesor, misma sección, etc.
	for _, existingSession := range sessionsInBlock {
		// Si comparten profesor, hay conflicto
		sessionTeachers := session.Class.GetTeachers()
		existingTeachers := existingSession.Class.GetTeachers()

		for _, t1 := range sessionTeachers {
			for _, t2 := range existingTeachers {
				if t1.ID == t2.ID {
					return true // Conflicto de profesor
				}
			}
		}

		// Si son de la misma clase (mismo ID), hay conflicto
		if session.Class.GetID() == existingSession.Class.GetID() {
			return true
		}
	}

	return false
}

// GetSessionsByColor retorna todas las sesiones asignadas a un color específico
func (s *Solution) GetSessionsByColor(color int) []*domain.ClassSession {
	return s.Schedule[color]
}

// IsFeasible verifica si la solución es factible
// Una solución es factible si usa ≤35 bloques (semana típica)
func (s *Solution) IsFeasible() bool {
	return s.TotalColors <= 35
}

// GetTotalSessions retorna el número total de sesiones en la solución
func (s *Solution) GetTotalSessions() int {
	total := 0
	for _, sessions := range s.Schedule {
		total += len(sessions)
	}
	return total
}
