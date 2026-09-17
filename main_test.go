package main

import "testing"

func TestDuplicatedPhysicalColumns(t *testing.T) {
	used := map[string]int{}
	ids := []string{
		uniqueNormalizedHeaderID("DESCRIPCION", used),
		uniqueNormalizedHeaderID("CODRETENCION", used),
		uniqueNormalizedHeaderID("N° CLIENTE", used),
		uniqueNormalizedHeaderID("DESCRIPCION", used),
		uniqueNormalizedHeaderID("CODRETENCION", used),
	}
	want := []string{"DESCRIPCION", "CODRETENCION", "N CLIENTE", "DESCRIPCION_2", "CODRETENCION_2"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("column id %d = %q, want %q; ids=%v", i, ids[i], want[i], ids)
		}
	}
	if ids[0] == ids[3] || ids[1] == ids[4] {
		t.Fatal("physical duplicate columns must have independent logical IDs")
	}
}
