package domain

// University es el contenedor principal de todos los datos del dominio
// Reemplaza al antiguo UniversityState
type University struct {
	// Entidades principales
	Courses  map[int]*Course
	Sections map[int]*Section
	Teachers map[int]*Teacher
	Rooms    map[int]*Room

	// Clases (generadas a partir de los datos cargados)
	Lectures  map[int]*Lecture
	Tutorials map[int]*Tutorial
	Labs      map[int]*Lab

	// Restricciones
	RoomConstraints *RoomConstraints
}

// NewUniversity crea una nueva instancia de University
func NewUniversity() *University {
	return &University{
		Courses:   make(map[int]*Course),
		Sections:  make(map[int]*Section),
		Teachers:  make(map[int]*Teacher),
		Rooms:     make(map[int]*Room),
		Lectures:  make(map[int]*Lecture),
		Tutorials: make(map[int]*Tutorial),
		Labs:      make(map[int]*Lab),
	}
}

// GetAllClasses retorna todas las clases (Lectures, Tutorials, Labs)
func (u *University) GetAllClasses() []Class {
	classes := make([]Class, 0, len(u.Lectures)+len(u.Tutorials)+len(u.Labs))

	for _, lecture := range u.Lectures {
		classes = append(classes, lecture)
	}

	for _, tutorial := range u.Tutorials {
		classes = append(classes, tutorial)
	}

	for _, lab := range u.Labs {
		classes = append(classes, lab)
	}

	return classes
}

// GetCoursesBySemester retorna todos los cursos de una carrera/semestre específico
func (u *University) GetCoursesBySemester(major Major, semester int) []*Course {
	courses := make([]*Course, 0)

	for _, course := range u.Courses {
		if course.BelongsToSemester(major, semester) {
			courses = append(courses, course)
		}
	}

	return courses
}

// GetSectionsByCourse retorna todas las secciones de un curso
func (u *University) GetSectionsByCourse(courseID int) []*Section {
	sections := make([]*Section, 0)

	for _, section := range u.Sections {
		if section.Course.ID == courseID {
			sections = append(sections, section)
		}
	}

	return sections
}
