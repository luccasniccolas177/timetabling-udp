package domain

// Section representa una sección específica de un curso
// Ejemplo: CIT1000 Sección 1, CIT1000 Sección 2
//
// Relaciones clave:
// - Múltiples secciones comparten la misma Cátedra (Lecture)
// - Múltiples secciones comparten la misma Ayudantía (Tutorial)
// - Cada sección tiene su propio Laboratorio (Lab) - si aplica
type Section struct {
	ID           int
	Course       *Course
	Number       int // Número de sección (1, 2, 3, etc.)
	StudentCount int

	// Relaciones con clases
	// Estas se establecen después de cargar los datos
	SharedLecture  *Lecture  // Cátedra compartida con otras secciones
	SharedTutorial *Tutorial // Ayudantía compartida (puede ser nil)
	OwnLab         *Lab      // Laboratorio propio (puede ser nil)
}

// GetAllClasses retorna todas las clases asociadas a esta sección
// Útil para iterar sobre todas las clases que un alumno de esta sección debe tomar
func (s *Section) GetAllClasses() []Class {
	classes := make([]Class, 0, 3)

	if s.SharedLecture != nil {
		classes = append(classes, s.SharedLecture)
	}

	if s.SharedTutorial != nil {
		classes = append(classes, s.SharedTutorial)
	}

	if s.OwnLab != nil {
		classes = append(classes, s.OwnLab)
	}

	return classes
}

// GetClassSessions retorna todas las sesiones de clase para esta sección
// Esto incluye todas las instancias semanales de cada clase
func (s *Section) GetClassSessions() []*ClassSession {
	sessions := make([]*ClassSession, 0)

	for _, class := range s.GetAllClasses() {
		classSessions := GenerateSessions(class)
		sessions = append(sessions, classSessions...)
	}

	return sessions
}

// GetFullName retorna el nombre completo de la sección
// Ejemplo: "CIT1000 - Programación (Sección 1)"
func (s *Section) GetFullName() string {
	return s.Course.Code + " - " + s.Course.Name + " (Sección " + string(rune(s.Number)) + ")"
}
