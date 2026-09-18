package main

import "encoding/json"

type visualDatasetSettings struct {
	RowHeight int `json:"row_height"`
	ColumnWidths map[string]int `json:"column_widths"`
}

var persistedVisualSettings = visualDatasetSettings{RowHeight: 28, ColumnWidths: map[string]int{}}

func normalizeVisualDatasetSettings() {
	if persistedVisualSettings.RowHeight < 18 || persistedVisualSettings.RowHeight > 60 { persistedVisualSettings.RowHeight = 28 }
	if persistedVisualSettings.ColumnWidths == nil { persistedVisualSettings.ColumnWidths = map[string]int{} }
	for id, width := range persistedVisualSettings.ColumnWidths { if width < 60 || width > 600 { delete(persistedVisualSettings.ColumnWidths, id) } }
}

func (s DatasetSettings) MarshalJSON() ([]byte, error) {
	normalizeVisualDatasetSettings()
	type alias DatasetSettings
	return json.Marshal(struct { alias; RowHeight int `json:"row_height"`; ColumnWidths map[string]int `json:"column_widths"` }{alias: alias(s), RowHeight: persistedVisualSettings.RowHeight, ColumnWidths: persistedVisualSettings.ColumnWidths})
}

func (s *DatasetSettings) UnmarshalJSON(data []byte) error {
	type alias DatasetSettings
	var aux struct { alias; RowHeight *int `json:"row_height"`; ColumnWidths map[string]int `json:"column_widths"` }
	if err := json.Unmarshal(data, &aux); err != nil { return err }
	*s = DatasetSettings(aux.alias)
	if aux.RowHeight != nil { persistedVisualSettings.RowHeight = *aux.RowHeight }
	if aux.ColumnWidths != nil { persistedVisualSettings.ColumnWidths = aux.ColumnWidths }
	normalizeVisualDatasetSettings()
	return nil
}

func currentRowHeight() int { normalizeVisualDatasetSettings(); return persistedVisualSettings.RowHeight }
func currentColumnWidths() map[string]int { normalizeVisualDatasetSettings(); out := make(map[string]int, len(persistedVisualSettings.ColumnWidths)); for id, width := range persistedVisualSettings.ColumnWidths { out[id] = width }; return out }
