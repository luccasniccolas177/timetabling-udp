package domain

import "time"

type Major string         // carreras de la FIC
type RoomType string      // tipo de sala
type EventCategory string // tipo de actividad (catedra, ayudantia, laboratorio)

const (
	// Carreras
	EIT Major = "CIVIL_INFORMATICA_TELECOMUNICACIONES"
	EOC Major = "CIVIL_OBRAS_CIVILES"
	IND Major = "CIVIL_INDUSTRIAL"

	// Categorías
	CAT EventCategory = "CATEDRA"
	AY  EventCategory = "AYUDANTIA"
	LAB EventCategory = "LABORATORIO"

	// Tipos de Sala
	RoomClassroom RoomType = "SALA"
	RoomLab       RoomType = "LABORATORIO"
)

const (
	// definición de bloques horarios
	BlocksPerDay = 7
	DaysPerWeek  = 5
	TotalBlocks  = BlocksPerDay * DaysPerWeek // 35 bloques

	// Duración clases (80 minutos)
	BlockDuration = 80 * time.Minute

	// miercoles 11:30-12:50 horario protegido
	ProtectedWednesdayBlock = 2*BlocksPerDay + 2 // = 16
)

// IsProtectedBlock verifica si un bloque es el horario protegido
func IsProtectedBlock(block int) bool {
	return block == ProtectedWednesdayBlock
}

// OccupiesProtectedBlock verifica si una actividad ocupa el bloque protegido
// especial para el caso de clases que duran más de un bloque
func OccupiesProtectedBlock(startBlock, duration int) bool {
	if duration < 1 {
		duration = 1
	}
	endBlock := startBlock + duration - 1

	return startBlock <= ProtectedWednesdayBlock && ProtectedWednesdayBlock <= endBlock
}

// valores de penalización para SA

const (
	PenaltyHard   = 100000 // Restricciones duras (profesor en 2 lugares)
	PenaltyMedium = 1000   // Preferible
	PenaltySoft   = 10     // Preferencia leve
)
