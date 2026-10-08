#!/usr/bin/env python3
"""Normalized case JSON -> Gherkin feature. Deterministic: every step and expected result is
copied verbatim (only markdown backticks are dropped), in order. No model involved.

    python3 gherkin_gen.py < case.json > case.feature
"""
import json, re, sys

KW = {
    "en": dict(feature="Feature", background="Background", scenario="Scenario",
               given="Given", when="When", then="Then", and_="And"),
    "es": dict(feature="Característica", background="Antecedentes", scenario="Escenario",
               given="Dado", when="Cuando", then="Entonces", and_="Y"),
}


def plain(s):
    return re.sub(r"\s+", " ", s.replace("`", "")).strip()


def tag(s):
    return re.sub(r"[^\w.-]+", "-", plain(s)).strip("-")


def generate(case):
    k = KW[case.get("language", "en")]
    meta = case.get("meta", {})
    tags = [case["id"]] + meta.get("labels", [])
    if meta.get("priority"):
        tags.append("priority-" + meta["priority"].lower())
    if meta.get("gate"):
        tags.append("gate-" + re.split(r"[.\s]", meta["gate"].strip())[0].lower())

    out = [f"# language: {case.get('language', 'en')}", " ".join("@" + tag(t) for t in tags if tag(t)),
           f"{k['feature']}: {plain(case['name'])}", ""]
    if case.get("notes"):
        out.insert(2, f"# Note: {plain(case['notes'])}")
    if meta.get("gate"):
        out.insert(2, f"# Pipeline gate: {plain(meta['gate'])}")
    if case.get("precondition_resolved_from"):
        out.insert(2, f"# Precondition resolved from {case['precondition_resolved_from']} (\"{plain(case['precondition_original'])}\")")
    elif case.get("precondition_ref"):  # "Same as TC-1": the text is kept verbatim, the reference is surfaced
        out.insert(2, f"# NEEDS-INPUT: precondition refers to {case['precondition_ref']}; resolve it from that case")

    if plain(case.get("precondition", "")):
        out += [f"  {k['background']}:", f"    {k['given']} {plain(case['precondition'])}", ""]
    out.append(f"  {k['scenario']}: {plain(case['name'])}")

    steps = case.get("steps", [])
    if not steps:
        out.append("    # NEEDS-INPUT: the case has no steps")
    first = True
    for s in steps:  # a step with its own expected result closes with a Then and restarts the When
        out.append(f"    {k['when'] if first else k['and_']} {plain(s['action'])}")
        first = False
        if s.get("expected"):
            out.append(f"    {k['then']} {plain(s['expected'])}")
            first = True

    expected = case.get("expected", [])
    if not expected and not any(s.get("expected") for s in steps):
        out.append("    # NEEDS-INPUT: the case has no expected results")
    for i, e in enumerate(expected):
        out.append(f"    {k['then'] if i == 0 else k['and_']} {plain(e['text'])}")
    return "\n".join(out) + "\n"


if __name__ == "__main__":
    sys.stdout.write(generate(json.load(sys.stdin)))
