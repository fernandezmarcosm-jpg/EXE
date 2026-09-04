package main

import "testing"

func TestEvaluateFormulaUsesArithmeticAndSourceQualifiedColumns(t *testing.T) {
	cols := []DatasetColumn{
		{ID: "1", Title: "COLUMNA 1", Source: "XLSX", Type: ValueNumber},
		{ID: "2", Title: "COLUMNA 2", Source: "XLSX", Type: ValueNumber},
		{ID: "3", Title: "COLUMNA 3", Source: "CSV", Type: ValueNumber},
	}
	r := DatasetRecord{SO: "1", Values: map[string]MemoryValue{
		"1": {ColumnID: "1", Type: ValueNumber, Number: 10},
		"2": {ColumnID: "2", Type: ValueNumber, Number: 2},
		"3": {ColumnID: "3", Type: ValueNumber, Number: 3},
	}}
	v, ok := evaluateFormula("[COLUMNA 1] / [COLUMNA 2] * [CSV:COLUMNA 3]", r, cols)
	if !ok || v != 15 {
		t.Fatalf("formula=%v ok=%v; want 15,true", v, ok)
	}
}

func makeTestDoc(rows ...[3]string) *xlsxDoc {
	columns := []MemoryColumn{
		{ID: "SO", Title: "SO", Index: 0, Type: ValueText},
		{ID: "ITEM", Title: "ITEM", Index: 1, Type: ValueText},
		{ID: "SKU", Title: "SKU", Index: 2, Type: ValueText},
	}
	memoryRows := make([]MemoryRow, 0, len(rows))
	for _, x := range rows {
		memoryRows = append(memoryRows, MemoryRow{Values: map[string]MemoryValue{
			"SO": {Raw: x[0], Type: ValueText},
			"ITEM": {Raw: x[1], Type: ValueText},
			"SKU": {Raw: x[2], Type: ValueText},
		}})
	}
	return &xlsxDoc{Memory: &MemoryWorkbook{Sheets: []MemorySheet{{Columns: columns, Rows: memoryRows}}}}
}

func TestMemoryDatasetKeepsAllItemsForSameSO(t *testing.T) {
	s := defaultDatasetSettings()
	s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{
		makeTestDoc(
			[3]string{"100", "1", "ACE0001"},
			[3]string{"100", "2", "ACE0002"},
			[3]string{"100", "3", "ACE0003"},
		),
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Records) != 3 || m.DuplicateSO != 0 {
		t.Fatalf("records=%d duplicates=%d; want 3,0", len(m.Records), m.DuplicateSO)
	}
}

func TestMemoryDatasetDeduplicatesExactSOItemLine(t *testing.T) {
	s := defaultDatasetSettings()
	s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{
		makeTestDoc(
			[3]string{"100", "1", "ACE0001"},
			[3]string{"100", "1", "ACE0001"},
			[3]string{"100", "2", "ACE0002"},
		),
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Records) != 2 || m.DuplicateSO != 1 {
		t.Fatalf("records=%d duplicates=%d; want 2,1", len(m.Records), m.DuplicateSO)
	}
}

func TestMemoryDatasetDoesNotDeduplicateBySOWhenITEMIsMissing(t *testing.T) {
	d := &xlsxDoc{Memory: &MemoryWorkbook{Sheets: []MemorySheet{{
		Columns: []MemoryColumn{{ID: "SO", Title: "SO", Index: 0, Type: ValueText}},
		Rows: []MemoryRow{
			{Values: map[string]MemoryValue{"SO": {Raw: "100", Type: ValueText}}},
			{Values: map[string]MemoryValue{"SO": {Raw: "100", Type: ValueText}}},
		},
	}}}}
	s := defaultDatasetSettings()
	s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{d}, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Records) != 2 || m.DuplicateSO != 0 {
		t.Fatalf("records=%d duplicates=%d; want 2,0", len(m.Records), m.DuplicateSO)
	}
}

func TestEmbeddedMasterCSVIsAvailable(t *testing.T) {
	m, _, err := loadMasterCSV("")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ByKey) == 0 {
		t.Fatal("embedded CSV master is empty")
	}
	if _, ok := m.ByKey["ACE0001"]; !ok {
		t.Fatal("expected ACE0001 in embedded master")
	}
}
