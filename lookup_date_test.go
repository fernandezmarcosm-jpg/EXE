package main

import (
	"testing"
	"time"
)

func TestParseDatasetDateAcceptsUnpaddedDateWithTime(t *testing.T) {
	want, ok := parseDatasetDate("13/9/2026 16:00:00")
	if !ok { t.Fatal("parseDatasetDate no aceptó 13/9/2026 16:00:00") }
	if got := want.Format("2006-01-02"); got != "2026-09-13" { t.Fatalf("fecha=%q; want 2026-09-13", got) }
}

func TestLookupRangeAcceptsDateTimeWithoutLeadingZeros(t *testing.T) {
	csvData := []byte("DESDE;HASTA;FECFACTURA;EJERCICIO\n1/9/2021;13/9/2026;;PRUEBA1\n")
	table, err := parseLookupTable(csvData, "EjercicioFechaHora", "EjercicioFechaHora.csv")
	if err != nil { t.Fatal(err) }
	sh := MemorySheet{Columns: []MemoryColumn{{ID: "FECFACTURA", Title: "FECFACTURA", Index: 0, Type: ValueDate}}}
	ids := map[string]map[string]string{"EjercicioFechaHora": {"EJERCICIO": "LOOKUP:EJERCICIOFECHAHORA:EJERCICIO"}}
	for _, raw := range []string{"13/9/2026 16:00:00", "13/09/2026 16:00:00", "13/9/2026"} {
		rec := DatasetRecord{Values: map[string]MemoryValue{}}
		row := MemoryRow{Values: map[string]MemoryValue{"FECFACTURA": {ColumnID: "FECFACTURA", Raw: raw, Type: ValueDate}}}
		if got := applyLookupEnrichment(&rec, sh, row, []lookupTable{table}, ids); got != 1 { t.Fatalf("fecha %q: matches=%d; want 1", raw, got) }
		if got := rec.Values["LOOKUP:EJERCICIOFECHAHORA:EJERCICIO"].Raw; got != "PRUEBA1" { t.Fatalf("fecha %q: atributo=%q; want PRUEBA1", raw, got) }
	}
}

func TestLookupRangeExcelSerialUsesCalendarDateInNegativeTimezone(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.FixedZone("ART", -3*60*60)
	defer func() { time.Local = oldLocal }()

	from, ok := parseDatasetDate("01/09/2021")
	if !ok { t.Fatal("could not parse range start") }
	to, ok := parseDatasetDate("13/09/2026")
	if !ok { t.Fatal("could not parse range end") }

	table := lookupTable{
		Name: "Ejercicio",
		KeyHeader: "FECFACTURA",
		DateKeyHeader: "FECFACTURA",
		Headers: []string{"DESDE", "HASTA", "FECFACTURA", "EJERCICIO"},
		IsRange: true,
		Ranges: []lookupRange{{From: from, To: to, Values: map[string]string{"EJERCICIO": "PRUEBA1"}}},
	}
	sh := MemorySheet{Columns: []MemoryColumn{{ID: "FEC", Title: "FECFACTURA", Index: 0, Type: ValueDate}}}
	row := MemoryRow{Values: map[string]MemoryValue{"FEC": {ColumnID: "FEC", Type: ValueDate, Raw: "46278"}}}
	rec := DatasetRecord{Values: map[string]MemoryValue{}}
	ids := map[string]map[string]string{"Ejercicio": {"EJERCICIO": "LOOKUP:EJERCICIO:EJERCICIO"}}

	if got := applyLookupEnrichment(&rec, sh, row, []lookupTable{table}, ids); got != 1 { t.Fatalf("matches=%d; want 1", got) }
	if got := rec.Values["LOOKUP:EJERCICIO:EJERCICIO"].Raw; got != "PRUEBA1" { t.Fatalf("attribute=%q; want PRUEBA1", got) }
}

func TestLookupDateOnlyAlwaysUsesUTC(t *testing.T) {
	local := time.FixedZone("ART", -3*60*60)
	got := lookupDateOnly(time.Date(2026, 9, 13, 0, 0, 0, 0, local))
	if got.Location() != time.UTC { t.Fatalf("location=%v; want UTC", got.Location()) }
	if got.Year() != 2026 || got.Month() != 9 || got.Day() != 13 || got.Hour() != 0 { t.Fatalf("date=%v; want 2026-09-13 00:00 UTC", got) }
}
