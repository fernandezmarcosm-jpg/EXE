package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMemoryDatasetLookupUsesMatchingNonFirstCSVKey(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	defer os.Chdir(oldWD)
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "Clientes.csv"), []byte("ATRIBUTO;CLIENTE\nCadena;80003285\n"), 0644); err != nil { t.Fatal(err) }

	doc := &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{
		Columns: []MemoryColumn{
			{ID:"SO", Title:"SO", Index:0, Type:ValueText},
			{ID:"CLIENTE", Title:"Nº CLIENTE", Index:1, Type:ValueNumber},
		},
		Rows: []MemoryRow{{Values:map[string]MemoryValue{
			"SO": {ColumnID:"SO", Raw:"100", Type:ValueText},
			"CLIENTE": {ColumnID:"CLIENTE", Raw:"80003285.00", Type:ValueNumber},
		}}},
	}}}}
	s := defaultDatasetSettings()
	s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{doc}, s)
	if err != nil { t.Fatal(err) }
	v, ok := m.Records[0].Values["LOOKUP:CLIENTES:ATRIBUTO"]
	if !ok || v.Raw != "Cadena" { t.Fatalf("lookup value: ok=%v raw=%q; want Cadena", ok, v.Raw) }
	if len(m.LookupDiagnostics) != 1 { t.Fatalf("diagnostics=%d; want 1", len(m.LookupDiagnostics)) }
	d := m.LookupDiagnostics[0]
	if d.KeyHeader != "CLIENTE" || d.MatchedColumns != 1 || d.EnrichedRows != 1 { t.Fatalf("diagnostic=%+v; want key CLIENTE, matched=1, enriched=1", d) }
}
