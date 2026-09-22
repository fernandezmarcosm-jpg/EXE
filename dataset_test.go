package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateFormulaUsesArithmeticAndSourceQualifiedColumns(t *testing.T) {
	cols := []DatasetColumn{{ID:"1",Title:"COLUMNA 1",Source:"XLSX",Type:ValueNumber},{ID:"2",Title:"COLUMNA 2",Source:"XLSX",Type:ValueNumber},{ID:"3",Title:"COLUMNA 3",Source:"CSV",Type:ValueNumber}}
	r:=DatasetRecord{SO:"1",Values:map[string]MemoryValue{"1":{ColumnID:"1",Type:ValueNumber,Number:10},"2":{ColumnID:"2",Type:ValueNumber,Number:2},"3":{ColumnID:"3",Type:ValueNumber,Number:3}}}
	v,ok:=evaluateFormula("[COLUMNA 1] / [COLUMNA 2] * [CSV:COLUMNA 3]",r,cols);if !ok||v!=15{t.Fatalf("formula=%v ok=%v; want 15,true",v,ok)}
}
func makeTestDoc(rows ...[3]string)*xlsxDoc{columns:=[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText},{ID:"SKU",Title:"SKU",Index:2,Type:ValueText}};memoryRows:=make([]MemoryRow,0,len(rows));for _,x:=range rows{memoryRows=append(memoryRows,MemoryRow{Values:map[string]MemoryValue{"SO":{Raw:x[0],Type:ValueText},"ITEM":{Raw:x[1],Type:ValueText},"SKU":{Raw:x[2],Type:ValueText}}})};return &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:columns,Rows:memoryRows}}}}}
func TestMemoryDatasetKeepsAllItemsForSameSO(t *testing.T){s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{makeTestDoc([3]string{"100","1","ACE0001"},[3]string{"100","2","ACE0002"},[3]string{"100","3","ACE0003"})},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=3||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 3,0",len(m.Records),m.DuplicateSO)}}
func TestMemoryDatasetDeduplicatesExactSOItemLine(t *testing.T){s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{makeTestDoc([3]string{"100","1","ACE0001"},[3]string{"100","1","ACE0001"},[3]string{"100","2","ACE0002"})},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=2||m.DuplicateSO!=1{t.Fatalf("records=%d duplicates=%d; want 2,1",len(m.Records),m.DuplicateSO)}}
func TestMemoryDatasetDoesNotDeduplicateBySOWhenITEMIsMissing(t *testing.T){d:=&xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText}},Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText}}},{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText}}}}}}}};s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{d},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=2||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 2,0",len(m.Records),m.DuplicateSO)}}
func TestEmbeddedMasterCSVIsAvailable(t *testing.T){m,_,err:=loadMasterCSV("");if err!=nil{t.Fatal(err)};if len(m.ByKey)==0{t.Fatal("embedded CSV master is empty")};if _,ok:=m.ByKey["ACE0001"];!ok{t.Fatal("expected ACE0001 in embedded master")}}
func TestMemoryDatasetReusesHeadersAcrossFilesAndStacksRows(t *testing.T){s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{makeTestDoc([3]string{"100","1","ACE0001"}),makeTestDoc([3]string{"200","2","ACE0002"})},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=2{t.Fatalf("records=%d; want 2",len(m.Records))};ids:=map[string]int{};for _,c:=range m.Columns{if c.Source=="XLSX"{ids[c.ID]++}};for _,id:=range []string{"SO","ITEM","SKU"}{if ids[id]!=1{t.Fatalf("column %q count=%d; want 1",id,ids[id])}};if ids["SO_2"]!=0||ids["ITEM_2"]!=0||ids["SKU_2"]!=0{t.Fatal("headers from the second file must not create _2 columns")};if _,ok:=m.Records[0].Values["ITEM"];!ok{t.Fatal("first row missing ITEM value")};if _,ok:=m.Records[1].Values["ITEM"];!ok{t.Fatal("second row missing shared ITEM value")}}
func TestBuildMemoryDatasetPreservesPhysicalDuplicates(t *testing.T){d:=&xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM1",Title:"ITEM",Index:1,Type:ValueText},{ID:"ITEM2",Title:"ITEM",Index:2,Type:ValueText}},Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText},"ITEM1":{Raw:"A",Type:ValueText},"ITEM2":{Raw:"B",Type:ValueText}}}}}}}};s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{d},s);if err!=nil{t.Fatal(err)};ids:=map[string]int{};for _,c:=range m.Columns{if c.Source=="XLSX"{ids[c.ID]++}};if ids["ITEM"]!=1||ids["ITEM_2"]!=1{t.Fatalf("physical duplicate IDs: ITEM=%d ITEM_2=%d; want 1,1",ids["ITEM"],ids["ITEM_2"]) };if _,ok:=m.Records[0].Values["ITEM"];!ok{t.Fatal("missing first physical ITEM value")};if _,ok:=m.Records[0].Values["ITEM_2"];!ok{t.Fatal("missing second physical ITEM value")}}

func TestBuildMemoryDatasetJoinsExcelCLAVEToCSVMaster(t *testing.T) {
	d := &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText},{ID:"CLAVE",Title:"CLAVE",Index:2,Type:ValueText}},Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText},"ITEM":{Raw:"1",Type:ValueText},"CLAVE":{Raw:"ACE0001",Type:ValueText}}}}}}}}
	s := defaultDatasetSettings(); s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{d}, s); if err != nil { t.Fatal(err) }
	if m.Enriched <= 0 { t.Fatalf("Enriched=%d; want > 0", m.Enriched) }
	v, ok := m.Records[0].Values["CSV:CLAVE"]; if !ok || strings.TrimSpace(v.Raw) == "" { t.Fatalf("CSV:CLAVE not populated: ok=%v raw=%q", ok, v.Raw) }
}

func TestBuildMemoryDatasetFallsBackToITEMForCSVJoin(t *testing.T) {
	d := &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{
		Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText}},
		Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText},"ITEM":{Raw:"ACE0001",Type:ValueText}}}},
	}}}}
	s := defaultDatasetSettings(); s.SOColumn = 1; s.JoinExcelColumn = "SKU"
	m, err := BuildMemoryDataset([]*xlsxDoc{d}, s); if err != nil { t.Fatal(err) }
	if m.Enriched <= 0 { t.Fatalf("Enriched=%d; want > 0 using ITEM fallback", m.Enriched) }
	v, ok := m.Records[0].Values["CSV:CLAVE"]; if !ok || strings.TrimSpace(v.Raw) == "" { t.Fatalf("CSV:CLAVE not populated through ITEM fallback: ok=%v raw=%q", ok, v.Raw) }
}

func TestCalculatedPercentIsAppliedOnlyByDisplayFormatting(t *testing.T) {
	old := appSettings
	defer func() { appSettings = old }()
	appSettings = defaultDatasetSettings()
	appSettings.CalculatedColumns = []CalculatedColumn{{Name:"MARGEN",Formula:"[BASE] / [TOTAL]",Percent:true}}
	d := &MemoryDataset{
		Columns: []DatasetColumn{
			{ID:"BASE",Title:"BASE",Source:"XLSX",Type:ValueNumber},
			{ID:"TOTAL",Title:"TOTAL",Source:"XLSX",Type:ValueNumber},
		},
		Records: []DatasetRecord{{Values: map[string]MemoryValue{
			"BASE":{ColumnID:"BASE",Type:ValueNumber,Number:1},
			"TOTAL":{ColumnID:"TOTAL",Type:ValueNumber,Number:2},
		}}},
	}
	applyDatasetFormula(d, appSettings)
	c, ok := d.columnByTitle("MARGEN")
	if !ok { t.Fatal("calculated column not created") }
	v := d.Records[0].Values[c.ID]
	if v.Number != 0.5 { t.Fatalf("stored value=%v; want 0.5", v.Number) }
	if got := datasetValueText(c, v); got != "50.00%" { t.Fatalf("display value=%q; want 50.00%%", got) }
}


func TestComputeSubtotalsUsesFormattedGroupValue(t *testing.T) {
	old := appSettings
	defer func() { appSettings = old }()
	appSettings = defaultDatasetSettings()
	appSettings.SubtotalColumn = "GRUPO"
	appSettings.SubtotalAgg = map[string]string{"VALOR":"suma"}
	appSettings.ColumnDecimals["GRUPO"] = 2
	d := &MemoryDataset{
		Columns: []DatasetColumn{
			{ID:"GRUPO",Title:"GRUPO",Source:"XLSX",Type:ValueNumber},
			{ID:"VALOR",Title:"VALOR",Source:"XLSX",Type:ValueNumber},
		},
		Records: []DatasetRecord{
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueNumber,Number:1.2,Raw:"1.2"},"VALOR":{ColumnID:"VALOR",Type:ValueNumber,Number:10}}},
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueNumber,Number:1.2,Raw:"1.2"},"VALOR":{ColumnID:"VALOR",Type:ValueNumber,Number:5}}},
		},
	}
	rows := computeSubtotals(d)
	if len(rows) != 2 { t.Fatalf("subtotal rows=%d; want 2", len(rows)) }
	if rows[0].GroupValue != "1.20" { t.Fatalf("group value=%q; want 1.20", rows[0].GroupValue) }
	if rows[0].Values["VALOR"] != "15.00" { t.Fatalf("group subtotal=%q; want 15.00", rows[0].Values["VALOR"]) }
}


func TestDatasetNumberFormattingGroupedAndCurrency(t *testing.T) {
	old := appSettings
	defer func(){ appSettings = old }()
	appSettings = defaultDatasetSettings()
	c := DatasetColumn{ID:"IMPORTE",Title:"IMPORTE",Source:"XLSX",Type:ValueNumber}
	appSettings.ColumnCurrency[c.ID] = true
	appSettings.ColumnDecimals[c.ID] = 2
	v := MemoryValue{ColumnID:c.ID,Type:ValueNumber,Number:-1234567.8}
	if got:=datasetValueText(c,v); got != "$-1.234.567,80" { t.Fatalf("currency=%q; want $-1.234.567,80",got) }
	delete(appSettings.ColumnCurrency,c.ID)
	appSettings.ColumnThousands[c.ID] = true
	if got:=datasetValueText(c,v); got != "-1.234.567,80" { t.Fatalf("grouped=%q; want -1.234.567,80",got) }
}


func TestComputeSubtotalsCountsUniqueTextValues(t *testing.T) {
	old := appSettings
	defer func(){ appSettings = old }()
	appSettings = defaultDatasetSettings()
	appSettings.SubtotalColumn = "GRUPO"
	appSettings.SubtotalAgg = map[string]string{"SDDOCO":"conteo_unico"}
	d := &MemoryDataset{
		Columns: []DatasetColumn{
			{ID:"GRUPO",Title:"GRUPO",Source:"XLSX",Type:ValueText},
			{ID:"SDDOCO",Title:"SDDOCO",Source:"XLSX",Type:ValueText},
		},
		Records: []DatasetRecord{
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueText,Raw:"A"},"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"100"}}},
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueText,Raw:"A"},"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"100"}}},
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueText,Raw:"A"},"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:" 200 "}}},
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueText,Raw:"B"},"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"300"}}},
			{Values:map[string]MemoryValue{"GRUPO":{ColumnID:"GRUPO",Type:ValueText,Raw:"B"},"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:""}}},
		},
	}
	rows := computeSubtotals(d)
	if len(rows) != 3 { t.Fatalf("subtotal rows=%d; want 3", len(rows)) }
	if rows[0].Values["SDDOCO"] != "2" { t.Fatalf("group A distinct count=%q; want 2", rows[0].Values["SDDOCO"]) }
	if rows[1].Values["SDDOCO"] != "1" { t.Fatalf("group B distinct count=%q; want 1", rows[1].Values["SDDOCO"]) }
	if rows[2].Values["SDDOCO"] != "3" { t.Fatalf("total distinct count=%q; want 3", rows[2].Values["SDDOCO"]) }
}

func TestComputeSubtotalsReportsUniqueGroupCount(t *testing.T) {
	old := appSettings
	defer func(){ appSettings = old }()
	appSettings = defaultDatasetSettings()
	appSettings.SubtotalColumn = "SDDOCO"
	appSettings.SubtotalAgg = map[string]string{"SDDOCO":"conteo_unico"}
	d := &MemoryDataset{
		Columns: []DatasetColumn{{ID:"SDDOCO",Title:"SDDOCO",Source:"XLSX",Type:ValueText}},
		Records: []DatasetRecord{
			{Values:map[string]MemoryValue{"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"100"}}},
			{Values:map[string]MemoryValue{"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"100"}}},
			{Values:map[string]MemoryValue{"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"200"}}},
			{Values:map[string]MemoryValue{"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:"300"}}},
			{Values:map[string]MemoryValue{"SDDOCO":{ColumnID:"SDDOCO",Type:ValueText,Raw:""}}},
		},
	}
	rows := computeSubtotals(d)
	var total SubtotalRow
	foundTotal := false
	for _, row := range rows {
		if row.Total { total = row; foundTotal = true; break }
	}
	if !foundTotal { t.Fatal("TOTAL GENERAL subtotal row not found") }
	if total.GroupCount != 3 { t.Fatalf("unique group count=%d; want 3",total.GroupCount) }
}


func TestDatasetValueTextFormatsConfiguredDate(t *testing.T) {
	old:=appSettings
	defer func(){appSettings=old}()
	appSettings=defaultDatasetSettings()
	appSettings.ColumnTypes["FECHA"]="fecha"
	c:=DatasetColumn{ID:"FECHA",Title:"FECHA",Source:"XLSX",Type:ValueText}
	cases:=[]struct{name string;v MemoryValue;want string}{
		{"date type",MemoryValue{ColumnID:"FECHA",Type:ValueDate,Raw:"2026-09-21"},"21/09/2026"},
		{"excel serial",MemoryValue{ColumnID:"FECHA",Type:ValueNumber,Number:45921,Raw:"45921"},"21/09/2025"},
		{"text",MemoryValue{ColumnID:"FECHA",Type:ValueText,Raw:"21/09/2026"},"21/09/2026"},
		{"invalid",MemoryValue{ColumnID:"FECHA",Type:ValueText,Raw:"sin fecha"},"sin fecha"},
	}
	for _,tc:=range cases{if got:=datasetValueText(c,tc.v);got!=tc.want{t.Fatalf("%s: got %q want %q",tc.name,got,tc.want)}}
}
func TestParseLookupTableSupportsSemicolonAndComma(t *testing.T) {
	semi,err:=parseLookupTable([]byte("Nº CLIENTE;PROVINCIA;CIUDAD\n123;Buenos Aires;Campana\n"),"Clientes","clientes.csv");if err!=nil{t.Fatal(err)}
	if semi.KeyHeader!="Nº CLIENTE"{t.Fatalf("key header=%q",semi.KeyHeader)};if semi.ByKey["123"]["PROVINCIA"]!="Buenos Aires"{t.Fatalf("semicolon lookup failed: %#v",semi.ByKey["123"])}
	comma,err:=parseLookupTable([]byte("SDSRP2,GENERATOR,CITY\nA1,Gen1,Campana\n"),"Generadores","generadores.csv");if err!=nil{t.Fatal(err)}
	if comma.ByKey["A1"]["CITY"]!="Campana"{t.Fatalf("comma lookup failed: %#v",comma.ByKey["A1"])}
}

func TestLookupColumnIDIsStableAndPrefixed(t *testing.T) {
	used:=map[string]int{}
	id1:=lookupColumnID("Clientes", "PROVINCIA", used);id2:=lookupColumnID("Clientes", "PROVINCIA", used)
	if id1!="LOOKUP:CLIENTES:PROVINCIA"||id2!="LOOKUP:CLIENTES:PROVINCIA_2"{t.Fatalf("ids=%q,%q",id1,id2)}
}

func TestLookupColumnForKeyUsesNormalizedHeader(t *testing.T) {
	sh:=MemorySheet{Columns:[]MemoryColumn{{ID:"C1",Title:"Nº CLIENTE",Index:0,Type:ValueText},{ID:"C2",Title:"CIUDAD",Index:1,Type:ValueText}}}
	if got:=lookupColumnForKey(sh,"N° CLIENTE");got!="C1"{t.Fatalf("column=%q; want C1",got)}
	if got:=lookupColumnForKey(sh,"PROVINCIA");got!=""{t.Fatalf("unexpected column=%q",got)}
}

func TestLookupEnrichmentByNumeroClienteNumericValue(t *testing.T) {
	table:=lookupTable{Name:"Clientes",KeyHeader:"Nº CLIENTE",Headers:[]string{"Nº CLIENTE","ATRIBUTO"},ByKey:map[string]map[string]string{}}
	table.ByKey[normalizeJoinKey("80003285")]=map[string]string{"Nº CLIENTE":"80003285","ATRIBUTO":"Cadena"}
	sh:=MemorySheet{Columns:[]MemoryColumn{{ID:"CLIENTE",Title:"Nº CLIENTE",Index:0,Type:ValueNumber}}}
	row:=MemoryRow{Values:map[string]MemoryValue{"CLIENTE":{ColumnID:"CLIENTE",Type:ValueNumber,Raw:"80003285.00",Number:80003285}}}
	rec:=DatasetRecord{Values:map[string]MemoryValue{}}
	ids:=map[string]map[string]string{"Clientes":{"ATRIBUTO":"LOOKUP:CLIENTES:ATRIBUTO"}}
	if got:=applyLookupEnrichment(&rec,sh,row,[]lookupTable{table},ids);got!=1{t.Fatalf("lookup matches=%d; want 1",got)}
	v,ok:=rec.Values["LOOKUP:CLIENTES:ATRIBUTO"];if !ok||v.Raw!="Cadena"{t.Fatalf("lookup value: ok=%v raw=%q",ok,v.Raw)}
}

func TestBuildMemoryDatasetJoinsPhysicalLookupNumeroClienteVariants(t *testing.T) {
	variants := []struct {
		name, header, value string
	}{
		{"N CLIENTE", "N CLIENTE", "80003285"},
		{"NRO CLIENTE", "NRO CLIENTE", "80.003.285"},
		{"Nº CLIENTE decimal", "Nº CLIENTE", "80003285.00"},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			oldWD, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWD)
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			csvData := tc.header + ";ATRIBUTO\n" + tc.value + ";Cadena\n"
			if err := os.WriteFile(filepath.Join(dir, "Clientes.csv"), []byte(csvData), 0644); err != nil {
				t.Fatal(err)
			}
			doc := &xlsxDoc{
				Memory: &MemoryWorkbook{
					Sheets: []MemorySheet{{
						Columns: []MemoryColumn{
							{ID: "SO", Title: "SO", Index: 0, Type: ValueText},
							{ID: "CLIENTE", Title: tc.header, Index: 1, Type: ValueNumber},
						},
						Rows: []MemoryRow{{
							Values: map[string]MemoryValue{
								"SO": {ColumnID: "SO", Raw: "100", Type: ValueText},
								"CLIENTE": {ColumnID: "CLIENTE", Raw: tc.value, Type: ValueNumber},
							},
						}},
					}},
				},
			}
			settings := defaultDatasetSettings()
			settings.SOColumn = 1
			m, err := BuildMemoryDataset([]*xlsxDoc{doc}, settings)
			if err != nil {
				t.Fatal(err)
			}
			v, ok := m.Records[0].Values["LOOKUP:CLIENTES:ATRIBUTO"]
			if !ok || v.Raw != "Cadena" {
				t.Fatalf("lookup value: ok=%v raw=%q; want Cadena", ok, v.Raw)
			}
		})
	}
}

func TestNormalizeJoinKeyLocaleNumbers(t *testing.T) {
	cases:=map[string]string{"80003285":"80003285","80003285.00":"80003285","80003285,00":"80003285","23.961,00":"23961","23,961.00":"23961"}
	for in,want:=range cases{if got:=normalizeJoinKey(in);got!=want{t.Fatalf("%q => %q; want %q",in,got,want)}}
}


func TestMakeMemoryValueCanonicalizesNumbers(t *testing.T) {
	cases := []struct{ raw, want string; number float64 }{
		{"8.0003285E7", "80003285", 80003285},
		{"80.003.285", "80003285", 80003285},
		{"23.961,00", "23961", 23961},
		{"534,68", "534.68", 534.68},
		{"534.68", "534.68", 534.68},
	}
	for _, tc := range cases {
		v := makeMemoryValue("C", tc.raw)
		if v.Type != ValueNumber || v.Raw != tc.want || v.Number != tc.number {
			t.Fatalf("%q => type=%v raw=%q number=%v; want raw=%q number=%v", tc.raw, v.Type, v.Raw, v.Number, tc.want, tc.number)
		}
	}
}

func TestCanonicalJoinScientificXLSXMatchesCSV(t *testing.T) {
	xlsx := makeMemoryValue("CLIENTE", "8.0003285E7")
	csv := normalizeJoinKey("80003285")
	if xlsx.Raw != "80003285" {
		t.Fatalf("canonical XLSX Raw=%q; want 80003285", xlsx.Raw)
	}
	if got := normalizeJoinKey(xlsx.Raw); got != csv {
		t.Fatalf("normalized keys differ: XLSX=%q CSV=%q", got, csv)
	}
	table := lookupTable{
		Name: "Clientes", KeyHeader: "Nº CLIENTE",
		Headers: []string{"Nº CLIENTE", "ATRIBUTO"},
		ByKey: map[string]map[string]string{csv: {"Nº CLIENTE": "80003285", "ATRIBUTO": "Cadena"}},
	}
	sh := MemorySheet{Columns: []MemoryColumn{{ID: "CLIENTE", Title: "Nº CLIENTE", Index: 0, Type: ValueNumber}}}
	row := MemoryRow{Values: map[string]MemoryValue{"CLIENTE": xlsx}}
	rec := DatasetRecord{Values: map[string]MemoryValue{}}
	ids := map[string]map[string]string{"Clientes": {"ATRIBUTO": "LOOKUP:CLIENTES:ATRIBUTO"}}
	if got := applyLookupEnrichment(&rec, sh, row, []lookupTable{table}, ids); got != 1 {
		t.Fatalf("lookup matches=%d; want 1", got)
	}
	if got := rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw; got != "Cadena" {
		t.Fatalf("lookup attribute=%q; want Cadena", got)
	}
}

func TestNormalizeJoinKeyScientificNotation(t *testing.T) {
	cases := map[string]string{
		"8.0003285E7": "80003285",
		"8.0003285e+07": "80003285",
		"8.0003285E+07": "80003285",
	}
	for in, want := range cases {
		if got := normalizeJoinKey(in); got != want {
			t.Fatalf("%q => %q; want %q", in, got, want)
		}
	}
}

func TestCanonicalConfiguredTypes(t *testing.T) {
	date := canonicalizeMemoryValue(MemoryValue{ColumnID: "F", Raw: "21/09/2026", Type: ValueText}, ValueDate)
	if date.Raw != "2026-09-21" || date.Type != ValueDate {
		t.Fatalf("date canonical=%q type=%v; want 2026-09-21/DATE", date.Raw, date.Type)
	}
	percent := canonicalizeMemoryValue(MemoryValue{ColumnID: "P", Raw: "12,50", Type: ValueText}, ValueNumber)
	if percent.Raw != "12.5" || percent.Number != 12.5 {
		t.Fatalf("percent canonical raw=%q number=%v; want 12.5", percent.Raw, percent.Number)
	}
	decimal := canonicalizeMemoryValue(MemoryValue{ColumnID: "D", Raw: "23.961,00", Type: ValueText}, ValueNumber)
	if decimal.Raw != "23961" || decimal.Number != 23961 {
		t.Fatalf("decimal canonical raw=%q number=%v; want 23961", decimal.Raw, decimal.Number)
	}
}

func TestLookupRangeByDateInclusive(t *testing.T) {
	csvData := []byte("DESDE;HASTA;FECHA;EJERCICIO\n01/01/2026;31/12/2026;FECHA;Ejercicio 2026\n")
	table, err := parseLookupTable(csvData, "Ejercicio", "Ejercicio.csv")
	if err != nil { t.Fatal(err) }
	if !table.IsRange || table.DateKeyHeader != "FECHA" || len(table.Ranges) != 1 { t.Fatalf("range table: %+v", table) }
	sh := MemorySheet{Columns: []MemoryColumn{{ID:"FECHA",Title:"FECHA",Index:0,Type:ValueDate}}}
	ids := map[string]map[string]string{"Ejercicio":{"EJERCICIO":"LOOKUP:EJERCICIO:EJERCICIO"}}
	for _, tc := range []struct{raw,want string}{{"01/01/2026","Ejercicio 2026"},{"31/12/2026","Ejercicio 2026"},{"31/12/2025",""}} {
		rec := DatasetRecord{Values: map[string]MemoryValue{}}
		row := MemoryRow{Values: map[string]MemoryValue{"FECHA":{ColumnID:"FECHA",Raw:tc.raw,Type:ValueDate}}}
		got := applyLookupEnrichment(&rec,sh,row,[]lookupTable{table},ids)
		v := rec.Values["LOOKUP:EJERCICIO:EJERCICIO"].Raw
		if tc.want == "" { if got != 0 || v != "" { t.Fatalf("date %q: matches=%d value=%q; want no match",tc.raw,got,v) } } else if got != 1 || v != tc.want { t.Fatalf("date %q: matches=%d value=%q; want %q",tc.raw,got,v,tc.want) }
	}
}

func TestBuildMemoryDatasetLookupRangeFiscalYear(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil { t.Fatal(err) }
	defer os.Chdir(oldWD)
	if err := os.Chdir(dir); err != nil { t.Fatal(err) }
	csvData := "DESDE;HASTA;FECFACTURA;EJERCICIO\n01/07/2024;30/06/2025;;Ejercicio 2025\n01/07/2025;30/06/2026;;Ejercicio 2026\n"
	if err := os.WriteFile(filepath.Join(dir, "Ejercicio.csv"), []byte(csvData), 0644); err != nil { t.Fatal(err) }
	doc := &xlsxDoc{Memory: &MemoryWorkbook{Sheets: []MemorySheet{{
		Columns: []MemoryColumn{
			{ID:"SO", Title:"SO", Index:0, Type:ValueText},
			{ID:"FECFACTURA", Title:"FECFACTURA", Index:1, Type:ValueDate},
		},
		Rows: []MemoryRow{
			{Values:map[string]MemoryValue{"SO":{ColumnID:"SO",Raw:"100",Type:ValueText},"FECFACTURA":{ColumnID:"FECFACTURA",Raw:"13/09/2025",Type:ValueDate}}},
			{Values:map[string]MemoryValue{"SO":{ColumnID:"SO",Raw:"101",Type:ValueText},"FECFACTURA":{ColumnID:"FECFACTURA",Raw:"15/03/2025",Type:ValueDate}}},
		},
	}}}}
	s := defaultDatasetSettings(); s.SOColumn = 1
	m, err := BuildMemoryDataset([]*xlsxDoc{doc}, s)
	if err != nil { t.Fatal(err) }
	if len(m.Records) != 2 { t.Fatalf("records=%d; want 2", len(m.Records)) }
	for _, tc := range []struct{so, want string}{{"100","Ejercicio 2026"},{"101","Ejercicio 2025"}} {
		var found *DatasetRecord
		for i := range m.Records { if m.Records[i].SO == tc.so { found = &m.Records[i]; break } }
		if found == nil { t.Fatalf("SO %s not found", tc.so) }
		v, ok := found.Values["LOOKUP:EJERCICIO:EJERCICIO"]
		if !ok || v.Raw != tc.want { t.Fatalf("SO %s lookup: ok=%v raw=%q; want %q", tc.so, ok, v.Raw, tc.want) }
	}
	if len(m.LookupDiagnostics) != 1 || m.LookupDiagnostics[0].EnrichedRows != 2 { t.Fatalf("range diagnostics=%+v; want 2 enriched rows", m.LookupDiagnostics) }
}
