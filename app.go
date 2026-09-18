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
	Subtotals []SubtotalRow `json:"subtotals"`
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

func (a *App) ListCalculatedColumns() []CalculatedColumn {
	out := make([]CalculatedColumn, len(appSettings.CalculatedColumns))
	copy(out, appSettings.CalculatedColumns)
	return out
}

func calculatedNameExists(name, except string) bool {
	n := strings.TrimSpace(name)
	for _, c := range viewDataset.Columns {
		if strings.EqualFold(strings.TrimSpace(c.Title), n) && !strings.EqualFold(strings.TrimSpace(c.Title), strings.TrimSpace(except)) {
			return true
		}
	}
	for _, c := range appSettings.CalculatedColumns {
		if strings.EqualFold(strings.TrimSpace(c.Name), n) && !strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(except)) {
			return true
		}
	}
	return false
}

func validateCalculatedFormula(name, formula string, original string) error {
	name = strings.TrimSpace(name)
	formula = strings.TrimSpace(formula)
	if name == "" { return fmt.Errorf("el nombre del campo calculado no puede estar vacío") }
	if formula == "" { return fmt.Errorf("la fórmula no puede estar vacía") }
	if viewDataset == nil || len(viewDataset.Records) == 0 { return fmt.Errorf("no hay datos cargados para validar la fórmula") }
	if calculatedNameExists(name, original) { return fmt.Errorf("ya existe una columna con el nombre %q", name) }
	if _, ok := evaluateFormula(formula, viewDataset.Records[0], viewDataset.Columns); !ok {
		return fmt.Errorf("fórmula inválida o referencia a una columna no numérica: %s", formula)
	}
	return nil
}

func syncCalculatedFormatSettings(ds *MemoryDataset) {
	if ds == nil { return }
	if appSettings.ColumnPercent == nil { appSettings.ColumnPercent = map[string]bool{} }
	if appSettings.ColumnTypes == nil { appSettings.ColumnTypes = map[string]string{} }
	for _, cc := range appSettings.CalculatedColumns {
		c, ok := ds.columnByTitle(cc.Name)
		if !ok { continue }
		appSettings.ColumnPercent[c.ID] = cc.Percent
		if cc.Percent {
			appSettings.ColumnTypes[c.ID] = "porcentaje"
		} else if strings.EqualFold(strings.TrimSpace(appSettings.ColumnTypes[c.ID]), "porcentaje") {
			appSettings.ColumnTypes[c.ID] = "decimal"
		}
	}
}

func rebuildCalculatedColumns() {
	if viewDataset == nil { return }
	for i := len(viewDataset.Columns)-1; i >= 0; i-- {
		if viewDataset.Columns[i].Source != "CALCULADA" { continue }
		id := viewDataset.Columns[i].ID
		for r := range viewDataset.Records { delete(viewDataset.Records[r].Values, id) }
		delete(appSettings.ColumnPercent, id)
		delete(appSettings.ColumnTypes, id)
		viewDataset.Columns = append(viewDataset.Columns[:i], viewDataset.Columns[i+1:]...)
	}
	ensureCalculatedDatasetColumns(viewDataset, appSettings)
	syncCalculatedFormatSettings(viewDataset)
	applyDatasetFormula(viewDataset, appSettings)
	applySavedColumnVisibility(viewDataset)
	applySavedColumnOrder(viewDataset)
	_ = saveDatasetSettings(appSettings)
}

func (a *App) AddCalculatedColumn(name, formula string, percent bool) (DatasetDTO, error) {
	name = strings.TrimSpace(name); formula = strings.TrimSpace(formula)
	if err := validateCalculatedFormula(name, formula, ""); err != nil { return DatasetDTO{}, err }
	appSettings.CalculatedColumns = append(appSettings.CalculatedColumns, CalculatedColumn{Name:name, Formula:formula, Percent:percent})
	if err := saveDatasetSettings(appSettings); err != nil { return DatasetDTO{}, err }
	rebuildCalculatedColumns()
	return datasetDTO(viewDataset), nil
}

func (a *App) UpdateCalculatedColumn(originalName, name, formula string, percent bool) (DatasetDTO, error) {
	originalName = strings.TrimSpace(originalName); name = strings.TrimSpace(name); formula = strings.TrimSpace(formula)
	idx := -1
	for i, c := range appSettings.CalculatedColumns {
		if strings.EqualFold(strings.TrimSpace(c.Name), originalName) { idx = i; break }
	}
	if idx < 0 { return DatasetDTO{}, fmt.Errorf("campo calculado no encontrado: %s", originalName) }
	if err := validateCalculatedFormula(name, formula, originalName); err != nil { return DatasetDTO{}, err }
	appSettings.CalculatedColumns[idx] = CalculatedColumn{Name:name, Formula:formula, Percent:percent}
	if err := saveDatasetSettings(appSettings); err != nil { return DatasetDTO{}, err }
	rebuildCalculatedColumns()
	return datasetDTO(viewDataset), nil
}

func (a *App) DeleteCalculatedColumn(name string) (DatasetDTO, error) {
	name = strings.TrimSpace(name)
	idx := -1
	for i, c := range appSettings.CalculatedColumns {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) { idx = i; break }
	}
	if idx < 0 { return DatasetDTO{}, fmt.Errorf("campo calculado no encontrado: %s", name) }
	appSettings.CalculatedColumns = append(appSettings.CalculatedColumns[:idx], appSettings.CalculatedColumns[idx+1:]...)
	if err := saveDatasetSettings(appSettings); err != nil { return DatasetDTO{}, err }
	rebuildCalculatedColumns()
	return datasetDTO(viewDataset), nil
}

func (a *App) SetColumnFormat(id string, decimals int, percent bool, kind string) (DatasetDTO, error) {
	id = strings.TrimSpace(id); kind = strings.ToLower(strings.TrimSpace(kind))
	if viewDataset == nil { return DatasetDTO{}, fmt.Errorf("no hay un dataset cargado") }
	found := false
	for _, c := range viewDataset.Columns { if c.ID == id { found = true; break } }
	if !found { return DatasetDTO{}, fmt.Errorf("columna desconocida: %s", id) }
	if kind != "entero" && kind != "decimal" && kind != "porcentaje" { return DatasetDTO{}, fmt.Errorf("tipo de formato inválido: %s", kind) }
	if kind == "entero" { decimals = 0 }
	if decimals < 0 { decimals = 0 }; if decimals > 8 { decimals = 8 }
	if appSettings.ColumnDecimals == nil { appSettings.ColumnDecimals = map[string]int{} }
	if appSettings.ColumnPercent == nil { appSettings.ColumnPercent = map[string]bool{} }
	if appSettings.ColumnTypes == nil { appSettings.ColumnTypes = map[string]string{} }
	appSettings.ColumnDecimals[id] = decimals
	appSettings.ColumnPercent[id] = percent || kind == "porcentaje"
	appSettings.ColumnTypes[id] = kind
	if err := saveDatasetSettings(appSettings); err != nil { return DatasetDTO{}, err }
	return datasetDTO(viewDataset), nil
}

func (a *App) SetSubtotals(groupColumnID string, agg map[string]string) (DatasetDTO, error) {
	groupColumnID = strings.TrimSpace(groupColumnID)
	if viewDataset == nil { return DatasetDTO{}, fmt.Errorf("no hay un dataset cargado") }
	if groupColumnID != "" {
		found := false
		for _, c := range viewDataset.Columns { if c.ID == groupColumnID { found = true; break } }
		if !found { return DatasetDTO{}, fmt.Errorf("columna de agrupación desconocida: %s", groupColumnID) }
	}
	clean := map[string]string{}
	for id, v := range agg {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" && v != "suma" && v != "promedio" { return DatasetDTO{}, fmt.Errorf("agregación inválida para %s: %s", id, v) }
		if v != "" {
			found := false; for _, c := range viewDataset.Columns { if c.ID == id { found = true; break } }
			if !found { return DatasetDTO{}, fmt.Errorf("columna de subtotal desconocida: %s", id) }
			clean[id] = v
		}
	}
	appSettings.SubtotalColumn = groupColumnID
	appSettings.SubtotalAgg = clean
	appSettings.SubtotalEnabled = groupColumnID != "" && len(clean) > 0
	if err := saveDatasetSettings(appSettings); err != nil { return DatasetDTO{}, err }
	return datasetDTO(viewDataset), nil
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
		Subtotals: computeSubtotals(ds),
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
		if datasetColumnIsPercent(c) { return formatDatasetNumber(v.Number*100, datasetColumnDecimals(c)) + "%" }
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
