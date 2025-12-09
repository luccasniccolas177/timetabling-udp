package data

import "timetabling-UDP/internal/models"

func LoadCourseRequirements() []models.Course {
	defaultDist := models.Distribution{
		NumLectures:        2,
		DurationLectures:   1,
		NumAssistants:      1,
		DurationAssistants: 1,
		NumLabs:            0,
		DurationLabs:       0,
	}

	EIT := models.CIT
	IND := models.CII
	EOC := models.COC

	return []models.Course{
		{
			Code: "CBM1000", Name: "álgebra y geometría", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 1}, {IND, 1}, {EOC, 1}},
		},
		{
			Code: "CBM1001", Name: "cálculo i", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 1}, {IND, 1}, {EOC, 1}},
		},
		{
			Code: "CBQ1000", Name: "química", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 1}, {IND, 1}, {EOC, 1}},
		},
		{
			Code: "FIC1000", Name: "comunicación para la ingeniería", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 1}, {IND, 1}, {EOC, 1}},
		},
		{
			Code: "CIT1000", Name: "programación", Distribution: defaultDist, // Python/C intro
			Requirements: []models.Requirement{{EIT, 1}, {IND, 1}, {EOC, 1}},
		},
		{
			Code: "CBM1002", Name: "álgebra lineal", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 2}, {IND, 2}, {EOC, 2}},
		},
		{
			Code: "CBM1003", Name: "cálculo ii", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 2}, {IND, 2}, {EOC, 2}},
		},
		{
			Code: "CBF1000", Name: "mecánica", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 2}, {IND, 2}, {EOC, 2}},
		},
		{
			Code: "CIT1010", Name: "programación avanzada", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 2}, {IND, 2}}, // EOC no lo toma
		},
		{
			Code: "CBM1005", Name: "ecuaciones diferenciales", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 3}, {IND, 3}, {EOC, 3}},
		},
		{
			Code: "CBM1006", Name: "cálculo iii", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 3}, {IND, 3}, {EOC, 3}},
		},
		{
			Code: "CBF1001", Name: "calor y ondas", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 3}, {IND, 3}, {EOC, 3}},
		},
		{
			Code: "CBF1002", Name: "electricidad y magnetismo", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 4}, {IND, 4}, {EOC, 4}},
		},
		{
			Code: "CII2750", Name: "optimización", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 5}, {IND, 5}, {EOC, 5}},
		},
		// =================================================================
		// RAMOS COMPARTIDOS ENTRE ALGUNAS CARRERAS (Industrial y Obras)
		// =================================================================
		{
			Code: "CII1000", Name: "contabilidad y costos", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 6}, {IND, 3}}, // EOC tiene su propio ramo de costos
		},
		{
			Code: "CBE2000", Name: "probabilidades y estadística", Distribution: defaultDist,
			Requirements: []models.Requirement{{IND, 4}, {EOC, 4}}, // EIT usa CIT-2204
		},
		{
			Code: "CII2100", Name: "introducción a la economía", Distribution: defaultDist,
			Requirements: []models.Requirement{{EIT, 8}, {IND, 4}, {EOC, 6}},
		},

		// =================================================================
		// INFORMÁTICA Y TELECOMUNICACIONES (EIT) - Exclusivos
		// =================================================================
		{Code: "CIT2006", Name: "estructuras de datos y algoritmos", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 3}}},
		{Code: "CIT2114", Name: "redes de datos", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 3}}},

		{Code: "CIT2107", Name: "electrónica y electrotecnia", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 4}}},
		{Code: "CIT2007", Name: "bases de datos", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 4}}},
		{Code: "CIT2008", Name: "desarrollo web y móvil", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 4}}},
		{Code: "CIT2204", Name: "probabilidades y estadística (info)", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 4}}},

		{Code: "CIT2205", Name: "proyecto en tics i", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 5}}},
		{Code: "CIT2108", Name: "taller de redes y servicios", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 5}}},
		{Code: "CIT2009", Name: "bases de datos avanzadas", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 5}}},

		{Code: "CIT2109", Name: "arquitectura y organización de computadores", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 6}}},
		{Code: "CIT2110", Name: "señales y sistemas", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 6}}},
		{Code: "CIT2010", Name: "sistemas operativos", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 6}}},

		{Code: "CIT2111", Name: "comunicaciones digitales", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 7}}},
		{Code: "CIT2011", Name: "sistemas distribuidos", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 7}}},
		{Code: "CIT2206", Name: "gestión organizacional", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 7}}},
		{Code: "CIT2012", Name: "ingeniería de software", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 7}}},

		{Code: "CIT2207", Name: "evaluación de proyectos tic", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 8}}},
		{Code: "CIT2113", Name: "criptografía y seguridad en redes", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 8}}},
		{Code: "CIT2112", Name: "tecnologías inalámbricas", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 8}}},
		{Code: "CIT2013", Name: "inteligencia artificial", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 8}}},

		{Code: "CIT3100", Name: "arquitecturas emergentes", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 9}}},
		{Code: "CIT3202", Name: "data science (info)", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 9}}},
		{Code: "CIT3000", Name: "arquitecturas de software", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 9}}},

		{Code: "CIT3203", Name: "proyecto en tics ii", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 10}}},

		// =================================================================
		// INDUSTRIAL (IND) - Exclusivos
		// =================================================================
		{Code: "CII1001", Name: "teoría organizacional", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 5}}},
		{Code: "CII2250", Name: "estática (ind)", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 4}}},

		{Code: "CII2751", Name: "inferencia estadística", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 5}}},
		{Code: "CII2401", Name: "mecánica de fluidos (ind)", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 5}}},

		{Code: "CII2501", Name: "bases de datos (ind)", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 6}}},
		{Code: "CII2402", Name: "termodinámica", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 6}}},
		{Code: "CII2755", Name: "modelos estocásticos", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 6}}},
		{Code: "CII2101", Name: "microeconomía", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 6}}},
		{Code: "CII2002", Name: "ingeniería económica", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 6}}},

		{Code: "CII2756", Name: "econometría", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 7}}},
		{Code: "CII2403", Name: "proyectos energéticos", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 7}}},
		{Code: "CII2253", Name: "producción", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 7}}},
		{Code: "CII2102", Name: "marketing", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 7}}},
		{Code: "CII2003", Name: "finanzas", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 7}}},

		{Code: "CII2504", Name: "data science (ind)", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 8}}},
		{Code: "CII2757", Name: "simulación", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 8}}},
		{Code: "CII2254", Name: "logística", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 8}}},
		{Code: "CII2103", Name: "gestión estratégica", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 8}}},
		{Code: "CII2004", Name: "evaluación de proyectos (ind)", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 8}}},

		{Code: "CII3101", Name: "liderazgo y emprendimiento", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 9}}},
		{Code: "CII3102", Name: "taller de ingeniería industrial", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 10}}},

		// =================================================================
		// OBRAS CIVILES (EOC) - Exclusivos
		// =================================================================
		{Code: "COC2001", Name: "ingeniería de materiales", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 2}}},

		{Code: "COC2108", Name: "estática (obras)", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 3}}},

		{Code: "COC2206", Name: "mecánica de fluidos (obras)", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 4}}},
		{Code: "COC2109", Name: "mecánica de sólidos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 4}}},
		{Code: "COC2202", Name: "edificación", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 4}}},

		{Code: "COC2207", Name: "hidráulica", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 5}}},
		{Code: "COC2006", Name: "topografía", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 5}}},
		{Code: "COC2102", Name: "análisis estructural", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 5}}},

		{Code: "COC2203", Name: "ingeniería ambiental", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 6}}},
		{Code: "COC2003", Name: "tecnología del hormigón", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 6}}},
		{Code: "COC2103", Name: "diseño estructural", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 6}}},
		{Code: "COC2305", Name: "seminario cs. ingeniería", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 6}}},

		{Code: "COC2104", Name: "mecánica de suelos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 7}}},
		{Code: "COC2204", Name: "hidrología", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 7}}},
		{Code: "COC2007", Name: "administración de proyectos civiles", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 7}}},
		{Code: "COC2105", Name: "diseño en hormigón", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 7}}},

		{Code: "COC2106", Name: "fundaciones", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 8}}},
		{Code: "COC2205", Name: "hidráulica urbana", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 8}}},
		{Code: "COC2008", Name: "planificación de proyectos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 8}}},
		{Code: "COC2107", Name: "diseño en acero", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 8}}},
		{Code: "COC3100", Name: "ingeniería sísmica", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 8}}},

		{Code: "COC2009", Name: "bim", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 9}}},
		{Code: "COC3000", Name: "ingeniería de costos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 9}}},

		{Code: "COC3300", Name: "taller de proyectos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 10}}},
		{Code: "COC3001", Name: "diseño de caminos", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 10}}},

		// =================================================================
		// ELECTIVOS (Placeholders Genéricos)
		// NOTA: Se crean códigos únicos ficticios para que el grafo los procese.
		// ================================================================

		// Electivos Informática
		{Code: "ELE-INF-01", Name: "electivo info ix-1", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 9}}},
		{Code: "ELE-TEL-01", Name: "electivo telco ix-1", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 9}}},
		{Code: "ELE-INF-02", Name: "electivo info x-1", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 10}}},
		{Code: "ELE-TEL-02", Name: "electivo telco x-1", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 10}}},
		{Code: "ELE-INF-03", Name: "electivo info x-2", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 10}}},
		{Code: "ELE-TEL-03", Name: "electivo telco x-2", Distribution: defaultDist, Requirements: []models.Requirement{{EIT, 10}}},

		// Electivos Industrial
		{Code: "ELE-IND-01", Name: "electivo prof. ind ix-1", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 9}}},
		{Code: "ELE-IND-02", Name: "electivo prof. ind ix-2", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 9}}},
		{Code: "ELE-IND-03", Name: "electivo prof. ind ix-3", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 9}}},
		{Code: "ELE-IND-04", Name: "electivo prof. ind ix-4", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 9}}},
		{Code: "ELE-IND-05", Name: "electivo prof. ind x-1", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 10}}},
		{Code: "ELE-IND-06", Name: "electivo prof. ind x-2", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 10}}},
		{Code: "ELE-IND-07", Name: "electivo prof. ind x-3", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 10}}},
		{Code: "ELE-IND-08", Name: "electivo prof. ind x-4", Distribution: defaultDist, Requirements: []models.Requirement{{IND, 10}}},

		// Electivos Obras
		{Code: "ELE-COC-01", Name: "electivo prof. obras ix-1", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 9}}},
		{Code: "ELE-COC-02", Name: "electivo prof. obras ix-2", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 9}}},
		{Code: "ELE-COC-03", Name: "electivo prof. obras ix-3", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 9}}},
		{Code: "ELE-COC-04", Name: "electivo prof. obras x-1", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 10}}},
		{Code: "ELE-COC-05", Name: "electivo prof. obras x-2", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 10}}},
		{Code: "ELE-COC-06", Name: "electivo prof. obras x-3", Distribution: defaultDist, Requirements: []models.Requirement{{EOC, 10}}},
	}
}

func RequirementsMap() map[string][]models.Requirement {
	reqMap := make(map[string][]models.Requirement)
	for _, c := range LoadCourseRequirements() {
		reqMap[c.Code] = c.Requirements
	}
	return reqMap
}
