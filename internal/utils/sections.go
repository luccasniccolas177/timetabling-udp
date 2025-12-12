package utils

import (
	"sort"
	"strconv"
)

// SectionGroupKey crea una clave única para un grupo de secciones.
// Se usa para identificar grupos de secciones fusionadas.
func SectionGroupKey(sections []int) string {
	if len(sections) == 0 {
		return "empty"
	}
	sorted := make([]int, len(sections))
	copy(sorted, sections)
	sort.Ints(sorted)

	key := ""
	for _, s := range sorted {
		if key != "" {
			key += "-"
		}
		key += strconv.Itoa(s)
	}
	return key
}
