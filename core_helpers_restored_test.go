package main

import "testing"

func TestParseNumberAcceptsEsARCommaDecimal(t *testing.T) {
	cases := map[string]float64{
		"1234,56": 1234.56,
		"1.234,56": 1234.56,
		"-12,5": -12.5,
		"210-": -210,
		"−210": -210,
		"-210": -210,
	}
	for raw,want:=range cases {
		got,ok:=parseNumber(raw)
		if !ok || got!=want { t.Fatalf("parseNumber(%q)=%v,%v; want %v,true",raw,got,ok,want) }
	}
}
