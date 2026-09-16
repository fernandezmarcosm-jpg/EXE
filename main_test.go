//go:build windows

package main

import "testing"

func TestDuplicatedPhysicalColumns(t *testing.T) {
    oldDataset := viewDataset
    oldSettings := appSettings
    defer func() { viewDataset = oldDataset; appSettings = oldSettings }()

    viewDataset = &MemoryDataset{
        Columns: []DatasetColumn{
            {ID: "DESCRIPCION", Title: "DESCRIPCION", Visible: true},
            {ID: "CODRETENCION", Title: "CODRETENCION", Visible: true},
            {ID: "N° CLIENTE", Title: "N° CLIENTE", Visible: true},
            {ID: "DESCRIPCION_2", Title: "DESCRIPCION", Visible: true},
            {ID: "CODRETENCION_2", Title: "CODRETENCION", Visible: true},
        },
        Records: []DatasetRecord{{Values: map[string]MemoryValue{
            "DESCRIPCION": {ColumnID: "DESCRIPCION", Type: ValueText, Raw: "primera"},
            "CODRETENCION": {ColumnID: "CODRETENCION", Type: ValueText, Raw: "A"},
            "N° CLIENTE": {ColumnID: "N° CLIENTE", Type: ValueText, Raw: "1"},
            "DESCRIPCION_2": {ColumnID: "DESCRIPCION_2", Type: ValueText, Raw: "segunda"},
            "CODRETENCION_2": {ColumnID: "CODRETENCION_2", Type: ValueText, Raw: "B"},
        }}}},
    }
    appSettings = defaultDatasetSettings()
    appSettings.MaxColumns = 20
    viewDataset.Columns[0].Visible = false

    visible := columnViewVisibleColumns()
    got := make([]string, 0, len(visible))
    for _, c := range visible { got = append(got, c.ID) }
    want := []string{"CODRETENCION", "N° CLIENTE", "DESCRIPCION_2", "CODRETENCION_2"}
    if len(got) != len(want) { t.Fatalf("visible columns = %v, want %v", got, want) }
    for i := range want { if got[i] != want[i] { t.Fatalf("visible columns = %v, want %v", got, want) } }

    r := viewDataset.Records[0]
    if r.Values["DESCRIPCION_2"].Raw == r.Values["DESCRIPCION"].Raw { t.Fatalf("duplicated description values are not independent") }
    if got := r.Values["DESCRIPCION_2"].Raw; got != "segunda" { t.Fatalf("second description = %q, want %q", got, "segunda") }
}
