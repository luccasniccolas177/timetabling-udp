package domain

// Course define una asignatura en el plan de estudios.
type Course struct {
	ID            int
	Code          string        // Código de curso (CBF1000)
	Name          string        // Nombre de la asignatura (mecanica)
	Prerequisites []int         // IDs de cursos prerequisito
	PlanLocation  map[Major]int // carrera y semestre en el que se debe tomar
	Distribution  Distribution  // Carga académica del curso
	IsElective    bool          // Si es un electivo
}

// Distribution define la carga semanal de un curso.
type Distribution struct {
	NumCAT      int // Cantidad de cátedras por semana
	NumAY       int // Cantidad de ayudantías por semana
	NumLAB      int // Cantidad de laboratorios por semana
	DurationCAT int // Duración en bloques de cada cátedra
	DurationAY  int // Duración en bloques de cada ayudantía
	DurationLAB int // Duración en bloques de cada laboratorio
}

// Teacher representa a un profesor
type Teacher struct {
	ID         int
	Name       string
	BusyBlocks []int // Bloques (0-34) donde NO puede hacer clases
}

// Room representa un espacio físico.
type Room struct {
	ID       int
	Code     string   // Identificador ("LAB D", "101")
	Capacity int      // Capacidad sala
	Type     RoomType // SALA o LABORATORIO
}

// Activity representa un evento, es decir una instancia de clase de cualquier tipo
// (catedra, ayudantía, laboratorio), es un nodo/vertice del grafo y se le debe asignar
// un horario (bloque) y sala. multiples secciones pueden asistir a la misma actividad,
// con esto nos referimos a que en la oferta academica existen multiples secciones de
// un mismo curso, donde muchas comparten catedra y ayudantía pero con diferentes labs.
type Activity struct {
	ID           int    // Identificador único
	Code         string // Código de actividad ("CBF1000-CAT-1-S1") -> mecanica- catedra 1-sesión 1
	CourseCode   string
	CourseName   string
	Type         EventCategory
	EventNumber  int      // Distingue entre las diferentes catedras, ayudantías o laboratorios
	Sections     []int    // lista de secciones que asisten a la actividad
	Students     int      // Total de estudiantes
	TeacherNames []string // Nombres de profesores asignados
	Duration     int      // Duración en bloques
	// SiblingGroupID agrupa actividades hermanas, en este caso agrupa las instancias de la catedra de una sección de un curso
	SiblingGroupID string
	Block          int    // Bloque temporal de INICIO
	Room           string // Sala asignada
}

// NewActivity crea una Activity con estado inicial sin asignar.
func NewActivity(id int, code, courseCode, courseName string, eventType EventCategory, eventNum int, sections []int, students int, teachers []string, siblingGroup string, duration int) Activity {
	if duration < 1 {
		duration = 1
	}
	return Activity{
		ID:             id,
		Code:           code,
		CourseCode:     courseCode,
		CourseName:     courseName,
		Type:           eventType,
		EventNumber:    eventNum,
		Sections:       sections,
		Students:       students,
		TeacherNames:   teachers,
		Duration:       duration,
		SiblingGroupID: siblingGroup,
		Block:          -1,
		Room:           "",
	}
}

// IsSiblingOf verifica si dos actividades son hermanas
func (a *Activity) IsSiblingOf(other *Activity) bool {
	return a.SiblingGroupID != "" && a.SiblingGroupID == other.SiblingGroupID && a.ID != other.ID
}

// IsAssigned indica si la actividad ya tiene bloque y sala asignados
func (a *Activity) IsAssigned() bool {
	return a.Block >= 0 && a.Room != ""
}

// Section representa una sección específica de un curso. cada actividad tiene una o más secciones asociadas
type Section struct {
	ID            int
	CourseID      int
	SectionNumber int
	Students      int   // Estimación de alumnos inscritos
	TeacherIDs    []int // IDs de profesores (co-docencia posible)
}

// NewSection crea una Section con los profesores indicados.
func NewSection(id, courseID, sectionNum, students int, teacherIDs ...int) Section {
	return Section{
		ID:            id,
		CourseID:      courseID,
		SectionNumber: sectionNum,
		Students:      students,
		TeacherIDs:    teacherIDs,
	}
}

// HasTeacher verifica si la actividad tiene asignado un profesor
func (a *Activity) HasTeacher(name string) bool {
	for _, t := range a.TeacherNames {
		if t == name {
			return true
		}
	}
	return false
}

// SharesTeacher verifica si dos actividades comparten al menos un profesor.
func (a *Activity) SharesTeacher(other *Activity) bool {
	for _, t := range a.TeacherNames {
		if other.HasTeacher(t) {
			return true
		}
	}
	return false
}

// SharesSection verifica si dos actividades comparten al menos una sección.
func (a *Activity) SharesSection(other *Activity) bool {
	// si las actividades no son del mismo curso, no comparten sección
	if a.CourseCode != other.CourseCode {
		return false
	}

	for _, s := range a.Sections {
		for _, os := range other.Sections {
			if s == os {
				return true
			}
		}
	}
	return false
}

// BlocksOccupied retorna la lista de bloques que ocupa la actividad, para actividades de más de un bloque
func (a *Activity) BlocksOccupied() []int {
	if a.Block < 0 {
		return nil
	}
	blocks := make([]int, a.Duration)
	for i := 0; i < a.Duration; i++ {
		blocks[i] = a.Block + i
	}
	return blocks
}

// OccupiesBlock verifica si la actividad ocupa un bloque específico.
func (a *Activity) OccupiesBlock(block int) bool {
	if a.Block < 0 || a.Duration < 1 {
		return false
	}
	return block >= a.Block && block < a.Block+a.Duration
}

// OverlapsInTime verifica si dos actividades se sobreponen en tiempo.
func (a *Activity) OverlapsInTime(other *Activity) bool {
	if a.Block < 0 || other.Block < 0 {
		return false
	}
	aEnd := a.Block + a.Duration
	otherEnd := other.Block + other.Duration
	// se sobreponen si una actividad termina antes de que la otra comience
	return a.Block < otherEnd && other.Block < aEnd
}
