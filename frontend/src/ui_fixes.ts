import { GetSettings, SaveSettings } from '../wailsjs/go/main/App'

type Settings = {
  subtotal_column?: string
  subtotal_agg?: Record<string,string>
  column_decimals?: Record<string,number>
  column_types?: Record<string,string>
  column_percent?: Record<string,boolean>
  column_currency?: Record<string,boolean>
  column_thousands?: Record<string,boolean>
  decimals?: number
}

let settings: Settings = {}
let lastSignature = ''
let refreshing = false
let cachedVisualSettings:any = null
let savingVisualSettings = false

function parseNumber(value:string):number|null {
  let s=value.trim().replace(/[%$\s]/g,'')
  if (!s) return null
  if (s.includes(',') && s.includes('.')) {
    if (s.lastIndexOf(',') > s.lastIndexOf('.')) s=s.replace(/\./g,'').replace(',','.')
    else s=s.replace(/,/g,'')
  } else if (s.includes(',')) {
    s=s.replace(/\./g,'').replace(',','.')
  }
  const n=Number(s)
  return Number.isFinite(n)?n:null
}

function formatNumber(columnId:string,value:number):string {
  const kind=settings.column_types?.[columnId] ?? (settings.column_percent?.[columnId]?'porcentaje':settings.column_currency?.[columnId]?'moneda':'decimal')
  const decimals=kind==='entero'?0:Number(settings.column_decimals?.[columnId] ?? settings.decimals ?? 2)
  const number=kind==='porcentaje'?value*100:value
  let text=number.toLocaleString('es-AR',{minimumFractionDigits:decimals,maximumFractionDigits:decimals})
  if (kind==='moneda') text='$'+text
  if (kind==='porcentaje') text+='%'
  return text
}

function refreshFilteredSubtotals() {
  if (refreshing) return
  const wrap=document.getElementById('table-wrap')
  const table=wrap?.querySelector('table')
  if (!wrap || !table) return
  const groupId=settings.subtotal_column?.trim() ?? ''
  const agg=settings.subtotal_agg ?? {}
  if (!groupId || !Object.keys(agg).length) return

  const headers=Array.from(table.querySelectorAll<HTMLTableCellElement>('thead th')).map(th=>({id:th.dataset.columnId??'',title:th.querySelector('.th-title')?.textContent??''}))
  if (!headers.some(h=>h.id===groupId)) return

  const dataRows=Array.from(table.querySelectorAll<HTMLTableSectionElement>('tbody tr')).filter(tr=>!tr.classList.contains('subtotal'))
  const rows=dataRows.map(tr=>{
    const cells=Array.from(tr.cells)
    const values:Record<string,string>={}
    headers.forEach((h,i)=>{values[h.id]=cells[i]?.textContent??''})
    return values
  })
  const signature=groupId+'|'+JSON.stringify(agg)+'|'+rows.map(r=>headers.map(h=>r[h.id]??'')).join('\u001e')
  if (signature===lastSignature) return
  lastSignature=signature

  const groups=new Map<string,{rows:Record<string,string>[]}>()
  const uniqueGroups=new Set<string>()
  rows.forEach(row=>{
    const key=row[groupId]??''
    if (!groups.has(key)) groups.set(key,{rows:[]})
    groups.get(key)!.rows.push(row)
    if (key.trim()!=='') uniqueGroups.add(key.trim())
  })

  const subtotalRows:HTMLTableRowElement[]=[]
  const makeSubtotal=(groupValue:string,groupRows:Record<string,string>[],total=false)=>{
    const tr=document.createElement('tr'); tr.className='subtotal'+(total?' subtotal-total':'')
    headers.forEach(h=>{
      const td=document.createElement('td')
      let value=''
      if (h.id===groupId) value=total ? `TOTAL GENERAL · ${uniqueGroups.size} únicos` : groupValue
      else {
        const operation=agg[h.id]
        if (operation==='conteo_unico') {
          const seen=new Set(groupRows.map(r=>(r[h.id]??'').trim()).filter(Boolean))
          value=String(seen.size)
        } else if (operation==='suma' || operation==='promedio') {
          const numbers=groupRows.map(r=>parseNumber(r[h.id]??'')).filter((n):n is number=>n!==null)
          if (numbers.length) {
            const sum=numbers.reduce((a,b)=>a+b,0)
            value=formatNumber(h.id,operation==='promedio'?sum/numbers.length:sum)
          }
        }
      }
      td.textContent=value
      tr.appendChild(td)
    })
    return tr
  }

  groups.forEach((group,groupValue)=>subtotalRows.push(makeSubtotal(groupValue,group.rows)))
  subtotalRows.push(makeSubtotal('TOTAL GENERAL',rows,true))

  refreshing=true
  const tbody=table.querySelector('tbody')!
  tbody.querySelectorAll('tr.subtotal').forEach(tr=>tr.remove())
  const dataTrs=Array.from(tbody.querySelectorAll<HTMLTableRowElement>('tr'))
  const byGroup=new Map<string,HTMLTableRowElement[]>()
  dataTrs.forEach(tr=>{
    const cells=Array.from(tr.cells)
    const groupIndex=headers.findIndex(h=>h.id===groupId)
    const key=cells[groupIndex]?.textContent??''
    if(!byGroup.has(key)) byGroup.set(key,[])
    byGroup.get(key)!.push(tr)
  })
  const groupSubtotals=subtotalRows.slice(0,-1)
  groupSubtotals.forEach((sr,i)=>{
    const key=Array.from(groups.keys())[i]
    const last=byGroup.get(key)?.at(-1)
    if(last) last.after(sr)
  })
  tbody.appendChild(subtotalRows.at(-1)!)
  refreshing=false

  const footer=document.getElementById('footer')
  if (footer) {
    const groupHeader=headers.find(h=>h.id===groupId)
    const base=footer.textContent?.replace(/ · Mostradas:.*$/,'') ?? ''
    footer.textContent=`${base} · Mostradas: ${rows.length} · ${groupHeader?.title||groupId} únicos: ${uniqueGroups.size}`
  }
}

function applyVisualSettings(s:any){
  cachedVisualSettings=s
  const types={...(s.column_types??{})}
  const font=Number(types.__visual_font_size)
  const row=Number(types.__visual_row_height)
  const wrap=document.getElementById('table-wrap')
  const fontInput=document.getElementById('font-size') as HTMLInputElement|null
  const rowInput=document.getElementById('row-height') as HTMLInputElement|null
  if(Number.isFinite(font)&&font>=6&&font<=28){if(fontInput)fontInput.value=String(font);wrap?.style.setProperty('--grid-font',`${font}px`)}
  if(Number.isFinite(row)&&row>=10&&row<=60){if(rowInput)rowInput.value=String(row);wrap?.style.setProperty('--grid-row-h',`${row}px`)}
}

async function saveVisualMinimums(){
  if(savingVisualSettings)return
  const fontInput=document.getElementById('font-size') as HTMLInputElement|null
  const rowInput=document.getElementById('row-height') as HTMLInputElement|null
  const font=Math.max(6,Math.min(28,Number(fontInput?.value)||14))
  const row=Math.max(10,Math.min(60,Number(rowInput?.value)||28))
  savingVisualSettings=true
  try{
    const s:any=await GetSettings()
    s.column_types={...(s.column_types??{}),__visual_font_size:font,__visual_row_height:row}
    await SaveSettings(s)
    applyVisualSettings(s)
  }finally{savingVisualSettings=false}
}

void GetSettings().then(s=>{settings=s as Settings;cachedVisualSettings=s;applyVisualSettings(s);refreshFilteredSubtotals()}).catch(()=>{})

document.addEventListener('change',event=>{
  const target=event.target as HTMLElement|null
  if(target?.id==='subtotal-group' || target?.matches('select[data-subtotal-id]')) {
    void GetSettings().then(s=>{settings=s as Settings;lastSignature='';refreshFilteredSubtotals()}).catch(()=>{})
  }
  if(target?.id==='font-size' || target?.id==='row-height') void saveVisualMinimums()
})

document.addEventListener('input',event=>{
  if ((event.target as HTMLElement|null)?.matches('input.filter')) requestAnimationFrame(refreshFilteredSubtotals)
})

const wrap=document.getElementById('table-wrap')
if (wrap) {
  const observer=new MutationObserver(()=>requestAnimationFrame(()=>{refreshFilteredSubtotals();if(cachedVisualSettings)applyVisualSettings(cachedVisualSettings)}))
  observer.observe(wrap,{childList:true,subtree:true})
}
