package main
func(v ValueType)String()string{switch v{case ValueText:return "texto";case ValueNumber:return "número";case ValueDate:return "fecha"};return "vacío"}
