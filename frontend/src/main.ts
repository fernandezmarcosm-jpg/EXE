import './style.css'
import { ImportXLSX, GetSettings, SaveSettings, SetVisibleColumns, SetColumnOrder, AddCalculatedColumn, UpdateCalculatedColumn, DeleteCalculatedColumn, ListCalculatedColumns, SetColumnFormat, SetSubtotals } from '../wailsjs/go/main/App'

type Column = { id:string; title:string; source:string; type:string; visible:boolean }
type CalculatedColumn = { Name:string; Formula:string; Percent:boolean }
type SubtotalRow = { group_value:string; values:Record<string,string>; total:boolean }
type Dataset = { columns:Column[]; rows:Record<string,string>[]; total_rows:number; duplicated:number; csv_rows:number; enriched:number; source_files:string[]; subtotals:SubtotalRow[] }
type VisualState = { fontSize:number; rowHeight:number; columnWidths:Record<string,number>; settings:any }

const state:{data:Dataset|null; filters:Record<string,string>; columnsOpen:boolean; panelMode:'columns'|'calculated'; visual:VisualState; calculated:{editingOriginal:string;list:CalculatedColumn[]}} = {
  data:null, filters:{}, columnsOpen:false, panelMode:'columns',
  visual:{fontSize:14,rowHeight:28,columnWidths:{},settings:null}, calculated:{editingOriginal:'',list:[]}
}

const app = document.querySelector<HTMLDivElement>('#app')!
app.innerHTML = `
<div class="shell">
  <header class="toolbar">
    <button id="open">ABRIR EXCEL</button>
    <button id="columns">COLUMNAS</button><button id="calculated">CAMPOS CALCULADOS</button>
    <label class="visual-control">Fuente <input id="font-size" type="number" min="10" max="28" step="1"></label>
    <label class="visual-control">Fila <input id="row-height" type="number" min="18" max="60" step="1"></label>
    <div id="status" class="status">Seleccione uno o varios archivos XLSX.</div>
  </header>
  <main class="table-wrap" id="table-wrap"><div class="empty">No hay datos cargados.</div></main>
  <footer id="footer" class="footer">Filas: 0 · Duplicadas: 0 · CSV: 0 · Enriquecidas: 0</footer>
</div>
<div id="backdrop" class="backdrop hidden"></div>
<aside id="column-panel" class="column-panel hidden">
  <div class="panel-head"><h2>Columnas</h2><button id="close">×</button></div>
  <div id="panel-actions" class="panel-actions">
    <button id="all">MARCAR TODAS</button><button id="none">DESMARCAR TODAS</button><button id="clear">LIMPIAR FILTROS</button>
  </div>
  <div id="column-list" class="column-list"></div><section id="calculated-section" class="panel-section"><h3>CAMPOS CALCULADOS</h3><div class="calc-form"><input id="calc-name" placeholder="Nombre"><input id="calc-formula" placeholder="Fórmula"><label><input id="calc-percent" type="checkbox"> Porcentaje</label><div id="formula-tokens" class="formula-tokens"></div><div><button id="calc-save">AGREGAR</button><button id="calc-cancel" class="hidden">CANCELAR</button></div></div><div id="calc-list"></div></section><section id="subtotal-section" class="panel-section"><h3>SUBTOTALES</h3><label class="subtotal-control">Agrupar por <select id="subtotal-group"></select></label><div id="subtotal-fields"></div></section>
</aside>`

const byId = <T extends Element>(id:string) => document.getElementById(id) as unknown as T
const openBtn = byId<HTMLButtonElement>('open')
const status = byId<HTMLDivElement>('status')
const tableWrap = byId<HTMLDivElement>('table-wrap')
const footer = byId<HTMLDivElement>('footer')
const panel = byId<HTMLElement>('column-panel')
const backdrop = byId<HTMLDivElement>('backdrop')
const columnList = byId<HTMLDivElement>('column-list')
const fontInput = byId<HTMLInputElement>('font-size')
const rowHeightInput = byId<HTMLInputElement>('row-height')
const calcName = byId<HTMLInputElement>('calc-name')
const calcFormula = byId<HTMLInputElement>('calc-formula')
const calcPercent = byId<HTMLInputElement>('calc-percent')
const calcSave = byId<HTMLButtonElement>('calc-save')
const calcCancel = byId<HTMLButtonElement>('calc-cancel')
const calcList = byId<HTMLDivElement>('calc-list')
const formulaTokens = byId<HTMLDivElement>('formula-tokens')
const subtotalGroup = byId<HTMLSelectElement>('subtotal-group')
const subtotalFields = byId<HTMLDivElement>('subtotal-fields')
const calculatedSection = byId<HTMLElement>('calculated-section')
const subtotalSection = byId<HTMLElement>('subtotal-section')
const panelTitle = panel.querySelector<HTMLHeadingElement>('.panel-head h2')!

function visibleColumns(){ return state.data?.columns.filter(c => c.visible) ?? [] }
function filteredRows(){
  const rows = state.data?.rows ?? []
  const active = Object.entries(state.filters).filter(([,v]) => v.trim() !== '')
  if (!active.length) return rows
  return rows.filter(row => active.every(([id,needle]) => (row[id] ?? '').toLocaleLowerCase().includes(needle.toLocaleLowerCase())))
}

function clamp(value:number,min:number,max:number){ return Math.max(min,Math.min(max,value)) }
function columnWidth(id:string){ return clamp(Number(state.visual.columnWidths[id] ?? 140),60,600) }
function isNumericColumn(c:Column){ return c.type.toLowerCase()==='number' || c.source==='CALCULADA' }
function columnFormatKind(c:Column){ const t=state.visual.settings?.column_types?.[c.id]; if(t==='entero'||t==='decimal'||t==='porcentaje')return t; if(state.visual.settings?.column_percent?.[c.id])return 'porcentaje'; return 'decimal' }

async function persistVisualSettings(){
  if (!state.visual.settings) return
  state.visual.settings.font_size = state.visual.fontSize
  state.visual.settings.row_height = state.visual.rowHeight
  state.visual.settings.column_widths = {...state.visual.columnWidths}
  await SaveSettings(state.visual.settings)
}

function applyVisualSettings(){
  tableWrap.style.setProperty('--grid-font', `${state.visual.fontSize}px`)
  tableWrap.style.setProperty('--grid-row-h', `${state.visual.rowHeight}px`)
  fontInput.value = String(state.visual.fontSize)
  rowHeightInput.value = String(state.visual.rowHeight)
}

function render(){
  applyVisualSettings()
  const cols = visibleColumns(), rows = filteredRows()
  if (!state.data) { tableWrap.innerHTML = '<div class="empty">No hay datos cargados.</div>'; return }
  if (!cols.length) { tableWrap.innerHTML = '<div class="empty">No hay columnas visibles.</div>'; return }
  const colgroup = cols.map(c => `<col style="width:${columnWidth(c.id)}px">`).join('')
  const head = cols.map(c => `<th data-column-id="${escAttr(c.id)}"><div class="th-title">${esc(c.title)}</div><input class="filter" data-filter="${escAttr(c.id)}" value="${escAttr(state.filters[c.id] ?? '')}" placeholder="Filtrar..."></th>`).join('')
  const groupId = state.visual.settings?.subtotal_column ?? ''
  const activeSubtotal = !!groupId && !!state.data.subtotals?.length
  const displayRows = activeSubtotal ? (() => {
    const firstSeen = new Map<string,number>()
    rows.forEach((row,index) => { const key = row[groupId] ?? ''; if (!firstSeen.has(key)) firstSeen.set(key,index) })
    return [...rows].map((row,index) => ({row,index,key:row[groupId] ?? ''}))
      .sort((a,b) => (firstSeen.get(a.key)! - firstSeen.get(b.key)!) || (a.index-b.index))
  })() : rows.map((row,index) => ({row,index,key:''}))
  const body = displayRows.map(item => `<tr>${cols.map(c => `<td>${esc(item.row[c.id] ?? '')}</td>`).join('')}</tr>`).join('')
  tableWrap.innerHTML = `<table><colgroup>${colgroup}</colgroup><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`
  const tbody=tableWrap.querySelector('tbody')!
  if (activeSubtotal) {
    const subtotalByGroup = new Map((state.data.subtotals ?? []).filter(sr => !sr.total).map(sr => [sr.group_value, sr]))
    displayRows.forEach((item,index) => {
      const nextKey = displayRows[index + 1]?.key
      if (nextKey !== item.key) {
        const sr = subtotalByGroup.get(item.key)
        if (!sr) return
        const tr=document.createElement('tr'); tr.className='subtotal'
        cols.forEach((c,i)=>{const td=document.createElement('td');td.textContent=sr.values[c.id] ?? (i===0?sr.group_value:'');tr.appendChild(td)})
        const rowNodes = tbody.querySelectorAll('tr')
        rowNodes[index]?.after(tr)
      }
    })
    const total = (state.data.subtotals ?? []).find(sr => sr.total)
    if (total) {
      const tr=document.createElement('tr'); tr.className='subtotal subtotal-total'
      cols.forEach((c,i)=>{const td=document.createElement('td');td.textContent=total.values[c.id] ?? (i===0?total.group_value:'');tr.appendChild(td)})
      tbody.appendChild(tr)
    }
  }
  tableWrap.querySelectorAll<HTMLInputElement>('.filter').forEach(input => input.addEventListener('input', () => { state.filters[input.dataset.filter!] = input.value; render() }))
  footer.textContent = `Filas: ${state.data.total_rows} · Duplicadas: ${state.data.duplicated} · CSV: ${state.data.csv_rows} · Enriquecidas: ${state.data.enriched} · Mostradas: ${rows.length}`
}

function renderColumnPanel(){
  if (!state.data) { columnList.innerHTML = '<div class="empty">Importe un Excel primero.</div>'; return }
  columnList.innerHTML = state.data.columns.map((c,i) => {
    const width = columnWidth(c.id)
    return `<div class="column-item">
      <input type="checkbox" data-index="${i}" ${c.visible?'checked':''}>
      <span title="${escAttr(c.title)}">${esc(c.title)}</span>
      <small>${esc(c.source)}</small>
      <label class="column-width">Ancho <input type="number" min="60" max="600" step="10" data-width-index="${i}" value="${width}"></label>
      <div class="column-move"><button type="button" data-move="up" data-index="${i}" ${i===0?'disabled':''}>▲</button><button type="button" data-move="down" data-index="${i}" ${i===state.data!.columns.length-1?'disabled':''}>▼</button></div>
    </div>`
  }).join('')

  columnList.querySelectorAll<HTMLInputElement>('input[type=checkbox]').forEach(box => box.addEventListener('change', async () => {
    const i = Number(box.dataset.index), c = state.data!.columns[i]
    c.visible = box.checked
    render()
    try { await SetVisibleColumns(state.data!.columns.filter(x=>x.visible).map(x=>x.id)); status.textContent = 'Configuración de columnas guardada.' }
    catch (e) { status.textContent = `Error guardando columnas: ${String(e)}` }
  }))

  columnList.querySelectorAll<HTMLDivElement>('.column-item').forEach((item,i)=>{
    const c=state.data!.columns[i]; if(!isNumericColumn(c)) return
    const controls=document.createElement('span'); controls.className='column-format-controls'
    const kind=columnFormatKind(c); const dec=Number(state.visual.settings?.column_decimals?.[c.id] ?? 2)
    controls.innerHTML='<select data-format-index="'+i+'"><option value="entero">Entero</option><option value="decimal">Decimal</option><option value="porcentaje">Porcentaje</option></select><input type="number" min="0" max="8" step="1" data-decimals-index="'+i+'" value="'+dec+'">'
    item.appendChild(controls); const sel=controls.querySelector<HTMLSelectElement>('select')!; sel.value=kind
  })
  columnList.querySelectorAll<HTMLSelectElement>('select[data-format-index]').forEach(sel=>sel.addEventListener('change',async()=>{
    const i=Number(sel.dataset.formatIndex),c=state.data!.columns[i];let d=Number(columnList.querySelector<HTMLInputElement>('input[data-decimals-index="'+i+'"]')!.value)||0;if(sel.value==='entero')d=0
    try{state.data=await SetColumnFormat(c.id,d,sel.value==='porcentaje',sel.value) as Dataset;state.visual.settings.column_decimals={...(state.visual.settings.column_decimals??{}),[c.id]:d};state.visual.settings.column_percent={...(state.visual.settings.column_percent??{}),[c.id]:sel.value==='porcentaje'};state.visual.settings.column_types={...(state.visual.settings.column_types??{}),[c.id]:sel.value};render();renderColumnPanel();renderSubtotalControls()}catch(e){status.textContent='Error guardando formato: '+String(e)}
  }))
  columnList.querySelectorAll<HTMLInputElement>('input[data-decimals-index]').forEach(inp=>inp.addEventListener('change',async()=>{
    const i=Number(inp.dataset.decimalsIndex),c=state.data!.columns[i],kind=columnFormatKind(c);let d=Math.max(0,Math.min(8,Number(inp.value)||0));if(kind==='entero')d=0;inp.value=String(d)
    try{state.data=await SetColumnFormat(c.id,d,kind==='porcentaje',kind) as Dataset;state.visual.settings.column_decimals={...(state.visual.settings.column_decimals??{}),[c.id]:d};render()}catch(e){status.textContent='Error guardando decimales: '+String(e)}
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[data-width-index]').forEach(input => input.addEventListener('change', async () => {
    const i = Number(input.dataset.widthIndex), c = state.data!.columns[i]
    const width = clamp(Number(input.value) || 140,60,600)
    state.visual.columnWidths[c.id] = width
    input.value = String(width)
    render()
    try { await persistVisualSettings(); status.textContent = 'Ancho de columna guardado.' }
    catch (e) { status.textContent = `Error guardando ancho: ${String(e)}` }
  }))

  columnList.querySelectorAll<HTMLButtonElement>('button[data-move]').forEach(button => button.addEventListener('click', async () => {
    const i = Number(button.dataset.index)
    const direction = button.dataset.move === 'up' ? -1 : 1
    const target = i + direction
    if (target < 0 || target >= state.data!.columns.length) return
    const columns = state.data!.columns
    ;[columns[i], columns[target]] = [columns[target], columns[i]]
    render()
    renderColumnPanel()
    try { await SetColumnOrder(columns.map(c=>c.id)); status.textContent = 'Orden de columnas guardado.' }
    catch (e) { status.textContent = `Error guardando orden: ${String(e)}` }
  }))
}

async function refreshCalculatedList(){state.calculated.list=await ListCalculatedColumns() as CalculatedColumn[];renderCalculatedPanel()}
function renderCalculatedPanel(){formulaTokens.innerHTML=state.data?.columns.map(c=>{const token=c.source==='CSV'?'[CSV:'+c.title+']':'['+c.title+']';return '<button type="button" data-token="'+escAttr(token)+'">'+esc(token)+'</button>'}).join('')??'';formulaTokens.querySelectorAll<HTMLButtonElement>('button[data-token]').forEach(b=>b.addEventListener('click',()=>{calcFormula.value+=(calcFormula.value&&!/[-+*/( ]$/.test(calcFormula.value)?' ':'')+b.dataset.token;calcFormula.focus()}));calcList.innerHTML=state.calculated.list.map((c,i)=>'<div class="calc-row"><span><strong>'+esc(c.Name)+'</strong><small>'+esc(c.Formula)+(c.Percent?' · %':'')+'</small></span><span><button data-edit="'+i+'">Editar</button> <button data-delete="'+i+'">Eliminar</button></span></div>').join('')||'<div class="empty">No hay campos calculados.</div>';calcList.querySelectorAll<HTMLButtonElement>('button[data-edit]').forEach(b=>b.addEventListener('click',()=>{const c=state.calculated.list[Number(b.dataset.edit)];state.calculated.editingOriginal=c.Name;calcName.value=c.Name;calcFormula.value=c.Formula;calcPercent.checked=c.Percent;calcSave.textContent='GUARDAR';calcCancel.classList.remove('hidden')}));calcList.querySelectorAll<HTMLButtonElement>('button[data-delete]').forEach(b=>b.addEventListener('click',async()=>{const c=state.calculated.list[Number(b.dataset.delete)];if(!confirm('Eliminar "'+c.Name+'"?'))return;try{state.data=await DeleteCalculatedColumn(c.Name) as Dataset;await refreshCalculatedList();render();renderColumnPanel();renderSubtotalControls()}catch(e){status.textContent='Error eliminando: '+String(e)}}))}
function renderSubtotalControls(){if(!state.data)return;const current=state.visual.settings?.subtotal_column??'';subtotalGroup.innerHTML='<option value="">Sin subtotales</option>'+state.data.columns.map(c=>'<option value="'+escAttr(c.id)+'" '+(c.id===current?'selected':'')+'>'+esc(c.title)+'</option>').join('');const agg=state.visual.settings?.subtotal_agg??{};subtotalFields.innerHTML=visibleColumns().filter(isNumericColumn).map(c=>'<label class="subtotal-field">'+esc(c.title)+' <select data-subtotal-id="'+escAttr(c.id)+'"><option value="">Nada</option><option value="suma">Suma</option><option value="promedio">Promedio</option></select></label>').join('');subtotalFields.querySelectorAll<HTMLSelectElement>('select[data-subtotal-id]').forEach(s=>{s.value=agg[s.dataset.subtotalId!]??'';s.addEventListener('change',saveSubtotals)})}
async function saveSubtotals(){const agg:Record<string,string>={};subtotalFields.querySelectorAll<HTMLSelectElement>('select[data-subtotal-id]').forEach(s=>{if(s.value)agg[s.dataset.subtotalId!]=s.value});try{state.data=await SetSubtotals(subtotalGroup.value,agg) as Dataset;state.visual.settings.subtotal_column=subtotalGroup.value;state.visual.settings.subtotal_agg=agg;render()}catch(e){status.textContent='Error guardando subtotales: '+String(e)}}
function resetCalc(){state.calculated.editingOriginal='';calcName.value='';calcFormula.value='';calcPercent.checked=false;calcSave.textContent='AGREGAR';calcCancel.classList.add('hidden')}
async function setPanel(open:boolean, mode: 'columns'|'calculated' = state.panelMode){
  state.columnsOpen=open; state.panelMode=mode
  panel.classList.toggle('hidden',!open); backdrop.classList.toggle('hidden',!open)
  if(!open)return
  panelTitle.textContent=mode==='columns'?'Columnas':'Campos calculados'
  byId<HTMLDivElement>('column-list').classList.toggle('hidden',mode!=='columns')
  byId<HTMLDivElement>('panel-actions').classList.toggle('hidden',mode!=='columns')
  calculatedSection.classList.toggle('hidden',mode!=='calculated')
  subtotalSection.classList.toggle('hidden',mode!=='columns')
  if(mode==='columns'){renderColumnPanel();renderSubtotalControls()}
  else {try{await refreshCalculatedList()}catch(e){status.textContent='Error leyendo calculados: '+String(e)}}
}

fontInput.addEventListener('change', async () => {
  state.visual.fontSize = clamp(Number(fontInput.value) || 14,10,28)
  applyVisualSettings(); render()
  try { await persistVisualSettings(); status.textContent = 'Tamaño de fuente guardado.' }
  catch (e) { status.textContent = `Error guardando fuente: ${String(e)}` }
})

rowHeightInput.addEventListener('change', async () => {
  state.visual.rowHeight = clamp(Number(rowHeightInput.value) || 28,18,60)
  applyVisualSettings(); render()
  try { await persistVisualSettings(); status.textContent = 'Alto de fila guardado.' }
  catch (e) { status.textContent = `Error guardando alto de fila: ${String(e)}` }
})

openBtn.addEventListener('click', async () => {
  openBtn.disabled = true; status.textContent = 'Importando Excel...'
  try {
    const data = await ImportXLSX() as Dataset
    if (!data.columns?.length) { status.textContent = 'Importación cancelada.'; return }
    state.data = data; state.filters = {}
    status.textContent = `${data.total_rows} filas · ${data.source_files?.length ?? 0} archivo(s)`
    render()
  } catch (e) { status.textContent = `ERROR: ${String(e)}` }
  finally { openBtn.disabled = false }
})
byId<HTMLButtonElement>('columns').addEventListener('click', () => setPanel(true,'columns'))
byId<HTMLButtonElement>('calculated').addEventListener('click', () => setPanel(true,'calculated'))
byId<HTMLButtonElement>('close').addEventListener('click', () => setPanel(false))
backdrop.addEventListener('click', () => setPanel(false))
byId<HTMLButtonElement>('all').addEventListener('click', async () => {
  if (!state.data) return
  state.data.columns.forEach(c=>c.visible=true); render(); renderColumnPanel(); await SetVisibleColumns(state.data.columns.map(c=>c.id))
})
byId<HTMLButtonElement>('none').addEventListener('click', async () => {
  if (!state.data) return
  state.data.columns.forEach(c=>c.visible=false); render(); renderColumnPanel(); await SetVisibleColumns([])
})
byId<HTMLButtonElement>('clear').addEventListener('click', () => { state.filters={}; render(); status.textContent='Filtros limpiados.' })
calcSave.addEventListener('click',async()=>{try{state.data=(state.calculated.editingOriginal?await UpdateCalculatedColumn(state.calculated.editingOriginal,calcName.value,calcFormula.value,calcPercent.checked):await AddCalculatedColumn(calcName.value,calcFormula.value,calcPercent.checked)) as Dataset;resetCalc();await refreshCalculatedList();render();renderColumnPanel();renderSubtotalControls();status.textContent='Campo calculado guardado.'}catch(e){status.textContent='Error guardando: '+String(e)}})
calcCancel.addEventListener('click',resetCalc)
subtotalGroup.addEventListener('change',saveSubtotals)

void GetSettings().then((settings:any) => {
  state.visual.settings = settings
  state.visual.fontSize = clamp(Number(settings.font_size) || 14,10,28)
  state.visual.rowHeight = clamp(Number(settings.row_height) || 28,18,60)
  state.visual.columnWidths = {...(settings.column_widths ?? {})}
  state.visual.settings = {...settings,column_decimals:{...(settings.column_decimals??{})},column_percent:{...(settings.column_percent??{})},column_types:{...(settings.column_types??{})},subtotal_agg:{...(settings.subtotal_agg??{})}}
  applyVisualSettings()
  render()
}).catch(() => render())

function esc(v:string){ return v.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;').replaceAll("'",'&#39;') }
function escAttr(v:string){ return esc(v) }
