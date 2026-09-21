import { GetSettings, SaveSettings } from '../wailsjs/go/main/App'

const FONT_MARK='__visual_font_size'
const ROW_MARK='__visual_row_height'
let applying=false
let cachedSettings:any=null

function clamp(value:number,min:number,max:number){return Math.max(min,Math.min(max,value))}

function applyPersistedMinimums(settings:any){
  cachedSettings=settings
  const types={...(settings.column_types??{})}
  const font=Number(types[FONT_MARK])
  const row=Number(types[ROW_MARK])
  const fontInput=document.getElementById('font-size') as HTMLInputElement|null
  const rowInput=document.getElementById('row-height') as HTMLInputElement|null
  const wrap=document.getElementById('table-wrap')
  if(Number.isFinite(font)&&font>=6&&font<=28){
    if(fontInput) fontInput.value=String(font)
    wrap?.style.setProperty('--grid-font',`${font}px`)
  }
  if(Number.isFinite(row)&&row>=10&&row<=60){
    if(rowInput) rowInput.value=String(row)
    wrap?.style.setProperty('--grid-row-h',`${row}px`)
  }
}

async function persistVisualMinimums(){
  if(applying)return
  const fontInput=document.getElementById('font-size') as HTMLInputElement|null
  const rowInput=document.getElementById('row-height') as HTMLInputElement|null
  const font=clamp(Number(fontInput?.value)||14,6,28)
  const row=clamp(Number(rowInput?.value)||28,10,60)
  applying=true
  try{
    const settings:any=await GetSettings()
    settings.column_types={...(settings.column_types??{}),[FONT_MARK]:font,[ROW_MARK]:row}
    await SaveSettings(settings)
    applyPersistedMinimums(settings)
  }finally{applying=false}
}

document.addEventListener('change',event=>{
  const target=event.target as HTMLElement|null
  if(target?.id==='font-size'||target?.id==='row-height') void persistVisualMinimums()
})

void GetSettings().then(settings=>applyPersistedMinimums(settings)).catch(()=>{})

const wrap=document.getElementById('table-wrap')
if(wrap){
  const observer=new MutationObserver(()=>{if(cachedSettings)requestAnimationFrame(()=>applyPersistedMinimums(cachedSettings))})
  observer.observe(wrap,{childList:true,subtree:true})
}
