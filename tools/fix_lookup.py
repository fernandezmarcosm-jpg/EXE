from pathlib import Path

root = Path(__file__).resolve().parents[1]
p = root / "dataset.go"
s = p.read_text(encoding="utf-8")

old = 'func lookupColumnForKey(sh MemorySheet,keyHeader string)string{if strings.TrimSpace(keyHeader)==""{return ""};for _,c:=range sh.Columns{if lookupHeadersMatch(c.Title,keyHeader){return c.ID}};return ""}'
new = 'func lookupColumnForKey(sh MemorySheet,keyHeader string)string{if strings.TrimSpace(keyHeader)==""{return ""};target:=normalizeHeader(keyHeader);for _,c:=range sh.Columns{if normalizeHeader(c.Title)==target{return c.ID}};for _,c:=range sh.Columns{if lookupHeadersMatch(c.Title,keyHeader){return c.ID}};return ""}'
if old not in s:
    raise SystemExit("lookupColumnForKey pattern not found")
s = s.replace(old, new, 1)

old = 'func applyLookupEnrichment(rec *DatasetRecord,sh MemorySheet,row MemoryRow,lookups []lookupTable,lookupIDs map[string]map[string]string)int{if rec==nil{return 0};return len(lookupMatchesForRow(sh,row,lookups,lookupIDs,rec))}'
new = 'var lookupDebugSeen map[string]bool\nfunc applyLookupEnrichment(rec *DatasetRecord,sh MemorySheet,row MemoryRow,lookups []lookupTable,lookupIDs map[string]map[string]string)int{if rec==nil{return 0};if lookupDebugSeen==nil{lookupDebugSeen=map[string]bool{}};for _,table:=range lookups{if lookupDebugSeen[table.Name]{continue};keyID:=lookupColumnForKey(sh,table.KeyHeader);xlsxTitle:="";for _,c:=range sh.Columns{if c.ID==keyID{xlsxTitle=c.Title;break}};csvKeys:=make([]string,0,5);for k:=range table.ByKey{csvKeys=append(csvKeys,k);if len(csvKeys)>=5{break}};keyRaw:="";if keyID!=""{if x,ok:=row.Values[keyID];ok{keyRaw=x.Raw}};log.Printf("[LOOKUP-DBG] base=%q keyHeader=%q xlsxCol=%q csvKeys=%v xlsxKey=%q->%q",table.Name,table.KeyHeader,xlsxTitle,csvKeys,keyRaw,normalizeJoinKey(keyRaw));lookupDebugSeen[table.Name]=true};return len(lookupMatchesForRow(sh,row,lookups,lookupIDs,rec))}'
if old not in s:
    raise SystemExit("applyLookupEnrichment pattern not found")
s = s.replace(old, new, 1)

old = 'func BuildMemoryDataset(docs []*xlsxDoc,s DatasetSettings)(*MemoryDataset,error){if len(docs)==0{return nil,fmt.Errorf("no hay archivos XLSX seleccionados")};m,_,e:=loadMasterCSV("")}'
new = 'func BuildMemoryDataset(docs []*xlsxDoc,s DatasetSettings)(*MemoryDataset,error){lookupDebugSeen=map[string]bool{};if len(docs)==0{return nil,fmt.Errorf("no hay archivos XLSX seleccionados")};m,_,e:=loadMasterCSV("")}'
if old not in s:
    raise SystemExit("BuildMemoryDataset prefix not found")
s = s.replace(old, new, 1)

old = 'matched:=lookupMatchesForRow(sh,row,lookups,lookupIDs,&rec);for _,idx:=range matched{ds.LookupDiagnostics[idx].EnrichedRows++}'
new = 'matched:=applyLookupEnrichment(&rec,sh,row,lookups,lookupIDs);for _,idx:=range matched{ds.LookupDiagnostics[idx].EnrichedRows++}'
if old not in s:
    raise SystemExit("lookup call pattern not found")
s = s.replace(old, new, 1)

p.write_text(s, encoding="utf-8")

test = root / "lookup_value_test.go"
if not test.exists():
    test.write_text('''package main

import "testing"

func TestLookupEnrichmentMatchesNumericClienteAndPrefersExactHeader(t *testing.T) {
\ttable, err := parseLookupTable([]byte("Nº CLIENTE;ATRIBUTO\\n80003285;Cadena\\n"), "Clientes", "Clientes.csv")
\tif err != nil { t.Fatal(err) }
\ttable = reindexLookupTable(table, "Nº CLIENTE")
\tsh := MemorySheet{
\t\tColumns: []MemoryColumn{
\t\t\t{ID:"NOMBRE", Title:"CLIENTE", Index:0, Type:ValueText},
\t\t\t{ID:"NUM", Title:"Nº CLIENTE", Index:1, Type:ValueNumber},
\t\t},
\t\tRows: []MemoryRow{{Values: map[string]MemoryValue{
\t\t\t"NOMBRE": {ColumnID:"NOMBRE", Raw:"Juan", Type:ValueText},
\t\t\t"NUM": {ColumnID:"NUM", Raw:"80003285.00", Type:ValueNumber},
\t\t}}},
\t}
\tif got := lookupColumnForKey(sh, "Nº CLIENTE"); got != "NUM" { t.Fatalf("lookup column=%q; want NUM", got) }
\trec := DatasetRecord{Values: map[string]MemoryValue{}}
\tids := map[string]map[string]string{"Clientes": {"ATRIBUTO": "LOOKUP:CLIENTES:ATRIBUTO"}}
\tmatched := lookupMatchesForRow(sh, sh.Rows[0], []lookupTable{table}, ids, &rec)
\tif len(matched) != 1 { t.Fatalf("matched=%d; want 1", len(matched)) }
\tif got := rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw; got != "Cadena" { t.Fatalf("lookup attribute=%q; want Cadena", got) }
}

func TestLookupEnrichmentNormalizesNumericThousands(t *testing.T) {
\ttable, err := parseLookupTable([]byte("NRO CLIENTE;ATRIBUTO\\n80.003.285;Cadena\\n"), "Clientes", "Clientes.csv")
\tif err != nil { t.Fatal(err) }
\ttable = reindexLookupTable(table, "NRO CLIENTE")
\tsh := MemorySheet{
\t\tColumns: []MemoryColumn{{ID:"NUM", Title:"NRO CLIENTE", Index:0, Type:ValueNumber}},
\t\tRows: []MemoryRow{{Values: map[string]MemoryValue{"NUM": {ColumnID:"NUM", Raw:"80003285.00", Type:ValueNumber}}}},
\t}
\trec := DatasetRecord{Values: map[string]MemoryValue{}}
\tids := map[string]map[string]string{"Clientes": {"ATRIBUTO": "LOOKUP:CLIENTES:ATRIBUTO"}}
\tif got := lookupMatchesForRow(sh, sh.Rows[0], []lookupTable{table}, ids, &rec); len(got) != 1 || rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw != "Cadena" { t.Fatalf("lookup failed: matches=%d value=%q", len(got), rec.Values["LOOKUP:CLIENTES:ATRIBUTO"].Raw) }
}
''', encoding="utf-8")

doc = root / "docs" / "REESCRITURA.md"
doc.parent.mkdir(parents=True, exist_ok=True)
text = doc.read_text(encoding="utf-8") if doc.exists() else "# REESCRITURA\n"
section = '''\n## Cruce por CSV auxiliar\n\nLos CSV auxiliares se colocan junto al ejecutable (o en el directorio de trabajo), excepto `GestionSO_Datos.csv`, que conserva su tratamiento como maestro. La clave se detecta por encabezado: primero se prioriza coincidencia exacta normalizada entre el encabezado del CSV y el encabezado físico del XLSX; si no existe, se admite la equivalencia de variantes como `Nº CLIENTE`, `N CLIENTE` y `NRO CLIENTE`. Los valores de clave se normalizan para tolerar espacios, separadores de miles y decimales enteros como `.00`. El enriquecimiento emite diagnóstico `[LOOKUP-DBG]` con base, encabezado, columna XLSX, primeras claves normalizadas y primera clave observada.\n'''
if "## Cruce por CSV auxiliar" not in text:
    doc.write_text(text.rstrip() + section + "\n", encoding="utf-8")
