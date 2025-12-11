package domain

// Course representa un curso universitario
// Ejemplo: CIT1000 - Programación
type Course struct {
	ID            int
	Code          string // CIT1000, CBM1001, etc.
	Name          string
	Curriculum    []CurriculumEntry
	Prerequisites []string
}

// CurriculumEntry vincula un curso con una carrera y semestre específico
// Un curso puede pertenecer a múltiples carreras/semestres
// Ejemplo: CBM1000 (Álgebra) está en CIT Semestre 1, CII Semestre 1, COC Semestre 1
type CurriculumEntry struct {
	Major    Major
	Semester int
}

// GetCurriculumKey retorna una clave única para buscar cursos por carrera/semestre
func (c *Course) GetCurriculumKey(major Major, semester int) string {
	return string(major) + "-" + string(rune(semester))
}

// BelongsToSemester verifica si el curso pertenece a una carrera/semestre específico
func (c *Course) BelongsToSemester(major Major, semester int) bool {
	for _, entry := range c.Curriculum {
		if entry.Major == major && entry.Semester == semester {
			return true
		}
	}
	return false
}
