package main

import (
	"strings"
	"testing"
)

const good = `# language: es
@FACT_0001 @High
Característica: Emisión

  Esquema del escenario: Emitir factura
    Dado el usuario ingresa con <usuario> y <contraseña>
    Cuando captura la boleta <boleta> en la tienda <tienda>
    Entonces el sistema muestra la pantalla
    # NEEDS-INPUT: ¿qué RFC usar?

    Ejemplos:
      | tienda | boleta |
      | 0002   | 47     |
`

func TestLint(t *testing.T) {
	if r := lint(good, Opts{}); !r.OK || len(r.NeedsInput) != 1 {
		t.Fatalf("good feature rejected: %+v", r)
	}
	cases := map[string]string{
		"language":    strings.Replace(good, "# language: es", "# language: en", 1),
		"Entonces":    strings.Replace(good, "Entonces el sistema muestra la pantalla", "Y algo", 1),
		"no column":   strings.Replace(good, "| tienda | boleta |", "| tienda | folio  |", 1),
		"credential":  strings.Replace(good, "con <usuario>", "con Contraseña: hunter2", 1),
		"parse error": "esto no es gherkin",
	}
	// new checks
	unused := strings.Replace(good, "| tienda | boleta |\n      | 0002   | 47     |", "| tienda | boleta | terminal |\n      | 0002   | 47     | 1        |", 1)
	if r := lint(unused, Opts{}); r.OK || !strings.Contains(strings.Join(r.Errors, "|"), "never used") {
		t.Errorf("unused Ejemplos column not caught: %+v", r)
	}
	if r := lint(good, Opts{MinThen: 2}); r.OK || !strings.Contains(strings.Join(r.Errors, "|"), "expected results") {
		t.Errorf("too few Entonces not caught: %+v", r)
	}
	two := good + strings.Replace(good[strings.Index(good, "  Esquema"):], "Emitir factura", "Otra", 1)
	if r := lint(two, Opts{MaxScenarios: 1}); r.OK || !strings.Contains(strings.Join(r.Errors, "|"), "2 scenarios") {
		t.Errorf("scenario cap not enforced: %+v", r)
	}
	for want, src := range cases {
		r := lint(src, Opts{})
		if r.OK || !strings.Contains(strings.Join(r.Errors, "|"), want) && want != "parse error" && want != "language" {
			t.Errorf("%s: expected an error mentioning it, got %+v", want, r)
		}
	}
}
