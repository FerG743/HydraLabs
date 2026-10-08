package main

import (
	"encoding/csv"
	"regexp"
	"strings"
	"testing"
)

// Mirrors the real layout: case-level cells only on a case's first row,
// continuation rows carry just the step.
const fixture = `NOMBRE DE CASO* [1],PRIORIDAD* [12],No. de Pasos,PASOS* [10],RESULTADOS ESPERADOS* [11],Nota
FACT_0001_Emision,High,01,El usuario ingresa a la liga,El sistema muestra Selección,
,,02,El usuario da clic en Facturación,El sistema muestra la pantalla,Puede variar
FACT_0008_Monederos,High,,,,
FACT_0011_Frontera,Low,01,Ingresa RFC,El sistema captura el dato,
FACT_0011.1_Sub,Low,01,x,y,
`

func TestParse(t *testing.T) {
	recs, _ := csv.NewReader(strings.NewReader(fixture)).ReadAll()
	cases, err := parse(recs, Config{
		Name: "NOMBRE DE CASO", Step: "PASOS", Expected: "RESULTADOS ESPERADOS",
		Num: "No. de Pasos", StepCols: []string{"Nota"}, IDRe: regexp.MustCompile(`(\d+(?:\.\d+)?)`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 4 || len(cases[0].Steps) != 2 || len(cases[1].Steps) != 0 || len(cases[2].Steps) != 1 {
		t.Fatalf("bad split: %+v", cases)
	}
	if cases[3].ID != "11.1" || cases[0].ID != "1" || cases[2].ID != "11" || cases[0].Meta["prioridad_12"] != "High" {
		t.Fatalf("bad id/meta: %+v", cases)
	}
	if cases[0].Steps[1].Extra["nota"] != "Puede variar" || cases[0].Steps[1].N != "02" {
		t.Fatalf("bad step: %+v", cases[0].Steps[1])
	}
}
