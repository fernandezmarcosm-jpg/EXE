export type ReportAgg = 'suma'|'promedio'|'conteo'|'ponderado'

export type ReportColumn = { id:string; title:string; type:string; source:string }
export type ReportGroup = { key:string; labels:string[]; value:number; count:number }
export type ReportResult = { groups:ReportGroup[]; total:number; totalCount:number }

export function reportNumber(value:string):number|null {
  let s=value.trim().replace(/[%$\s]/g,'')
  if(!s)return null
  if(s.includes(',')&&s.includes('.')){
    if(s.lastIndexOf(',')>s.lastIndexOf('.'))s=s.replace(/\./g,'').replace(',','.')
    else s=s.replace(/,/g,'')
  } else if(s.includes(',')) s=s.replace(/\./g,'').replace(',','.')
  const n=Number(s)
  return Number.isFinite(n)?n:null
}

export function pivotReport(rows:Record<string,string>[],groupBy:string[],measureID:string,agg:ReportAgg,weightID=''):ReportResult{
  type Acc={labels:string[];sum:number;count:number;weightSum:number;weightedSum:number}
  const groups=new Map<string,Acc>()
  const values=(row:Record<string,string>,id:string)=>row[id]??''
  for(const row of rows){
    const labels=groupBy.map(id=>values(row,id))
    const key=JSON.stringify(labels)
    let acc=groups.get(key)
    if(!acc){acc={labels,sum:0,count:0,weightSum:0,weightedSum:0};groups.set(key,acc)}
    const measure=reportNumber(values(row,measureID))
    if(agg==='conteo'){acc.count++;continue}
    if(measure===null)continue
    acc.sum+=measure
    acc.count++
    if(agg==='ponderado'&&weightID){
      const weight=reportNumber(values(row,weightID))
      if(weight!==null){acc.weightedSum+=measure*weight;acc.weightSum+=weight}
    }
  }
  const makeValue=(a:Acc)=>{
    if(agg==='conteo')return a.count
    if(agg==='promedio')return a.count?a.sum/a.count:0
    if(agg==='ponderado')return a.weightSum?a.weightedSum/a.weightSum:0
    return a.sum
  }
  const groupsOut=[...groups.values()].map(a=>({key:JSON.stringify(a.labels),labels:a.labels,value:makeValue(a),count:a.count}))
  let total=0,totalCount=0,weightSum=0,weightedSum=0
  for(const row of rows){
    if(agg==='conteo'){totalCount++;continue}
    const m=reportNumber(values(row,measureID)); if(m===null)continue
    total+=m; totalCount++
    if(agg==='ponderado'&&weightID){const w=reportNumber(values(row,weightID));if(w!==null){weightedSum+=m*w;weightSum+=w}}
  }
  const totalValue=agg==='conteo'?totalCount:agg==='promedio'?(totalCount?total/totalCount:0):agg==='ponderado'?(weightSum?weightedSum/weightSum:0):total
  return {groups:groupsOut,total:totalValue,totalCount}
}
