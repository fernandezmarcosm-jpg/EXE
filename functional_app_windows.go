//go:build windows

package main

import (
    "fmt"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "syscall"
    "time"
    "unsafe"
)

const (
    appIDOpen = 2001
    appIDStatus = 2006
    appIDGrid = 2007
    appTimerRefresh = 3001
    WM_CREATE = 0x0001
    WM_SIZE = 0x0005
    WM_CLOSE = 0x0010
    WM_DESTROY = 0x0002
    WM_TIMER = 0x0113
    WM_COMMAND = 0x0111
    WM_APP_REFRESH = 0x8001
    WS_OVERLAPPEDWINDOW = 0x00CF0000
    WS_VISIBLE = 0x10000000
    WS_CHILD = 0x40000000
    WS_TABSTOP = 0x00010000
    WS_BORDER = 0x00800000
    WS_VSCROLL = 0x00200000
    WS_HSCROLL = 0x00100000
    BS_PUSHBUTTON = 0
    LBS_NOINTEGRALHEIGHT = 0x0100
    LBS_USETABSTOPS = 0x0080
    LB_RESETCONTENT = 0x0184
    LB_ADDSTRING = 0x0180
    LB_SETHORIZONTALEXTENT = 0x0194
    LB_SETCURSEL = 0x0186
    ofnExplorer = 0x00080000
    ofnPathMustExist = 0x00000800
    ofnFileMustExist = 0x00001000
    ofnHideReadOnly = 0x00000004
    ofnAllowMultiSelect = 0x00000200
)

type appRect struct { Left, Top, Right, Bottom int32 }
type appWndClass struct { CbSize uint32; Style uint32; LpfnWndProc uintptr; CbClsExtra, CbWndExtra int32; HInstance, HIcon, HCursor, HbrBackground uintptr; LpszMenuName, LpszClassName *uint16; HIconSm uintptr }
type appOpenFile struct { LStructSize uint32; _ uint32; HwndOwner uintptr; HInstance uintptr; Filter uintptr; CustomFilter uintptr; MaxCustom uint32; FilterIndex uint32; File uintptr; MaxFile uint32; _ uint32; FileTitle uintptr; MaxFileTitle uint32; _ uint32; InitialDir uintptr; Title uintptr; Flags uint32; FileOffset uint16; FileExtension uint16; DefExt uintptr; CustData uintptr; Hook uintptr; Template uintptr; Reserved uintptr; Reserved2 uint32; FlagsEx uint32 }
type appRow struct { Values []string }

var (
    appHwnd, appView, appStatus uintptr
    appHInstance uintptr
    appStateMu sync.Mutex
    appStatusText string
    appViewText string
    appSource string
    appLoading bool
    appRefreshPending bool
)

func appU16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func appSetText(h uintptr, s string) { if h==0{return}; start:=time.Now(); p:=appU16(s); user32.NewProc("SetWindowTextW").Call(h,uintptr(unsafe.Pointer(p))); if d:=time.Since(start);d>500*time.Millisecond{appLog("DIAGNOSTICO: SetWindowTextW tardó %s; caracteres=%d",d,len(s))} }
func appSetListBox(h uintptr,s string){if h==0{return};start:=time.Now();send:=user32.NewProc("SendMessageW");send.Call(h,LB_RESETCONTENT,0,0);maxExtent:=0;lines:=strings.Split(strings.ReplaceAll(s,"\r\n","\n"),"\n");for _,line:=range lines{if line==""{continue};p:=appU16(line);send.Call(h,LB_ADDSTRING,0,uintptr(unsafe.Pointer(p)));if n:=len([]rune(line));n>maxExtent{maxExtent=n}};if maxExtent>0{send.Call(h,LB_SETHORIZONTALEXTENT,uintptr(maxExtent*8),0)};if len(lines)>1{send.Call(h,LB_SETCURSEL,0,0)};if d:=time.Since(start);d>500*time.Millisecond{appLog("DIAGNOSTICO: actualización LISTBOX tardó %s; líneas=%d caracteres=%d",d,len(lines),len(s))}}
func appMake(parent uintptr,cls,text string,style uint32,x,y,w,h int,id uintptr)uintptr{c:=appU16(cls);t:=appU16(text);r,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(c)),uintptr(unsafe.Pointer(t)),uintptr(style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),parent,id,appHInstance,0);return r}

func crearVentana() uintptr { appHInstance,_,_=kernel32.NewProc("GetModuleHandleW").Call(0);cls:=appU16("GestionSOExcelImporter");wc:=appWndClass{CbSize:uint32(unsafe.Sizeof(appWndClass{})),LpfnWndProc:syscall.NewCallback(appWndProcLogged),HInstance:appHInstance,HCursor:func()uintptr{r,_,_:=user32.NewProc("LoadCursorW").Call(0,32512);return r}(),LpszClassName:cls};user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)));title:=appU16("GestionSO V57 - Importar Excel");h,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(cls)),uintptr(unsafe.Pointer(title)),WS_OVERLAPPEDWINDOW|WS_VISIBLE,0x80000000,0x80000000,1400,820,0,0,appHInstance,0);return h}

func appWndProc(hwnd uintptr,msg uint32,wp,lp uintptr)uintptr{switch msg{case WM_CREATE:appHwnd=hwnd;appBuildControls(hwnd);user32.NewProc("SetTimer").Call(hwnd,appTimerRefresh,200,0);appLog("EVENTO: timer de interfaz iniciado");return 0;case WM_SIZE:appLayout(hwnd);return 0;case WM_TIMER:if wp==appTimerRefresh{appStateMu.Lock();pending:=appRefreshPending;appRefreshPending=false;appStateMu.Unlock();if pending{appLog("EVENTO: timer detectó actualización pendiente");appRefreshView()}};return 0;case WM_COMMAND:id:=int(wp&0xffff);code:=uint32((wp>>16)&0xffff);if id==appIDOpen{appOpenXLSX(hwnd);return 0};if id==appIDGrid&&code==256{return 0};return 0;case WM_APP_REFRESH:appLog("EVENTO: WM_APP_REFRESH recibido (compatibilidad)");appStateMu.Lock();appRefreshPending=false;appStateMu.Unlock();appRefreshView();return 0;case WM_CLOSE:appLog("EVENTO: WM_CLOSE hwnd=0x%X",hwnd);user32.NewProc("KillTimer").Call(hwnd,appTimerRefresh);user32.NewProc("DestroyWindow").Call(hwnd);return 0;case WM_DESTROY:appLog("EVENTO: WM_DESTROY hwnd=0x%X",hwnd);user32.NewProc("PostQuitMessage").Call(0);return 0};r,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,uintptr(msg),wp,lp);return r}

func appBuildControls(hwnd uintptr){appMake(hwnd,"BUTTON","ABRIR EXCEL",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,10,10,130,32,appIDOpen);appStatus=appMake(hwnd,"STATIC","Listo. Seleccione uno o varios archivos XLSX para comenzar.",WS_CHILD|WS_VISIBLE,155,15,1100,24,appIDStatus);appView=appMake(hwnd,"LISTBOX","",WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|WS_HSCROLL|LBS_NOINTEGRALHEIGHT|LBS_USETABSTOPS,10,55,1360,700,appIDGrid)}
func appLayout(hwnd uintptr){var r appRect;user32.NewProc("GetClientRect").Call(hwnd,uintptr(unsafe.Pointer(&r)));w:=int(r.Right-r.Left);h:=int(r.Bottom-r.Top);if w<600{w=600};if h<300{h=300};user32.NewProc("MoveWindow").Call(appStatus,155,15,uintptr(maxInt(300,w-170)),24,1);user32.NewProc("MoveWindow").Call(appView,10,55,uintptr(w-20),uintptr(h-65),1)}
func maxInt(a,b int)int{if a>b{return a};return b}

func appPickXLSX(owner uintptr)[]string{f1,_:=syscall.UTF16FromString("Archivos Excel (*.xlsx)");f2,_:=syscall.UTF16FromString("*.xlsx");f3,_:=syscall.UTF16FromString("Todos los archivos (*.*)");f4,_:=syscall.UTF16FromString("*.*");filter:=make([]uint16,0,len(f1)+len(f2)+len(f3)+len(f4)+2);filter=append(filter,f1...);filter=append(filter,f2...);filter=append(filter,f3...);filter=append(filter,f4...);filter=append(filter,0,0);buffer:=make([]uint16,65536);title:=appU16("Seleccionar archivo Excel");defExt:=appU16("xlsx");ofn:=appOpenFile{LStructSize:uint32(unsafe.Sizeof(appOpenFile{})),HwndOwner:owner,Filter:uintptr(unsafe.Pointer(&filter[0])),FilterIndex:1,File:uintptr(unsafe.Pointer(&buffer[0])),MaxFile:uint32(len(buffer)),Title:uintptr(unsafe.Pointer(title)),Flags:ofnExplorer|ofnFileMustExist|ofnPathMustExist|ofnHideReadOnly|ofnAllowMultiSelect,DefExt:uintptr(unsafe.Pointer(defExt))};appLog("EVENTO: abrir selector XLSX");ret,_,_=comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&ofn)));if ret==0{appLog("EVENTO: selector XLSX cancelado");return nil};parts:=make([]string,0,8);pos:=0;for pos<len(buffer){end:=pos;for end<len(buffer)&&buffer[end]!=0{end++};if end==pos{break};s:=strings.TrimSpace(syscall.UTF16ToString(buffer[pos:end]));if s!=""{parts=append(parts,s)};pos=end+1};if len(parts)==0{return nil};if len(parts)>1{dir:=parts[0];out:=make([]string,0,len(parts)-1);for _,name:=range parts[1:]{if filepath.IsAbs(name){out=append(out,name)}else{out=append(out,filepath.Join(dir,name))}};appLog("EVENTO: selector devolvio %d archivos",len(out));return out};appLog("EVENTO: selector devolvio 1 archivo");return []string{parts[0]}}

func appOpenXLSX(owner uintptr){appStateMu.Lock();if appLoading{appStateMu.Unlock();return};appStateMu.Unlock();files:=appPickXLSX(owner);if len(files)==0{return};appStateMu.Lock();appLoading=true;appStatusText=fmt.Sprintf("Leyendo %d archivo(s)...",len(files));appRefreshPending=true;appStateMu.Unlock();appRefreshView();selected:=append([]string(nil),files...);go func(){defer appRecover("importacion XLSX");defer func(){appStateMu.Lock();appLoading=false;appStateMu.Unlock()}();appLog("EVENTO: inicio lectura XLSX; archivos=%d",len(selected));headers,rows,source,err:=loadPreviewXLSX(selected[0]);appStateMu.Lock();if err!=nil{appLog("ERROR importando XLSX: %v",err);appStatusText="ERROR: "+err.Error();appViewText="";appSource=""}else{appStatusText=fmt.Sprintf("%d fila(s) cargadas — %s",len(rows),filepath.Base(source));appViewText=renderPreview(headers,rows);appSource=source;appLog("EVENTO: XLSX cargado correctamente; filas=%d columnas=%d",len(rows),len(headers))};appRefreshPending=true;appStateMu.Unlock();r,_,e:=user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_REFRESH,0,0);appLog("EVENTO: PostMessageW refresh => ret=%d err=%v hwnd=0x%X",r,e,appHwnd)}()}

func loadPreviewXLSX(path string)([]string,[]appRow,string,error){doc,err:=ReadXLSX(path);if err!=nil{return nil,nil,"",err};if len(doc.Sheets)==0{return nil,nil,"",fmt.Errorf("el Excel no contiene hojas legibles")};names:=make([]string,0,len(doc.Sheets));for n:=range doc.Sheets{names=append(names,n)};sort.Strings(names);rows:=doc.Sheets[names[0]];if len(rows)==0{return nil,nil,"",fmt.Errorf("la hoja %q no contiene filas de datos",names[0])};n:=maxPreviewColumns(rows);headers:=makeHeaders(rows[0],n);const maxRows=5000;if len(rows)-1>maxRows{rows=rows[:maxRows+1]};out:=make([]appRow,0,len(rows)-1);for i:=1;i<len(rows);i++{vals:=make([]string,n);for j:=0;j<n;j++{if j<len(rows[i]){vals[j]=rows[i][j]}};out=append(out,appRow{Values:vals})};return headers,out,path,nil}
func maxPreviewColumns(rows[][]string)int{n:=0;for _,r:=range rows{if len(r)>n{n=len(r)}};if n>80{n=80};return n}
func makeHeaders(row[]string,n int)[]string{out:=make([]string,n);seen:=map[string]int{};for i:=0;i<n;i++{name:="Columna "+fmt.Sprint(i+1);if i<len(row)&&strings.TrimSpace(row[i])!=""{name=strings.TrimSpace(row[i])};base:=name;seen[base]++;if seen[base]>1{name=fmt.Sprintf("%s (%d)",base,seen[base])};out[i]=name};return out}
func renderPreview(headers[]string,rows[]appRow)string{var b strings.Builder;const maxChars=256*1024;for i,h:=range headers{if i>0{b.WriteByte('\t')};b.WriteString(cleanCell(h))};b.WriteString("\r\n");for _,row:=range rows{for i:=range headers{if i>0{b.WriteByte('\t')};if i<len(row.Values){b.WriteString(cleanCell(row.Values[i]))}};b.WriteString("\r\n");if b.Len()>=maxChars{b.WriteString("\r\n[Vista limitada a 256 KB para mantener la interfaz estable]\r\n");break}};return b.String()}
func cleanCell(s string)string{return strings.NewReplacer("\r"," ","\n"," ","\t"," ").Replace(s)}
func appRefreshView(){appLog("EVENTO: appRefreshView inicio");appStateMu.Lock();status:=appStatusText;view:=appViewText;appStateMu.Unlock();appSetText(appStatus,status);appSetListBox(appView,view);appLog("EVENTO: appRefreshView fin")}
