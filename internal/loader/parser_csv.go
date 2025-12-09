package loader

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
)

func LoadCSV(filepath string) ([][]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error al abrir el archivo %s: %v\n", filepath, err))
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return [][]string{}, errors.New(fmt.Sprintf("Error al leer el archivo %s: %v\n", filepath, err))
	}

	return records, nil
}
