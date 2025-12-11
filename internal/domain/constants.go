package domain

import "time"

// --- TUS TIPOS EXISTENTES ---
type Major string
type RoomType string
type EventCategory string

// --- TUS CONSTANTES EXISTENTES ---
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
	// Definición de la grilla horaria
	BlocksPerDay = 7
	DaysPerWeek  = 5
	TotalBlocks  = BlocksPerDay * DaysPerWeek // 35 bloques

	// Duración base (referencia)
	BlockDuration = 80 * time.Minute
)

// Estos valores ayudan al algoritmo a decidir qué es "grave" y qué es "aceptable".

const (
	PenaltyHard   = 100000 // Inviolable (ej: Profesor en dos lugares)
	PenaltyMedium = 1000   // Preferible evitar (ej: Ventanas de 3 horas)
	PenaltySoft   = 10     // Preferencia leve
)
