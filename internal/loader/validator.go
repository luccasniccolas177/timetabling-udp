package loader

import (
	"fmt"
	"strings"
	"timetabling-UDP/internal/models"
)

// ValidationError agrupa múltiples errores encontrados durante la carga.
// Esto permite corregir todo de una vez en lugar de un error a la vez.
type ValidationError struct {
	Errors []string
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("Se encontraron %d errores de validación:\n- %s", len(v.Errors), strings.Join(v.Errors, "\n- "))
}

// ValidateState verifica la integridad lógica de los datos cargados
func ValidateState(state *UniversityState) error {
	var errs []string

	// 1. Validaciones de Existencia Básica
	if len(state.Courses) == 0 {
		errs = append(errs, "CRÍTICO: No se cargaron cursos.")
	}
	if len(state.Rooms) == 0 {
		errs = append(errs, "CRÍTICO: No se cargaron salas (rooms.csv).")
	}
	if len(state.Teachers) == 0 {
		errs = append(errs, "CRÍTICO: No se cargaron profesores.")
	}

	// 2. Pre-cálculo de capacidades máximas (Crucial para el Paper Sección 2.3)
	maxCapacityNormal := 0
	maxCapacityLab := 0

	for _, room := range state.Rooms {
		if room.RoomType == models.LR {
			if room.Capacity > maxCapacityLab {
				maxCapacityLab = room.Capacity
			}
		} else {
			if room.Capacity > maxCapacityNormal {
				maxCapacityNormal = room.Capacity
			}
		}
	}

	// 3. Validación de Secciones
	for _, section := range state.Sections {
		// Validar que la sección tenga alumnos
		if section.StudentsNumber <= 0 {
			courseName := state.Courses[section.CourseID].Name
			errs = append(errs, fmt.Sprintf("Sección %d del curso '%s' tiene 0 alumnos.", section.SectionNumber, courseName))
		}

		// Validar "Bin Packing constraint": El curso debe caber en ALGUNA sala de la universidad.
		// Si es un curso enorme, el algoritmo nunca encontrará solución.
		// Asumimos por defecto que cátedras van a salas normales.
		if section.StudentsNumber > maxCapacityNormal {
			courseName := state.Courses[section.CourseID].Name
			errs = append(errs, fmt.Sprintf("IMPOSIBLE FÍSICO: Sección %d de '%s' tiene %d alumnos, pero la sala más grande hace %d.",
				section.SectionNumber, courseName, section.StudentsNumber, maxCapacityNormal))
		}
	}

	// 4. Integridad Referencial de Cursos
	for _, course := range state.Courses {
		// Verificar que el curso tenga al menos una sección
		hasSections := false
		for _, s := range state.Sections {
			if s.CourseID == course.ID {
				hasSections = true
				break
			}
		}
		if !hasSections {
			// Esto es un warning, no un error fatal, pero ensucia el grafo.
			// errs = append(errs, fmt.Sprintf("Warning: Curso '%s' (%s) cargado pero sin secciones activas.", course.Name, course.Code))
		}
	}

	// 5. Validar Profesores (Integridad de Nombres)
	// Esto detecta si en oferta_academica.json hay un profe "Juan Perez"
	// que NO existe en profesores.json (o está escrito distinto "Juan A. Perez")
	teacherNames := make(map[string]bool)
	for _, t := range state.Teachers {
		teacherNames[strings.ToLower(strings.TrimSpace(t.Name))] = true
	}

	// NOTA: Como aún no procesamos los eventos a fondo en el loader,
	// esta validación se hará más fuerte cuando conectemos LogicalEvents.
	// Por ahora, validamos que la lista de profesores sea consistente en sí misma.
	for _, t := range state.Teachers {
		if t.Name == "" {
			errs = append(errs, fmt.Sprintf("Profesor ID %d no tiene nombre.", t.ID))
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	return nil
}
