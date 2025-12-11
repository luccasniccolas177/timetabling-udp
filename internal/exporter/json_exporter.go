package exporter

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"timetabling-UDP/internal/domain"
)

// ScheduleExport es la estructura del JSON exportado.
type ScheduleExport struct {
	GeneratedAt string           `json:"generated_at"`
	Summary     ScheduleSummary  `json:"summary"`
	Schedule    []DaySchedule    `json:"schedule"`
	Activities  []ActivityExport `json:"activities"`
}

// ScheduleSummary contiene estadísticas del horario.
type ScheduleSummary struct {
	TotalActivities  int     `json:"total_activities"`
	TotalCourses     int     `json:"total_courses"`
	TotalRooms       int     `json:"total_rooms"`
	AYOnWednesday    float64 `json:"ay_on_wednesday_percent"`
	MirrorCompliance float64 `json:"mirror_compliance_percent"`
}

// DaySchedule representa un día de la semana.
type DaySchedule struct {
	Day    string      `json:"day"`
	Blocks []BlockSlot `json:"blocks"`
}

// BlockSlot representa un bloque horario.
type BlockSlot struct {
	Block      int              `json:"block"`
	Time       string           `json:"time"`
	Activities []ActivityExport `json:"activities"`
}

// ActivityExport representa una actividad en el JSON.
type ActivityExport struct {
	Code       string   `json:"code"`
	CourseCode string   `json:"course_code"`
	CourseName string   `json:"course_name"`
	Type       string   `json:"type"`
	Room       string   `json:"room"`
	Block      int      `json:"block"`
	Day        string   `json:"day"`
	TimeSlot   string   `json:"time_slot"`
	Students   int      `json:"students"`
	Teachers   []string `json:"teachers"`
	Sections   []int    `json:"sections"`
}

// Days names in Spanish
var dayNames = []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes"}

// Time slots - Horarios UDP
var timeSlots = []string{
	"08:30-09:50",
	"10:00-11:20",
	"11:30-12:50",
	"13:00-14:20",
	"14:30-15:50",
	"16:00-17:20",
	"17:25-18:45",
}

// ExportScheduleToJSON exporta el horario completo a un archivo JSON.
func ExportScheduleToJSON(activities []domain.Activity, filename string) error {
	// Crear export
	export := ScheduleExport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Summary:     calculateSummary(activities),
		Schedule:    buildDaySchedule(activities),
		Activities:  buildActivityList(activities),
	}

	// Escribir JSON
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func calculateSummary(activities []domain.Activity) ScheduleSummary {
	courses := make(map[string]bool)
	rooms := make(map[string]bool)
	ayOnWed := 0
	totalAY := 0

	for _, a := range activities {
		courses[a.CourseCode] = true
		if a.Room != "" {
			rooms[a.Room] = true
		}
		if a.Type == domain.AY {
			totalAY++
			day := a.Block / domain.BlocksPerDay
			if day == 2 { // Miércoles
				ayOnWed++
			}
		}
	}

	ayPercent := 0.0
	if totalAY > 0 {
		ayPercent = float64(ayOnWed) / float64(totalAY) * 100
	}

	return ScheduleSummary{
		TotalActivities:  len(activities),
		TotalCourses:     len(courses),
		TotalRooms:       len(rooms),
		AYOnWednesday:    ayPercent,
		MirrorCompliance: 0, // TODO: calcular
	}
}

func buildDaySchedule(activities []domain.Activity) []DaySchedule {
	schedule := make([]DaySchedule, 5)

	for d := 0; d < 5; d++ {
		schedule[d] = DaySchedule{
			Day:    dayNames[d],
			Blocks: make([]BlockSlot, domain.BlocksPerDay),
		}

		for s := 0; s < domain.BlocksPerDay; s++ {
			block := d*domain.BlocksPerDay + s
			schedule[d].Blocks[s] = BlockSlot{
				Block:      block,
				Time:       timeSlots[s],
				Activities: []ActivityExport{},
			}
		}
	}

	// Agregar actividades a sus slots
	for _, a := range activities {
		if a.Block < 0 || a.Block >= domain.TotalBlocks {
			continue
		}
		day := a.Block / domain.BlocksPerDay
		slot := a.Block % domain.BlocksPerDay

		ae := activityToExport(a)
		schedule[day].Blocks[slot].Activities = append(
			schedule[day].Blocks[slot].Activities,
			ae,
		)
	}

	return schedule
}

func buildActivityList(activities []domain.Activity) []ActivityExport {
	result := make([]ActivityExport, 0, len(activities))
	for _, a := range activities {
		result = append(result, activityToExport(a))
	}

	// Ordenar por curso y código
	sort.Slice(result, func(i, j int) bool {
		if result[i].CourseCode != result[j].CourseCode {
			return result[i].CourseCode < result[j].CourseCode
		}
		return result[i].Code < result[j].Code
	})

	return result
}

func activityToExport(a domain.Activity) ActivityExport {
	day := 0
	slot := 0
	dayName := ""
	timeSlot := ""

	if a.Block >= 0 && a.Block < domain.TotalBlocks {
		day = a.Block / domain.BlocksPerDay
		slot = a.Block % domain.BlocksPerDay
		dayName = dayNames[day]
		timeSlot = timeSlots[slot]
	}

	typeStr := "CATEDRA"
	switch a.Type {
	case domain.AY:
		typeStr = "AYUDANTIA"
	case domain.LAB:
		typeStr = "LABORATORIO"
	}

	return ActivityExport{
		Code:       a.Code,
		CourseCode: a.CourseCode,
		CourseName: a.CourseName,
		Type:       typeStr,
		Room:       a.Room,
		Block:      a.Block,
		Day:        dayName,
		TimeSlot:   timeSlot,
		Students:   a.Students,
		Teachers:   a.TeacherNames,
		Sections:   a.Sections,
	}
}
