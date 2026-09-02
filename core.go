// RECONSTRUCCION DE GestionSO V57.
// Este archivo NO es el fuente original. Es una reimplementacion basada en
// simbolos, strings y comportamiento observable del binario V57.
//
// HECHO VERIFICADO: el binario contiene simbolos para XLSX, persistencia,
// vistas, configuracion, opciones y simulador.
// INFERENCIA: las estructuras internas y formulas no recuperables se mantienen
// conservadoras y se marcan como tales.

package main

import (
    "archive/zip"
    "bytes"
    "crypto/sha256"
    "encoding/csv"
    "encoding/xml"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
)

type MasterRow map[string]string
type MasterData struct { Headers []string; Rows []MasterRow; Path string }
type Line struct { Values map[string]string; Source string; RowNumber int }
type ColumnDef struct { Name string; Width int; Hidden bool }
type xlsxDoc struct { SharedStrings []string; Sheets map[string][][]string }

// INFERENCIA: el tipo exacto no es recuperable del fuente original.
type configData struct { MasterPath string; EnginePath string; Mode string }

func panicGuard(fn func()) { defer func(){ if r:=recover(); r!=nil { logf("panicGuard recovered: %v",r) } }(); fn() }
func defaultConfig() configData { return configData{Mode:"MODO: FACTURAS"} }
func LoadConfig() configData {
    c:=defaultConfig(); p:=filepath.Join(os.TempDir(),"GestionSO-config.txt")
    b,err:=os.ReadFile(p); if err!=nil{return c}
    for _,ln:=range strings.Split(string(b),"\n") { kv:=strings.SplitN(strings.TrimSpace(ln),"=",2); if len(kv)!=2{continue}; switch kv[0]{case "MasterPath":c.MasterPath=kv[1];case "EnginePath":c.EnginePath=kv[1];case "Mode":c.Mode=kv[1]} }
    return c
}
func SaveConfig(c configData) error { return saveCfg(filepath.Join(os.TempDir(),"GestionSO-config.txt"),c) }
func saveCfg(path string,c configData) error { return os.WriteFile(path,[]byte(fmt.Sprintf("MasterPath=%s\nEnginePath=%s\nMode=%s\n",c.MasterPath,c.EnginePath,c.Mode)),0644) }

func ReadXLSX(path string)(*xlsxDoc,error){return readXLSXDoc(path)}
func readXLSXDoc(path string)(*xlsxDoc,error){
    z,err:=zip.OpenReader(path); if err!=nil{return nil,err}; defer z.Close(); d:=&xlsxDoc{Sheets:map[string][][]string{}}
    if ss,e:=readZipEntry(z,"xl/sharedStrings.xml");e==nil{d.SharedStrings=parseSharedStrings(ss)}
    for _,f:=range z.File { if strings.HasPrefix(f.Name,"xl/worksheets/")&&strings.HasSuffix(f.Name,".xml"){if b,e:=readZipEntry(z,f.Name);e==nil{d.Sheets[filepath.Base(f.Name)]=decodeSheet(b,d.SharedStrings)}} }
    return d,nil
}
func readZipEntry(z *zip.ReadCloser,name string)([]byte,error){for _,f:=range z.File{if f.Name==name{r,e:=f.Open();if e!=nil{return nil,e};defer r.Close();return io.ReadAll(r)}};return nil,os.ErrNotExist}
type sharedXML struct{SI []struct{T string `xml:"t"`;R []struct{T string `xml:"t"`} `xml:"r"`} `xml:"si"`}
func parseSharedStrings(b []byte)[]string{var x sharedXML;if xml.Unmarshal(b,&x)!=nil{return nil};out:=make([]string,len(x.SI));for i,s:=range x.SI{if s.T!=""{out[i]=s.T}else{for _,r:=range s.R{out[i]+=r.T}}};return out}
type sheetXML struct{Rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`} `xml:"row"`}
func decodeSheet(b []byte,ss []string)[][]string{var x sheetXML;if xml.Unmarshal(b,&x)!=nil{return nil};return parseRows(x.Rows,ss)}
func parseRows(rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`},ss []string)[][]string{
    out:=make([][]string,0,len(rows)); for _,r:=range rows{m:=map[int]string{};max:=0;for _,c:=range r.Cells{col:=colFromRef(c.Ref);if col<1{continue};v:=c.Value;switch c.Type{case "s":if n,e:=strconv.Atoi(v);e==nil&&n>=0&&n<len(ss){v=ss[n]};case "inlineStr":v=c.Inline.T};m[col]=v;if col>max{max=col}};row:=make([]string,max);for c,v:=range m{row[c-1]=v};out=append(out,row)};return normalizeRows(out)
}
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
func uniqueHeaders(row []string)[]string{out:=make([]string,len(row));seen:=map[string]int{};for i,v:=range row{h:=normalizedHeader(v,i);if n:=seen[h];n>0{n++;seen[h]=n;h=fmt.Sprintf("%s_%d",h,n)}else{seen[h]=1};out[i]=h};return out}
func mergeXLSX(paths []string)([][]string,error){
    var all [][]string
    for _,p:=range paths{d,e:=ReadXLSX(p);if e!=nil{return nil,e};var best [][]string;bestScore:=-1;for _,rows:=range d.Sheets{s:=-1;if len(rows)>0{s=headerScore(rows[0])};if s>bestScore{bestScore=s;best=rows}};if len(best)==0{continue};if len(all)==0{all=append(all,best...);continue};start:=0;if len(best)>0&&len(all)>0&&hashRow(best[0])==hashRow(all[0]){start=1};all=append(all,best[start:]...)};return normalizeRows(all),nil
}

func locateMaster()string{c:=LoadConfig();if c.MasterPath!=""{return c.MasterPath};return filepath.Join(os.TempDir(),"GestionSO_Datos.csv")}
func openMasterCSV()(*os.File,error){return os.OpenFile(locateMaster(),os.O_CREATE|os.O_RDWR|os.O_APPEND,0644)}
func ensureMasterHeaders(path string,headers []string)error{f,e:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0644);if e!=nil{return e};defer f.Close();st,e:=f.Stat();if e!=nil{return e};if st.Size()!=0{return nil};w:=csv.NewWriter(f);if e=w.Write(headers);e!=nil{return e};w.Flush();return w.Error()}
func LoadMaster(path ...string)(*MasterData,error){p:=locateMaster();if len(path)>0&&path[0]!=""{p=path[0]};f,e:=os.Open(p);if e!=nil{return nil,e};defer f.Close();r:=csv.NewReader(f);rows,e:=r.ReadAll();if e!=nil{return nil,e};m:=&MasterData{Path:p};if len(rows)==0{return m,nil};m.Headers=rows[0];for _,rr:=range rows[1:]{mr:=MasterRow{};for i,h:=range m.Headers{if i<len(rr){mr[h]=rr[i]}};m.Rows=append(m.Rows,mr)};return m,nil}
func SaveWithBackup(path string,data []byte)error{if b,e:=os.ReadFile(path);e==nil{if e=os.WriteFile(path+".bak",b,0644);e!=nil{return e}};return os.WriteFile(path,data,0644)}
func EnsureSKU(m *MasterData,sku string)MasterRow{for _,r:=range m.Rows{if r["SKU"]==sku{return r}};r:=MasterRow{"SKU":sku};m.Rows=append(m.Rows,r);return r}
func SetSO(r MasterRow,so string){r["SO"]=so}
func SOState(r MasterRow)string{return r["SO"]}

// BuildLines: HECHO VERIFICADO, el binario contiene BuildLines y headerScore.
// INFERENCIA CONSERVADORA: la primera fila con mayor headerScore se trata como
// encabezado; los valores de cada Line usan esos nombres reales, no C1/C2...
func BuildLines(rows [][]string,source string)[]Line{
    if len(rows)==0{return nil};hi:=headerRowIndex(rows);headers:=uniqueHeaders(rows[hi]);out:=make([]Line,0,len(rows)-1)
    for i,r:=range rows{if i==hi{continue};m:=map[string]string{};for j,v:=range r{if j<len(headers){m[headers[j]]=v}};out=append(out,Line{Values:m,Source:source,RowNumber:i+1})};return out
}
func findFieldKey(l Line,candidates ...string)string{for _,cand:=range candidates{cl:=strings.ToLower(cand);for k:=range l.Values{kl:=strings.ToLower(strings.TrimSpace(k));if kl==cl||strings.Contains(kl,cl){return k}}};keys:=make([]string,0,len(l.Values));for k:=range l.Values{keys=append(keys,k)};sort.Strings(keys);if len(keys)>0{return keys[0]};return ""}
func GroupLines(lines []Line)map[string][]Line{g:=map[string][]Line{};for _,l:=range lines{k:=findFieldKey(l,"factura","cliente");g[l.Values[k]]=append(g[l.Values[k]],l)};return g}
func BuildFilteredSortedView(lines []Line,filter string)[]Line{out:=make([]Line,0,len(lines));for _,l:=range lines{if filter==""||FilterValue(l,filter){out=append(out,l)}};sort.SliceStable(out,func(i,j int)bool{return lineSortKey(out[i])<lineSortKey(out[j])});return out}
func FilterValue(l Line,filter string)bool{f:=strings.ToLower(filter);for _,v:=range l.Values{if strings.Contains(strings.ToLower(v),f){return true}};return false}
func DisplayValue(l Line,col string)string{return l.Values[col]}
func rawVal(l Line,col string)string{return l.Values[col]}
func rawDisplay(l Line,col string)string{return l.Values[col]}
func availableColumns(lines []Line)[]ColumnDef{if len(lines)==0{return nil};keys:=make([]string,0,len(lines[0].Values));for k:=range lines[0].Values{keys=append(keys,k)};sort.Strings(keys);out:=make([]ColumnDef,0,len(keys));for _,k:=range keys{out=append(out,ColumnDef{Name:k,Width:120})};return out}
func defaultVisible(cols []ColumnDef)[]ColumnDef{return cols}
func defaultHeaderFilters(cols []ColumnDef)map[string]string{m:=map[string]string{};for _,c:=range cols{m[c.Name]=""};return m}
func lineSortKey(l Line)string{return strings.ToLower(l.Values[findFieldKey(l,"factura","cliente")])}
func groupSortKey(l Line)string{return lineSortKey(l)}
func keyText(l Line)string{return lineSortKey(l)}
func keyAuto(l Line)string{return lineSortKey(l)}
func cmpKey(a,b string)int{a=strings.ToLower(a);b=strings.ToLower(b);if a<b{return -1};if a>b{return 1};return 0}
func fmtPct(v float64)string{return fmt.Sprintf("%.2f%%",v)}
func parseNumber(s string)(float64,bool){s=strings.TrimSpace(s);if s==""{return 0,false};s=strings.ReplaceAll(s,"$","");s=strings.ReplaceAll(s," ","");if strings.Contains(s,",")&&strings.Contains(s,"."){if strings.LastIndex(s,",")>strings.LastIndex(s,"."){s=strings.ReplaceAll(s,".","");s=strings.ReplaceAll(s,",",".")}else{s=strings.ReplaceAll(s,",","")}}else if strings.Contains(s,","){s=strings.ReplaceAll(s,",",".")};v,e:=strconv.ParseFloat(s,64);return v,e==nil}
func SimDisplay(v float64)string{return fmt.Sprintf("%.2f",v)}
func exportVisible(lines []Line,path string)error{f,e:=os.Create(path);if e!=nil{return e};defer f.Close();w:=csv.NewWriter(f);cols:=availableColumns(lines);headers:=make([]string,len(cols));for i,c:=range cols{headers[i]=c.Name};if e=w.Write(headers);e!=nil{return e};for _,l:=range lines{r:=make([]string,len(cols));for i,c:=range cols{r[i]=l.Values[c.Name]};if e=w.Write(r);e!=nil{return e}};w.Flush();return w.Error()}
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
