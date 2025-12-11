package solver

import (
	"sort"

	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/graph"
	"timetabling-UDP/internal/loader"
)

// Period representa un periodo completo con actividades y sus salas asignadas.
type Period struct {
	Number      int                // Número de periodo (0-based)
	Block       int                // Bloque temporal asignado
	Assignments []RoomAssignment   // Asignaciones de salas
	Unassigned  []*domain.Activity // DUD local de este periodo
}

// TimetableResult es el resultado del algoritmo integrado.
type TimetableResult struct {
	Periods      []Period           // Periodos programados
	FinalDUD     []*domain.Activity // Actividades que no pudieron ser programadas
	TotalPeriods int                // Total de periodos usados
}

// IntegratedSchedulerWithConstraints implementa el Algoritmo Integrado con restricciones de salas.
func IntegratedSchedulerWithConstraints(activities []domain.Activity, rooms []domain.Room, constraints loader.RoomConstraints) TimetableResult {
	// Separar salas por tipo
	classrooms := GetRoomsByType(rooms, domain.RoomClassroom)
	labs := GetRoomsByType(rooms, domain.RoomLab)
	allRooms := append(classrooms, labs...)

	// Crear grafo de conflictos mutable
	G := graph.BuildFromActivities(activities)

	var periods []Period
	periodNum := 0

	// Mientras queden vértices en el grafo
	for G.NumVertices() > 0 && periodNum < domain.TotalBlocks {
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
		period := assignRoomsToPeriodWithConstraints(periodActivities, allRooms, constraints, periodNum)

		periods = append(periods, period)

		// Eliminar vértices asignados exitosamente
		for _, ra := range period.Assignments {
			for _, a := range ra.Activities {
				removeVertex(G, a.ID)
				a.Block = periodNum
			}
		}

		periodNum++
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
	return IntegratedSchedulerWithConstraints(activities, rooms, nil)
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
			if allowedCodes == nil || contains(allowedCodes, r.Code) {
				availableRooms = append(availableRooms, r)
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

// assignCoursesToRooms implementa el Algoritmo 2 para CURSOS (Section 3.1).
// Regla: 1 curso por sala, con desplazamiento a salas más grandes.
func assignCoursesToRooms(courses []*domain.Activity, rooms []domain.Room) RoomAssignmentResult {
	if len(courses) == 0 || len(rooms) == 0 {
		return RoomAssignmentResult{DUD: courses}
	}

	// Paso 1: Ordenar cursos por tamaño (estudiantes), menor primero
	sortedCourses := make([]*domain.Activity, len(courses))
	copy(sortedCourses, courses)
	sort.Slice(sortedCourses, func(i, j int) bool {
		return sortedCourses[i].Students < sortedCourses[j].Students
	})

	// Paso 1: Ordenar salas por capacidad, menor primero
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
			Activities: nil,
			Used:       0,
		}
	}

	// Pasos 2-3: Colocar cada curso en la sala más pequeña donde quepa
	for _, course := range sortedCourses {
		for j := range assignments {
			if course.Students <= assignments[j].Capacity {
				assignments[j].Activities = append(assignments[j].Activities, course)
				assignments[j].Used += course.Students
				break
			}
		}
	}

	// Pasos 4-6: Desplazar para que quede máximo 1 curso por sala
	var dud []*domain.Activity
	for j := range assignments {
		if len(assignments[j].Activities) <= 1 {
			continue
		}

		// Ordenar por tamaño, menor primero
		sort.Slice(assignments[j].Activities, func(i, k int) bool {
			return assignments[j].Activities[i].Students < assignments[j].Activities[k].Students
		})

		// Encontrar el curso más pequeño que NO cabe en la sala anterior
		keepIdx := -1
		for i, a := range assignments[j].Activities {
			if j == 0 || a.Students > assignments[j-1].Capacity {
				keepIdx = i
				break
			}
		}

		// Si ninguno cumple, quedarse con el más pequeño
		if keepIdx == -1 {
			keepIdx = 0
		}

		// Mover los demás a la siguiente sala o DUD
		for i, a := range assignments[j].Activities {
			if i == keepIdx {
				continue
			}

			moved := false
			if j+1 < len(assignments) {
				nextRoom := &assignments[j+1]
				// Agregar a la siguiente sala (se resolverá en la siguiente iteración)
				nextRoom.Activities = append(nextRoom.Activities, a)
				nextRoom.Used += a.Students
				moved = true
			}

			if !moved {
				dud = append(dud, a)
			}
		}

		// Mantener solo el curso seleccionado
		kept := assignments[j].Activities[keepIdx]
		assignments[j].Activities = []*domain.Activity{kept}
		assignments[j].Used = kept.Students
		kept.Room = assignments[j].RoomCode
	}

	// Filtrar asignaciones válidas y asignar Room a las actividades
	var validAssignments []RoomAssignment
	for _, a := range assignments {
		if len(a.Activities) == 1 {
			a.Activities[0].Room = a.RoomCode
			validAssignments = append(validAssignments, a)
		} else if len(a.Activities) > 1 {
			// Esto no debería pasar después del desplazamiento
			// pero por seguridad, solo quedarnos con el primero
			a.Activities[0].Room = a.RoomCode
			a.Activities = a.Activities[:1]
			validAssignments = append(validAssignments, a)
			dud = append(dud, a.Activities[1:]...)
		}
	}

	return RoomAssignmentResult{
		Assignments: validAssignments,
		DUD:         dud,
	}
}
