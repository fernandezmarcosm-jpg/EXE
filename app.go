package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct{ ctx context.Context }

type ColumnDTO struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Source string `json:"source"`
	Type string `json:"type"`
	Visible bool `json:"visible"`
}

type DatasetDTO struct {
	Columns []ColumnDTO `json:"columns"`
	Rows []map[string]string `json:"rows"`
	TotalRows int `json:"total_rows"`
	Duplicated int `json:"duplicated"`
	CSVRows int `json:"csv_rows"`
	Enriched int `json:"enriched"`
	SourceFiles []string `json:"source_files"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	appSettings = loadDatasetSettings()
}

func (a *App) ImportXLSX() (DatasetDTO, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Seleccionar uno o varios archivos Excel",
		Filters: []runtime.FileFilter{{DisplayName: "Archivos Excel (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil { return DatasetDTO{}, err }
	if len(paths) == 0 { return DatasetDTO{}, nil }

	docs := make([]*xlsxDoc, 0, len(paths))
	for _, path := range paths {
		doc, err := ReadXLSX(path)
		if err != nil { return DatasetDTO{}, fmt.Errorf("%s: %w", filepath.Base(path), err) }
		decorateXLSXDates(doc, path)
		docs = append(docs, doc)
	}
	ds, err := BuildMemoryDataset(docs, appSettings)
	if err != nil { return DatasetDTO{}, err }
	applyDatasetFormula(ds, appSettings)
	ds.SourceFiles = append([]string(nil), paths...)
	viewDataset = ds
	applySavedColumnVisibility(ds)
	applySavedColumnOrder(ds)
	return datasetDTO(ds), nil
}

func (a *App) GetSettings() DatasetSettings {
	appSettings = loadDatasetSettings()
	return appSettings
}

func (a *App) SaveSettings(s DatasetSettings) error {
	datasetSettingsNormalize(&s)
	appSettings = s
	return saveDatasetSettings(s)
}

func (a *App) SetFontSize(px int) error {
	if px < 10 { px = 10 }
	if px > 28 { px = 28 }
	appSettings.FontSize = px
	return saveDatasetSettings(appSettings)
}

func (a *App) SetRowHeight(px int) error {
	if px < 18 { px = 18 }
	if px > 60 { px = 60 }
	appSettings.RowHeight = px
	return saveDatasetSettings(appSettings)
}

func (a *App) SetColumnWidth(id string, px int) error {
	id = strings.TrimSpace(id)
	if id == "" { return fmt.Errorf("id de columna vacío") }
	if viewDataset != nil {
		found := false
		for _, c := range viewDataset.Columns { if c.ID == id { found = true; break } }
		if !found { return fmt.Errorf("columna desconocida: %s", id) }
	}
	if px < 60 { px = 60 }
	if px > 600 { px = 600 }
	if appSettings.ColumnWidths == nil { appSettings.ColumnWidths = map[string]int{} }
	appSettings.ColumnWidths[id] = px
	return saveDatasetSettings(appSettings)
}

func (a *App) SetVisibleColumns(ids []string) error {
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" { clean = append(clean, id) }
	}
	if len(clean) == 0 { appSettings.VisibleColumns = []string{"__NONE__"} } else { appSettings.VisibleColumns = clean }
	if viewDataset != nil {
		allowed := make(map[string]bool, len(clean))
		for _, id := range clean { allowed[id] = true }
		for i := range viewDataset.Columns { viewDataset.Columns[i].Visible = allowed[viewDataset.Columns[i].ID] }
	}
	return saveDatasetSettings(appSettings)
}

func (a *App) SetColumnOrder(ids []string) error {
	if viewDataset == nil { return fmt.Errorf("no hay un dataset cargado") }

	known := make(map[string]bool, len(viewDataset.Columns))
	for _, c := range viewDataset.Columns { known[c.ID] = true }
	seen := make(map[string]bool, len(ids))
	order := make([]string, 0, len(viewDataset.Columns))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] { continue }
		if !known[id] { return fmt.Errorf("columna desconocida: %s", id) }
		seen[id] = true
		order = append(order, id)
	}
	for _, c := range viewDataset.Columns {
		if !seen[c.ID] { order = append(order, c.ID) }
	}

	byID := make(map[string]DatasetColumn, len(viewDataset.Columns))
	for _, c := range viewDataset.Columns { byID[c.ID] = c }
	reordered := make([]DatasetColumn, 0, len(viewDataset.Columns))
	for _, id := range order { reordered = append(reordered, byID[id]) }
	viewDataset.Columns = reordered
	appSettings.ColumnOrder = append([]string(nil), order...)
	return saveDatasetSettings(appSettings)
}

func applySavedColumnOrder(ds *MemoryDataset) {
	if ds == nil || len(ds.Columns) == 0 || len(appSettings.ColumnOrder) == 0 { return }

	byID := make(map[string]DatasetColumn, len(ds.Columns))
	for _, c := range ds.Columns { byID[c.ID] = c }
	seen := make(map[string]bool, len(ds.Columns))
	reordered := make([]DatasetColumn, 0, len(ds.Columns))
	for _, id := range appSettings.ColumnOrder {
		if c, ok := byID[id]; ok && !seen[id] {
			reordered = append(reordered, c)
			seen[id] = true
		}
	}
	for _, c := range ds.Columns {
		if !seen[c.ID] { reordered = append(reordered, c) }
	}
	ds.Columns = reordered
}

func applySavedColumnVisibility(ds *MemoryDataset) {
	if ds == nil { return }
	if len(appSettings.VisibleColumns) == 0 {
		for i := range ds.Columns { ds.Columns[i].Visible = true }
		return
	}
	allowed := make(map[string]bool, len(appSettings.VisibleColumns))
	for _, id := range appSettings.VisibleColumns { if id != "__NONE__" { allowed[id] = true } }
	for i := range ds.Columns { ds.Columns[i].Visible = allowed[ds.Columns[i].ID] }
}

func datasetDTO(ds *MemoryDataset) DatasetDTO {
	out := DatasetDTO{
		Columns: make([]ColumnDTO, 0, len(ds.Columns)),
		Rows: make([]map[string]string, 0, len(ds.Records)),
		TotalRows: len(ds.Records), Duplicated: ds.DuplicateSO, CSVRows: ds.CSVRows, Enriched: ds.Enriched,
		SourceFiles: append([]string(nil), ds.SourceFiles...),
	}
	for _, c := range ds.Columns {
		out.Columns = append(out.Columns, ColumnDTO{ID:c.ID, Title:datasetColumnDisplayTitle(c), Source:c.Source, Type:valueTypeName(c.Type), Visible:c.Visible})
	}
	for _, record := range ds.Records {
		row := make(map[string]string, len(ds.Columns))
		for _, c := range ds.Columns {
			v, ok := record.Values[c.ID]
			if !ok { row[c.ID] = ""; continue }
			row[c.ID] = datasetValueText(c, v)
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

func datasetValueText(c DatasetColumn, v MemoryValue) string {
	switch v.Type {
	case ValueNumber:
		if datasetColumnIsPercent(c) { return formatDatasetNumber(v.Number*100, datasetColumnDecimals(c)) }
		return formatDatasetNumber(v.Number, datasetColumnDecimals(c))
	case ValueDate:
		return v.Raw
	default:
		return v.Raw
	}
}

func valueTypeName(t ValueType) string {
	switch t { case ValueNumber: return "number"; case ValueDate: return "date"; case ValueText: return "text"; default: return "empty" }
}
