package domain

// Room representa una sala de clase
type Room struct {
	ID       int
	Code     string   // Código de la sala (ej: "101", "LAB D", "AUDITORIO 3")
	Capacity int      // Capacidad máxima de estudiantes
	Type     RoomType // Tipo de sala (SALA o LABORATORIO)
}

// CanAccommodate verifica si la sala puede acomodar un número de estudiantes
func (r *Room) CanAccommodate(studentCount int) bool {
	return r.Capacity >= studentCount
}

// IsLaboratory indica si esta sala es un laboratorio
func (r *Room) IsLaboratory() bool {
	return r.Type == RoomTypeLaboratory
}

// Teacher representa un profesor
type Teacher struct {
	ID   int
	Name string
	// Futuro: Agregar disponibilidad horaria
	// UnavailableSlots map[TimeSlot]bool
}
