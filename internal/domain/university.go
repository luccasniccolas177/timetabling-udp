package domain

// Course define una asignatura en el plan de estudios.
type Course struct {
	ID            int
	Code          string        // Código institucional (ej: "CBF1000")
	Name          string        // Nombre de la asignatura
	Prerequisites []int         // IDs de cursos prerequisito
	PlanLocation  map[Major]int // Carrera -> Semestre en la malla
	Distribution  Distribution  // Carga académica (CAT, AY, LAB)
	IsElective    bool          // Si es una asignatura electiva
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

// Teacher representa a un docente.
type Teacher struct {
	ID         int
	Name       string
	BusyBlocks []int // Bloques (0-34) donde NO puede hacer clases
}

// Room representa un espacio físico.
type Room struct {
	ID       int
	Code     string   // Identificador (ej: "LAB D", "101")
	Capacity int      // Capacidad máxima de estudiantes
	Type     RoomType // SALA o LABORATORIO
}

// Activity es la unidad fundamental de programación (vértice en el grafo).
// Cada Activity corresponde a un evento que debe asignarse a un bloque y sala.
type Activity struct {
	ID           int           // Identificador único
	Code         string        // Código de actividad (ej: "CBF1000-CAT-1")
	CourseCode   string        // Código del curso padre
	CourseName   string        // Nombre del curso (para reportes)
	Type         EventCategory // CATEDRA, AYUDANTIA, LABORATORIO
	EventNumber  int           // Distingue CAT-1, CAT-2, etc.
	Sections     []int         // Secciones vinculadas (super-vértice)
	Students     int           // Total de estudiantes (para Bin Packing)
	TeacherNames []string      // Nombres de profesores asignados

	// SiblingGroupID agrupa cátedras que deben ser "espejo" (mismo horario, días distintos).
	// Formato: "COURSE_CODE-TYPE-SECTIONS" (ej: "CBF1000-CAT-1,2")
	SiblingGroupID string

	// --- Estado del Scheduler (se llena durante la programación) ---
	Block int    // Bloque temporal asignado (-1 = sin asignar)
	Room  string // Sala asignada ("" = sin asignar)
}

// NewActivity crea una Activity con estado inicial sin asignar.
func NewActivity(id int, code, courseCode, courseName string, eventType EventCategory, eventNum int, sections []int, students int, teachers []string, siblingGroup string) Activity {
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
		SiblingGroupID: siblingGroup,
		Block:          -1,
		Room:           "",
	}
}

// IsSiblingOf verifica si dos actividades son hermanas (mismo grupo espejo).
func (a *Activity) IsSiblingOf(other *Activity) bool {
	return a.SiblingGroupID != "" && a.SiblingGroupID == other.SiblingGroupID && a.ID != other.ID
}

// IsAssigned indica si la actividad ya tiene bloque y sala asignados.
func (a *Activity) IsAssigned() bool {
	return a.Block >= 0 && a.Room != ""
}

// Section representa una sección específica de un curso.
// Las Activities referencian secciones a través de sus IDs.
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

// HasTeacher verifica si la actividad tiene asignado un profesor específico.
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
// IMPORTANTE: Solo considera conflicto si son del MISMO curso, ya que las
// secciones son independientes entre cursos diferentes.
func (a *Activity) SharesSection(other *Activity) bool {
	// Secciones solo son compartidas si es el mismo curso
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
