package loader

import (
	"fmt"
	"timetabling-UDP/internal/domain"
)

// LoadUniversity carga todos los datos y retorna el modelo de dominio nuevo
// Esta es la función principal que debe usarse en lugar de LoadFicData
func LoadUniversity(basePath string) (*domain.University, error) {
	// 1. Cargar datos usando el loader antiguo
	oldState, err := LoadFicData(basePath)
	if err != nil {
		return nil, fmt.Errorf("error loading data: %w", err)
	}

	// 2. Construir modelo de dominio usando DomainBuilder
	builder := NewDomainBuilder()
	university, err := builder.BuildFromOldModel(oldState)
	if err != nil {
		return nil, fmt.Errorf("error building domain model: %w", err)
	}

	return university, nil
}
