package main

import (
	"sort"
	"strconv"
	"strings"
)

// Funciones de soporte que ya formaban parte de la reconstruccion y que se
// mantienen separadas del lector XLSX/maestro para no mezclar responsabilidades.
func BuildFilteredSortedViewByHeaders(lines []Line, filters map[string]string) []Line {
	o := make([]Line, 0, len(lines))
	for _, l := range lines {
		ok := true
		for field, value := range filters {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if field == "__all__" {
				if !FilterValue(l, value) {
					ok = false
					break
				}
			} else if !filterFieldValue(l, field, value) {
				ok = false
				break
			}
		}
		if ok {
			o = append(o, l)
		}
	}
	sort.SliceStable(o, func(i, j int) bool {
		return strings.ToLower(lineSortKey(lCopy(o[i]))) < strings.ToLower(lineSortKey(lCopy(o[j])))
	})
	return o
}

func lCopy(l Line) Line { return l }

func filterFieldValue(l Line, field, filter string) bool {
	v := fieldValue(l, field)
	if v == "" {
		return false
	}
	return strings.Contains(strings.ToLower(v), strings.ToLower(filter))
}

func fieldValue(l Line, field string) string {
	if v, ok := l.Values[field]; ok {
		return v
	}
	for k, v := range l.Values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(field)) {
			return v
		}
	}
	return ""
}

func FilterValue(l Line, filter string) bool {
	f := strings.ToLower(filter)
	for _, v := range l.Values {
		if strings.Contains(strings.ToLower(v), f) {
			return true
		}
	}
	return false
}

func lineSortKey(l Line) string {
	return strings.ToLower(l.Values[findFieldKey(l, "so", "factura", "cliente")])
}

// parseNumber acepta el formato es-AR: 1.234,56 -> 1234.56.
// Cuando solo hay coma, se interpreta como separador decimal, que es el
// formato usado por GestionSO_Datos.csv.
func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	accountingNegative := strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
	if accountingNegative {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "'", "")

	// La notación científica debe resolverse antes de aplicar heurísticas
	// locales: el separador decimal puede ser un punto y la E no es un
	// separador de miles.
	if strings.ContainsAny(s, "eE") {
		if x, err := strconv.ParseFloat(s, 64); err == nil {
			if accountingNegative { x = -math.Abs(x) }
			return x, true
		}
		return 0, false
	}

	// Con ambos separadores, el último determina el separador decimal.
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
		if x, err := strconv.ParseFloat(s, 64); err == nil {
			if accountingNegative { x = -math.Abs(x) }
			return x, true
		}
		return 0, false
	}

	// Solo coma: en es-AR es separador decimal.
	if strings.Contains(s, ",") {
		// Varias comas con grupos de tres son miles: 80,003,285.
		parts := strings.Split(s, ",")
		if len(parts) > 2 && allThousandGroups(parts) {
			if x, err := strconv.ParseFloat(strings.Join(parts, ""), 64); err == nil {
				if accountingNegative { x = -math.Abs(x) }
				return x, true
			}
		}
		s = strings.ReplaceAll(s, ",", ".")
		if x, err := strconv.ParseFloat(s, 64); err == nil {
			if accountingNegative { x = -math.Abs(x) }
			return x, true
		}
		return 0, false
	}

	// Solo punto: un único grupo de tres dígitos después de un entero corto
	// se interpreta como miles (23.961 => 23961). Los demás puntos son
	// decimales (534.68, 1234.5).
	parts := strings.Split(s, ".")
	if len(parts) > 2 && allThousandGroups(parts) {
		if x, err := strconv.ParseFloat(strings.Join(parts, ""), 64); err == nil {
			if accountingNegative { x = -math.Abs(x) }
			return x, true
		}
	}
	if len(parts) == 2 && len(parts[1]) == 3 && len(parts[0]) > 0 && len(parts[0]) <= 3 {
		if x, err := strconv.ParseFloat(parts[0]+parts[1], 64); err == nil {
			if accountingNegative { x = -math.Abs(x) }
			return x, true
		}
	}
	if x, err := strconv.ParseFloat(s, 64); err == nil {
		if accountingNegative { x = -math.Abs(x) }
		return x, true
	}
	return 0, false
}

func allThousandGroups(parts []string) bool {
	if len(parts) < 2 || parts[0] == "" {
		return false
	}
	for _, p := range parts[1:] {
		if len(p) != 3 {
			return false
		}
	}
	return true
}

// maxColumns returns the widest row in a decoded XLSX sheet.
// It belongs in the platform-independent helpers because BuildMemoryWorkbook
// is compiled and tested on Windows as part of the complete package.
func maxColumns(rows [][]string) int {
	max := 0
	for _, row := range rows {
		if len(row) > max {
			max = len(row)
		}
	}
	return max
}
