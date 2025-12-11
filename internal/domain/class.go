package domain

// Class es la interfaz común para todos los tipos de clase
// Permite tratar Lecture, Tutorial y Lab de forma polimórfica
type Class interface {
	GetID() int
	GetCourse() *Course
	GetType() ClassType
	GetSections() []*Section
	GetDuration() int  // Bloques por sesión
	GetFrequency() int // Sesiones por semana
	GetTeachers() []*Teacher
	GetStudentCount() int
}

// =============================================================================
// LECTURE (Cátedra)
// =============================================================================

// Lecture representa una cátedra
// Características:
// - Compartida entre múltiples secciones
// - Puede tener 1, 2 o 3 clases por semana (Frequency)
// - Duración fija de 1 bloque por clase
// - Todos los alumnos de las secciones asisten juntos
type Lecture struct {
	ID        int
	Course    *Course
	Sections  []*Section // Secciones que comparten esta cátedra
	Number    int        // Número de cátedra (1, 2, etc.) para el mismo curso
	Frequency int        // 1, 2 o 3 clases por semana
	Teachers  []*Teacher
}

func (l *Lecture) GetID() int              { return l.ID }
func (l *Lecture) GetCourse() *Course      { return l.Course }
func (l *Lecture) GetType() ClassType      { return ClassTypeLecture }
func (l *Lecture) GetSections() []*Section { return l.Sections }
func (l *Lecture) GetDuration() int        { return 1 } // Siempre 1 bloque
func (l *Lecture) GetFrequency() int       { return l.Frequency }
func (l *Lecture) GetTeachers() []*Teacher { return l.Teachers }

func (l *Lecture) GetStudentCount() int {
	total := 0
	for _, section := range l.Sections {
		total += section.StudentCount
	}
	return total
}

// GetUniqueID retorna un identificador único para esta cátedra
// Formato: CIT1000-L1 (Lecture 1)
func (l *Lecture) GetUniqueID() string {
	return l.Course.Code + "-L" + string(rune('0'+l.Number))
}

// =============================================================================
// TUTORIAL (Ayudantía)
// =============================================================================

// Tutorial representa una ayudantía
// Características:
// - Compartida entre múltiples secciones
// - Siempre 1 clase por semana (Frequency = 1)
// - Duración fija de 1 bloque
// - Opcional (un curso puede no tener ayudantía)
type Tutorial struct {
	ID       int
	Course   *Course
	Sections []*Section // Secciones que comparten esta ayudantía
	Number   int        // Número de ayudantía (1, 2, etc.)
	Teachers []*Teacher
}

func (t *Tutorial) GetID() int              { return t.ID }
func (t *Tutorial) GetCourse() *Course      { return t.Course }
func (t *Tutorial) GetType() ClassType      { return ClassTypeTutorial }
func (t *Tutorial) GetSections() []*Section { return t.Sections }
func (t *Tutorial) GetDuration() int        { return 1 } // Siempre 1 bloque
func (t *Tutorial) GetFrequency() int       { return 1 } // Siempre 1 vez por semana
func (t *Tutorial) GetTeachers() []*Teacher { return t.Teachers }

func (t *Tutorial) GetStudentCount() int {
	total := 0
	for _, section := range t.Sections {
		total += section.StudentCount
	}
	return total
}

// GetUniqueID retorna un identificador único para esta ayudantía
// Formato: CIT1000-T1 (Tutorial 1)
func (t *Tutorial) GetUniqueID() string {
	return t.Course.Code + "-T" + string(rune('0'+t.Number))
}

// =============================================================================
// LAB (Laboratorio)
// =============================================================================

// Lab representa un laboratorio
// Características:
// - ESPECÍFICO de UNA sección (no compartido)
// - Siempre 1 clase por semana (Frequency = 1)
// - Duración VARIABLE (puede ser > 1 bloque)
// - Tiene restricciones de sala específicas
// - Opcional (un curso puede no tener laboratorio)
type Lab struct {
	ID              int
	Course          *Course
	Section         *Section // ⚠️ IMPORTANTE: Solo UNA sección
	Number          int      // Número de laboratorio (1, 2, etc.)
	Duration        int      // Bloques por sesión (puede ser > 1)
	Teachers        []*Teacher
	RoomConstraints []string // Salas específicas permitidas (ej: ["LAB MECANICA"])
}

func (l *Lab) GetID() int              { return l.ID }
func (l *Lab) GetCourse() *Course      { return l.Course }
func (l *Lab) GetType() ClassType      { return ClassTypeLab }
func (l *Lab) GetSections() []*Section { return []*Section{l.Section} }
func (l *Lab) GetDuration() int        { return l.Duration }
func (l *Lab) GetFrequency() int       { return 1 } // Siempre 1 vez por semana
func (l *Lab) GetTeachers() []*Teacher { return l.Teachers }
func (l *Lab) GetStudentCount() int    { return l.Section.StudentCount }

// GetUniqueID retorna un identificador único para este laboratorio
// Formato: CIT1000-LAB1 (Lab 1)
func (l *Lab) GetUniqueID() string {
	return l.Course.Code + "-LAB" + string(rune('0'+l.Number))
}

// RequiresMultipleBlocks indica si este laboratorio necesita bloques consecutivos
func (l *Lab) RequiresMultipleBlocks() bool {
	return l.Duration > 1
}
