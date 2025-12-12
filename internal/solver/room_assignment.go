package solver

import (
	"sort"

	"timetabling-UDP/internal/domain"
)

// RoomAssignment representa la asignación de actividades a salas para un periodo.
type RoomAssignment struct {
	RoomCode   string             // código de la sala
	Activities []*domain.Activity // actividades asignadas a esta sala
	Capacity   int                // capacidad total de la sala
	Used       int                // capacidad utilizada
}

// RoomAssignmentResult almacenara el resultado de la asignación de salas
type RoomAssignmentResult struct {
	Assignments []RoomAssignment   // asignaciones exitosas
	DUD         []*domain.Activity // actividades sin sala
}

// AssignRoomsToColorSet implementamos el Algoritmo 2 del paper.
// ordena actividades y salas por tamaño y asigna.
func AssignRoomsToColorSet(activities []*domain.Activity, rooms []domain.Room) RoomAssignmentResult {
	if len(activities) == 0 {
		return RoomAssignmentResult{}
	}

	sortedActivities := make([]*domain.Activity, len(activities))
	copy(sortedActivities, activities)
	sort.Slice(sortedActivities, func(i, j int) bool {
		return sortedActivities[i].Students < sortedActivities[j].Students
	})

	sortedRooms := make([]domain.Room, len(rooms))
	copy(sortedRooms, rooms)
	sort.Slice(sortedRooms, func(i, j int) bool {
		return sortedRooms[i].Capacity < sortedRooms[j].Capacity
	})

	assignments := make([]RoomAssignment, len(sortedRooms))
	for i, r := range sortedRooms {
		assignments[i] = RoomAssignment{
			RoomCode:   r.Code,
			Capacity:   r.Capacity,
			Activities: []*domain.Activity{},
			Used:       0,
		}
	}

	var dud []*domain.Activity

	for _, activity := range sortedActivities {
		placed := false

		// Buscar la sala más pequeña donde entre
		for j := range assignments {
			if len(assignments[j].Activities) == 0 && activity.Students <= assignments[j].Capacity {
				assignments[j].Activities = append(assignments[j].Activities, activity)
				assignments[j].Used = activity.Students
				activity.Room = assignments[j].RoomCode
				placed = true
				break
			}
		}

		// Si no se pudo colocar, va a DUD
		if !placed {
			dud = append(dud, activity)
		}
	}

	// Filtrar asignaciones vacías
	var nonEmptyAssignments []RoomAssignment
	for _, a := range assignments {
		if len(a.Activities) > 0 {
			nonEmptyAssignments = append(nonEmptyAssignments, a)
		}
	}

	return RoomAssignmentResult{
		Assignments: nonEmptyAssignments,
		DUD:         dud,
	}
}

// GetRoomsByType filtra salas por tipo (SALA o LABORATORIO).
func GetRoomsByType(rooms []domain.Room, roomType domain.RoomType) []domain.Room {
	var filtered []domain.Room
	for _, r := range rooms {
		if r.Type == roomType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
