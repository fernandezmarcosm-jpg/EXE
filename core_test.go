package main

import "testing"

func TestParseNumberLocaleFormats(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"1.234,56", 1234.56},
		{"1,234.56", 1234.56},
		{"1234,5", 1234.5},
		{"1234.5", 1234.5},
	}
	for _, tc := range tests {
		got, ok := parseNumber(tc.in)
		if !ok || got != tc.want {
			t.Fatalf("parseNumber(%q) = %v, %v; want %v, true", tc.in, got, ok, tc.want)
		}
	}
}

func TestParseRowsPreservesSparseColumns(t *testing.T) {
	rows := []sheetRowXML{{
		Cells: []sheetCellXML{
			{Ref: "A1", Value: "SKU"},
			{Ref: "C1", Value: "NETO SO"},
		},
	}, {
		Cells: []sheetCellXML{
			{Ref: "A2", Value: "100"},
			{Ref: "C2", Value: "1.234,50"},
		},
	}}
	got := parseRows(rows, nil)
	if len(got) != 2 || len(got[1]) != 3 {
		t.Fatalf("unexpected parsed dimensions: %#v", got)
	}
	if got[1][1] != "" || got[1][2] != "1.234,50" {
		t.Fatalf("sparse columns were shifted: %#v", got[1])
	}
}

func TestHeaderFiltersAreFieldSpecific(t *testing.T) {
	lines := []Line{
		{Values: map[string]string{"SO": "100", "Estado": "LIBERADA", "SKU": "ABC"}},
		{Values: map[string]string{"SO": "200", "Estado": "RETENIDA", "SKU": "XYZ"}},
	}
	got := BuildFilteredSortedViewByHeaders(lines, map[string]string{"SKU": "ABC"})
	if len(got) != 1 || got[0].Values["SO"] != "100" {
		t.Fatalf("field filter matched the wrong field: %#v", got)
	}
}
