import { strict as assert } from 'node:assert'
import { buildMonthlyPivot } from './report_pivot.ts'

const rows=[
  {FECHA:'01/01/2026',CADENA:'A',VENTA:'10',PESO:'2'},
  {FECHA:'2026-01-31 23:59:59',CADENA:'A',VENTA:'20',PESO:'1'},
  {FECHA:'01/02/2026',CADENA:'B',VENTA:'-30',PESO:'3'},
  {FECHA:'01/03/2026',CADENA:'A',VENTA:'40',PESO:'2'},
  {FECHA:'fecha inválida',CADENA:'B',VENTA:'5',PESO:'1'}
]
const suma=buildMonthlyPivot(rows,'FECHA','CADENA','VENTA','suma')
assert.deepEqual(suma.months,['2026-01','2026-02','2026-03'])
assert.equal(suma.groups.find(g=>g.label==='A')?.values['2026-01'],30)
assert.equal(suma.groups.find(g=>g.label==='B')?.values['2026-02'],-30)
assert.equal(suma.grandTotal,40)
assert.equal(suma.skippedInvalidDates,1)

const promedio=buildMonthlyPivot(rows.slice(0,2),'FECHA','CADENA','VENTA','promedio')
assert.equal(promedio.groups.find(g=>g.label==='A')?.values['2026-01'],15)

const conteo=buildMonthlyPivot(rows,'FECHA','CADENA','VENTA','conteo')
assert.equal(conteo.groups.find(g=>g.label==='A')?.values['2026-01'],2)
assert.equal(conteo.grandTotal,4)

const ponderado=buildMonthlyPivot(rows.slice(0,2),'FECHA','CADENA','VENTA','ponderado','PESO')
assert.equal(ponderado.groups.find(g=>g.label==='A')?.values['2026-01'],40/3)

const serial=buildMonthlyPivot([{FECHA:'46023',CADENA:'S',VENTA:'7'}],'FECHA','CADENA','VENTA','suma')
assert.deepEqual(serial.months,['2025-12'])
assert.equal(serial.grandTotal,7)

console.log('report pivot tests: OK')
