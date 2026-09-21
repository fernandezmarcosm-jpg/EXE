package main

import("bytes";_ "embed";"encoding/csv";"encoding/json";"fmt";"log";"math";"os";"path/filepath";"strconv";"strings")

//go:embed "acceso chatgpt/GestionSO_Datos.csv"
var embeddedMasterCSV []byte

type CalculatedColumn struct{Name string; Formula string; Percent bool}
type DatasetSettings struct{Decimals int `json:"decimals"`;FontSize int `json:"font_size"`;RowHeight int `json:"row_height"`;ColumnWidths map[string]int `json:"column_widths"`;SOColumn int `json:"so_column"`;JoinExcelColumn string `json:"join_excel_column"`;FormulaTitle string `json:"formula_title"`;Formula string `json:"formula"`;SubtotalColumn string `json:"subtotal_column"`;SubtotalEnabled bool `json:"subtotal_enabled"`;ColumnTitles map[string]string `json:"column_titles"`;ColumnOrder []string `json:"column_order"`;MaxColumns int `json:"max_columns"`;VisibleColumns []string `json:"visible_columns"`;ColumnDecimals map[string]int `json:"column_decimals"`;SubtotalColumns []string `json:"subtotal_columns"`;ColumnPercent map[string]bool `json:"column_percent"`;ColumnCurrency map[string]bool `json:"column_currency"`;ColumnThousands map[string]bool `json:"column_thousands"`;HighlightNegative map[string]bool `json:"highlight_negative"`;ColumnHighlightSign map[string]bool `json:"column_highlight_sign"`;ColumnBackground map[string]string `json:"column_background"`;ColumnAlign map[string]string `json:"column_align"`;ColumnTypes map[string]string `json:"column_types"`;CalculatedColumns []CalculatedColumn `json:"calculated_columns"`;SubtotalAgg map[string]string `json:"subtotal_agg"`}
func defaultDatasetSettings()DatasetSettings{return DatasetSettings{Decimals:2,FontSize:10,RowHeight:28,ColumnWidths:map[string]int{},SOColumn:5,JoinExcelColumn:"CLAVE",FormulaTitle:"CALCULADA",Formula:"",SubtotalEnabled:false,ColumnTitles:map[string]string{},ColumnOrder:[]string{},MaxColumns:500,VisibleColumns:[]string{},ColumnDecimals:map[string]int{},ColumnPercent:map[string]bool{},ColumnCurrency:map[string]bool{},ColumnThousands:map[string]bool{},HighlightNegative:map[string]bool{},ColumnHighlightSign:map[string]bool{},ColumnBackground:map[string]string{},ColumnAlign:map[string]string{},ColumnTypes:map[string]string{},CalculatedColumns:[]CalculatedColumn{},SubtotalAgg:map[string]string{}}}
func datasetSettingsPortablePath()string{
	if x,e:=os.Executable();e==nil&&strings.TrimSpace(x)!=""{
		return filepath.Join(filepath.Dir(x),"dataset.txt")
	}
	if d,e:=os.Getwd();e==nil&&strings.TrimSpace(d)!=""{
		return filepath.Join(d,"dataset.txt")
	}
	return ""
}
func datasetSettingsFallbackPath()string{
	if d,e:=os.UserConfigDir();e==nil&&strings.TrimSpace(d)!=""{
		return filepath.Join(d,"GestionSO V57","dataset.txt")
	}
	return filepath.Join(os.TempDir(),"GestionSO-V57-dataset.txt")
}
func datasetSettingsPaths()[]string{
	paths:=make([]string,0,2)
	add:=func(p string){if strings.TrimSpace(p)==""{return};for _,x:=range paths{if filepath.Clean(x)==filepath.Clean(p){return}};paths=append(paths,p)}
	add(datasetSettingsPortablePath())
	add(datasetSettingsFallbackPath())
	return paths
}
func datasetSettingsPath()string{
	paths:=datasetSettingsPaths()
	if len(paths)>0{return paths[0]}
	return filepath.Join(os.TempDir(),"GestionSO-V57-dataset.txt")
}
func loadDatasetSettings()DatasetSettings{
	s:=defaultDatasetSettings()
	for _,p:=range datasetSettingsPaths(){
		b,e:=os.ReadFile(p)
		if e!=nil{continue}
		if json.Unmarshal(b,&s)==nil{
			datasetSettingsNormalize(&s)
			return s
		}
	}
	return s
}
func datasetSettingsNormalize(s *DatasetSettings){if s.Decimals<0||s.Decimals>8{s.Decimals=2};if s.FontSize<=0{s.FontSize=14};if s.FontSize<6{s.FontSize=6};if s.FontSize>28{s.FontSize=28};if s.RowHeight<10||s.RowHeight>60{s.RowHeight=28};if s.ColumnWidths==nil{s.ColumnWidths=map[string]int{}};if s.SOColumn<1||s.SOColumn>1024{s.SOColumn=5};if s.MaxColumns<1||s.MaxColumns>100{s.MaxColumns=20};if s.ColumnTitles==nil{s.ColumnTitles=map[string]string{}};if s.ColumnDecimals==nil{s.ColumnDecimals=map[string]int{}};if s.ColumnPercent==nil{s.ColumnPercent=map[string]bool{}};if s.ColumnCurrency==nil{s.ColumnCurrency=map[string]bool{}};if s.ColumnThousands==nil{s.ColumnThousands=map[string]bool{}};if s.ColumnHighlightSign==nil{s.ColumnHighlightSign=map[string]bool{}};if s.ColumnBackground==nil{s.ColumnBackground=map[string]string{}};if s.ColumnAlign==nil{s.ColumnAlign=map[string]string{}};for id,v:=range s.ColumnAlign{v=strings.ToLower(strings.TrimSpace(v));if v!="left"&&v!="center"&&v!="right"{delete(s.ColumnAlign,id)}else{s.ColumnAlign[id]=v}};if s.HighlightNegative==nil{s.HighlightNegative=map[string]bool{}};if s.ColumnTypes==nil{s.ColumnTypes=map[string]string{}};if s.SubtotalAgg==nil{s.SubtotalAgg=map[string]string{}}}
func saveDatasetSettings(s DatasetSettings)error{
	datasetSettingsNormalize(&s)
	if viewDataset!=nil{
		keys:=make([]string,0,len(viewDataset.Columns))
		for _,c:=range viewDataset.Columns{if c.Visible{keys=append(keys,datasetColumnKey(c))}}
		if len(keys)==0{keys=[]string{"__NONE__"}}
		s.VisibleColumns=keys
	}
	b,e:=json.MarshalIndent(s,"","  ")
	if e!=nil{return e}
	paths:=datasetSettingsPaths()
	var lastErr error
	for _,p:=range paths{
		if e:=os.MkdirAll(filepath.Dir(p),0755);e!=nil{lastErr=e;continue}
		if e:=os.WriteFile(p,b,0644);e==nil{return nil}else{lastErr=e}
	}
	if lastErr!=nil{return lastErr}
	return fmt.Errorf("no hay una ruta disponible para guardar dataset.txt")
}

type DatasetColumn struct{ID,Title,Source string;Type ValueType;Width int;Visible bool}
type DatasetRecord struct{SO string;Values map[string]MemoryValue}
type MemoryDataset struct{Columns []DatasetColumn;Records []DatasetRecord;CSVRows,Enriched,DuplicateSO int;SourceFiles []string}
type csvMaster struct{Headers []string;ByKey map[string]map[string]string}
func normalizeJoinKey(v string)string{return strings.ToUpper(strings.TrimSpace(v))}
func normalizeHeader(s string) string{s=strings.TrimSpace(s);s=strings.ReplaceAll(s,"Nº","N");s=strings.ReplaceAll(s,"N°","N");s=strings.ReplaceAll(s,"º","o");s=strings.ReplaceAll(s,"°","o");s=strings.ToUpper(s);s=strings.Join(strings.Fields(s)," ");return s}
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
func datasetColumnCurrency(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnCurrency,c) || strings.EqualFold(strings.TrimSpace(datasetSettingString(appSettings.ColumnTypes,c)),"moneda")}
func datasetColumnThousands(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnThousands,c) || datasetColumnCurrency(c)}
func datasetColumnHighlightSign(c DatasetColumn)bool{return datasetSettingBool(appSettings.ColumnHighlightSign,c)}
func datasetColumnBackground(c DatasetColumn)string{return strings.TrimSpace(appSettings.ColumnBackground[datasetColumnKey(c)])}
func datasetColumnAlign(c DatasetColumn)string{if v:=strings.ToLower(strings.TrimSpace(appSettings.ColumnAlign[datasetColumnKey(c)]));v=="left"||v=="center"||v=="right"{return v};if c.Type==ValueNumber||c.Source=="CALCULADA"{return "right"};return "left"}
func parseConfiguredType(v string)ValueType{switch strings.ToLower(strings.TrimSpace(v)){case "entero","integer":return ValueNumber;case "decimal","number","numero":return ValueNumber;case "fecha","date":return ValueDate;case "porcentaje","percent":return ValueNumber};return ValueEmpty}
func applyConfiguredColumnType(c DatasetColumn)DatasetColumn{t:=parseConfiguredType(datasetSettingString(appSettings.ColumnTypes,c));if t!=ValueEmpty{c.Type=t};return c}
func loadMasterCSV(path string)(*csvMaster,string,error){data:=embeddedMasterCSV;source:="CSV maestro integrado: GestionSO_Datos.csv";if path!=""{if b,e:=os.ReadFile(path);e==nil{data=b;source=path}}else{var cs []string;if x,e:=os.Executable();e==nil{cs=append(cs,filepath.Join(filepath.Dir(x),"GestionSO_Datos.csv"))};if x,e:=os.Getwd();e==nil{cs=append(cs,filepath.Join(x,"GestionSO_Datos.csv"),filepath.Join(x,"acceso chatgpt","GestionSO_Datos.csv"))};for _,p:=range cs{if b,e:=os.ReadFile(p);e==nil{data=b;source=p;break}}};r:=csv.NewReader(bytes.NewReader(data));r.Comma=';';r.FieldsPerRecord=-1;rows,e:=r.ReadAll();if e!=nil{return nil,source,e};if len(rows)==0{return nil,source,fmt.Errorf("CSV maestro vacío")};h:=make([]string,len(rows[0]));ki:=-1;for i,x:=range rows[0]{h[i]=strings.TrimPrefix(x,"\ufeff");n:=normalizeHeader(h[i]);if n=="CLAVE"||n=="SKU"{if ki<0{ki=i}}};if ki<0{return nil,source,fmt.Errorf("CSV maestro sin columna CLAVE/SKU")};m:=&csvMaster{Headers:h,ByKey:map[string]map[string]string{}};for _,row:=range rows[1:]{if ki>=len(row){continue};k:=normalizeJoinKey(row[ki]);if k==""{continue};v:=map[string]string{};for i,x:=range h{if i<len(row){v[x]=strings.TrimSpace(row[i])}};m.ByKey[k]=v};return m,source,nil}
func csvHeaderTypes(m *csvMaster)map[string]ValueType{out:=map[string]ValueType{};if m==nil{return out};for _,h:=range m.Headers{out[h]=ValueText};for _,row:=range m.ByKey{for h,raw:=range row{if raw==""{continue};t:=inferValueType(raw);if t==ValueNumber{out[h]=ValueNumber}}};return out}
func findConfiguredSOColumn(sh MemorySheet,configured int)string{if configured>0{for _,c:=range sh.Columns{if c.Index==configured-1{return c.ID}}};for _,c:=range sh.Columns{n:=normalizeHeader(c.Title);switch n{case "SO","NRO SO","N SO","NUMERO SO","NÚMERO SO","ORDEN DE VENTA","ORDENVENTA","SALES ORDER":return c.ID}};return ""}
func BuildMemoryDataset(docs []*xlsxDoc,s DatasetSettings)(*MemoryDataset,error){if len(docs)==0{return nil,fmt.Errorf("no hay archivos XLSX seleccionados")};m,_,e:=loadMasterCSV("");if e!=nil{return nil,e};ds:=&MemoryDataset{CSVRows:len(m.ByKey)};addCSV:=func(t string,typ ValueType)string{x:=fmt.Sprintf("CSV:%s",normalizeHeader(t));c:=DatasetColumn{ID:x,Title:strings.TrimSpace(t),Source:"CSV",Type:typ,Width:140,Visible:false};c=applyConfiguredColumnType(c);ds.Columns=append(ds.Columns,c);return x};csvTypes:=csvHeaderTypes(m);csvIDs:=map[string]string{};csvJoinHeader:="";for _,h:=range m.Headers{if _,exists:=csvIDs[h];exists{continue};csvIDs[h]=addCSV(h,csvTypes[h]);if csvJoinHeader==""&&(normalizeHeader(h)=="CLAVE"||normalizeHeader(h)=="SKU"){csvJoinHeader=h}};seenLines:=map[string]bool{};createdColumns:=map[string]bool{}
	for _,doc:=range docs{if doc==nil||doc.Memory==nil{continue};for _,sh:=range doc.Memory.Sheets{soID:=findConfiguredSOColumn(sh,s.SOColumn);joinID,itemID:="","";mapID:=map[string]string{};usedIDs:=map[string]int{};explicitJoinMatched:=false;for _,c:=range sh.Columns{id:=uniqueNormalizedHeaderID(c.Title,usedIDs);if !createdColumns[id]{dc:=DatasetColumn{ID:id,Title:c.Title,Source:"XLSX",Type:c.Type,Width:c.Width,Visible:true};if strings.TrimSpace(c.Title)==""{dc.Title=fmt.Sprintf("C%d",c.Index+1)};if dc.Width<1{dc.Width=140};dc=applyConfiguredColumnType(dc);ds.Columns=append(ds.Columns,dc);createdColumns[id]=true};mapID[c.ID]=id;normalizedTitle:=normalizeHeader(c.Title);if strings.TrimSpace(s.JoinExcelColumn)!=""&&normalizedTitle==normalizeHeader(s.JoinExcelColumn){joinID=c.ID;explicitJoinMatched=true};if normalizedTitle=="ITEM"{itemID=c.ID}}
		if !explicitJoinMatched {for _,synonym:=range []string{"CLAVE","SKU","CODIGO","COD"}{for _,c:=range sh.Columns{if normalizeHeader(c.Title)==synonym{joinID=c.ID;break}};if joinID!=""{break}}}
		if joinID=="" {joinID=itemID}
		joinExcelTitle:="";for _,c:=range sh.Columns{if c.ID==joinID{joinExcelTitle=c.Title;break}}
		if soID==""{continue};sheetEnriched:=0;for _,row:=range sh.Rows{v,ok:=row.Values[soID];if !ok||strings.TrimSpace(v.Raw)==""{continue};lineKey:="";if itemID!=""{if item,ok:=row.Values[itemID];ok&&strings.TrimSpace(item.Raw)!=""{lineKey=normalizeJoinKey(v.Raw)+"\x1f"+normalizeJoinKey(item.Raw)}};if lineKey!=""{if seenLines[lineKey]{ds.DuplicateSO++;continue};seenLines[lineKey]=true};rec:=DatasetRecord{SO:v.Raw,Values:map[string]MemoryValue{}};join:="";if joinID!=""{if x,ok:=row.Values[joinID];ok{join=normalizeJoinKey(x.Raw)}};for old,newID:=range mapID{if x,ok:=row.Values[old];ok{x.ColumnID=newID;rec.Values[newID]=x}};if item,ok:=m.ByKey[join];ok{ds.Enriched++;sheetEnriched++;for _,h:=range m.Headers{raw:=item[h];if raw==""{continue};id,ok:=csvIDs[h];if !ok{continue};rec.Values[id]=makeMemoryValue(id,raw)}};ds.Records=append(ds.Records,rec)};log.Printf("[JOIN] Excel=%q CSV=%q filas_match=%d Enriched=%d",joinExcelTitle,csvJoinHeader,sheetEnriched,ds.Enriched)}}
	if len(ds.Records)==0{return nil,fmt.Errorf("no se encontraron filas con SO: columna configurada N°%d y tampoco se detectó una cabecera SO válida",s.SOColumn)}
	order:=make([]string,0,len(ds.Columns));for _,c:=range ds.Columns{order=append(order,c.ID)};s.ColumnOrder=order;s.VisibleColumns=order
	if s.MaxColumns<len(ds.Columns){s.MaxColumns=len(ds.Columns)+50};ensureCalculatedDatasetColumns(ds,s);log.Printf("[JOIN] final: filas_csv=%d Enriched=%d",len(m.ByKey),ds.Enriched);return ds,nil}
func ensureCalculatedDatasetColumns(ds *MemoryDataset,s DatasetSettings){if ds==nil{return};ensure:=func(name string)DatasetColumn{for _,c:=range ds.Columns{if c.Source=="CALCULADA"&&strings.EqualFold(c.Title,name){return c}};c:=DatasetColumn{ID:fmt.Sprintf("D%03d",len(ds.Columns)+1),Title:name,Source:"CALCULADA",Type:ValueNumber,Width:150,Visible:true};ds.Columns=append(ds.Columns,c);return c};if strings.TrimSpace(s.FormulaTitle)!=""&&strings.TrimSpace(s.Formula)!=""{ensure(strings.TrimSpace(s.FormulaTitle))};for _,cc:=range s.CalculatedColumns{if strings.TrimSpace(cc.Name)!=""&&strings.TrimSpace(cc.Formula)!=""{ensure(strings.TrimSpace(cc.Name))}}}
func(e *MemoryDataset)columnByTitle(t string)(DatasetColumn,bool){for _,c:=range e.Columns{if strings.EqualFold(strings.TrimSpace(c.Title),strings.TrimSpace(t))||strings.EqualFold(strings.TrimSpace(datasetColumnDisplayTitle(c)),strings.TrimSpace(t)){return c,true}};return DatasetColumn{},false}
func evaluateFormula(expr string,r DatasetRecord,cols []DatasetColumn)(float64,bool){vals:=map[string]float64{};for _,c:=range cols{if v,ok:=r.Values[c.ID];ok&&(v.Type==ValueNumber||v.Type==ValueDate){n:=v.Number;title:=strings.ToLower(strings.TrimSpace(c.Title));vals[title]=n;vals[strings.ToLower(strings.TrimSpace(c.Source+":"+c.Title))]=n;if c.Source=="CSV"{vals["csv:"+title]=n}}};p:=&formulaParser{s:expr,values:vals};v,ok:=p.expr();p.skip();return v,ok&&p.pos==len(p.s)}
type formulaParser struct{s string;values map[string]float64;pos int};func(p *formulaParser)skip(){for p.pos<len(p.s)&&(p.s[p.pos]==' '||p.s[p.pos]=='\t'){p.pos++}};func(p *formulaParser)expr()(float64,bool){a,ok:=p.term();if !ok{return 0,false};for{p.skip();if p.pos>=len(p.s){return a,true};o:=p.s[p.pos];if o!='+'&&o!='-'{return a,true};p.pos++;b,ok:=p.term();if !ok{return 0,false};if o=='+'{a+=b}else{a-=b}}};func(p *formulaParser)term()(float64,bool){a,ok:=p.factor();if !ok{return 0,false};for{p.skip();if p.pos>=len(p.s){return a,true};o:=p.s[p.pos];if o!='*'&&o!='/'{return a,true};p.pos++;b,ok:=p.factor();if !ok{return 0,false};if o=='*'{a*=b}else{if b==0{return 0,false};a/=b}}};func(p *formulaParser)factor()(float64,bool){p.skip();if p.pos>=len(p.s){return 0,false};if p.s[p.pos]=='(' {p.pos++;v,ok:=p.expr();p.skip();if p.pos>=len(p.s)||p.s[p.pos]!=')'{return 0,false};p.pos++;return v,ok};st:=p.pos;if p.s[p.pos]=='['{if e:=strings.IndexByte(p.s[st:],']');e>=0{e+=st;key:=strings.ToLower(strings.TrimSpace(p.s[st+1:e]));p.pos=e+1;v,ok:=p.values[key];return v,ok}};for p.pos<len(p.s)&&((p.s[p.pos]>='0'&&p.s[p.pos]<='9')||p.s[p.pos]=='.'||p.s[p.pos]==','){p.pos++};if p.pos>st{v,e:=strconv.ParseFloat(strings.ReplaceAll(p.s[st:p.pos],",","."),64);return v,e==nil};return 0,false}
func applyDatasetFormula(ds *MemoryDataset,s DatasetSettings){if ds==nil{return};ensureCalculatedDatasetColumns(ds,s);if s.ColumnPercent==nil{s.ColumnPercent=map[string]bool{}};if s.ColumnTypes==nil{s.ColumnTypes=map[string]string{}};for _,cc:=range s.CalculatedColumns{if c,ok:=ds.columnByTitle(cc.Name);ok{s.ColumnPercent[c.ID]=cc.Percent;if cc.Percent{s.ColumnTypes[c.ID]="porcentaje"}}};if s.Formula!=""&&s.FormulaTitle!=""{if c,ok:=ds.columnByTitle(s.FormulaTitle);ok{for i:=range ds.Records{if v,ok:=evaluateFormula(s.Formula,ds.Records[i],ds.Columns);ok{ds.Records[i].Values[c.ID]=MemoryValue{ColumnID:c.ID,Type:ValueNumber,Number:v,Raw:formatDatasetNumber(v,datasetColumnDecimals(c))}}}}};for _,cc:=range s.CalculatedColumns{c,ok:=ds.columnByTitle(cc.Name);if !ok{continue};for i:=range ds.Records{if v,ok:=evaluateFormula(cc.Formula,ds.Records[i],ds.Columns);ok{ds.Records[i].Values[c.ID]=MemoryValue{ColumnID:c.ID,Type:ValueNumber,Number:v,Raw:formatDatasetNumber(v,datasetColumnDecimals(c))}}}}}
type SubtotalRow struct{GroupValue string `json:"group_value"`;GroupCount int `json:"group_count"`;Values map[string]string `json:"values"`;Total bool `json:"total"`}

func computeSubtotals(ds *MemoryDataset)[]SubtotalRow{
	if ds==nil||strings.TrimSpace(appSettings.SubtotalColumn)==""||len(appSettings.SubtotalAgg)==0{return nil}
	group,ok:=ds.columnByTitle(appSettings.SubtotalColumn);if !ok{for _,c:=range ds.Columns{if c.ID==appSettings.SubtotalColumn{group=c;ok=true;break}}};if !ok{return nil}
	type accum struct{sum map[string]float64;count map[string]int;unique map[string]map[string]struct{}}
	groups:=map[string]*accum{};order:=[]string{};seenGroups:=map[string]struct{}{}
	total:=&accum{sum:map[string]float64{},count:map[string]int{},unique:map[string]map[string]struct{}{}}
	for _,r:=range ds.Records{
		gv:="";if v,ok:=r.Values[group.ID];ok{gv=datasetValueText(group,v)};if strings.TrimSpace(gv)!=""{if _,seen:=seenGroups[gv];!seen{seenGroups[gv]=struct{}{}}};if _,ok:=groups[gv];!ok{groups[gv]=&accum{sum:map[string]float64{},count:map[string]int{},unique:map[string]map[string]struct{}{}};order=append(order,gv)}
		g:=groups[gv]
		for id,agg:=range appSettings.SubtotalAgg{
			if agg!="suma"&&agg!="promedio"&&agg!="conteo_unico"{continue}
			for _,c:=range ds.Columns{if c.ID!=id{continue}
				v,has:=r.Values[id]
				if agg=="conteo_unico"{raw:=strings.TrimSpace(v.Raw);if has&&raw!=""{if g.unique[id]==nil{g.unique[id]=map[string]struct{}{}};if total.unique[id]==nil{total.unique[id]=map[string]struct{}{}};g.unique[id][raw]=struct{}{};total.unique[id][raw]=struct{}{}};break}
				if has&&v.Type==ValueNumber{g.sum[id]+=v.Number;g.count[id]++;total.sum[id]+=v.Number;total.count[id]++}
				break
			}
		}
	}
	makeRow:=func(label string,a *accum,totalRow bool)SubtotalRow{vals:=map[string]string{};for id,agg:=range appSettings.SubtotalAgg{if agg!="suma"&&agg!="promedio"&&agg!="conteo_unico"{continue};if agg=="conteo_unico"{vals[id]=strconv.Itoa(len(a.unique[id]));continue};v:=a.sum[id];if agg=="promedio"{if a.count[id]==0{continue};v/=float64(a.count[id])};for _,c:=range ds.Columns{if c.ID==id{vals[id]=datasetValueText(c,MemoryValue{ColumnID:id,Type:ValueNumber,Number:v});break}}};return SubtotalRow{GroupValue:label,Values:vals,Total:totalRow}}
	out:=make([]SubtotalRow,0,len(order)+1);for _,gv:=range order{out=append(out,makeRow(gv,groups[gv],false))};out=append(out,makeRow("TOTAL GENERAL",total,true));out[len(out)-1].GroupCount=len(seenGroups);return out
}

func formatDatasetNumber(v float64,d int)string{if d<0{d=0};if d>8{d=8};if math.IsNaN(v)||math.IsInf(v,0){return ""};return strconv.FormatFloat(v,'f',d,64)}
func formatDatasetNumberGrouped(v float64,d int)string{raw:=formatDatasetNumber(math.Abs(v),d);parts:=strings.SplitN(raw,".",2);intPart:=parts[0];for i:=len(intPart)-3;i>0;i-=3{intPart=intPart[:i]+"."+intPart[i:]};out:=intPart;if len(parts)==2{out+=","+parts[1]};if v<0{out="-"+out};return out}
