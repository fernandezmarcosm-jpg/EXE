import './style.css'
import { ImportXLSX, GetSettings, SetVisibleColumns } from '../../wailsjs/go/main/App'

type Column = { id:string; title:string; source:string; type:string; visible:boolean }
type Dataset = { columns:Column[]; rows:Record<string,string>[]; total_rows:number; duplicated:number; csv_rows:number; enriched:number; source_files:string[] }

const state:{data:Dataset|null; filters:Record<string,string>; columnsOpen:boolean} = {data:null, filters:{}, columnsOpen:false}

const app = document.querySelector<HTMLDivElement>('#app')!
app.innerHTML = `
<div class="shell">
  <header class="toolbar">
    <button id="open">ABRIR EXCEL</button>
    <button id="columns">COLUMNAS</button>
    <div id="status" class="status">Seleccione uno o varios archivos XLSX.</div>
  </header>
  <main class="table-wrap" id="table-wrap"><div class="empty">No hay datos cargados.</div></main>
  <footer id="footer" class="footer">Filas: 0 · Duplicadas: 0 · CSV: 0 · Enriquecidas: 0</footer>
</div>
<div id="backdrop" class="backdrop hidden"></div>
<aside id="column-panel" class="column-panel hidden">
  <div class="panel-head"><h2>Columnas</h2><button id="close">×</button></div>
  <div class="panel-actions">
    <button id="all">MARCAR TODAS</button><button id="none">DESMARCAR TODAS</button><button id="clear">LIMPIAR FILTROS</button>
  </div>
  <div id="column-list" class="column-list"></div>
</aside>`

const $ = <T extends Element>(id:string) => document.getElementById(id) as T
const openBtn = $('open') as HTMLButtonElement
const status = $('status') as HTMLDivElement
const tableWrap = $('table-wrap') as HTMLDivElement
const footer = $('footer') as HTMLDivElement
const panel = $('column-panel') as HTMLElement
const backdrop = $('backdrop') as HTMLDivElement
const columnList = $('column-list') as HTMLDivElement

function visibleColumns(){ return state.data?.columns.filter(c => c.visible) ?? [] }
function filteredRows(){
  const rows = state.data?.rows ?? []
  const active = Object.entries(state.filters).filter(([,v]) => v.trim() !== '')
  if (!active.length) return rows
  return rows.filter(row => active.every(([id,needle]) => (row[id] ?? '').toLocaleLowerCase().includes(needle.toLocaleLowerCase())))
}

function render(){
  const cols = visibleColumns(), rows = filteredRows()
  if (!state.data) { tableWrap.innerHTML = '<div class="empty">No hay datos cargados.</div>'; return }
  if (!cols.length) { tableWrap.innerHTML = '<div class="empty">No hay columnas visibles.</div>'; return }
  const head = cols.map(c => `<th><div class="th-title">${esc(c.title)}</div><input class="filter" data-filter="${escAttr(c.id)}" value="${escAttr(state.filters[c.id] ?? '')}" placeholder="Filtrar..."></th>`).join('')
  const body = rows.map(row => `<tr>${cols.map(c => `<td>${esc(row[c.id] ?? '')}</td>`).join('')}</tr>`).join('')
  tableWrap.innerHTML = `<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`
  tableWrap.querySelectorAll<HTMLInputElement>('.filter').forEach(input => input.addEventListener('input', () => { state.filters[input.dataset.filter!] = input.value; render() }))
  footer.textContent = `Filas: ${state.data.total_rows} · Duplicadas: ${state.data.duplicated} · CSV: ${state.data.csv_rows} · Enriquecidas: ${state.data.enriched} · Mostradas: ${rows.length}`
}

function renderColumnPanel(){
  if (!state.data) { columnList.innerHTML = '<div class="empty">Importe un Excel primero.</div>'; return }
  columnList.innerHTML = state.data.columns.map((c,i) => `<label class="column-item"><input type="checkbox" data-index="${i}" ${c.visible?'checked':''}><span>${esc(c.title)}</span><small>${esc(c.source)}</small></label>`).join('')
  columnList.querySelectorAll<HTMLInputElement>('input[type=checkbox]').forEach(box => box.addEventListener('change', async () => {
    const i = Number(box.dataset.index), c = state.data!.columns[i]
    c.visible = box.checked
    render()
    try { await SetVisibleColumns(state.data!.columns.filter(x=>x.visible).map(x=>x.id)); status.textContent = 'Configuración de columnas guardada.' }
    catch (e) { status.textContent = `Error guardando columnas: ${String(e)}` }
  }))
}

function setPanel(open:boolean){ state.columnsOpen=open; panel.classList.toggle('hidden',!open); backdrop.classList.toggle('hidden',!open); if(open) renderColumnPanel() }

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
$('columns').addEventListener('click', () => setPanel(true))
$('close').addEventListener('click', () => setPanel(false))
backdrop.addEventListener('click', () => setPanel(false))
$('all').addEventListener('click', async () => {
  if (!state.data) return
  state.data.columns.forEach(c => c.visible=true); render(); renderColumnPanel(); await SetVisibleColumns(state.data.columns.map(c=>c.id))
})
$('none').addEventListener('click', async () => {
  if (!state.data) return
  state.data.columns.forEach(c => c.visible=false); render(); renderColumnPanel(); await SetVisibleColumns([])
})
$('clear').addEventListener('click', () => { state.filters={}; render(); status.textContent='Filtros limpiados.' })

void GetSettings().then(() => render()).catch(() => render())

function esc(v:string){ return v.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;').replaceAll("'",'&#39;') }
function escAttr(v:string){ return esc(v) }
