package main

import "testing"

func TestEvaluateFormulaUsesArithmeticAndSourceQualifiedColumns(t *testing.T){
	cols:=[]DatasetColumn{{ID:"1",Title:"COLUMNA 1",Source:"XLSX",Type:ValueNumber},{ID:"2",Title:"COLUMNA 2",Source:"XLSX",Type:ValueNumber},{ID:"3",Title:"COLUMNA 3",Source:"CSV",Type:ValueNumber}}
	r:=DatasetRecord{SO:"1",Values:map[string]MemoryValue{"1":{ColumnID:"1",Type:ValueNumber,Number:10},"2":{ColumnID:"2",Type:ValueNumber,Number:2},"3":{ColumnID:"3",Type:ValueNumber,Number:3}}}
	v,ok:=evaluateFormula("[COLUMNA 1] / [COLUMNA 2] * [CSV:COLUMNA 3]",r,cols)
	if !ok||v!=15{t.Fatalf("formula=%v ok=%v; want 15,true",v,ok)}
}

func TestMemoryDatasetDeduplicatesSO(t *testing.T){
	d:=func(so,sku string)*xlsxDoc{return &xlsxDoc{Memory:&MemoryWorkbook{Sheets:[]MemorySheet{{Columns:[]MemoryColumn{{ID:"SO",Title:"SO",Type:ValueText},{ID:"SKU",Title:"SKU",Type:ValueText}},Rows:[]MemoryRow{{Values:map[string]MemoryValue{"SO":{Raw:so,Type:ValueText},"SKU":{Raw:sku,Type:ValueText}}}}}}}}}
	a:=d("100","ACE0001");b:=d("100","ACE0002")
	m,e:=BuildMemoryDataset([]*xlsxDoc{a,b},defaultDatasetSettings());if e!=nil{t.Fatal(e)};if len(m.Records)!=1||m.DuplicateSO!=1{t.Fatalf("records=%d duplicates=%d; want 1,1",len(m.Records),m.DuplicateSO)}
}

func TestEmbeddedMasterCSVIsAvailable(t *testing.T){m,_,e:=loadMasterCSV("");if e!=nil{t.Fatal(e)};if len(m.ByKey)==0{t.Fatal("embedded CSV master is empty")};if _,ok:=m.ByKey["ACE0001"];!ok{t.Fatal("expected ACE0001 in embedded master")}}
