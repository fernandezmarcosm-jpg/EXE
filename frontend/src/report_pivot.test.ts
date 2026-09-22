import { strict as assert } from 'node:assert'
import { pivotReport } from './report_pivot.ts'

const rows=[
  {CADENA:'A',MES:'01',VENTA:'10',PESO:'2'},
  {CADENA:'A',MES:'02',VENTA:'20',PESO:'1'},
  {CADENA:'B',MES:'01',VENTA:'30',PESO:'3'}
]
assert.equal(pivotReport(rows,['CADENA'],'VENTA','suma').groups.find(g=>g.labels[0]==='A')?.value,30)
assert.equal(pivotReport(rows,['CADENA'],'VENTA','promedio').groups.find(g=>g.labels[0]==='A')?.value,15)
assert.equal(pivotReport(rows,['CADENA'],'VENTA','conteo').groups.find(g=>g.labels[0]==='A')?.value,2)
assert.equal(pivotReport(rows,['CADENA'],'VENTA','ponderado','PESO').groups.find(g=>g.labels[0]==='A')?.value,40/3)
assert.equal(pivotReport(rows,['CADENA','MES'],'VENTA','suma').groups.length,3)
console.log('report pivot tests: OK')
