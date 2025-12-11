package domain

import "fmt"

// ClassSession representa una sesión concreta de una clase
// Ejemplo: Cátedra 1, Instancia del Lunes (si la cátedra tiene 3 clases semanales)
//
// Relación con Class:
// - Una Lecture con Frequency=3 genera 3 ClassSessions (Lunes, Miércoles, Viernes)
// - Una Tutorial con Frequency=1 genera 1 ClassSession
// - Un Lab con Frequency=1 genera 1 ClassSession
//
// Este es el equivalente al antiguo "EventInstance"
type ClassSession struct {
	ID           string // UUID único: CIT1000-L1-W1 (Lecture 1, Week instance 1)
	Class        Class  // Referencia a la clase (Lecture, Tutorial o Lab)
	WeekInstance int    // 1, 2 o 3 (para clases con frecuencia > 1)

	// Asignación (resultado del solver)
	AssignedSlot TimeSlot // Bloque horario asignado (-1 si no asignado)
	AssignedRoom *Room    // Sala asignada (nil si no asignado)
	Color        int      // Color asignado por el algoritmo de coloración
}

// GetCourse retorna el curso al que pertenece esta sesión
func (cs *ClassSession) GetCourse() *Course {
	return cs.Class.GetCourse()
}

// GetType retorna el tipo de clase (Lecture, Tutorial, Lab)
func (cs *ClassSession) GetType() ClassType {
	return cs.Class.GetType()
}

// GetSections retorna las secciones asociadas a esta sesión
func (cs *ClassSession) GetSections() []*Section {
	return cs.Class.GetSections()
}

// IsAssigned indica si esta sesión ya tiene slot y sala asignados
func (cs *ClassSession) IsAssigned() bool {
	return cs.AssignedSlot != TimeSlotUnassigned && cs.AssignedRoom != nil
}

// HasTimeSlot indica si esta sesión tiene un slot asignado
func (cs *ClassSession) HasTimeSlot() bool {
	return cs.AssignedSlot != TimeSlotUnassigned
}

// GenerateSessions crea todas las sesiones para una clase dada
// Ejemplo: Una Lecture con Frequency=3 genera 3 ClassSessions
func GenerateSessions(class Class) []*ClassSession {
	frequency := class.GetFrequency()
	sessions := make([]*ClassSession, 0, frequency)

	for i := 1; i <= frequency; i++ {
		session := &ClassSession{
			ID:           generateSessionID(class, i),
			Class:        class,
			WeekInstance: i,
			AssignedSlot: TimeSlotUnassigned,
			AssignedRoom: nil,
			Color:        0,
		}
		sessions = append(sessions, session)
	}

	return sessions
}

// generateSessionID genera un ID único para una sesión
// Formato: CODIGO-TIPO-NUMERO-WX
// Ejemplos:
//   - CIT1000-L1-W1 (Lecture 1, Week 1)
//   - CIT1000-L1-W2 (Lecture 1, Week 2)
//   - CIT1000-T1-W1 (Tutorial 1, Week 1)
//   - CIT1000-LAB1-W1 (Lab 1, Week 1)
func generateSessionID(class Class, weekInstance int) string {
	course := class.GetCourse()

	var typePrefix string
	var number int

	switch c := class.(type) {
	case *Lecture:
		typePrefix = "L"
		number = c.Number
	case *Tutorial:
		typePrefix = "T"
		number = c.Number
	case *Lab:
		typePrefix = "LAB"
		number = c.Number
	default:
		typePrefix = "X"
		number = 0
	}

	return fmt.Sprintf("%s-%s%d-W%d", course.Code, typePrefix, number, weekInstance)
}
