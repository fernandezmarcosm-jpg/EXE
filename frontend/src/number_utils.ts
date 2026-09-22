export function parseNumber(value:string):number|null {
  let s=value.trim().replace(/[%$\s]/g,'')
  if (!s) return null

  let negative=false
  if (/^\(.*\)$/.test(s)) {
    negative=true
    s=s.slice(1,-1).trim()
  }

  if (s.includes(',') && s.includes('.')) {
    if (s.lastIndexOf(',') > s.lastIndexOf('.')) s=s.replace(/\./g,'').replace(',','.')
    else s=s.replace(/,/g,'')
  } else if (s.includes(',')) {
    s=s.replace(/\./g,'').replace(',','.')
  }

  const n=Number(s)
  if (!Number.isFinite(n)) return null
  return negative ? -Math.abs(n) : n
}
