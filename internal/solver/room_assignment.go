package solver

import (
	"sort"

	"timetabling-UDP/internal/domain"
)

// RoomAssignment representa la asignación de actividades a salas para un periodo.
type RoomAssignment struct {
	RoomCode   string             // Código de la sala
	Activities []*domain.Activity // Actividades asignadas a esta sala
	Capacity   int                // Capacidad total de la sala
	Used       int                // Capacidad utilizada
}

// RoomAssignmentResult es el resultado del algoritmo de asignación de salas.
type RoomAssignmentResult struct {
	Assignments []RoomAssignment   // Asignaciones exitosas
	DUD         []*domain.Activity // Actividades sin sala (Displaced Unassigned Duties)
}

// AssignRoomsToColorSet implementa el Algoritmo 2 del paper.
// Para CURSOS (no exámenes): generalmente 1 actividad por sala.
// Ordena actividades y salas por tamaño (menor primero) y asigna.
func AssignRoomsToColorSet(activities []*domain.Activity, rooms []domain.Room) RoomAssignmentResult {
	if len(activities) == 0 {
		return RoomAssignmentResult{}
	}

	// Paso 1: Ordenar actividades por tamaño (estudiantes), menor primero
	sortedActivities := make([]*domain.Activity, len(activities))
	copy(sortedActivities, activities)
	sort.Slice(sortedActivities, func(i, j int) bool {
		return sortedActivities[i].Students < sortedActivities[j].Students
	})

	// Paso 2: Ordenar salas por capacidad, menor primero
	sortedRooms := make([]domain.Room, len(rooms))
	copy(sortedRooms, rooms)
	sort.Slice(sortedRooms, func(i, j int) bool {
		return sortedRooms[i].Capacity < sortedRooms[j].Capacity
	})

	// Inicializar asignaciones (una por sala)
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

	// Paso 3: Para cada actividad, buscar sala
	for _, activity := range sortedActivities {
		placed := false

		// Buscar la sala más pequeña donde quepa (1 actividad por sala para cursos)
		for j := range assignments {
			// Para cursos: 1 actividad por sala
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

// AssignRoomsToColorSetShared implementa el Algoritmo 2 para EXÁMENES.
// Permite múltiples actividades por sala si caben por capacidad.
func AssignRoomsToColorSetShared(activities []*domain.Activity, rooms []domain.Room) RoomAssignmentResult {
	if len(activities) == 0 {
		return RoomAssignmentResult{}
	}

	// Ordenar actividades por tamaño, menor primero
	sortedActivities := make([]*domain.Activity, len(activities))
	copy(sortedActivities, activities)
	sort.Slice(sortedActivities, func(i, j int) bool {
		return sortedActivities[i].Students < sortedActivities[j].Students
	})

	// Ordenar salas por capacidad, menor primero
	sortedRooms := make([]domain.Room, len(rooms))
	copy(sortedRooms, rooms)
	sort.Slice(sortedRooms, func(i, j int) bool {
		return sortedRooms[i].Capacity < sortedRooms[j].Capacity
	})

	// Inicializar asignaciones
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

	// Para cada actividad
	for _, activity := range sortedActivities {
		placed := false

		// Buscar sala donde quepa (puede compartir)
		for j := range assignments {
			remainingCapacity := assignments[j].Capacity - assignments[j].Used
			if activity.Students <= remainingCapacity {
				assignments[j].Activities = append(assignments[j].Activities, activity)
				assignments[j].Used += activity.Students
				activity.Room = assignments[j].RoomCode
				placed = true
				break
			}
		}

		// Si no cabe en ninguna, intentar desplazamiento
		if !placed {
			placed = tryDisplacement(activity, assignments, sortedRooms)
		}

		// Si aún no se pudo, va a DUD
		if !placed {
			dud = append(dud, activity)
		}
	}

	// Filtrar vacías
	var nonEmpty []RoomAssignment
	for _, a := range assignments {
		if len(a.Activities) > 0 {
			nonEmpty = append(nonEmpty, a)
		}
	}

	return RoomAssignmentResult{
		Assignments: nonEmpty,
		DUD:         dud,
	}
}

// tryDisplacement intenta desplazar actividades a salas más grandes.
// Implementa el paso 4 del algoritmo del paper.
func tryDisplacement(newActivity *domain.Activity, assignments []RoomAssignment, rooms []domain.Room) bool {
	// Para cada sala desde la más pequeña
	for j := range assignments {
		// Si la nueva actividad cabe sola en esta sala
		if newActivity.Students <= assignments[j].Capacity {
			// Calcular cuánto espacio necesitamos liberar
			currentUsed := assignments[j].Used
			totalNeeded := currentUsed + newActivity.Students
			overflow := totalNeeded - assignments[j].Capacity

			if overflow <= 0 {
				// Ya cabe, agregar directamente
				assignments[j].Activities = append(assignments[j].Activities, newActivity)
				assignments[j].Used += newActivity.Students
				newActivity.Room = assignments[j].RoomCode
				return true
			}

			// Intentar desplazar actividades pequeñas a la siguiente sala
			if j+1 < len(assignments) {
				displaced := displaceSmallest(assignments, j, overflow)
				if displaced {
					assignments[j].Activities = append(assignments[j].Activities, newActivity)
					assignments[j].Used += newActivity.Students
					newActivity.Room = assignments[j].RoomCode
					return true
				}
			}
		}
	}
	return false
}

// displaceSmallest desplaza las actividades más pequeñas desde roomIdx a roomIdx+1.
func displaceSmallest(assignments []RoomAssignment, roomIdx, overflow int) bool {
	if roomIdx+1 >= len(assignments) {
		return false
	}

	source := &assignments[roomIdx]
	target := &assignments[roomIdx+1]

	// Ordenar actividades en la sala por tamaño
	sort.Slice(source.Activities, func(i, j int) bool {
		return source.Activities[i].Students < source.Activities[j].Students
	})

	// Desplazar las más pequeñas hasta liberar suficiente espacio
	freedSpace := 0
	var toMove []*domain.Activity

	for i := 0; i < len(source.Activities) && freedSpace < overflow; i++ {
		a := source.Activities[i]
		// Verificar si cabe en la sala destino
		if target.Used+a.Students <= target.Capacity {
			toMove = append(toMove, a)
			freedSpace += a.Students
		}
	}

	if freedSpace >= overflow {
		// Realizar el desplazamiento
		for _, a := range toMove {
			// Quitar de source
			for k, act := range source.Activities {
				if act.ID == a.ID {
					source.Activities = append(source.Activities[:k], source.Activities[k+1:]...)
					source.Used -= a.Students
					break
				}
			}
			// Agregar a target
			target.Activities = append(target.Activities, a)
			target.Used += a.Students
			a.Room = target.RoomCode
		}
		return true
	}

	return false
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
