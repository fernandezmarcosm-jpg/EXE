import './style.css'
import { ImportXLSX, GetSettings, SaveSettings, SetVisibleColumns, SetColumnOrder, AddCalculatedColumn, UpdateCalculatedColumn, DeleteCalculatedColumn, ListCalculatedColumns, SetColumnFormat, SetSubtotals, SetColumnSignHighlight, SetColumnBackground, SetColumnAlign, SetColumnTitle } from '../wailsjs/go/main/App'

type Column = { id:string; title:string; source:string; type:string; visible:boolean; highlight_sign?:boolean; background?:string; align?:string }
type CalculatedColumn = { Name:string; Formula:string; Percent:boolean }
type SubtotalRow = { group_value:string; group_count:number; values:Record<string,string>; total:boolean }
type Dataset = { columns:Column[]; rows:Record<string,string>[]; total_rows:number; duplicated:number; csv_rows:number; enriched:number; source_files:string[]; subtotals:SubtotalRow[] }
type VisualState = { fontSize:number; rowHeight:number; columnWidths:Record<string,number>; settings:any }

let draggedColumnId = ''

const state:{data:Dataset|null; filters:Record<string,string>; valueFilters:Record<string,Set<string>>; columnsOpen:boolean; panelMode:'columns'|'calculated'; visual:VisualState; calculated:{editingOriginal:string;list:CalculatedColumn[]}} = {
  data:null, filters:{}, valueFilters:{}, columnsOpen:false, panelMode:'columns',
  visual:{fontSize:14,rowHeight:28,columnWidths:{},settings:null}, calculated:{editingOriginal:'',list:[]}
}

const app = document.querySelector<HTMLDivElement>('#app')!
app.innerHTML = `
<div class="shell">
  <header class="toolbar">
    <button id="open">ABRIR EXCEL</button>
    <button id="columns">COLUMNAS</button><button id="calculated">CAMPOS CALCULADOS</button>
    <label class="visual-control">Fuente <input id="font-size" type="number" min="6" max="28" step="1"></label>
    <label class="visual-control">Fila <input id="row-height" type="number" min="10" max="60" step="1"></label>
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
  const activeText = Object.entries(state.filters).filter(([,v]) => v.trim() !== '')
  const activeValues = Object.entries(state.valueFilters)
  if (!activeText.length && !activeValues.length) return rows
  return rows.filter(row => {
    if (!activeText.every(([id,needle]) => (row[id] ?? '').toLocaleLowerCase().includes(needle.toLocaleLowerCase()))) return false
    return activeValues.every(([id,allowed]) => allowed.has(row[id] ?? ''))
  })
}

function clamp(value:number,min:number,max:number){ return Math.max(min,Math.min(max,value)) }
function columnWidth(id:string){ return Math.round(clamp(Number(state.visual.columnWidths[id] ?? 140),8,600)) }
function isNumericColumn(c:Column){ return c.type.toLowerCase()==='number' || c.source==='CALCULADA' }
function isDateColumn(c:Column){ const t=String(state.visual.settings?.column_types?.[c.id]??'').toLowerCase(); return c.type.toLowerCase()==='date' || t==='fecha' || t==='date' }
function columnFormatKind(c:Column){ const t=String(state.visual.settings?.column_types?.[c.id]??'').toLowerCase(); if(t==='fecha'||t==='date')return 'fecha'; if(t==='entero'||t==='decimal'||t==='porcentaje'||t==='moneda')return t; if(state.visual.settings?.column_percent?.[c.id])return 'porcentaje'; if(state.visual.settings?.column_currency?.[c.id])return 'moneda'; return 'decimal' }
function columnThousands(c:Column){ return !!state.visual.settings?.column_thousands?.[c.id] || columnFormatKind(c)==='moneda' }
function parseCellNumber(value:string){ const s=value.trim().replace(/[%$\s]/g,'').replace(/\./g,'').replace(',', '.'); const n=Number(s); return Number.isFinite(n)?n:null }
function closeValueFilterMenu(){ document.querySelector('.value-filter-menu')?.remove() }
function openValueFilterMenu(id:string,anchor:HTMLButtonElement){
  closeValueFilterMenu()
  const values=[...new Set((state.data?.rows??[]).map(r=>r[id]??''))]
  values.sort((a,b)=>a.localeCompare(b,'es',{numeric:true,sensitivity:'base'}))
  let draft=new Set(state.valueFilters[id]??values)
  const menu=document.createElement('div'); menu.className='value-filter-menu'
  const rect=anchor.getBoundingClientRect()
  menu.style.left=Math.min(rect.left,Math.max(8,window.innerWidth-300))+'px'
  menu.style.top=Math.min(rect.bottom+4,Math.max(8,window.innerHeight-420))+'px'
  menu.innerHTML='<input class="value-filter-search" placeholder="Buscar valor..."><div class="value-filter-actions"><button type="button" data-value-action="all">Seleccionar todo</button><button type="button" data-value-action="none">Limpiar</button></div><div class="value-filter-values"></div><div class="value-filter-footer"><button type="button" data-value-action="apply">Aplicar</button></div>'
  document.body.appendChild(menu)
  const list=menu.querySelector<HTMLDivElement>('.value-filter-values')!
  const search=menu.querySelector<HTMLInputElement>('.value-filter-search')!
  const renderValues=()=>{
    const needle=search.value.toLocaleLowerCase()
    list.innerHTML=values.filter(v=>(v||'(en blanco)').toLocaleLowerCase().includes(needle)).map((v,i)=>{
      const label=v||'(en blanco)', checked=draft.has(v)
      return '<label class="value-filter-option"><input type="checkbox" data-value-index="'+i+'" '+(checked?'checked':'')+'><span>'+esc(label)+'</span></label>'
    }).join('')||'<div class="empty">Sin coincidencias.</div>'
    list.querySelectorAll<HTMLInputElement>('input[data-value-index]').forEach(cb=>cb.addEventListener('change',()=>{
      const v=values[Number(cb.dataset.valueIndex)]
      if(cb.checked)draft.add(v);else draft.delete(v)
    }))
  }
  renderValues()
  search.addEventListener('input',renderValues)
  menu.querySelector<HTMLButtonElement>('[data-value-action="all"]')!.addEventListener('click',()=>{draft=new Set(values);renderValues()})
  menu.querySelector<HTMLButtonElement>('[data-value-action="none"]')!.addEventListener('click',()=>{draft=new Set();renderValues()})
  menu.querySelector<HTMLButtonElement>('[data-value-action="apply"]')!.addEventListener('click',()=>{
    if(draft.size===values.length){delete state.valueFilters[id]}else{state.valueFilters[id]=draft}
    closeValueFilterMenu();render()
  })
}
document.addEventListener('keydown',event=>{if(event.key==='Escape')closeValueFilterMenu()})
document.addEventListener('mousedown',event=>{
  const target=event.target as Element|null
  if(target?.closest('.value-filter-menu')||target?.closest('.filter-menu-btn'))return
  closeValueFilterMenu()
})

function cellMarkup(c:Column,value:string){ const bg=c.background||state.visual.settings?.column_background?.[c.id]||''; const sign=c.highlight_sign && isNumericColumn(c) ? parseCellNumber(value) : null; const cls=sign!==null?(sign<0?'cell-neg':sign>0?'cell-pos':''):''; const align=c.align||'left'; const styles=(bg?'background:'+escAttr(bg)+';':'')+'text-align:'+align+';'; return '<td class="'+cls+'" style="'+styles+'">'+esc(value)+'</td>' }

async function persistVisualSettings(){
  if (!state.visual.settings) return
  state.visual.settings.font_size = state.visual.fontSize
  state.visual.settings.row_height = state.visual.rowHeight
  state.visual.settings.column_widths = Object.fromEntries(Object.entries(state.visual.columnWidths).map(([id,width]) => [id,Math.round(clamp(Number(width),8,600))]))
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
  const tableWidth = cols.reduce((sum, c) => sum + columnWidth(c.id), 0)
  const colgroup = cols.map(c => `<col style="width:${columnWidth(c.id)}px">`).join('')
  const head = cols.map(c => {
    const active=Object.prototype.hasOwnProperty.call(state.valueFilters,c.id)
    return `<th data-column-id="${escAttr(c.id)}" draggable="true" style="text-align:${c.align||'left'}"><div class="th-title-row"><div class="th-title">${esc(c.title)}</div><button type="button" class="filter-menu-btn${active?' filter-active':''}" draggable="false" data-value-filter="${escAttr(c.id)}" title="Filtrar por valores">▾</button></div><input class="filter" data-filter="${escAttr(c.id)}" draggable="false" value="${escAttr(state.filters[c.id] ?? '')}" placeholder="Filtrar..."><span class="col-resizer" data-resize-id="${escAttr(c.id)}" draggable="false"></span></th>`
  }).join('')
  const groupId = state.visual.settings?.subtotal_column ?? ''
  const activeSubtotal = !!groupId && !!state.data.subtotals?.length
  const displayRows = activeSubtotal ? (() => {
    const firstSeen = new Map<string,number>()
    rows.forEach((row,index) => { const key = row[groupId] ?? ''; if (!firstSeen.has(key)) firstSeen.set(key,index) })
    return [...rows].map((row,index) => ({row,index,key:row[groupId] ?? ''}))
      .sort((a,b) => (firstSeen.get(a.key)! - firstSeen.get(b.key)!) || (a.index-b.index))
  })() : rows.map((row,index) => ({row,index,key:''}))
  const subtotalByGroup = new Map((state.data.subtotals ?? []).filter(sr => !sr.total).map(sr => [sr.group_value, sr]))
  const bodyParts:string[] = []
  displayRows.forEach((item,index) => {
    bodyParts.push(`<tr>${cols.map(c => `<td>${esc(item.row[c.id] ?? '')}</td>`).join('')}</tr>`)
    if (activeSubtotal && displayRows[index + 1]?.key !== item.key) {
      const sr = subtotalByGroup.get(item.key)
      if (sr) bodyParts.push(`<tr class="subtotal">${cols.map((c,i) => `<td>${esc(sr.values[c.id] ?? (i===0?sr.group_value:''))}</td>`).join('')}</tr>`)
    }
  })
  if (activeSubtotal) {
    const total = (state.data.subtotals ?? []).find(sr => sr.total)
    if (total) bodyParts.push(`<tr class="subtotal subtotal-total">${cols.map((c,i) => { const value=c.id===groupId && total.group_count>0 ? `${total.group_value} · ${total.group_count} únicos` : (total.values[c.id] ?? (i===0?total.group_value:'')); return `<td>${esc(value)}</td>` }).join('')}</tr>`)
  }
  tableWrap.innerHTML = `<table style="width:${tableWidth}px"><colgroup>${colgroup}</colgroup><thead><tr>${head}</tr></thead><tbody>${bodyParts.join('')}</tbody></table>`
  tableWrap.querySelectorAll<HTMLInputElement>('.filter').forEach(input => {
    input.addEventListener('dragstart', event => event.stopPropagation())
    input.addEventListener('mousedown', event => event.stopPropagation())
    input.addEventListener('input', () => { state.filters[input.dataset.filter!] = input.value; renderBody() })
  })
  tableWrap.querySelectorAll<HTMLButtonElement>('.filter-menu-btn').forEach(button => {
    button.addEventListener('click',event => {
      event.preventDefault(); event.stopPropagation()
      openValueFilterMenu(button.dataset.valueFilter ?? '',button)
    })
  })
  tableWrap.querySelectorAll<HTMLSpanElement>('.col-resizer').forEach(handle => {
    const id = handle.dataset.resizeId ?? ''
    handle.addEventListener('dragstart', event => event.stopPropagation())
    handle.addEventListener('pointerdown', event => {
      event.preventDefault()
      event.stopPropagation()
      if (!id || !state.data) return
      const cols = visibleColumns()
      const index = cols.findIndex(c => c.id === id)
      const col = tableWrap.querySelectorAll<HTMLTableColElement>('col')[index]
      if (index < 0 || !col) return
      const startX = event.clientX
      const startWidth = columnWidth(id)
      let currentWidth = startWidth
      const move = (moveEvent: PointerEvent) => {
        moveEvent.preventDefault()
        currentWidth = Math.round(clamp(startWidth + (moveEvent.clientX - startX), 8, 600))
        state.visual.columnWidths[id] = currentWidth
        col.style.width = `${currentWidth}px`
        const table = tableWrap.querySelector<HTMLTableElement>('table')
        if (table) table.style.width = `${cols.reduce((sum, c) => sum + columnWidth(c.id), 0)}px`
      }
      const finish = async () => {
        window.removeEventListener('pointermove', move)
        window.removeEventListener('pointerup', finish)
        window.removeEventListener('pointercancel', finish)
        try {
          render()
          renderColumnPanel()
          await persistVisualSettings()
          status.textContent = 'Ancho de columna guardado.'
        } catch (e) {
          status.textContent = `Error guardando ancho: ${String(e)}`
        }
      }
      window.addEventListener('pointermove', move)
      window.addEventListener('pointerup', finish)
      window.addEventListener('pointercancel', finish)
    })
  })

  tableWrap.querySelectorAll<HTMLTableCellElement>('th[data-column-id][draggable="true"]').forEach(th => {
    th.addEventListener('dragstart', event => {
      draggedColumnId = th.dataset.columnId ?? ''
      th.classList.add('column-dragging')
      if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = 'move'
        event.dataTransfer.setData('text/plain', draggedColumnId)
      }
    })
    th.addEventListener('dragend', () => {
      draggedColumnId = ''
      th.classList.remove('column-dragging')
      tableWrap.querySelectorAll('th.column-drop-target').forEach(target => target.classList.remove('column-drop-target'))
    })
    th.addEventListener('dragover', event => {
      event.preventDefault()
      if (!draggedColumnId || draggedColumnId === th.dataset.columnId) return
      if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
      th.classList.add('column-drop-target')
    })
    th.addEventListener('dragleave', () => th.classList.remove('column-drop-target'))
    th.addEventListener('drop', async event => {
      event.preventDefault()
      th.classList.remove('column-drop-target')
      const sourceId = draggedColumnId || event.dataTransfer?.getData('text/plain') || ''
      const targetId = th.dataset.columnId ?? ''
      draggedColumnId = ''
      if (!state.data || !sourceId || !targetId || sourceId === targetId) return
      const columns = state.data.columns
      const sourceIndex = columns.findIndex(c => c.id === sourceId)
      const targetIndex = columns.findIndex(c => c.id === targetId)
      if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) return
      const [moved] = columns.splice(sourceIndex, 1)
      columns.splice(targetIndex, 0, moved)
      render()
      renderColumnPanel()
      try {
        await SetColumnOrder(columns.map(c => c.id))
        status.textContent = 'Orden de columnas guardado.'
      } catch (e) {
        status.textContent = `Error guardando orden: ${String(e)}`
      }
    })
  })
  renderBody()
  const subtotalColumn = state.visual.settings?.subtotal_column ?? ''
  const subtotalGroup = subtotalColumn ? state.data.columns.find(c => c.id === subtotalColumn) : undefined
  const subtotalTotal = (state.data.subtotals ?? []).find(sr => sr.total)
  const uniqueSuffix = subtotalColumn && subtotalGroup && subtotalTotal && subtotalTotal.group_count > 0 ? ` · ${subtotalGroup.title} únicos: ${subtotalTotal.group_count}` : ''
  footer.textContent = `Filas: ${state.data.total_rows} · Duplicadas: ${state.data.duplicated} · CSV: ${state.data.csv_rows} · Enriquecidas: ${state.data.enriched} · Mostradas: ${rows.length}${uniqueSuffix}`
}

function renderBody(){
  if(!state.data)return
  const cols=visibleColumns(), rows=filteredRows(), groupId=state.visual.settings?.subtotal_column??''
  const activeSubtotal=!!groupId&&!!state.data.subtotals?.length
  const displayRows=activeSubtotal?(()=>{const firstSeen=new Map<string,number>();rows.forEach((row,index)=>{const key=row[groupId]??'';if(!firstSeen.has(key))firstSeen.set(key,index)});return [...rows].map((row,index)=>({row,index,key:row[groupId]??''})).sort((a,b)=>(firstSeen.get(a.key)!-firstSeen.get(b.key)!)||(a.index-b.index))})():rows.map((row,index)=>({row,index,key:''}))
  const subtotalByGroup=new Map((state.data.subtotals??[]).filter(sr=>!sr.total).map(sr=>[sr.group_value,sr]))
  const bodyParts:string[]=[]
  displayRows.forEach((item,index)=>{bodyParts.push('<tr>'+cols.map(c=>cellMarkup(c,item.row[c.id]??'')).join('')+'</tr>');if(activeSubtotal&&displayRows[index+1]?.key!==item.key){const sr=subtotalByGroup.get(item.key);if(sr)bodyParts.push('<tr class="subtotal">'+cols.map((c,i)=>cellMarkup(c,sr.values[c.id]??(i===0?sr.group_value:'')).replace('class=""','')).join('')+'</tr>')}})
  if(activeSubtotal){const total=(state.data.subtotals??[]).find(sr=>sr.total);if(total)bodyParts.push('<tr class="subtotal subtotal-total">'+cols.map((c,i)=>{const value=c.id===groupId&&total.group_count>0?`${total.group_value} · ${total.group_count} únicos`:(total.values[c.id]??(i===0?total.group_value:''));return cellMarkup(c,value).replace('class=""','')}).join('')+'</tr>')}
  const tbody=tableWrap.querySelector('tbody');if(tbody)tbody.innerHTML=bodyParts.join('')
  const total=activeSubtotal?(state.data.subtotals??[]).find(sr=>sr.total):undefined
  const group=groupId?state.data.columns.find(c=>c.id===groupId):undefined
  const uniqueSuffix=activeSubtotal&&total&&group&&total.group_count>0?` · ${group.title} únicos: ${total.group_count}`:''
  footer.textContent='Filas: '+state.data.total_rows+' · Duplicadas: '+state.data.duplicated+' · CSV: '+state.data.csv_rows+' · Enriquecidas: '+state.data.enriched+' · Mostradas: '+rows.length+uniqueSuffix
}

function renderColumnPanel(){
  if (!state.data) { columnList.innerHTML = '<div class="empty">Importe un Excel primero.</div>'; return }
  columnList.innerHTML = state.data.columns.map((c,i) => {
    const width = columnWidth(c.id)
    return `<div class="column-item">
      <input type="checkbox" data-index="${i}" ${c.visible?'checked':''}>
      <span title="${escAttr(c.title)}">${esc(c.title)}</span>
      <label class="column-alias">Alias <input type="text" data-alias-index="${i}" value="${escAttr(c.title)}"></label>
      <small>${esc(c.source)}</small>
      <label class="column-alignment">Alineación <select data-align-index="${i}"><option value="left">Izq.</option><option value="center">Centro</option><option value="right">Der.</option></select></label>
      <label class="column-width">Ancho <input type="number" min="8" max="600" step="10" data-width-index="${i}" value="${width}"></label>
      <label class="column-background">Fondo <input type="color" data-background-index="${i}" value="${c.background||state.visual.settings?.column_background?.[c.id]||'#ffffff'}"></label>
      <div class="column-move"><button type="button" data-move="up" data-index="${i}" ${i===0?'disabled':''}>▲</button><button type="button" data-move="down" data-index="${i}" ${i===state.data!.columns.length-1?'disabled':''}>▼</button></div>
    </div>`
  }).join('')

  columnList.querySelectorAll<HTMLInputElement>('input[data-alias-index]').forEach(input => input.addEventListener('change', async () => {
    const i = Number(input.dataset.aliasIndex), col = state.data!.columns[i]
    try { state.data = await SetColumnTitle(col.id, input.value) as Dataset; render(); renderColumnPanel() }
    catch(e) { status.textContent = 'Error guardando alias: '+String(e) }
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[type=checkbox]').forEach(box => box.addEventListener('change', async () => {
    const i = Number(box.dataset.index), c = state.data!.columns[i]
    c.visible = box.checked
    render()
    try { await SetVisibleColumns(state.data!.columns.filter(x=>x.visible).map(x=>x.id)); status.textContent = 'Configuración de columnas guardada.' }
    catch (e) { status.textContent = `Error guardando columnas: ${String(e)}` }
  }))

  columnList.querySelectorAll<HTMLDivElement>('.column-item').forEach((item,i)=>{
    const c=state.data!.columns[i]; if(!isNumericColumn(c) && !isDateColumn(c)) return
    const controls=document.createElement('span'); controls.className='column-format-controls'
    const kind=columnFormatKind(c); const dec=Number(state.visual.settings?.column_decimals?.[c.id] ?? 2); const thousands=columnThousands(c); const sign=!!c.highlight_sign||!!state.visual.settings?.column_highlight_sign?.[c.id]
    controls.innerHTML='<select data-format-index="'+i+'"><option value="entero">Entero</option><option value="decimal">Decimal</option><option value="porcentaje">Porcentaje</option><option value="moneda">Moneda ($)</option><option value="fecha">Fecha</option></select>'+(!isDateColumn(c)?'<input type="number" min="0" max="8" step="1" data-decimals-index="'+i+'" value="'+dec+'"><label><input type="checkbox" data-thousands-index="'+i+'" '+(thousands?'checked':'')+'> Miles</label><label><input type="checkbox" data-sign-index="'+i+'" '+(sign?'checked':'')+'> Color +/-</label>':'')
    item.appendChild(controls); const sel=controls.querySelector<HTMLSelectElement>('select')!; sel.value=kind
  })
  columnList.querySelectorAll<HTMLSelectElement>('select[data-align-index]').forEach(sel => sel.addEventListener('change', async () => {
    const i = Number(sel.dataset.alignIndex), col = state.data!.columns[i]
    try { state.data = await SetColumnAlign(col.id, sel.value) as Dataset; render(); renderColumnPanel() }
    catch(e) { status.textContent = 'Error guardando alineación: '+String(e) }
  }))
  columnList.querySelectorAll<HTMLSelectElement>('select[data-align-index]').forEach((sel,i) => { sel.value = state.data!.columns[i].align || 'left' })

  columnList.querySelectorAll<HTMLSelectElement>('select[data-format-index]').forEach(sel=>sel.addEventListener('change',async()=>{
    const i=Number(sel.dataset.formatIndex),c=state.data!.columns[i];const decInput=columnList.querySelector<HTMLInputElement>('input[data-decimals-index="'+i+'"]');const thousandsInput=columnList.querySelector<HTMLInputElement>('input[data-thousands-index="'+i+'"]');let d=Number(decInput?.value)||0;if(sel.value==='entero'||sel.value==='fecha')d=0
    try{state.data=await SetColumnFormat(c.id,d,sel.value==='porcentaje',sel.value,!!thousandsInput?.checked) as Dataset;state.visual.settings.column_decimals={...(state.visual.settings.column_decimals??{}),[c.id]:d};state.visual.settings.column_percent={...(state.visual.settings.column_percent??{}),[c.id]:sel.value==='porcentaje'};state.visual.settings.column_types={...(state.visual.settings.column_types??{}),[c.id]:sel.value};state.visual.settings.column_currency={...(state.visual.settings.column_currency??{}),[c.id]:sel.value==='moneda'};state.visual.settings.column_thousands={...(state.visual.settings.column_thousands??{}),[c.id]:sel.value==='moneda'||!!thousandsInput?.checked};render();renderColumnPanel();renderSubtotalControls()}catch(e){status.textContent='Error guardando formato: '+String(e)}
  }))
  columnList.querySelectorAll<HTMLInputElement>('input[data-decimals-index]').forEach(inp=>inp.addEventListener('change',async()=>{
    const i=Number(inp.dataset.decimalsIndex),c=state.data!.columns[i],kind=columnFormatKind(c);let d=Math.max(0,Math.min(8,Number(inp.value)||0));if(kind==='entero')d=0;inp.value=String(d)
    try{state.data=await SetColumnFormat(c.id,d,kind==='porcentaje',kind,!!columnList.querySelector<HTMLInputElement>('input[data-thousands-index="'+i+'"]')!.checked) as Dataset;state.visual.settings.column_decimals={...(state.visual.settings.column_decimals??{}),[c.id]:d};render()}catch(e){status.textContent='Error guardando decimales: '+String(e)}
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[data-thousands-index]').forEach(inp => inp.addEventListener('change', async () => {
    const i=Number(inp.dataset.thousandsIndex), c=state.data!.columns[i], kind=columnFormatKind(c), d=kind==='entero'?0:Number(columnList.querySelector<HTMLInputElement>('input[data-decimals-index="'+i+'"]')!.value)||0
    try { state.data=await SetColumnFormat(c.id,d,kind==='porcentaje',kind,inp.checked) as Dataset; state.visual.settings.column_thousands={...(state.visual.settings.column_thousands??{}),[c.id]:inp.checked||kind==='moneda'}; render(); renderColumnPanel() }
    catch(e){ status.textContent='Error guardando separador: '+String(e) }
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[data-sign-index]').forEach(inp => inp.addEventListener('change', async () => {
    const i=Number(inp.dataset.signIndex), c=state.data!.columns[i]
    try { state.data=await SetColumnSignHighlight(c.id,inp.checked) as Dataset; state.visual.settings.column_highlight_sign={...(state.visual.settings.column_highlight_sign??{}),[c.id]:inp.checked}; render(); renderColumnPanel() }
    catch(e){ status.textContent='Error guardando color de signo: '+String(e) }
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[data-background-index]').forEach(inp => inp.addEventListener('change', async () => {
    const i=Number(inp.dataset.backgroundIndex), c=state.data!.columns[i]
    try { state.data=await SetColumnBackground(c.id,inp.value) as Dataset; state.visual.settings.column_background={...(state.visual.settings.column_background??{}),[c.id]:inp.value}; render(); renderColumnPanel() }
    catch(e){ status.textContent='Error guardando fondo: '+String(e) }
  }))

  columnList.querySelectorAll<HTMLInputElement>('input[data-width-index]').forEach(input => input.addEventListener('change', async () => {
    const i = Number(input.dataset.widthIndex), c = state.data!.columns[i]
    const width = clamp(Number(input.value) || 140,8,600)
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
function renderSubtotalControls(){if(!state.data)return;const current=state.visual.settings?.subtotal_column??'';subtotalGroup.innerHTML='<option value="">Sin subtotales</option>'+state.data.columns.map(c=>'<option value="'+escAttr(c.id)+'" '+(c.id===current?'selected':'')+'>'+esc(c.title)+'</option>').join('');const agg=state.visual.settings?.subtotal_agg??{};subtotalFields.innerHTML=state.data.columns.map(c=>{const numeric=isNumericColumn(c);const isGroup=c.id===current;return '<label class="subtotal-field">'+esc(c.title)+' <select data-subtotal-id="'+escAttr(c.id)+'"><option value="">Nada</option>'+(!isGroup&&numeric?'<option value="suma">Suma</option><option value="promedio">Promedio</option>':'')+'<option value="conteo_unico">Contador (únicos)</option></select></label>'}).join('');subtotalFields.querySelectorAll<HTMLSelectElement>('select[data-subtotal-id]').forEach(s=>{s.value=agg[s.dataset.subtotalId!]??'';s.addEventListener('change',saveSubtotals)})}
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
  state.visual.fontSize = clamp(Number(fontInput.value) || 14,6,28)
  applyVisualSettings(); render()
  try { await persistVisualSettings(); status.textContent = 'Tamaño de fuente guardado.' }
  catch (e) { status.textContent = `Error guardando fuente: ${String(e)}` }
})

rowHeightInput.addEventListener('change', async () => {
  state.visual.rowHeight = clamp(Number(rowHeightInput.value) || 28,10,60)
  applyVisualSettings(); render()
  try { await persistVisualSettings(); status.textContent = 'Alto de fila guardado.' }
  catch (e) { status.textContent = `Error guardando alto de fila: ${String(e)}` }
})

openBtn.addEventListener('click', async () => {
  openBtn.disabled = true; status.textContent = 'Importando Excel...'
  try {
    const data = await ImportXLSX() as Dataset
    if (!data.columns?.length) { status.textContent = 'Importación cancelada.'; return }
    state.data = data; state.filters = {}; state.valueFilters = {}
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
byId<HTMLButtonElement>('clear').addEventListener('click', () => { state.filters={}; state.valueFilters={}; closeValueFilterMenu(); render(); status.textContent='Filtros limpiados.' })
calcSave.addEventListener('click',async()=>{try{state.data=(state.calculated.editingOriginal?await UpdateCalculatedColumn(state.calculated.editingOriginal,calcName.value,calcFormula.value,calcPercent.checked):await AddCalculatedColumn(calcName.value,calcFormula.value,calcPercent.checked)) as Dataset;resetCalc();await refreshCalculatedList();render();renderColumnPanel();renderSubtotalControls();status.textContent='Campo calculado guardado.'}catch(e){status.textContent='Error guardando: '+String(e)}})
calcCancel.addEventListener('click',resetCalc)
subtotalGroup.addEventListener('change',saveSubtotals)

void GetSettings().then((settings:any) => {
  state.visual.settings = settings
  state.visual.fontSize = clamp(Number(settings.font_size) || 14,6,28)
  state.visual.rowHeight = clamp(Number(settings.row_height) || 28,10,60)
  state.visual.columnWidths = {...(settings.column_widths ?? {})}
  state.visual.settings = {...settings,column_decimals:{...(settings.column_decimals??{})},column_percent:{...(settings.column_percent??{})},column_types:{...(settings.column_types??{})},subtotal_agg:{...(settings.subtotal_agg??{})}}
  applyVisualSettings()
  render()
}).catch(() => render())

function esc(v:string){ return v.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;').replaceAll("'",'&#39;') }
function escAttr(v:string){ return esc(v) }
