package loader

import (
	"fmt"
	"timetabling-UDP/internal/domain"
	"timetabling-UDP/internal/models"
)

// DomainBuilder construye el modelo de dominio a partir de los datos crudos del loader
// Separa la responsabilidad de cargar datos (loader.go) de construir el modelo (domain_builder.go)
type DomainBuilder struct {
	// Datos crudos del loader antiguo
	oldCourses  map[int]models.Course
	oldSections map[int]models.Section
	oldTeachers map[int]models.Teacher
	oldRooms    map[int]models.Room
	oldEvents   []models.LogicalEvent

	// Modelo de dominio nuevo
	university *domain.University

	// Mapas de conversión (ID antiguo → objeto nuevo)
	courseMap  map[int]*domain.Course
	sectionMap map[int]*domain.Section
	teacherMap map[int]*domain.Teacher
	roomMap    map[int]*domain.Room

	// Contadores para IDs de clases
	nextLectureID  int
	nextTutorialID int
	nextLabID      int
}

// NewDomainBuilder crea un nuevo builder
func NewDomainBuilder() *DomainBuilder {
	return &DomainBuilder{
		university:     domain.NewUniversity(),
		courseMap:      make(map[int]*domain.Course),
		sectionMap:     make(map[int]*domain.Section),
		teacherMap:     make(map[int]*domain.Teacher),
		roomMap:        make(map[int]*domain.Room),
		nextLectureID:  1,
		nextTutorialID: 1,
		nextLabID:      1,
	}
}

// BuildFromOldModel construye el modelo de dominio desde UniversityState
func (b *DomainBuilder) BuildFromOldModel(oldState *UniversityState) (*domain.University, error) {
	// Guardar datos crudos
	b.oldCourses = oldState.Courses
	b.oldSections = oldState.Sections
	b.oldTeachers = oldState.Teachers
	b.oldRooms = oldState.Rooms
	b.oldEvents = oldState.RawEvents

	// Construir en orden de dependencias
	if err := b.buildTeachers(); err != nil {
		return nil, fmt.Errorf("error building teachers: %w", err)
	}

	if err := b.buildRooms(); err != nil {
		return nil, fmt.Errorf("error building rooms: %w", err)
	}

	if err := b.buildCourses(); err != nil {
		return nil, fmt.Errorf("error building courses: %w", err)
	}

	if err := b.buildSections(); err != nil {
		return nil, fmt.Errorf("error building sections: %w", err)
	}

	if err := b.buildClasses(); err != nil {
		return nil, fmt.Errorf("error building classes: %w", err)
	}

	if err := b.linkSectionsToClasses(); err != nil {
		return nil, fmt.Errorf("error linking sections to classes: %w", err)
	}

	// Copiar restricciones de salas
	if oldState.RoomConstraints != nil {
		b.buildRoomConstraints(oldState.RoomConstraints)
	}

	return b.university, nil
}

// buildTeachers convierte teachers del modelo antiguo al nuevo
func (b *DomainBuilder) buildTeachers() error {
	for id, oldTeacher := range b.oldTeachers {
		teacher := &domain.Teacher{
			ID:   oldTeacher.ID,
			Name: oldTeacher.Name,
		}
		b.teacherMap[id] = teacher
		b.university.Teachers[id] = teacher
	}
	return nil
}

// buildRooms convierte rooms del modelo antiguo al nuevo
func (b *DomainBuilder) buildRooms() error {
	for id, oldRoom := range b.oldRooms {
		room := &domain.Room{
			ID:       oldRoom.ID,
			Code:     oldRoom.Code,
			Capacity: oldRoom.Capacity,
			Type:     convertRoomType(oldRoom.RoomType),
		}
		b.roomMap[id] = room
		b.university.Rooms[id] = room
	}
	return nil
}

// buildCourses convierte courses del modelo antiguo al nuevo
func (b *DomainBuilder) buildCourses() error {
	for id, oldCourse := range b.oldCourses {
		// Convertir requirements
		curriculum := make([]domain.CurriculumEntry, len(oldCourse.Requirements))
		for i, req := range oldCourse.Requirements {
			curriculum[i] = domain.CurriculumEntry{
				Major:    convertMajor(req.Major),
				Semester: req.Semester,
			}
		}

		course := &domain.Course{
			ID:            oldCourse.ID,
			Code:          oldCourse.Code,
			Name:          oldCourse.Name,
			Curriculum:    curriculum,
			Prerequisites: oldCourse.Prerequisites,
		}

		b.courseMap[id] = course
		b.university.Courses[id] = course
	}
	return nil
}

// buildSections convierte sections del modelo antiguo al nuevo
func (b *DomainBuilder) buildSections() error {
	for id, oldSection := range b.oldSections {
		course, ok := b.courseMap[oldSection.CourseID]
		if !ok {
			return fmt.Errorf("course %d not found for section %d", oldSection.CourseID, id)
		}

		section := &domain.Section{
			ID:           oldSection.ID,
			Course:       course,
			Number:       oldSection.SectionNumber,
			StudentCount: oldSection.StudentsNumber,
			// Las relaciones con clases se establecen después
		}

		b.sectionMap[id] = section
		b.university.Sections[id] = section
	}
	return nil
}

// buildClasses crea Lectures, Tutorials y Labs a partir de LogicalEvents
func (b *DomainBuilder) buildClasses() error {
	// Agrupar eventos por tipo y ParentSectionIDs
	// Eventos con los mismos ParentSectionIDs son la misma clase compartida
	lectureGroups := make(map[string]*domain.Lecture)
	tutorialGroups := make(map[string]*domain.Tutorial)

	for _, oldEvent := range b.oldEvents {
		course, ok := b.courseMap[oldEvent.CourseID]
		if !ok {
			continue // Skip eventos sin curso
		}

		// Obtener secciones
		sections := make([]*domain.Section, 0, len(oldEvent.ParentSectionIDs))
		for _, secID := range oldEvent.ParentSectionIDs {
			if sec, ok := b.sectionMap[secID]; ok {
				sections = append(sections, sec)
			}
		}

		if len(sections) == 0 {
			continue // Skip eventos sin secciones
		}

		// Obtener teachers
		teachers := make([]*domain.Teacher, 0, len(oldEvent.TeachersIDs))
		for _, teacherID := range oldEvent.TeachersIDs {
			if teacher, ok := b.teacherMap[teacherID]; ok {
				teachers = append(teachers, teacher)
			}
		}

		switch oldEvent.Type {
		case models.CAT:
			// Cátedra: compartida entre secciones
			key := b.getLectureKey(course.ID, oldEvent.ParentSectionIDs, oldEvent.EventNumber)

			if lecture, exists := lectureGroups[key]; exists {
				// Ya existe, solo verificar consistencia
				_ = lecture
			} else {
				// Crear nueva lecture
				lecture := &domain.Lecture{
					ID:        b.nextLectureID,
					Course:    course,
					Sections:  sections,
					Number:    oldEvent.EventNumber,
					Frequency: oldEvent.Frequency,
					Teachers:  teachers,
				}
				b.nextLectureID++
				lectureGroups[key] = lecture
				b.university.Lectures[lecture.ID] = lecture
			}

		case models.AY:
			// Ayudantía: compartida entre secciones
			key := b.getTutorialKey(course.ID, oldEvent.ParentSectionIDs, oldEvent.EventNumber)

			if tutorial, exists := tutorialGroups[key]; exists {
				_ = tutorial
			} else {
				tutorial := &domain.Tutorial{
					ID:       b.nextTutorialID,
					Course:   course,
					Sections: sections,
					Number:   oldEvent.EventNumber,
					Teachers: teachers,
				}
				b.nextTutorialID++
				tutorialGroups[key] = tutorial
				b.university.Tutorials[tutorial.ID] = tutorial
			}

		case models.LAB:
			// Laboratorio: específico de UNA sección
			if len(sections) != 1 {
				// Labs deben tener exactamente 1 sección
				// Si tiene más, crear uno por sección
				for _, section := range sections {
					lab := &domain.Lab{
						ID:              b.nextLabID,
						Course:          course,
						Section:         section,
						Number:          oldEvent.EventNumber,
						Duration:        oldEvent.DurationBlocks,
						Teachers:        teachers,
						RoomConstraints: oldEvent.RoomsConstraints,
					}
					b.nextLabID++
					b.university.Labs[lab.ID] = lab
				}
			} else {
				lab := &domain.Lab{
					ID:              b.nextLabID,
					Course:          course,
					Section:         sections[0],
					Number:          oldEvent.EventNumber,
					Duration:        oldEvent.DurationBlocks,
					Teachers:        teachers,
					RoomConstraints: oldEvent.RoomsConstraints,
				}
				b.nextLabID++
				b.university.Labs[lab.ID] = lab
			}
		}
	}

	return nil
}

// linkSectionsToClasses establece las relaciones bidireccionales
func (b *DomainBuilder) linkSectionsToClasses() error {
	// Para cada sección, encontrar sus clases
	for _, section := range b.university.Sections {
		// Buscar lecture compartida
		for _, lecture := range b.university.Lectures {
			if lecture.Course.ID == section.Course.ID {
				for _, lecSec := range lecture.Sections {
					if lecSec.ID == section.ID {
						section.SharedLecture = lecture
						break
					}
				}
			}
		}

		// Buscar tutorial compartida
		for _, tutorial := range b.university.Tutorials {
			if tutorial.Course.ID == section.Course.ID {
				for _, tutSec := range tutorial.Sections {
					if tutSec.ID == section.ID {
						section.SharedTutorial = tutorial
						break
					}
				}
			}
		}

		// Buscar lab propio
		for _, lab := range b.university.Labs {
			if lab.Section.ID == section.ID {
				section.OwnLab = lab
				break
			}
		}
	}

	return nil
}

// buildRoomConstraints convierte las restricciones de salas
func (b *DomainBuilder) buildRoomConstraints(oldConstraints *models.RoomConstraints) {
	newConstraints := &domain.RoomConstraints{
		CourseConstraints: make(map[string]map[domain.ClassType][]string),
		Defaults:          make(map[domain.ClassType][]string),
	}

	// Convertir CourseConstraints
	for courseCode, typeMap := range oldConstraints.CourseConstraints {
		newConstraints.CourseConstraints[courseCode] = make(map[domain.ClassType][]string)
		for eventType, rooms := range typeMap {
			classType := convertEventTypeToClassType(eventType)
			newConstraints.CourseConstraints[courseCode][classType] = rooms
		}
	}

	// Convertir Defaults
	for eventType, rooms := range oldConstraints.Defaults {
		classType := convertEventTypeToClassType(eventType)
		newConstraints.Defaults[classType] = rooms
	}

	b.university.RoomConstraints = newConstraints
}

// Helper functions

func (b *DomainBuilder) getLectureKey(courseID int, sectionIDs []int, number int) string {
	return fmt.Sprintf("L-%d-%v-%d", courseID, sectionIDs, number)
}

func (b *DomainBuilder) getTutorialKey(courseID int, sectionIDs []int, number int) string {
	return fmt.Sprintf("T-%d-%v-%d", courseID, sectionIDs, number)
}

// Conversion functions

func convertMajor(old models.Major) domain.Major {
	switch old {
	case models.CIT:
		return domain.MajorCIT
	case models.COC:
		return domain.MajorCOC
	case models.CII:
		return domain.MajorCII
	default:
		return domain.MajorCIT
	}
}

func convertRoomType(old models.RoomType) domain.RoomType {
	switch old {
	case models.CR:
		return domain.RoomTypeClassroom
	case models.LR:
		return domain.RoomTypeLaboratory
	default:
		return domain.RoomTypeClassroom
	}
}

func convertEventTypeToClassType(old models.EventType) domain.ClassType {
	switch old {
	case models.CAT:
		return domain.ClassTypeLecture
	case models.AY:
		return domain.ClassTypeTutorial
	case models.LAB:
		return domain.ClassTypeLab
	default:
		return domain.ClassTypeLecture
	}
}
