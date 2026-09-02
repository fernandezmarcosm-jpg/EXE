// RECONSTRUCCION DE GestionSO V57.
// Este archivo NO es el fuente original. Es una reimplementacion basada en
// simbolos, strings y comportamiento observable del binario V57.
// HECHO VERIFICADO: existen simbolos para XLSX, persistencia, vistas,
// configuracion, opciones y simulador.
// INFERENCIA: estructuras, formulas y conteos no recuperables son conservadores.
package main

import("archive/zip";"bytes";"crypto/sha256";"encoding/csv";"encoding/xml";"fmt";"io";"os";"path/filepath";"sort";"strconv";"strings")

type MasterRow map[string]string
type MasterData struct{Headers []string;Rows []MasterRow;Path string}
type Line struct{Values map[string]string;Source string;RowNumber int}
type ColumnDef struct{Name string;Width int;Hidden bool}
type xlsxDoc struct{SharedStrings []string;Sheets map[string][][]string}
type configData struct{MasterPath string;EnginePath string;Mode string}

func panicGuard(fn func()){defer func(){if r:=recover();r!=nil{logf("panicGuard recovered: %v",r)}}();fn()}
func defaultConfig()configData{return configData{Mode:"MODO: FACTURAS"}}
func LoadConfig()configData{c:=defaultConfig();p:=filepath.Join(os.TempDir(),"GestionSO-config.txt");b,e:=os.ReadFile(p);if e!=nil{return c};for _,ln:=range strings.Split(string(b),"\n"){kv:=strings.SplitN(strings.TrimSpace(ln),"=",2);if len(kv)!=2{continue};switch kv[0]{case "MasterPath":c.MasterPath=kv[1];case "EnginePath":c.EnginePath=kv[1];case "Mode":c.Mode=kv[1]}};return c}
func SaveConfig(c configData)error{return saveCfg(filepath.Join(os.TempDir(),"GestionSO-config.txt"),c)}
func saveCfg(path string,c configData)error{return os.WriteFile(path,[]byte(fmt.Sprintf("MasterPath=%s\nEnginePath=%s\nMode=%s\n",c.MasterPath,c.EnginePath,c.Mode)),0644)}

func ReadXLSX(path string)(*xlsxDoc,error){return readXLSXDoc(path)}
func readXLSXDoc(path string)(*xlsxDoc,error){z,e:=zip.OpenReader(path);if e!=nil{return nil,e};defer z.Close();d:=&xlsxDoc{Sheets:map[string][][]string{}};if ss,e:=readZipEntry(z,"xl/sharedStrings.xml");e==nil{d.SharedStrings=parseSharedStrings(ss)};for _,f:=range z.File{if strings.HasPrefix(f.Name,"xl/worksheets/")&&strings.HasSuffix(f.Name,".xml"){if b,e:=readZipEntry(z,f.Name);e==nil{d.Sheets[filepath.Base(f.Name)]=decodeSheet(b,d.SharedStrings)}}};return d,nil}
func readZipEntry(z *zip.ReadCloser,name string)([]byte,error){for _,f:=range z.File{if f.Name==name{r,e:=f.Open();if e!=nil{return nil,e};defer r.Close();return io.ReadAll(r)}};return nil,os.ErrNotExist}
type sharedXML struct{SI []struct{T string `xml:"t"`;R []struct{T string `xml:"t"`} `xml:"r"`} `xml:"si"`}
func parseSharedStrings(b []byte)[]string{var x sharedXML;if xml.Unmarshal(b,&x)!=nil{return nil};o:=make([]string,len(x.SI));for i,s:=range x.SI{if s.T!=""{o[i]=s.T}else{for _,r:=range s.R{o[i]+=r.T}}};return o}
type sheetXML struct{Rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`} `xml:"row"`}
func decodeSheet(b []byte,ss []string)[][]string{var x sheetXML;if xml.Unmarshal(b,&x)!=nil{return nil};return parseRows(x.Rows,ss)}
func parseRows(rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`},ss []string)[][]string{o:=make([][]string,0,len(rows));for _,r:=range rows{m:=map[int]string{};mx:=0;for _,c:=range r.Cells{col:=colFromRef(c.Ref);if col<1{continue};v:=c.Value;switch c.Type{case "s":if n,e:=strconv.Atoi(v);e==nil&&n>=0&&n<len(ss){v=ss[n]};case "inlineStr":v=c.Inline.T};m[col]=v;if col>mx{mx=col}};row:=make([]string,mx);for c,v:=range m{row[c-1]=v};o=append(o,row)};return normalizeRows(o)}
func normalizeRows(rows [][]string)[][]string{for i:=range rows{for len(rows[i])>0&&strings.TrimSpace(rows[i][len(rows[i])-1])==""{rows[i]=rows[i][:len(rows[i])-1]}};return rows}
func normalizeRowXML(b []byte)[]byte{return bytes.TrimSpace(b)}
func buildMergedSheet(rows [][]string)[]byte{var b bytes.Buffer;for i,r:=range rows{if i>0{b.WriteByte('\n')};for j,v:=range r{if j>0{b.WriteByte(',')};b.WriteString(xmlEscape(v))}};return b.Bytes()}
func rewriteRowNumber(ref string,n int)string{if n<1{return ref};i:=0;for i<len(ref)&&((ref[i]>='A'&&ref[i]<='Z')||(ref[i]>='a'&&ref[i]<='z')){i++};return ref[:i]+strconv.Itoa(n)}
func xmlEscape(s string)string{return strings.NewReplacer("&","&amp;","<","&lt;",">","&gt;","\"","&quot;","'","&apos;").Replace(s)}
func hashRow(r []string)string{h:=sha256.New();for _,v:=range r{_,_=h.Write([]byte(v));_,_=h.Write([]byte{0})};return fmt.Sprintf("%x",h.Sum(nil))}
func colFromRef(ref string)int{n:=0;for _,r:=range strings.ToUpper(ref){if r<'A'||r>'Z'{break};n=n*26+int(r-'A'+1)};return n}
func headerScore(row []string)int{score:=0;keys:=[]string{"factura","cliente","fecha","cuit","sku","producto","cantidad","importe","so"};for _,v:=range row{s:=strings.ToLower(strings.TrimSpace(v));for _,k:=range keys{if strings.Contains(s,k){score++;break}}};return score}
func headerRowIndex(rows [][]string)int{best,bestScore:=0,-1;for i,r:=range rows{if s:=headerScore(r);s>bestScore{best,bestScore=i,s}};return best}
func normalizedHeader(v string,index int)string{v=strings.TrimSpace(v);if v==""{return fmt.Sprintf("C%d",index+1)};return v}
func uniqueHeaders(row []string)[]string{o:=make([]string,len(row));seen:=map[string]int{};for i,v:=range row{h:=normalizedHeader(v,i);n:=seen[h];if n>0{n++;seen[h]=n;h=fmt.Sprintf("%s_%d",h,n)}else{seen[h]=1};o[i]=h};return o}
func mergeXLSX(paths []string)([][]string,error){var all [][]string;for _,p:=range paths{d,e:=ReadXLSX(p);if e!=nil{return nil,e};var best [][]string;bs:=-1;for _,rows:=range d.Sheets{s:=-1;if len(rows)>0{s=headerScore(rows[0])};if s>bs{bs=s;best=rows}};if len(best)==0{continue};if len(all)==0{all=append(all,best...);continue};start:=0;if hashRow(best[0])==hashRow(all[0]){start=1};all=append(all,best[start:]...)};return normalizeRows(all),nil}

func locateMaster()string{c:=LoadConfig();if c.MasterPath!=""{return c.MasterPath};return filepath.Join(os.TempDir(),"GestionSO_Datos.csv")}
func openMasterCSV()(*os.File,error){return os.OpenFile(locateMaster(),os.O_CREATE|os.O_RDWR|os.O_APPEND,0644)}
func ensureMasterHeaders(path string,headers []string)error{f,e:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0644);if e!=nil{return e};defer f.Close();st,e:=f.Stat();if e!=nil{return e};if st.Size()!=0{return nil};w:=csv.NewWriter(f);if e=w.Write(headers);e!=nil{return e};w.Flush();return w.Error()}
func LoadMaster(path ...string)(*MasterData,error){p:=locateMaster();if len(path)>0&&path[0]!=""{p=path[0]};f,e:=os.Open(p);if e!=nil{return nil,e};defer f.Close();r:=csv.NewReader(f);rows,e:=r.ReadAll();if e!=nil{return nil,e};m:=&MasterData{Path:p};if len(rows)==0{return m,nil};m.Headers=rows[0];for _,rr:=range rows[1:]{mr:=MasterRow{};for i,h:=range m.Headers{if i<len(rr){mr[h]=rr[i]}};m.Rows=append(m.Rows,mr)};return m,nil}
func SaveWithBackup(path string,data []byte)error{if b,e:=os.ReadFile(path);e==nil{if e=os.WriteFile(path+".bak",b,0644);e!=nil{return e}};return os.WriteFile(path,data,0644)}
func EnsureSKU(m *MasterData,sku string)MasterRow{for _,r:=range m.Rows{if r["SKU"]==sku{return r}};r:=MasterRow{"SKU":sku};m.Rows=append(m.Rows,r);return r}
func SetSO(r MasterRow,so string){r["SO"]=so}
func SOState(r MasterRow)string{return r["SO"]}

func BuildLines(rows [][]string,source string)[]Line{if len(rows)==0{return nil};hi:=headerRowIndex(rows);headers:=uniqueHeaders(rows[hi]);out:=make([]Line,0,len(rows)-1);for i,r:=range rows{if i==hi{continue};m:=map[string]string{};for j,v:=range r{if j<len(headers){m[headers[j]]=v}};out=append(out,Line{Values:m,Source:source,RowNumber:i+1})};return out}
func findFieldKey(l Line,candidates ...string)string{for _,cand:=range candidates{cl:=strings.ToLower(cand);for k:=range l.Values{kl:=strings.ToLower(strings.TrimSpace(k));if kl==cl||strings.Contains(kl,cl){return k}}};keys:=make([]string,0,len(l.Values));for k:=range l.Values{keys=append(keys,k)};sort.Strings(keys);if len(keys)>0{return keys[0]};return ""}
func GroupLines(lines []Line)map[string][]Line{g:=map[string][]Line{};for _,l:=range lines{k:=findFieldKey(l,"so","factura","cliente");g[l.Values[k]]=append(g[l.Values[k]],l)};return g}
func BuildFilteredSortedView(lines []Line,filter string)[]Line{return BuildFilteredSortedViewByHeaders(lines,map[string]string{"__all__":filter})}
func BuildFilteredSortedViewByHeaders(lines []Line,filters map[string]string)[]Line{o:=make([]Line,0,len(lines));for _,l:=range lines{ok:=true;for field,value:=range filters{value=strings.TrimSpace(value);if value==""{continue};if field=="__all__"{if !FilterValue(l,value){ok=false;break}}else if !strings.Contains(strings.ToLower(fieldValue(l,field)),strings.ToLower(value)){ok=false;break}};if ok{o=append(o,l)}};sort.SliceStable(o,func(i,j int)bool{return lineSortKey(o[i])<lineSortKey(o[j])});return o}
func fieldValue(l Line,field string)string{if v,ok:=l.Values[field];ok{return v};for k,v:=range l.Values{if strings.EqualFold(strings.TrimSpace(k),strings.TrimSpace(field)){return v}};return ""}
func FilterValue(l Line,filter string)bool{f:=strings.ToLower(filter);for _,v:=range l.Values{if strings.Contains(strings.ToLower(v),f){return true}};return false}
func DisplayValue(l Line,col string)string{return fieldValue(l,col)}
func rawVal(l Line,col string)string{return fieldValue(l,col)}
func rawDisplay(l Line,col string)string{return fieldValue(l,col)}
func availableColumns(lines []Line)[]ColumnDef{if len(lines)==0{return nil};keys:=make([]string,0,len(lines[0].Values));for k:=range lines[0].Values{keys=append(keys,k)};sort.Strings(keys);o:=make([]ColumnDef,0,len(keys));for _,k:=range keys{o=append(o,ColumnDef{Name:k,Width:120})};return o}
func defaultVisible(c []ColumnDef)[]ColumnDef{return c}
func defaultHeaderFilters(c []ColumnDef)map[string]string{m:=map[string]string{};for _,x:=range c{m[x.Name]=""};return m}
func lineSortKey(l Line)string{return strings.ToLower(l.Values[findFieldKey(l,"so","factura","cliente")])}
func groupSortKey(l Line)string{return lineSortKey(l)}
func keyText(l Line)string{return lineSortKey(l)}
func keyAuto(l Line)string{return lineSortKey(l)}
func cmpKey(a,b string)int{a=strings.ToLower(a);b=strings.ToLower(b);if a<b{return -1};if a>b{return 1};return 0}
func fmtPct(v float64)string{return fmt.Sprintf("%.2f%%",v)}
func parseNumber(s string)(float64,bool){s=strings.TrimSpace(s);if s==""{return 0,false};s=strings.ReplaceAll(s,"$","");s=strings.ReplaceAll(s," ","");if strings.Contains(s,",")&&strings.Contains(s,"."){if strings.LastIndex(s,",")>strings.LastIndex(s,"."){s=strings.ReplaceAll(s,".","");s=strings.ReplaceAll(s,",",".")}else{s=strings.ReplaceAll(s,",","")}}else if strings.Contains(s,","){s=strings.ReplaceAll(s,",",".")};v,e:=strconv.ParseFloat(s,64);return v,e==nil}
func numericByNames(l Line,names ...string)float64{for k,v:=range l.Values{kl:=strings.ToLower(strings.TrimSpace(k));for _,n:=range names{if kl==n||strings.Contains(kl,n){if x,ok:=parseNumber(v);ok{return x}}}};return 0}

type GroupSubtotal struct{Key string;Lines int;Bultos,Pallets,PK,Unidades,Neto,TN,CMG,PPP float64}
func CalculateSOSubtotals(lines []Line)[]GroupSubtotal{groups:=GroupLines(lines);keys:=make([]string,0,len(groups));for k:=range groups{keys=append(keys,k)};sort.Strings(keys);o:=make([]GroupSubtotal,0,len(keys));for _,k:=range keys{g:=groups[k];s:=GroupSubtotal{Key:k,Lines:len(g)};for _,l:=range g{s.Bultos+=numericByNames(l,"bultos","bulto");s.Pallets+=numericByNames(l,"pall","pallets","pallet");s.PK+=numericByNames(l,"pk");s.Unidades+=numericByNames(l,"unidades","unidad","cantidad");s.Neto+=numericByNames(l,"neto so","neto");s.TN+=numericByNames(l,"tn so","tn","toneladas");s.CMG+=numericByNames(l,"cmg","margen");s.PPP+=numericByNames(l,"ppp so","ppp")};o=append(o,s)};return o}
func FormatSOSubtotal(s GroupSubtotal)string{return fmt.Sprintf("SUBTOTAL SO %s | RET %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s",s.Key,s.Lines,"","","","",formatNumber(s.Unidades),formatNumber(s.Pallets),formatNumber(s.PK),formatNumber(s.Neto),formatNumber(s.TN),formatNumber(s.CMG),formatNumber(s.PPP))}
func formatNumber(v float64)string{if v==0{return "0"};return fmt.Sprintf("%.2f",v)}
func findAnyValue(l Line,names ...string)string{for _,n:=range names{if v:=fieldValue(l,n);v!=""{return v};for k,v:=range l.Values{if strings.Contains(strings.ToLower(k),strings.ToLower(n)){return v}}};return ""}
// INFERENCIA: conteos derivados de datos cargados, no formulas internas V54.
func BuildStatusBar(mode string,lines []Line,filterCount int,detail string)string{ret,lib:=0,0;soSet:=map[string]struct{}{};for _,l:=range lines{st:=strings.ToLower(findAnyValue(l,"estado"));if strings.Contains(st,"ret"){ret++};if strings.Contains(st,"lib"){lib++};if so:=findAnyValue(l,"so");so!=""{soSet[so]=struct{}{}}};if detail==""{detail="Detalle de Descuentos Aplicados..."};return fmt.Sprintf("%s | RETENIDAS %d | LIBERADAS %d | SO %d | LINEAS %d | %d filtros | %s | CSV",mode,ret,lib,len(soSet),len(lines),filterCount,detail)}

func SimDisplay(v float64)string{return fmt.Sprintf("%.2f",v)}
func exportVisible(lines []Line,path string)error{f,e:=os.Create(path);if e!=nil{return e};defer f.Close();w:=csv.NewWriter(f);cols:=availableColumns(lines);h:=make([]string,len(cols));for i,c:=range cols{h[i]=c.Name};if e=w.Write(h);e!=nil{return e};for _,l:=range lines{r:=make([]string,len(cols));for i,c:=range cols{r[i]=l.Values[c.Name]};if e=w.Write(r);e!=nil{return e}};w.Flush();return w.Error()}
func csvWriter(w io.Writer,rows [][]string)error{cw:=csv.NewWriter(w);for _,r:=range rows{if e:=cw.Write(r);e!=nil{return e}};cw.Flush();return cw.Error()}
func newModeConfig()configData{return defaultConfig()}
func openOption(_ uintptr){}
func optWndProc(hwnd,msg,w,l uintptr)uintptr{return 0}
func createOptControls(_ uintptr){}
func layoutOpt(_ uintptr){}
func optChecked(_ uintptr)bool{return false}
func applyOption(_ uintptr){}
func openSimulator(_ uintptr){}
func simWndProc(hwnd,msg,w,l uintptr)uintptr{return 0}
func createSimControls(_ uintptr){}
func layoutSim(_ uintptr){}
func rebuildSimColumns(_ uintptr){}
func captureSimState(_ uintptr)[]string{return nil}
func simKey(_ Line)string{return ""}
func simAdd(_ Line){}
func simApply(){}
func simRemove(_ int){}
func simPopulate(_ []Line){}
func handleSimNotify(_ uintptr,_ uintptr,_ uintptr)uintptr{return 0}
func CalcSimFromMaster(_ *MasterData)float64{return 0}
func masterScore(_ MasterRow)int{return 0}
func insertAfter(s []string,after string,value string)[]string{for i,v:=range s{if v==after{return append(append(append([]string{},s[:i+1]...),value),s[i+1:]...)}};return append(s,value)}
