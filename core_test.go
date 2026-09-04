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

func TestBuildMemoryWorkbookUsesOnlyRowsBelowHeaders(t *testing.T) {
	doc := &xlsxDoc{Sheets: map[string][][]string{
		"sheet1": {
			{"INFORME DE DESCUENTOS", "", ""},
			{"otra linea informativa", "", ""},
			{"SKU", "DESCRIPCION", "PRECIOUNIFC"},
			{"001", "HARINA", "1.234,50"},
			{"002", "AZUCAR", "800"},
		},
	}}
	m := BuildMemoryWorkbook(doc)
	if len(m.Sheets) != 1 {
		t.Fatalf("expected one sheet, got %d", len(m.Sheets))
	}
	s := m.Sheets[0]
	if s.HeaderIndex != 2 {
		t.Fatalf("expected header row 2 (zero based), got %d", s.HeaderIndex)
	}
	if len(s.Columns) != 3 || len(s.Rows) != 2 {
		t.Fatalf("unexpected memory dimensions: columns=%d rows=%d", len(s.Columns), len(s.Rows))
	}
	if s.Columns[0].ID != "S001C001" || s.Columns[1].ID != "S001C002" || s.Columns[2].ID != "S001C003" {
		t.Fatalf("column IDs are not stable: %#v", s.Columns)
	}
	if s.Rows[0].ID != "S001R000001" || s.Rows[1].ID != "S001R000002" {
		t.Fatalf("row IDs are not stable: %#v", s.Rows)
	}
	v := s.Rows[0].Values[s.Columns[2].ID]
	if v.Type != ValueNumber || v.Number != 1234.5 || v.Raw != "1.234,50" {
		t.Fatalf("numeric value was not normalized while preserving raw value: %#v", v)
	}
	if s.Rows[0].Values[s.Columns[0].ID].Type != ValueText {
		t.Fatalf("SKU must remain text to preserve identifiers")
	}
}

func TestMemoryWorkbookKeepsDuplicateTitlesUnique(t *testing.T) {
	doc := &xlsxDoc{Sheets: map[string][][]string{
		"sheet1": {
			{"SKU", "IMPORTE", "IMPORTE"},
			{"1", "10", "20"},
		},
	}}
	m := BuildMemoryWorkbook(doc)
	cols := m.Sheets[0].Columns
	if len(cols) != 3 || cols[1].ID == cols[2].ID || cols[1].Title == cols[2].Title {
		t.Fatalf("duplicate column titles were not made unique: %#v", cols)
	}
}
