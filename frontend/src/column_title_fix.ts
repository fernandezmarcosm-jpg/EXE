// El texto del título se puede seleccionar/copiar. El drag de reordenamiento
// queda temporalmente desactivado mientras el puntero está sobre el título.
function titleCell(target:EventTarget|null):HTMLDivElement|null {
  return target instanceof Element ? target.closest<HTMLDivElement>('.th-title') : null
}

document.addEventListener('mousedown',event=>{
  const title=titleCell(event.target)
  if (!title) return
  const th=title.closest<HTMLTableCellElement>('th[data-column-id]')
  if (th) th.draggable=false
})

document.addEventListener('mouseup',event=>{
  const title=titleCell(event.target)
  const th=title?.closest<HTMLTableCellElement>('th[data-column-id]')
  if (th) th.draggable=true
})

document.addEventListener('mouseleave',()=>{
  document.querySelectorAll<HTMLTableCellElement>('th[data-column-id]').forEach(th=>th.draggable=true)
})
