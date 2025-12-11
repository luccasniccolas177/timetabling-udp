package domain

// Major representa una carrera universitaria
type Major string

const (
	MajorCIT Major = "CIVIL_INFORMATICA_TELECOMUNICACIONES"
	MajorCOC Major = "CIVIL_OBRAS_CIVILES"
	MajorCII Major = "CIVIL_INDUSTRIAL"
)

// ClassType representa el tipo de clase
type ClassType string

const (
	ClassTypeLecture  ClassType = "CATEDRA"
	ClassTypeTutorial ClassType = "AYUDANTIA"
	ClassTypeLab      ClassType = "LABORATORIO"
)

// RoomType representa el tipo de sala
type RoomType string

const (
	RoomTypeClassroom  RoomType = "SALA"
	RoomTypeLaboratory RoomType = "LABORATORIO"
)

// TimeSlot representa un bloque horario asignado
// -1 significa no asignado
type TimeSlot int

const (
	TimeSlotUnassigned TimeSlot = -1
)
