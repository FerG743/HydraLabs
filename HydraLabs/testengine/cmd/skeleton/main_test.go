package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const caseJSON = `{
 "id":"1","name":"FACT_0001_Emision","meta":{"prioridad":"High","ciclo":"Ciclo 1","precondicion":"Usuario activo."},
 "steps":[
  {"n":"01","action":"Ingresa a la liga\nUsuario:\nContraseña:\nY da clic en Tienda","expected":"Muestra pantalla"},
  {"n":"02","action":"Coloca en el campo Tienda: 0103- Interlomas","expected":"Captura el dato"},
  {"n":"03","action":"Coloca en el campo Boleta: 71","expected":"Captura el dato"},
  {"n":"04","action":"Ingresa el RFC de ejemplo XAXX010101000 en el sistema","expected":""},
  {"n":"06","action":"Coloca en el campo TR (Boleta) e ingresa\nTR: 355","expected":"Captura"},
  {"n":"05","action":"1.- Abre el reporte\n2.- Descarga el PDF","expected":"Se descarga"}
 ],
 "data":{"shared":{"tienda":"0002","rfc":"XAXX010101000"},
         "rows":[{"boleta":"47","terminal":"1"},{"boleta":"48","terminal":"1"}],
         "lists":{"notas":["a","b"]}}}`

func run(t *testing.T, m map[string][]Entry) (Result, map[string]bool) {
	var c Case
	if err := json.Unmarshal([]byte(caseJSON), &c); err != nil {
		t.Fatal(err)
	}
	r, _, used := build(&c, Opts{Map: m, Tags: []string{"prioridad", "ciclo"}})
	return r, used
}

func TestSkeleton(t *testing.T) {
	r, _ := run(t, nil)
	f := r.Feature
	for _, want := range []string{
		"# language: es", "@FACT_0001 @High @Ciclo_1",
		"Dado Usuario activo.",
		"Usuario: <usuario> Contraseña: <contraseña>", // credentials by rule
		"Tienda: 0002",                                // shared field: sheet value wins over matrix example
		"Boleta: <boleta>",                            // row field: placeholder
		"TR: <boleta>",                                // abbreviation named by a parenthetical
		"Esquema del escenario",
		"| boleta |", "| 48 |",
		"Cuando Abre el reporte\n    Cuando Descarga el PDF\n    Entonces Se descarga", // numbered list split
	} {
		if !strings.Contains(f, want) {
			t.Errorf("missing %q in:\n%s", want, f)
		}
	}
	if strings.Contains(f, "terminal") { // unused column must be dropped, not left dangling
		t.Errorf("unused column kept:\n%s", f)
	}
	if strings.Count(f, "Entonces") != 5 { // step 04 has no expected: no Entonces
		t.Errorf("want 4 Entonces:\n%s", f)
	}

	// LM layer: map a literal the automatic pass cannot see; a bad literal only warns.
	r, used := run(t, map[string][]Entry{"04": {{"rfc", "XAXX010101000"}}, "02": {{"terminal", "nope"}}})
	if !used["rfc"] || !strings.Contains(r.Feature, "RFC de ejemplo XAXX010101000 en") {
		t.Errorf("shared literal should be replaced with the sheet value:\n%s", r.Feature)
	}
	if !strings.Contains(strings.Join(r.Warnings, "|"), `literal "nope" not found`) {
		t.Errorf("bad literal should warn: %v", r.Warnings)
	}
}
