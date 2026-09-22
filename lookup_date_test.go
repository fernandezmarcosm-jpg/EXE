package main

import "testing"

func TestParseDatasetDateAcceptsUnpaddedDateWithTime(t *testing.T) {
	want, ok := parseDatasetDate("13/9/2026 16:00:00")
	if !ok {
		t.Fatal("parseDatasetDate no aceptó 13/9/2026 16:00:00")
	}
	if got := want.Format("2006-01-02"); got != "2026-09-13" {
		t.Fatalf("fecha=%q; want 2026-09-13", got)
	}
}

func TestLookupRangeAcceptsDateTimeWithoutLeadingZeros(t *testing.T) {
	csvData := []byte("DESDE;HASTA;FECFACTURA;EJERCICIO\n1/9/2021;13/9/2026;;PRUEBA1\n")
	table, err := parseLookupTable(csvData, "EjercicioFechaHora", "EjercicioFechaHora.csv")
	if err != nil {
		t.Fatal(err)
	}
	sh := MemorySheet{Columns: []MemoryColumn{{ID: "FECFACTURA", Title: "FECFACTURA", Index: 0, Type: ValueDate}}}
	ids := map[string]map[string]string{"EjercicioFechaHora": {"EJERCICIO": "LOOKUP:EJERCICIOFECHAHORA:EJERCICIO"}}
	for _, raw := range []string{"13/9/2026 16:00:00", "13/09/2026 16:00:00", "13/9/2026"} {
		rec := DatasetRecord{Values: map[string]MemoryValue{}}
		row := MemoryRow{Values: map[string]MemoryValue{"FECFACTURA": {ColumnID: "FECFACTURA", Raw: raw, Type: ValueDate}}}
		if got := applyLookupEnrichment(&rec, sh, row, []lookupTable{table}, ids); got != 1 {
			t.Fatalf("fecha %q: matches=%d; want 1", raw, got)
		}
		if got := rec.Values["LOOKUP:EJERCICIOFECHAHORA:EJERCICIO"].Raw; got != "PRUEBA1" {
			t.Fatalf("fecha %q: atributo=%q; want PRUEBA1", raw, got)
		}
	}
}
