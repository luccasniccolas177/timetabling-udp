package models

// Constantes para los nombres de las carreras y tipos de salas
type Major string
type EventType string
type RoomType string

const (
	CIT Major = "CIVIL_INFORMATICA_TELECOMUNICACIONES"
	COC Major = "CIVIL_OBRAS_CIVILES"
	CII Major = "CIVIL_INDUSTRIAL"

	CAT EventType = "CATEDRA"
	AY  EventType = "AYUDANTIA"
	LAB EventType = "LABORATORIO"

	CR RoomType = "SALA"
	LR RoomType = "LABORATORIO"
)
