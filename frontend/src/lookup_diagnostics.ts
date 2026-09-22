import { EventsOn } from '../wailsjs/runtime/runtime'

type LookupSummary = { name:string; key_header:string; rows:number; matched_columns:number; enriched_rows:number; date_parse_failures:number }

function renderLookupDiagnostics(items:LookupSummary[]){
  const status=document.getElementById('status')
  if(!status||!items?.length)return
  const total=items.map(d=>{
    const base=`base ${d.name}.csv: clave=${d.key_header}, ${d.rows} filas, ${d.enriched_rows} enriquecidas`
    if(d.matched_columns===0)return `${base} · AVISO: la clave no coincide con ninguna columna XLSX`
    if(d.date_parse_failures>0)return `${base} · AVISO: ${d.date_parse_failures} fecha(s) del XLSX no se pudieron interpretar`
    if(d.enriched_rows===0)return `${base} · AVISO: la clave coincide, pero ningún valor coincide`
    return base
  }).join(' · ')
  status.textContent=`${status.textContent} · ${total}`
}

const install=()=>EventsOn('lookup-diagnostics',(items:LookupSummary[])=>renderLookupDiagnostics(items))
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',install,{once:true})
else install()
