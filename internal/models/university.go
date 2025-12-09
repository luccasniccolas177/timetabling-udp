package models

// Requirement vincula un curso con la carrera(s) a la(s) que pertenece y el semestre en el que se toma
type Requirement struct {
	Major    Major
	Semester int
}

// Distribution guarda el numero de eventos (CAT, AY, LAB) de cada tipo y la duración en bloques horarios
type Distribution struct {
	NumLectures   int `json:"num_lectures"`
	NumAssistants int `json:"num_assistants"`
	NumLabs       int `json:"num_labs"`

	DurationLectures   int `json:"duration_lectures"`
	DurationAssistants int `json:"duration_assistants"`
	DurationLabs       int `json:"duration_labs"`
}

// Course almacena toda la información general sobre un curso, no almacena los eventos en sí
// Como existen multiples cursos que se comparten dentro de la carrera como los de ciencias basicas se define los requerimientos como
// un slice para almacenar los distintos semestres donde se imparte el ramo en cada carrera
// Como la distribucion de eventos (catedras, ayudantía y labs) es general a un curso se almacena aca esa información
type Course struct {
	ID           int
	Name         string `json:"name"`
	Code         string `json:"code"`
	Requirements []Requirement
	Distribution Distribution
}

// Section almacena la metadata de una sección en especifico (numero de sección, numero estudiantes)ñ
type Section struct {
	ID             int
	CourseID       int
	SectionNumber  int
	StudentsNumber int
}

// LogicalEvent Almacena solo la información de un evento, hice esta distinción debido a que una catedra normalmente se materializan en 2 o 3 clases
// en la semana por lo que es necesario indicar esto para que el modelo lo resuelva correctamente
// ParentSectionIDs almacena las secciones relacionadas con el evento, esto es clave, pues multiples catedras y ayudantias suelen pertenecer a
// más de una sección, donde lo que las diferencia son el laboratorio que realizan
type LogicalEvent struct {
	ID               int
	CourseID         int
	Type             EventType
	ParentSectionIDs []int    // secciones relacionadas al evento
	EventNumber      int      // (CATEDRA 1, AYUDANTIA 1, LAB 2)
	EventSize        int      // numero de alumnos
	DurationBlocks   int      // duración en bloques del evento
	Frequency        int      // numero de clases
	TeachersIDs      []int    // lista de profesores que imparten el evento, suele ser uno, pero en la malla salen 2 en algunos casos
	RoomType         RoomType // tipo de sala a utilizar
	RoomsConstraints []string // salas donde se puede impartir, se lee del json rooms_constraints.json
}

// EventInstance es un evento en especifico, normalmente las catedras tienen 2-3 secciones, por lo que esta estructura es clave para poder representar a los nodos
type EventInstance struct {
	UUID           string // CIT1000-CX-IX
	LogicalEventID int
	Index          int
	Data           *LogicalEvent
	FixedBlock     int // -1 no fijo, >= 0 indica el bloque fijo

	AssignedSlot TimeSlot
	Color        int
}

// Room representa las salas de clase de la universidad, se dividen en salas normales y laboratorios
type Room struct {
	ID       int
	RoomType RoomType // sala - lab
	Code     string   // 402
	Capacity int
}

// Teacher representa un profesor, más adelante se colocaran más campos para agregar restricciones (tiempo, cursos, etc)
type Teacher struct {
	ID   int
	Name string
	// UnvailableBlocks map[int]bool restricciones de tiempo para profesores
}
