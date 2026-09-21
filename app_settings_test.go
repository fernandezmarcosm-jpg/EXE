package main

import "testing"

func TestMergeProtectedColumnSettingsPreservesBackendOrderAndTitles(t *testing.T) {
	old := appSettings
	defer func() { appSettings = old }()

	appSettings = defaultDatasetSettings()
	appSettings.ColumnOrder = []string{"B", "A"}
	appSettings.ColumnTitles = map[string]string{"A": "Alias A"}

	incoming := defaultDatasetSettings()
	incoming.ColumnOrder = []string{"A", "B"}
	incoming.ColumnTitles = map[string]string{"A": "Alias viejo"}
	incoming.FontSize = 18

	mergeProtectedColumnSettings(&incoming)

	if len(incoming.ColumnOrder) != 2 || incoming.ColumnOrder[0] != "B" || incoming.ColumnOrder[1] != "A" {
		t.Fatalf("orden sobrescrito: %#v", incoming.ColumnOrder)
	}
	if incoming.ColumnTitles["A"] != "Alias A" {
		t.Fatalf("alias sobrescrito: %#v", incoming.ColumnTitles)
	}
	if incoming.FontSize != 18 {
		t.Fatalf("se modificó un campo visual no relacionado: %d", incoming.FontSize)
	}
}
