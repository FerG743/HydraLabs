package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheBoardShowsAndChangesACasesDataNextToItsSteps(t *testing.T) {
	c := casesRepo(t)
	c.App, c.Out = "Shop", filepath.Join(t.TempDir(), "out")
	envFile := filepath.Join(t.TempDir(), ".hydra.env")
	old := hydraEnvPath
	hydraEnvPath = envFile
	t.Cleanup(func() { hydraEnvPath = old; os.Unsetenv("HYDRA_SHOP_5_BOLETA") })

	writeJSON(filepath.Join(c.bundleDir("SHOP-5"), "steps.json"), []StepMap{
		{N: "1", Text: "Navigate to /cobro."}, {N: "2", Text: "Enter boleta B-00456.", Column: "boleta"}, {N: "3", Text: `Click "Buscar".`}})
	if d := c.caseData("SHOP-5"); d.Delivered || !strings.Contains(d.Note, "not delivered yet") {
		t.Errorf("before delivery the values are the case's own: %+v", d)
	}
	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"boleta": {Value: "B-1"}}); err == nil {
		t.Error("nothing to save into before delivery")
	}

	csvPath := filepath.Join(c.Home, c.dataRel("SHOP-5"))
	os.MkdirAll(filepath.Dir(csvPath), 0o755)
	os.WriteFile(csvPath, []byte("caso_id,boleta\nSHOP-5,B-00456\nSHOP-9,X\n"), 0o644)
	git(c.Home, "add", "-A")
	git(c.Home, append(gitIdentity(c.Home), "commit", "-q", "-m", "delivered")...)

	d := c.caseData("SHOP-5")
	if !d.Delivered || len(d.Steps) != 3 || d.Steps[1].Column != "boleta" || d.Steps[1].Value != "B-00456" || d.Steps[0].Column != "" {
		t.Fatalf("each value next to the step that uses it: %+v", d)
	}

	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"boleta": {Value: "B-00789"}}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(csvPath); string(b) != "caso_id,boleta\nSHOP-5,B-00789\nSHOP-9,X\n" {
		t.Errorf("only this case's row changes: %q", b)
	}
	if st, _ := git(c.Home, "status", "--porcelain"); st != "" {
		t.Errorf("saved means committed: %q", st)
	}

	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"boleta": {Value: "B-00999", Env: true}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(csvPath)
	env, _ := os.ReadFile(envFile)
	if !strings.Contains(string(b), "SHOP-5,ENV:HYDRA_SHOP_5_BOLETA") || strings.Contains(string(b), "B-00999") || !strings.Contains(string(env), "HYDRA_SHOP_5_BOLETA=B-00999") {
		t.Errorf("kept out of the repo: csv %q env %q", b, env)
	}
	if d := c.caseData("SHOP-5"); d.Steps[1].Env != "HYDRA_SHOP_5_BOLETA" || !d.Steps[1].EnvSet || d.Steps[1].Value != "" {
		t.Errorf("an env value is shown by name, never by value: %+v", d.Steps[1])
	}

	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"store": {Value: "1"}}); err == nil {
		t.Error("a column the case does not use is refused")
	}
	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"boleta": {Value: "a\nb"}}); err == nil {
		t.Error("a value on two lines would break the CSV")
	}
	git(c.Home, "checkout", "-q", "-b", "hydra/delivery")
	if err := c.saveCaseData("SHOP-5", map[string]DataValue{"boleta": {Value: "B-2"}}); err == nil || !strings.Contains(err.Error(), "delivery") {
		t.Errorf("never commit onto a delivery branch: %v", err)
	}
}
