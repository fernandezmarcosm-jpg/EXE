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

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "'", "")
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		if len(parts) == 2 && len(parts[1]) <= 2 {
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	}
	if x, err := strconv.ParseFloat(s, 64); err == nil {
		return x, true
	}
	return 0, false
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
