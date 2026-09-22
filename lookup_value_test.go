package main

import "testing"

func TestLookupEnrichmentMatchesNumericClienteAndPrefersExactHeader(t *testing.T) {
	table, err := parseLookupTable([]byte("Nº CLIENTE;ATRIBUTO\n80003285;Cadena\n"), "Clientes", "Clientes.csv")
	if err != nil { t.Fatal(err) }
	table = reindexLookupTable(table, "Nº CLIENTE")
	sh := MemorySheet{
		Columns: []MemoryColumn{
			{ID: "NOMBRE", Title: "CLIENTE", Index: 0, Type: ValueText},
			{ID: "NUM", Title: "Nº CLIENTE", Index: 1, Type: ValueNumber},
		},
		Rows: []MemoryRow{{Values: map[string]MemoryValue{
			"NOMBRE": {ColumnID: "NOMBRE", Raw: "Juan", Type: ValueText},
			"NUM": {ColumnID: "NUM", Raw: "80003285.00", Type: ValueNumber},
		}}},
	}
	if got := lookupColumnForKey(sh, "Nº CLIENTE"); got != "NUM" { t.Fatalf("lookup column=%q; want NUM", got) }
	rec := DatasetRecord{Values: map[string]MemoryValue{}}
	ids := map[string]map[string]string{"Clientes": {"ATRIBUTO": "LOOKUP:CLIENTES:ATRIBUTO"}}
	matched := applyLookupEnrichment(&rec, sh, sh.Rows[0], []lookupTable{table}, ids)
	if matched != 1 { t.Fatalf("matched=%d; want 1", matched) }
	if got := rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw; got != "Cadena" { t.Fatalf("lookup attribute=%q; want Cadena", got) }
}

func TestLookupEnrichmentNormalizesNumericThousands(t *testing.T) {
	table, err := parseLookupTable([]byte("NRO CLIENTE;ATRIBUTO\n80.003.285;Cadena\n"), "Clientes", "Clientes.csv")
	if err != nil { t.Fatal(err) }
	table = reindexLookupTable(table, "NRO CLIENTE")
	sh := MemorySheet{
		Columns: []MemoryColumn{{ID: "NUM", Title: "NRO CLIENTE", Index: 0, Type: ValueNumber}},
		Rows: []MemoryRow{{Values: map[string]MemoryValue{"NUM": {ColumnID: "NUM", Raw: "80003285.00", Type: ValueNumber}}}},
	}
	rec := DatasetRecord{Values: map[string]MemoryValue{}}
	ids := map[string]map[string]string{"Clientes": {"ATRIBUTO": "LOOKUP:CLIENTES:ATRIBUTO"}}
	if got := applyLookupEnrichment(&rec, sh, sh.Rows[0], []lookupTable{table}, ids); got != 1 || rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw != "Cadena" {
		t.Fatalf("lookup failed: matches=%d value=%q", got, rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw)
	}
}
