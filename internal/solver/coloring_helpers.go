package solver

import "timetabling-UDP/internal/domain"

// findMaxUsedColor encuentra el color máximo usado en la solución
func findMaxUsedColor(solution *Solution) int {
	maxColor := 0
	for color := range solution.Schedule {
		if color > maxColor {
			maxColor = color
		}
	}
	return maxColor
}

// Helper para verificar si una sesión es ayudantía
func isTutorial(session *domain.ClassSession) bool {
	return session.GetType() == domain.ClassTypeTutorial
}
