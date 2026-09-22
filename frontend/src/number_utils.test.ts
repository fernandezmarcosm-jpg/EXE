import { parseNumber } from './number_utils'

function assertEqual(actual:number|null,expected:number,message:string){
  if(actual!==expected) throw new Error(`${message}: got ${actual}, want ${expected}`)
}

assertEqual(parseNumber('-16'),-16,'-16')
assertEqual(parseNumber('-16,00'),-16,'-16,00')
assertEqual(parseNumber('-1.234,56'),-1234.56,'-1.234,56')
assertEqual(parseNumber('($1.234,56)'),-1234.56,'accounting negative')

const cantidad=parseNumber('-16')
const kg=parseNumber('96')
assertEqual(cantidad===null||kg===null?null:cantidad*kg,-1536,'CANTIDAD*KG')

console.log('signed number tests: OK')
