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
func TestEvaluateFormulaPreservesNegativeValues(t *testing.T) {
	cols := []DatasetColumn{{ID:"cantidad",Title:"CANTIDAD",Source:"XLSX",Type:ValueNumber},{ID:"kg",Title:"KG",Source:"XLSX",Type:ValueNumber}}
	r := DatasetRecord{Values:map[string]MemoryValue{"cantidad":{ColumnID:"cantidad",Type:ValueNumber,Number:-210},"kg":{ColumnID:"kg",Type:ValueNumber,Number:1260}}}
	if got,ok:=evaluateFormula("[CANTIDAD]*[KG]",r,cols); !ok || got != -264600 { t.Fatalf("multiplication=%v ok=%v; want -264600,true",got,ok) }
	if got,ok:=evaluateFormula("[CANTIDAD]/[KG]",r,cols); !ok || got != -1.0/6.0 { t.Fatalf("division=%v ok=%v; want -1/6,true",got,ok) }
	if got,ok:=evaluateFormula("[CANTIDAD]+[KG]",r,cols); !ok || got != 1050 { t.Fatalf("addition=%v ok=%v; want 1050,true",got,ok) }
	if got,ok:=evaluateFormula("-[CANTIDAD]",r,cols); !ok || got != 210 { t.Fatalf("unary minus=%v ok=%v; want 210,true",got,ok) }
}
func makeTestDoc(rows ...[3]string)*xlsxDoc{columns:=[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText},{ID:"SKU",Title:"SKU",Index:2,Type:ValueText}};memoryRows:=make([]MemoryRow,0,len(rows));for _,x:=range rows{memoryRows=append(memoryRows,MemoryRow{Values:map[string]MemoryValue{"SO":{Raw:x[0],Type:ValueText},"ITEM":{Raw:x[1],Type:ValueText},"SKU":{Raw:x[2],Type:ValueText}}})};return &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:columns,Rows:memoryRows}}}}}
func TestBuildMemoryDatasetCalculatedColumnPreservesNegativeSignEndToEnd(t *testing.T) {
	old := appSettings
	defer func() { appSettings = old }()
	appSettings = defaultDatasetSettings()
	appSettings.CalculatedColumns = []CalculatedColumn{{Name:"TN",Formula:"[CANTIDAD]*[KG]"}}

	d := &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{
		Columns: []MemoryColumn{
			{ID:"SO",Title:"SO",Index:0,Type:ValueText},
			{ID:"CANTIDAD",Title:"CANTIDAD",Index:1,Type:ValueNumber},
			{ID:"KG",Title:"KG",Index:2,Type:ValueNumber},
		},
		Rows: []MemoryRow{{Values: map[string]MemoryValue{
			"SO": {ColumnID:"SO",Raw:"100",Type:ValueText},
			"CANTIDAD": makeMemoryValue("CANTIDAD","-105"),
			"KG": makeMemoryValue("KG","630"),
		}}},
	}}}}
	s := defaultDatasetSettings()
	s.SOColumn = 1
	ds, err := BuildMemoryDataset([]*xlsxDoc{d}, s)
	if err != nil { t.Fatal(err) }
	applyDatasetFormula(ds, appSettings)

	c, ok := ds.columnByTitle("TN")
	if !ok { t.Fatal("calculated TN column not created") }
	v, ok := ds.Records[0].Values[c.ID]
	if !ok { t.Fatal("calculated TN value missing") }
	if v.Number != -66150 { t.Fatalf("TN.Number=%v; want -66150", v.Number) }
	if got := datasetValueText(c, v); !strings.HasPrefix(strings.TrimSpace(got), "-") {
		t.Fatalf("TN display=%q; want negative sign", got)
	}
}

func TestMakeMemoryValuePreservesNegativeLocaleAndAccountingSign(t *testing.T) {
	for _, raw := range []string{"-105,00", "(105)"} {
		v := makeMemoryValue("CANTIDAD", raw)
		if v.Type != ValueNumber || v.Number >= 0 {
			t.Fatalf("%q => type=%v number=%v; want numeric negative", raw, v.Type, v.Number)
		}
		if raw == "-105,00" && v.Number != -105 {
			t.Fatalf("%q => number=%v; want -105", raw, v.Number)
		}
		if raw == "(105)" && v.Number != -105 {
			t.Fatalf("%q => number=%v; want -105", raw, v.Number)
		}
	}
}

func TestMemoryDatasetKeepsAllItemsForSameSO(t *testing.T){s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{makeTestDoc([3]string{"100","1","ACE0001"},[3]string{"100","2","ACE0002"},[3]string{"100","3","ACE0003"})},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=3||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 3,0",len(m.Records),m.DuplicateSO)}}
func TestMemoryDatasetDeduplicatesExactSOItemLine(t *testing.T){
	s:=defaultDatasetSettings();s.SOColumn=1
	m,err:=BuildMemoryDataset([]*xlsxDoc{makeTestDoc([3]string{"100","1","ACE0001"},[3]string{"100","1","ACE0001"},[3]string{"100","2","ACE0002"})},s);if err!=nil{t.Fatal(err)}
	if len(m.Records)!=2||m.DuplicateSO!=1{t.Fatalf("records=%d duplicates=%d; want 2,1",len(m.Records),m.DuplicateSO)}
}
func TestMemoryDatasetDoesNotDeduplicateSameSOItemWhenAnotherColumnDiffers(t *testing.T){
	d:=&xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText},{ID:"CANTIDAD",Title:"CANTIDAD",Index:2,Type:ValueNumber}},Rows:[]MemoryRow{
		{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText},"ITEM":{Raw:"1",Type:ValueText},"CANTIDAD":makeMemoryValue("CANTIDAD","10")}},
		{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText},"ITEM":{Raw:"1",Type:ValueText},"CANTIDAD":makeMemoryValue("CANTIDAD","20")}},
	}}}}}
	s:=defaultDatasetSettings();s.SOColumn=1
	m,err:=BuildMemoryDataset([]*xlsxDoc{d},s);if err!=nil{t.Fatal(err)}
	if len(m.Records)!=2||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 2,0",len(m.Records),m.DuplicateSO)}
}
func TestMemoryDatasetDoesNotDeduplicateBySOWhenITEMIsMissing(t *testing.T){d:=&xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText}},Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText}}},{Values:map[string]MemoryValue{"SO":{Raw:"100",Type:ValueText}}}}}}}};s:=defaultDatasetSettings();s.SOColumn=1;m,err:=BuildMemoryDataset([]*xlsxDoc{d},s);if err!=nil{t.Fatal(err)};if len(m.Records)!=2||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 2,0",len(m.Records),m.DuplicateSO)}}
func TestBuildMemoryDatasetPreservesRepeatedSOItemQuantitiesAndSum(t *testing.T){
	d:=&xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Index:0,Type:ValueText},{ID:"ITEM",Title:"ITEM",Index:1,Type:ValueText},{ID:"CANTIDAD",Title:"CANTIDAD",Index:2,Type:ValueNumber},{ID:"KG",Title:"KG",Index:3,Type:ValueNumber}},Rows:[]MemoryRow{
		{Values:map[string]MemoryValue{"SO":{Raw:"500",Type:ValueText},"ITEM":{Raw:"A",Type:ValueText},"CANTIDAD":makeMemoryValue("CANTIDAD","10"),"KG":makeMemoryValue("KG","100")}},
		{Values:map[string]MemoryValue{"SO":{Raw:"500",Type:ValueText},"ITEM":{Raw:"A",Type:ValueText},"CANTIDAD":makeMemoryValue("CANTIDAD","20"),"KG":makeMemoryValue("KG","200")}},
		{Values:map[string]MemoryValue{"SO":{Raw:"500",Type:ValueText},"ITEM":{Raw:"A",Type:ValueText},"CANTIDAD":makeMemoryValue("CANTIDAD","30"),"KG":makeMemoryValue("KG","300")}},
	}}}}}
	s:=defaultDatasetSettings();s.SOColumn=1
	m,err:=BuildMemoryDataset([]*xlsxDoc{d},s);if err!=nil{t.Fatal(err)}
	if len(m.Records)!=3||m.DuplicateSO!=0{t.Fatalf("records=%d duplicates=%d; want 3,0",len(m.Records),m.DuplicateSO)}
	var qtySum,kgSum float64
	for _,r:=range m.Records{qtySum+=r.Values["CANTIDAD"].Number;kgSum+=r.Values["KG"].Number}
	if qtySum!=60||kgSum!=600{t.Fatalf("sums cantidad=%v kg=%v; want 60,600",qtySum,kgSum)}
}
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
	v, ok := m.Records[0].Values["CSV:CLAVE"]; if !ok || strings.TrimSpace(v.Raw) == "" { t.Fatalf("CSV:CLAVE not populated: ok=%v raw=%q", ok, v.Raw) }
}

func TestCanonicalizeMemoryValueNormalizesDateAndNumber(t *testing.T) {
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
	for _, tc := range []struct{raw,want string}{{"01/01/2026","Ejercicio 2026"},{"31/12/2026","Ejercicio 2026"},{"31/12/2026 23:59:59","Ejercicio 2026"},{"31/12/2025",""}} {
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

func TestLookupRangeBoundariesAreInclusiveWithTime(t *testing.T) {
	csvData := []byte("DESDE;HASTA;FECFACTURA;EJERCICIO\n12/09/2026;13/09/2026;;PRUEBA1\n")
	table, err := parseLookupTable(csvData, "EjercicioBorde", "EjercicioBorde.csv")
	if err != nil { t.Fatal(err) }
	sh := MemorySheet{Columns: []MemoryColumn{{ID:"FECFACTURA", Title:"FECFACTURA", Index:0, Type:ValueDate}}}
	ids := map[string]map[string]string{"EjercicioBorde":{"EJERCICIO":"LOOKUP:EJERCICIOBORDE:EJERCICIO"}}
	cases := []struct{raw,want string}{{"12/09/2026","PRUEBA1"},{"13/09/2026","PRUEBA1"},{"13/09/2026 23:59:59","PRUEBA1"},{"11/09/2026",""}}
	for _, tc := range cases {
		rec := DatasetRecord{Values: map[string]MemoryValue{}}
		row := MemoryRow{Values: map[string]MemoryValue{"FECFACTURA":{ColumnID:"FECFACTURA",Raw:tc.raw,Type:ValueDate}}}
		got := applyLookupEnrichment(&rec,sh,row,[]lookupTable{table},ids)
		v := rec.Values["LOOKUP:EJERCICIOBORDE:EJERCICIO"].Raw
		if tc.want == "" { if got != 0 || v != "" { t.Fatalf("date %q: matches=%d value=%q; want no match",tc.raw,got,v) } } else if got != 1 || v != tc.want { t.Fatalf("date %q: matches=%d value=%q; want %q",tc.raw,got,v,tc.want) }
	}
}
