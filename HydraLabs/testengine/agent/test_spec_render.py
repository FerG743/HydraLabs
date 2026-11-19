import copy, http.server, json, os, socketserver, sys, tempfile, threading, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira"))
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira", "support"))
import jira2case, render, spec as specmod, structure_check  # noqa: E402

NAMES = {"DWQ-130": "TC-1 – Manual order entry and processing (Happy Path)",
         "DWQ-131": "TC-2 – Bulk CSV upload using the portal's sample file (Happy Path)",
         "DWQ-132": "TC-3 – Invalid input and backend failure under duplicate submission (Negative / Resilience)"}


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def case(key):
    with open(os.path.join(HERE, "..", "tools", "jira", "fixtures", "rendered", key + ".html"), encoding="utf-8") as f:
        return jira2case.convert(f.read(), key, NAMES[key], "en", "DWQ-129", "Medium")


def golden(key):
    with open(os.path.join(HERE, "golden", key + ".spec.json"), encoding="utf-8") as f:
        return json.load(f)


class Validator(unittest.TestCase):
    def test_golden_specs_are_valid(self):
        for key in NAMES:
            self.assertEqual(specmod.validate(golden(key), case(key)), [], key)

    def test_shorthand_shapes_get_the_exact_format_back(self):
        # the real mistake: {"do": {"goto": "/x"}} instead of {"do": {"action": "goto", "path": "/x"}}
        s = copy.deepcopy(golden("DWQ-130"))
        s["steps"][0]["do"] = {"goto": "/cobro_ordenes"}
        real = next(c for c in s["checks"] if "skip" not in c)  # a check that is actually validated, not a skipped one
        real["check"] = {"visible": {"text": "x"}}
        out = "\n".join(specmod.validate(s, case("DWQ-130")))
        self.assertIn("le falta la clave \"action\" (recibí las claves ['goto'])", out)
        self.assertIn('"action": "goto", "path"', out)
        self.assertIn("recibí las claves ['visible']", out)

    def test_malformed_shapes_come_back_as_errors_not_crashes(self):
        # the shapes that crashed the validator in real runs: a list where a name goes, a null step, a string 'do'
        c = case("DWQ-130")
        cases = {
            "steps with a null item": lambda s: s.__setitem__("steps", [None] + s["steps"][1:]),
            "a step n that is a list": lambda s: s["steps"][0].update(n=[1]),
            "do as a string": lambda s: s["steps"][0].update(do="goto /x"),
            "locator as a list": lambda s: s["steps"][4]["do"].update(locator=["BOTON"]),
            "checks as a dict": lambda s: s.__setitem__("checks", {"n": 1}),
            "a check that is a string": lambda s: next(c for c in s["checks"] if "skip" not in c).update(check="visible"),
            "after_step as a list": lambda s: next(c for c in s["checks"] if "skip" not in c).update(after_step=[3]),
            "locators as a list": lambda s: s.__setitem__("locators", [["BOTON"]]),
            "a selector that is a list": lambda s: s["locators"].update(BOTON_X={"selector": ["x"]}),
            "data as a list": lambda s: s.__setitem__("data", ["order_number"]),
            "mock as a list": lambda s: s.__setitem__("mock", [{"pattern": "x"}]),
        }
        for name, mutate in cases.items():
            s = copy.deepcopy(golden("DWQ-130"))
            mutate(s)
            try:
                out = specmod.validate(s, c)
            except Exception as e:  # noqa: BLE001 - the point of the test
                self.fail(f"{name}: validate raised {type(e).__name__}: {e}")
            self.assertTrue(out, f"{name}: a malformed spec must come back with errors")

    def test_wrong_value_types_say_what_the_field_needs(self):
        c = case("DWQ-130")
        def broken(mutate):
            s = copy.deepcopy(golden("DWQ-130"))
            mutate(s)
            return "\n".join(specmod.validate(s, c))
        # the real mistakes: from_download sent as true, data sent as a list
        self.assertIn("\"from_download\" debe ser el nombre", broken(lambda s: s["steps"][4]["do"].update(action="upload_file", from_download=True, locator="BOTON_PROCESAR")))
        self.assertIn("\"data\" debe ser el NOMBRE", broken(lambda s: next(x["do"] for x in s["steps"] if x.get("do", {}).get("action") == "fill").update(data=["LIVR"])))

    def test_a_save_as_that_is_not_a_python_name_is_refused_before_rendering(self):
        # the real crash: save_as "ejemplo-ordenes.csv" became a variable and the generated module did not compile
        c = case("DWQ-131")
        s = copy.deepcopy(golden("DWQ-131"))
        dl = next(x["do"] for x in s["steps"] if x.get("do", {}).get("action") == "download")
        dl["save_as"] = "ejemplo-ordenes.csv"
        out = "\n".join(specmod.validate(s, c))
        self.assertIn("save_as debe ser un nombre de variable", out)

    def test_unknown_root_keys_are_dropped_and_reported(self):
        s = copy.deepcopy(golden("DWQ-130"))
        s["currentPage"] = "cobro_ordenes"
        notes = specmod.normalize(s)
        self.assertNotIn("currentPage", s)
        self.assertTrue(any("currentPage" in n for n in notes), notes)

    def test_a_select_with_values_becomes_its_data_key(self):
        s = copy.deepcopy(golden("DWQ-131"))
        s.setdefault("data", {})["store"] = "LIVR"
        s["steps"].append({"n": 99, "do": {"action": "select", "locator": "SELECT_TIENDA", "values": ["LIVR"]}})
        s["locators"]["SELECT_TIENDA"] = "#store"
        notes = specmod.normalize(s)
        step = next(x for x in s["steps"] if x["n"] == 99)
        self.assertEqual(step["do"].get("data"), "store")
        self.assertNotIn("values", step["do"])
        self.assertTrue(notes)

    def test_a_wrong_repo_locator_is_refused_unless_its_replacement_was_verified(self):
        s = copy.deepcopy(golden("DWQ-130"))
        name = specmod.used_locators(s)[0]
        s["locators"][name] = "role=textbox[name=\"Nuevo\"]"
        c = case("DWQ-130")
        repo = tempfile.mkdtemp()
        framework(repo)
        loc = os.path.join(repo, "apps", "PortalQA", "locators", f"{s['page']}_locators.py")
        os.makedirs(os.path.dirname(loc), exist_ok=True)
        with open(loc, "w", encoding="utf-8") as f:
            f.write(f"class X:\n    {name} = 'input[name=\"viejo\"]'\n")
        import render
        refused = render.render(repo, "PortalQA", c, s)
        self.assertTrue(any("ya existe con otro valor" in e for e in refused["errors"]), refused)
        fixed = render.render(repo, "PortalQA", c, s, replace=frozenset({name}))
        self.assertEqual(fixed["errors"], [])
        self.assertTrue(any("verificado en la página" in w for w in fixed["warnings"]), fixed["warnings"])
        self.assertIn("Nuevo", read(loc))

    def test_snapshot_refs_are_not_selectors(self):
        s = copy.deepcopy(golden("DWQ-130"))
        s["locators"]["BOTON_X"] = 'option[ref="e195"]'
        out = "\n".join(specmod.validate(s, case("DWQ-130")))
        self.assertIn("referencia del snapshot", out)

    def test_names_the_repo_already_has_are_declared_from_it(self):
        s = copy.deepcopy(golden("DWQ-130"))
        name = specmod.used_locators(s)[0]
        s["locators"].pop(name, None)
        existing = {name.split("__", 1)[-1]: ("role=button[name=\"Procesar\"]", False)}
        self.assertEqual(specmod.resolve(s, existing), [name])
        self.assertEqual(s["locators"][name], "role=button[name=\"Procesar\"]")
        self.assertNotIn(name, "\n".join(specmod.validate(s, case("DWQ-130"))))

    def test_a_near_miss_name_gets_a_did_you_mean(self):
        s = copy.deepcopy(golden("DWQ-130"))
        name = specmod.used_locators(s)[0]
        s["locators"].pop(name, None)
        s["locators"][name + "S"] = "#x"  # declared, but the step says the name without the S
        out = "\n".join(specmod.validate(s, case("DWQ-130")))
        self.assertIn(f"¿quisiste decir {name}S?", out)

    def test_a_spec_wrapped_in_pages_says_where_the_fields_go(self):
        # the real mistake of a small model: page and locators nested under "pages"
        s = copy.deepcopy(golden("DWQ-130"))
        s["pages"] = {"page": s.pop("page"), "locators": s.pop("locators")}
        out = "\n".join(specmod.validate(s, case("DWQ-130")))
        self.assertIn("no lleva 'pages'", out)
        self.assertIn("en la raíz del objeto", out)

    def test_bad_specs_are_rejected_with_actionable_errors(self):
        c = case("DWQ-130")
        def broken(mutate):
            s = copy.deepcopy(golden("DWQ-130"))
            mutate(s)
            return "\n".join(specmod.validate(s, c))
        self.assertIn("acción 'teleport' inválida", broken(lambda s: s["steps"][0]["do"].update(action="teleport")))
        self.assertIn("no está declarado en locators", broken(lambda s: s["steps"][4]["do"].update(locator="BOTON_FANTASMA")))
        self.assertIn("no inventes valores", broken(lambda s: s["data"].update(order_number="1234567890")))   # not in the step text
        self.assertIn("falta el paso 6", broken(lambda s: s["steps"].pop()))
        self.assertIn("falta el resultado esperado 3", broken(lambda s: s["checks"].pop()))
        self.assertIn("XPath absoluto", broken(lambda s: s["locators"].update(BOTON_X="/html/body/div[2]/button")))
        self.assertIn("requiere 'mock'", broken(lambda s: s["checks"].__setitem__(2, {"n": 3, "check": {"kind": "max_requests", "n": 1}})))
        self.assertIn("argumentos no permitidos", broken(lambda s: s["steps"][4]["do"].update(color="rojo")))
        self.assertIn("MAYÚSCULAS", broken(lambda s: s["locators"].update(boton="#b")))


def framework(repo):
    for rel, text in {"config/__init__.py": "", "utils/__init__.py": "", "config/settings.py": "class Config:\n    BASE_URL = ''\n",
                      "utils/templete_report.py": "def set_business_mode(v): pass\ndef add_step(msg, status=None, level=None): pass\n",
                      "utils/csv_reader.py": "import csv\ndef cargar_datos_csv(p):\n    with open(p, newline='', encoding='utf-8') as f:\n        return [tuple(r) for r in list(csv.reader(f))[1:]]\n",
                      "utils/driver_wrapper.py": "class DriverWrapper:\n    def __init__(self, d): self.page = d.page\n    def click(self, s, description=None): self.page.click(s)\n    def send_keys(self, s, t, description=None, is_secret=False): self.page.fill(s, t)\n",
                      "conftest.py": "import pytest\n@pytest.fixture\ndef driver():\n    raise RuntimeError('unused')\n"}.items():
        p = os.path.join(repo, rel)
        os.makedirs(os.path.dirname(p), exist_ok=True)
        with open(p, "w") as f:
            f.write(text)


class Renderer(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.repo = self.tmp.name
        framework(self.repo)
        render.scaffold(self.repo, "Portal")

    def tearDown(self):
        self.tmp.cleanup()

    def render_all(self):
        out = {}
        for key in NAMES:
            out[key] = render.render(self.repo, "Portal", case(key), golden(key))
            self.assertEqual(out[key]["errors"], [], key)
        return out

    def test_all_three_cases_render_readme_clean_and_importable(self):
        self.render_all()
        self.assertEqual(structure_check.check(self.repo, "Portal"), [])      # README rules hold by construction
        import runner
        self.assertEqual(runner.run_checks(self.repo, "Portal"), [])          # compiles and pytest can collect it

    def test_cases_share_one_locators_class_and_module_without_duplicates(self):
        self.render_all()
        loc = read(os.path.join(self.repo, "apps/Portal/locators/cobro_ordenes_locators.py"))
        self.assertEqual(loc.count("INPUT_ARCHIVO ="), 1)                       # declared by TC-2 and TC-3, kept once
        self.assertEqual(loc.count("BOTON_SUBIR_Y_PROCESAR ="), 1)
        self.assertIn("# dinámico", loc)                                          # toast / spinner flagged
        mod = read(os.path.join(self.repo, "apps/Portal/modules/cobro_ordenes_module.py"))
        self.assertEqual(mod.count("def navegar_a_cobro_ordenes_flujo"), 1)
        self.assertIn('get_by_role("option", name=valor, exact=True)', mod)     # combobox flow
        self.assertTrue(os.path.exists(os.path.join(self.repo, "utils/network_mock.py")))   # generic helpers copied when needed
        self.assertTrue(os.path.exists(os.path.join(self.repo, "utils/test_files.py")))

    def test_step_texts_come_from_the_case_verbatim(self):
        self.render_all()
        t = read(os.path.join(self.repo, "tests/Portal/test_dwq_132.py"))
        self.assertIn("Upload evil.csv.exe (binary content, application/octet-stream).", t)
        self.assertIn("'; DROP TABLE x;--", t)
        self.assertIn("'9' * 10000", t)
        self.assertIn("mock = co.mockear_peticion_post_flujo(driver, 500, 1500)", t)

    def test_expected_results_and_data_files(self):
        self.render_all()
        import csv
        with open(os.path.join(self.repo, "apps/Portal/data/expected_result.csv"), encoding="utf-8", newline="") as f:
            rows = list(csv.reader(f))
        self.assertEqual(rows[0], ["test_id", "expected_result"])
        self.assertEqual(sorted(r[0] for r in rows[1:]), ["DWQ-130", "DWQ-131", "DWQ-132"])
        with open(os.path.join(self.repo, "apps/Portal/data/dwq_130_data.csv"), encoding="utf-8") as f:
            self.assertEqual(f.read(), "caso_id,order_number,store\nDWQ-130,9800074297,LIVR\n")

    def test_rerendering_is_idempotent_and_conflicts_are_refused(self):
        self.render_all()
        before = read(os.path.join(self.repo, "apps/Portal/locators/cobro_ordenes_locators.py"))
        self.render_all()
        self.assertEqual(before, read(os.path.join(self.repo, "apps/Portal/locators/cobro_ordenes_locators.py")))
        clash = golden("DWQ-130")
        clash["locators"]["BOTON_AGREGAR"] = "#otro"
        res = render.render(self.repo, "Portal", case("DWQ-130"), clash)
        self.assertTrue(any("ya existe con otro valor" in e for e in res["errors"]), res)

    def test_skipped_steps_make_the_test_skip_honestly(self):
        s = golden("DWQ-130")
        s["steps"][5] = {"n": 6, "skip": "no se pudo ubicar el botón"}
        render.render(self.repo, "Portal", case("DWQ-130"), s)
        t = read(os.path.join(self.repo, "tests/Portal/test_dwq_130.py"))
        self.assertIn("pytest.skip('Pasos sin automatizar: 6:", t)
        self.assertIn("# PENDIENTE: no se pudo ubicar el botón", t)

    def test_scaffold_warns_when_the_suite_is_not_registered(self):
        with open(os.path.join(self.repo, "runner_central.py"), "w") as f:
            f.write("choices=['Portal', 'all']\n")
        self.assertEqual(render.scaffold(self.repo, "Portal"), [])
        self.assertTrue(render.scaffold(self.repo, "OtraApp")[0].startswith("registra la suite 'OtraApp'"))


# ---- end to end in a real browser, three behaviours of a fake portal -------------------------------------
PAGE = """<!doctype html><title>Cobro</title><h1>Cobro de ordenes</h1>
<input type=file id=file-upload><button disabled id=go>Subir y Procesar</button>
<ol><li role=status id=toast hidden>Error interno del servidor</li></ol><div class=animate-spin id=spin hidden></div>
<script>
const f = document.getElementById('file-upload'), go = document.getElementById('go'); let pending = false;
f.addEventListener('change', () => { const x = f.files[0]; go.disabled = !(x && x.name.toLowerCase().endsWith('.csv')); });
go.addEventListener('click', async () => {
  if (%(guard)s && pending) return;
  pending = true; if (%(guard)s) go.disabled = true; spin.hidden = false;
  try { const r = await fetch('/uploadFile', {method: 'POST', body: f.files[0]}); if (!r.ok) throw new Error(); }
  catch (e) { toast.hidden = false; }
  finally { pending = false; spin.hidden = true; if (%(reenable)s) go.disabled = false; else { f.value = ''; go.disabled = true; } }
});
</script>"""


class Portal(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        variant = self.path.split("/")[1]
        body = (PAGE % {"guard": "false" if variant == "B" else "true", "reenable": "false" if variant == "C" else "true"}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


class EndToEnd(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            from playwright.sync_api import sync_playwright  # noqa: F401
        except ImportError:
            raise unittest.SkipTest("playwright not installed")
        cls.tmp = tempfile.TemporaryDirectory()
        cls.repo = cls.tmp.name
        framework(cls.repo)
        render.scaffold(cls.repo, "Portal")
        for key in NAMES:
            render.render(cls.repo, "Portal", case(key), golden(key))
        cls.srv = socketserver.TCPServer(("127.0.0.1", 0), Portal)
        threading.Thread(target=cls.srv.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.srv.shutdown()
        cls.srv.server_close()
        cls.tmp.cleanup()

    def run_tc3(self, variant):
        import harness
        # tests/ lives under repo/tests/Portal; the harness copies the tree and stubs the framework around it
        return harness.run(self.repo, "dwq_132", f"http://127.0.0.1:{self.srv.server_address[1]}/{variant}")

    def test_portal_with_guard_and_reenable_passes(self):
        r = self.run_tc3("A")
        self.assertEqual((r["status"], r["error"]), ("passed", None), r)

    def test_portal_without_the_guard_fails_the_double_charge_check(self):
        r = self.run_tc3("B")
        self.assertEqual(r["status"], "failed")
        self.assertIn("posible doble cobro", r["error"])

    def test_portal_that_stays_disabled_fails_only_the_enabled_check_like_the_real_one(self):
        r = self.run_tc3("C")
        self.assertEqual(r["status"], "failed")
        self.assertIn("debería estar enabled", r["error"])
        self.assertEqual(r["aborted_writes"], [])   # the mocked POST never reached the (fake) server


if __name__ == "__main__":
    unittest.main()
