package solver

import (
	"sort"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
)

// Period representa un bloque completo con actividades y sus salas asignadas.
type Period struct {
	Number      int                // Número de periodo (0-based)
	Block       int                // Bloque temporal asignado
	Assignments []RoomAssignment   // Asignaciones de salas
	Unassigned  []*domain.Activity // DUD local de este periodo
}

// TimetableResult es el resultado del algoritmo integrado.
type TimetableResult struct {
	Periods      []Period           // bloques programados
	FinalDUD     []*domain.Activity // Actividades que no pudieron ser programadas
	TotalPeriods int                // Total de periodos usados
}

// IntegratedSchedulerWithConstraints implementa el Algoritmo Integrado con restricciones de salas.
// Recibe el grafo ya construido (con cliques) para no reconstruirlo.
func IntegratedSchedulerWithConstraints(activities []domain.Activity, G *graph.ConflictGraph, rooms []domain.Room, constraints loader.RoomConstraints) TimetableResult {
	// Separar salas por tipo
	classrooms := GetRoomsByType(rooms, domain.RoomClassroom)
	labs := GetRoomsByType(rooms, domain.RoomLab)
	allRooms := append(classrooms, labs...)

	// El grafo G ya viene construido desde main (con cliques)

	var periods []Period
	periodNum := 0
	blockNum := 0 // Bloque temporal real (0-34), puede saltar el protegido

	// Mientras queden vértices en el grafo
	for G.NumVertices() > 0 && blockNum < domain.TotalBlocks {
		// Saltar el bloque protegido del miércoles (11:30-12:50)
		if domain.IsProtectedBlock(blockNum) {
			blockNum++
			continue
		}

		colorSet := findMaxIndependentSet(G)

		if len(colorSet) == 0 {
			break
		}

		// Obtener actividades del colorSet
		var periodActivities []*domain.Activity
		for _, id := range colorSet {
			periodActivities = append(periodActivities, G.Vertices[id])
		}

		// Asignar salas usando Algoritmo 2 CON restricciones
		period := assignRoomsToPeriodWithConstraints(periodActivities, allRooms, constraints, blockNum)

		periods = append(periods, period)

		// Eliminar vértices asignados exitosamente
		for _, ra := range period.Assignments {
			for _, a := range ra.Activities {
				removeVertex(G, a.ID)
				a.Block = blockNum // Usar bloque real, no periodNum
			}
		}

		periodNum++
		blockNum++
	}

	// DUD final
	var finalDUD []*domain.Activity
	for _, a := range G.Vertices {
		finalDUD = append(finalDUD, a)
	}

	return TimetableResult{
		Periods:      periods,
		FinalDUD:     finalDUD,
		TotalPeriods: len(periods),
	}
}

// IntegratedScheduler versión sin restricciones (legacy).
func IntegratedScheduler(activities []domain.Activity, rooms []domain.Room) TimetableResult {
	G := graph.BuildFromActivities(activities)
	return IntegratedSchedulerWithConstraints(activities, G, rooms, nil)
}

// assignRoomsToPeriodWithConstraints asigna salas respetando restricciones.
func assignRoomsToPeriodWithConstraints(activities []*domain.Activity, rooms []domain.Room, constraints loader.RoomConstraints, periodNum int) Period {
	var allAssignments []RoomAssignment
	var allDUD []*domain.Activity

	// Agrupar actividades por restricción de sala
	roomAvailability := make(map[string]bool) // Salas disponibles en este periodo
	for _, r := range rooms {
		roomAvailability[r.Code] = true
	}

	// Procesar cada actividad individualmente respetando su restricción
	for _, activity := range activities {
		// Obtener salas permitidas para esta actividad
		eventType := eventTypeToString(activity.Type)
		allowedCodes := constraints.GetAllowedRooms(activity.CourseCode, eventType)

		// Filtrar salas permitidas que estén disponibles
		var availableRooms []domain.Room
		for _, r := range rooms {
			if !roomAvailability[r.Code] {
				continue // Ya usada en este periodo
			}

			// Si hay restricción explícita, usar solo esas salas
			if allowedCodes != nil {
				if contains(allowedCodes, r.Code) {
					availableRooms = append(availableRooms, r)
				}
			} else {
				// Sin restricción explícita: respetar tipo de sala
				// CAT/AY → solo aulas normales, LAB → solo laboratorios
				if activity.Type == domain.LAB {
					// Laboratorios solo pueden ir a salas tipo LAB
					if r.Type == domain.RoomLab {
						availableRooms = append(availableRooms, r)
					}
				} else {
					// Cátedras y Ayudantías solo pueden ir a aulas normales
					if r.Type == domain.RoomClassroom {
						availableRooms = append(availableRooms, r)
					}
				}
			}
		}

		// Ordenar por capacidad y buscar sala adecuada
		sort.Slice(availableRooms, func(i, j int) bool {
			return availableRooms[i].Capacity < availableRooms[j].Capacity
		})

		assigned := false
		for _, room := range availableRooms {
			if activity.Students <= room.Capacity {
				activity.Room = room.Code
				roomAvailability[room.Code] = false // Marcar como usada
				allAssignments = append(allAssignments, RoomAssignment{
					RoomCode:   room.Code,
					Capacity:   room.Capacity,
					Activities: []*domain.Activity{activity},
					Used:       activity.Students,
				})
				assigned = true
				break
			}
		}

		if !assigned {
			allDUD = append(allDUD, activity)
		}
	}

	return Period{
		Number:      periodNum,
		Block:       periodNum % domain.TotalBlocks,
		Assignments: allAssignments,
		Unassigned:  allDUD,
	}
}

// eventTypeToString convierte EventCategory a string para buscar en constraints.
func eventTypeToString(t domain.EventCategory) string {
	switch t {
	case domain.CAT:
		return "CATEDRA"
	case domain.AY:
		return "AYUDANTIA"
	case domain.LAB:
		return "LABORATORIO"
	default:
		return "CATEDRA"
	}
}

// contains verifica si un slice contiene un string.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
