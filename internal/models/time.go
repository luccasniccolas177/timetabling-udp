package models

import "fmt"

// Dimensiones de tiempo
const (
	DaysPerWeek  = 5
	BlocksPerDay = 7
	TotalSlots   = DaysPerWeek * DaysPerWeek
)

// Mapeo de índices a strings para visualización
var blockTimeStrings = [BlocksPerDay]string{
	"08:30 - 09:50", // Bloque 0
	"10:00 - 11:20", // Bloque 1
	"11:30 - 12:50", // Bloque 2
	"13:00 - 14:20", // Bloque 3
	"14:30 - 15:50", // Bloque 4
	"16:00 - 17:20", // Bloque 5
	"17:25 - 18:45", // Bloque 6
}

var dayStrings = [DaysPerWeek]string{
	"LUNES", "MARTES", "MIERCOLES", "JUEVES", "VIERNES",
}

type TimeSlot int

// ToHumanReadable convierte el ID del slot (0-34) a texto legible
func (t TimeSlot) ToHumanReadable() string {
	if t < 0 || int(t) >= TotalSlots {
		return "INVALID_SLOT"
	}
	dayIndex := int(t) / BlocksPerDay
	blockIndex := int(t) % BlocksPerDay

	return fmt.Sprintf("%s %s", dayStrings[dayIndex], blockTimeStrings[blockIndex])
}

// GetDayAndBlock retorna los índices separados (útil para validaciones)
func (t TimeSlot) GetDayAndBlock() (dayIndex int, blockIndex int) {
	return int(t) / BlocksPerDay, int(t) % BlocksPerDay
}

// Helper para crear un TimeSlot desde día y bloque
func NewTimeSlot(day int, block int) (TimeSlot, error) {
	if day < 0 || day >= DaysPerWeek || block < 0 || block >= BlocksPerDay {
		return -1, fmt.Errorf("dia o bloque fuera de rango")
	}
	return TimeSlot((day * BlocksPerDay) + block), nil
}
