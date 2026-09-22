package main

import("bytes";_ "embed";"encoding/csv";"encoding/json";"fmt";"math";"os";"path/filepath";"sort";"strconv";"strings";"time")

//go:embed "acceso chatgpt/GestionSO_Datos.csv"
var embeddedMasterCSV []byte

type CalculatedColumn struct{Name string; Formula string; Percent bool}
type DatasetSettings struct{Decimals int `json:"decimals"`;FontSize int `json:"font_size"`;RowHeight int `json:"row_height"`;ColumnWidths map[string]int `json:"column_widths"`;SOColumn int `json:"so_column"`;JoinExcelColumn string `json:"join_excel_column"`;FormulaTitle string `json:"formula_title"`;Formula string `json:"formula"`;SubtotalColumn string `json:"subtotal_column"`;SubtotalEnabled bool `json:"subtotal_enabled"`;ColumnTitles map[string]string `json:"column_titles"`;ColumnOrder []string `json:"column_order"`;MaxColumns int `json:"max_columns"`;VisibleColumns []string `json:"visible_columns"`;ColumnDecimals map[string]int `json:"column_decimals"`;SubtotalColumns []string `json:"subtotal_columns"`;ColumnPercent map[string]bool `json:"column_percent"`;ColumnCurrency map[string]bool `json:"column_currency"`;ColumnThousands map[string]bool `json:"column_thousands"`;HighlightNegative map[string]bool `json:"highlight_negative"`;ColumnHighlightSign map[string]bool `json:"column_highlight_sign"`;ColumnBackground map[string]string `json:"column_background"`;ColumnAlign map[string]string `json:"column_align"`;ColumnTypes map[string]string `json:"column_types"`;CalculatedColumns []CalculatedColumn `json:"calculated_columns"`;SubtotalAgg map[string]string `json:"subtotal_agg"`}
func defaultDatasetSettings()DatasetSettings{return DatasetSettings{Decimals:2,FontSize:10,RowHeight:28,ColumnWidths:map[string]int{},SOColumn:5,JoinExcelColumn:"CLAVE",FormulaTitle:"CALCULADA",Formula:"",SubtotalEnabled:false,ColumnTitles:map[string]string{},ColumnOrder:[]string{},MaxColumns:500,VisibleColumns:[]string{},ColumnDecimals:map[string]int{},ColumnPercent:map[string]bool{},ColumnCurrency:map[string]bool{},ColumnThousands:map[string]bool{},HighlightNegative:map[string]bool{},ColumnHighlightSign:map[string]bool{},ColumnBackground:map[string]string{},ColumnAlign:map[string]string{},ColumnTypes:map[string]string{},CalculatedColumns:[]CalculatedColumn{},SubtotalAgg:map[string]string{}}}
func datasetSettingsPortablePath()string{if x,e:=os.Executable();e==nil&&strings.TrimSpace(x)!=""{return filepath.Join(filepath.Dir(x),"dataset.txt")};if d,e:=os.Getwd();e==nil&&strings.TrimSpace(d)!=""{return filepath.Join(d,"dataset.txt")};return ""}
func datasetSettingsFallbackPath()string{if d,e:=os.UserConfigDir();e==nil&&strings.TrimSpace(d)!=""{return filepath.Join(d,"GestionSO V57","dataset.txt")};return filepath.Join(os.TempDir(),"GestionSO-V57-dataset.txt")}
func datasetSettingsPaths()[]string{paths:=make([]string,0,2);add:=func(p string){if strings.TrimSpace(p)==""{return};for _,x:=range paths{if filepath.Clean(x)==filepath.Clean(p){return}};paths=append(paths,p)};add(datasetSettingsPortablePath());add(datasetSettingsFallbackPath());return paths}
func datasetSettingsPath()string{paths:=datasetSettingsPaths();if len(paths)>0{return paths[0]};return filepath.Join(os.TempDir(),"GestionSO-V57-dataset.txt")}
func loadDatasetSettings()DatasetSettings{s:=defaultDatasetSettings();for _,p:=range datasetSettingsPaths(){b,e:=os.ReadFile(p);if e!=nil{continue};if json.Unmarshal(b,&s)==nil{datasetSettingsNormalize(&s);return s}};return s}
func datasetSettingsNormalize(s *DatasetSettings){if s.Decimals<0||s.Decimals>8{s.Decimals=2};if s.FontSize<=0{s.FontSize=14};if s.FontSize<6{s.FontSize=6};if s.FontSize>28{s.FontSize=28};if s.RowHeight<10||s.RowHeight>60{s.RowHeight=28};if s.ColumnWidths==nil{s.ColumnWidths=map[string]int{}};if s.SOColumn<1||s.SOColumn>1024{s.SOColumn=5};if s.MaxColumns<1||s.MaxColumns>100{s.MaxColumns=20};if s.ColumnTitles==nil{s.ColumnTitles=map[string]string{}};if s.ColumnDecimals==nil{s.ColumnDecimals=map[string]int{}};if s.ColumnPercent==nil{s.ColumnPercent=map[string]bool{}};if s.ColumnCurrency==nil{s.ColumnCurrency=map[string]bool{}};if s.ColumnThousands==nil{s.ColumnThousands=map[string]bool{}};if s.ColumnHighlightSign==nil{s.ColumnHighlightSign=map[string]bool{}};if s.ColumnBackground==nil{s.ColumnBackground=map[string]string{}};if s.ColumnAlign==nil{s.ColumnAlign=map[string]string{}};for id,v:=range s.ColumnAlign{v=strings.ToLower(strings.TrimSpace(v));if v!="left"&&v!="center"&&v!="right"{delete(s.ColumnAlign,id)}else{s.ColumnAlign[id]=v}};if s.HighlightNegative==nil{s.HighlightNegative=map[string]bool{}};if s.ColumnTypes==nil{s.ColumnTypes=map[string]string{}};if s.SubtotalAgg==nil{s.SubtotalAgg=map[string]string{}}}
func saveDatasetSettings(s DatasetSettings)error{datasetSettingsNormalize(&s);if viewDataset!=nil{keys:=make([]string,0,len(viewDataset.Columns));for _,c:=range viewDataset.Columns{if c.Visible{keys=append(keys,datasetColumnKey(c))}};if len(keys)==0{keys=[]string{"__NONE__"}};s.VisibleColumns=keys};b,e:=json.MarshalIndent(s,"","  ");if e!=nil{return e};paths:=datasetSettingsPaths();var lastErr error;for _,p:=range paths{if e:=os.MkdirAll(filepath.Dir(p),0755);e!=nil{lastErr=e;continue};if e:=os.WriteFile(p,b,0644);e==nil{return nil}else{lastErr=e}};if lastErr!=nil{return lastErr};return fmt.Errorf("no hay una ruta disponible para guardar dataset.txt")}

type DatasetColumn struct{ID,Title,Source string;Type ValueType;Width int;Visible bool}
type DatasetRecord struct{SO string;Values map[string]MemoryValue}
type LookupSummary struct{Name string `json:"name"`;KeyHeader string `json:"key_header"`;Rows int `json:"rows"`;MatchedColumns int `json:"matched_columns"`;EnrichedRows int `json:"enriched_rows"`;DateParseFailures int `json:"date_parse_failures"`}
type MemoryDataset struct{Columns []DatasetColumn;Records []DatasetRecord;CSVRows,Enriched,DuplicateSO int;SourceFiles []string;LookupDiagnostics []LookupSummary `json:"lookup_diagnostics"`}
type csvMaster struct{Headers []string;ByKey map[string]map[string]string}
func normalizeJoinKey(v string) string {
	s := strings.TrimSpace(strings.ToUpper(v))
	if s == "" { return "" }
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	// Scientific notation is handled first because E is not a separator.
	if n, err := strconv.ParseFloat(s, 64); err == nil && strings.ContainsAny(s, "E") {
		if math.Trunc(n) == n { return strconv.FormatInt(int64(n), 10) }
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	numeric := true
	for _, r := range s {
		if !(r >= '0' && r <= '9') && r != '.' && r != ',' && r != '-' && r != '+' { numeric = false; break }
	}
	if !numeric { return s }
	sign, body := "", s
	if strings.HasPrefix(body, "-") || strings.HasPrefix(body, "+") { sign, body = body[:1], body[1:] }
	if body == "" { return s }
	lastComma, lastDot := strings.LastIndex(body, ","), strings.LastIndex(body, ".")
	decimalPos := -1
	if lastComma >= 0 || lastDot >= 0 {
		last := lastComma
		if lastDot > last { last = lastDot }
		fracLen := len(body) - last - 1
		if lastComma >= 0 && lastDot >= 0 {
			if fracLen > 0 && fracLen <= 2 { decimalPos = last }
		} else {
			count := strings.Count(body, ",") + strings.Count(body, ".")
			if fracLen > 0 && fracLen <= 2 { decimalPos = last
			} else if count == 1 && fracLen == 3 && len(body[:last]) <= 3 {
				decimalPos = -1
			} else if count > 1 { decimalPos = -1 }
		}
	}
	if decimalPos >= 0 {
		integer := strings.NewReplacer(".", "", ",", "").Replace(body[:decimalPos])
		fraction := strings.NewReplacer(".", "", ",", "").Replace(body[decimalPos+1:])
		if fraction == "" { return sign + integer }
		if n, err := strconv.ParseFloat(sign+integer+"."+fraction, 64); err == nil {
			if math.Trunc(n) == n { return strconv.FormatInt(int64(n), 10) }
			return strconv.FormatFloat(n, 'f', -1, 64)
		}
	}
	digits := strings.NewReplacer(".", "", ",", "").Replace(body)
	if digits != "" {
		if n, err := strconv.ParseInt(sign+digits, 10, 64); err == nil { return strconv.FormatInt(n, 10) }
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		if math.Trunc(n) == n { return strconv.FormatInt(int64(n), 10) }
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return s
}

func normalizeHeader(s string)string{s=strings.TrimSpace(s);s=strings.ReplaceAll(s,"Nº","N");s=strings.ReplaceAll(s,"N°","N");s=strings.ReplaceAll(s,"º","o");s=strings.ReplaceAll(s,"°","o");s=strings.ToUpper(s);s=strings.Join(strings.Fields(s)," ");return s}
func lookupHeaderEquivalent(s string)string{n:=normalizeHeader(s);n=strings.TrimSpace(n);for _,prefix:=range []string{"NRO ","Nº ","N° ","N "}{if strings.HasPrefix(n,prefix){return strings.TrimSpace(n[len(prefix):])}};return n}
func lookupHeadersMatch(a,b string)bool{aN,bN:=normalizeHeader(a),normalizeHeader(b);if aN==bN{return true};return lookupHeaderEquivalent(aN)==lookupHeaderEquivalent(bN)}
func uniqueNormalizedHeaderID(title string,used map[string]int)string{base:=normalizeHeader(title);if base==""{base="COLUMNA"};used[base]++;if used[base]==1{return base};return fmt.Sprintf("%s_%d",base,used[base])}
func datasetColumnKey(c DatasetColumn)string{return c.ID}
func legacyDatasetColumnKey(c DatasetColumn)string{return c.Source+"|"+c.Title}
func datasetSettingString(m map[string]string,c DatasetColumn)string{if v:=strings.TrimSpace(m[datasetColumnKey(c)]);v!=""{return v};return strings.TrimSpace(m[legacyDatasetColumnKey(c)])}
func datasetSettingInt(m map[string]int,c DatasetColumn,def int)int{if v,ok:=m[datasetColumnKey(c)];ok{return v};if v,ok:=m[legacyDatasetColumnKey(c)];ok{return v};return def}
func datasetSettingBool(m map[string]bool,c DatasetColumn)bool{if v,ok:=m[datasetColumnKey(c)];ok{return v};return m[legacyDatasetColumnKey(c)]}
func datasetColumnDisplayTitle(c DatasetColumn)string{if t:=datasetSettingString(appSettings.ColumnTitles,c);t!=""{return t};return c.Title}
func datasetColumnDecimals(c DatasetColumn)int{if d:=datasetSettingInt(appSettings.ColumnDecimals,c,appSettings.Decimals);d>=0&&d<=8{return d};return appSettings.Decimals}
func datasetColumnIsPercent(c DatasetColumn)bool{if datasetSettingBool(appSettings.ColumnPercent,c){return true};return strings.EqualFold(strings.TrimSpace(datasetSettingString(appSettings.ColumnTypes,c)),"porcentaje")}
func datasetColumnHighlightNegative(c DatasetColumn)bool{return datasetSettingBool(appSettings.HighlightNegative,c)}
func datasetColumnCurrency(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnCurrency,c)||strings.EqualFold(strings.TrimSpace(datasetSettingString(appSettings.ColumnTypes,c)),"moneda")}
func datasetColumnIsDate(c DatasetColumn)bool{return strings.EqualFold(strings.TrimSpace(datasetSettingString(appSettings.ColumnTypes,c)),"fecha")}
func datasetColumnThousands(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnThousands,c)||datasetColumnCurrency(c)}
func datasetColumnHighlightSign(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnHighlightSign,c)}
func datasetColumnBackground(c DatasetColumn)string{return strings.TrimSpace(appSettings.ColumnBackground[datasetColumnKey(c)])}
func datasetColumnAlign(c DatasetColumn)string{if v:=strings.ToLower(strings.TrimSpace(appSettings.ColumnAlign[datasetColumnKey(c)]));v=="left"||v=="center"||v=="right"{return v};if c.Type==ValueNumber||c.Source=="CALCULADA"{return "right"};return "left"}
func parseConfiguredType(v string)ValueType{switch strings.ToLower(strings.TrimSpace(v)){case "entero","integer":return ValueNumber;case "decimal","number","numero":return ValueNumber;case "fecha","date":return ValueDate;case "porcentaje","percent":return ValueNumber};return ValueEmpty}
// canonicalDateRaw convierte fechas de entrada a una representacion interna
// estable. La presentacion (dd/mm/aaaa) se aplica solamente al mostrar.
func canonicalDateRaw(raw string) string {