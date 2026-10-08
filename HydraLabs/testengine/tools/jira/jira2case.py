#!/usr/bin/env python3
"""Jira issue description (HTML) -> normalized case JSON. Deterministic, stdlib only.

The description is a bullet list whose items are "Label: value" sections; Steps and
Expected Result carry a nested list. Section names are synonyms, so other boards
(Spanish, different wording) only need entries in SECTIONS, not new code.

    python3 jira2case.py --key DWQ-130 --summary "TC-1 - ..." [--lang en] [--parent DWQ-129]
                         [--jira-priority Medium] < description.html
"""
import argparse, json, re, sys
from html.parser import HTMLParser

SECTIONS = {  # label (lowercase) -> field
    "preconditions": "precondition", "precondition": "precondition", "precondiciones": "precondition",
    "precondición": "precondition", "precondicion": "precondition",
    "steps": "steps", "pasos": "steps",
    "expected result": "expected", "expected results": "expected",
    "resultado esperado": "expected", "resultados esperados": "expected",
    "pipeline gate": "gate", "gate": "gate",
    "note": "notes", "notes": "notes", "nota": "notes", "notas": "notes",
    "type": "_meta", "tipo": "_meta",
}
VOID = {"br", "hr", "img"}


class Node:
    def __init__(self, tag, attrs=()):
        self.tag, self.attrs, self.kids = tag, dict(attrs), []


class Tree(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.root = Node("root")
        self.stack = [self.root]

    def handle_starttag(self, tag, attrs):
        n = Node(tag, attrs)
        self.stack[-1].kids.append(n)
        if tag not in VOID:
            self.stack.append(n)

    def handle_endtag(self, tag):
        for i in range(len(self.stack) - 1, 0, -1):
            if self.stack[i].tag == tag:
                del self.stack[i:]
                break

    def handle_data(self, d):
        self.stack[-1].kids.append(d)


def text(n):
    """Inline text of a node, skipping nested lists; <code> keeps its backticks (value anchors)."""
    if isinstance(n, str):
        return n
    if n.tag == "br":
        return " "
    if n.tag in ("ul", "ol"):
        return ""
    if "lozenge" in n.attrs.get("class", ""):  # status label Jira appends to auto-linked issue keys ("Backlog")
        return ""
    inner = "".join(text(k) for k in n.kids)
    return f"`{inner}`" if n.tag in ("code", "tt") else inner  # connector HTML uses <code>, Jira's renderer <tt>


def clean(s):
    return re.sub(r"\s+([.,;:!?])(?=\s|$)", r"\1", re.sub(r"\s+", " ", s)).strip()  # Jira's link markup leaves "TC-1 ."


def sublists(li):
    return [k for k in li.kids if not isinstance(k, str) and k.tag in ("ul", "ol")]


def item(li):
    """One list item: its own text, plus any nested bullets appended as '; '-joined text."""
    base = clean(text(li))
    subs = [item(k) for lst in sublists(li) for k in lst.kids if not isinstance(k, str) and k.tag == "li"]
    return f"{base} {'; '.join(subs)}".strip() if subs else base


def convert(html, key, summary, lang="en", parent=None, jira_priority=None):
    tree = Tree()
    tree.feed(html)
    case = {"id": key, "name": summary, "language": lang,
            "meta": {"parent": parent} if parent else {}, "precondition": "", "steps": [], "expected": []}
    warnings, extra = [], {}

    def walk(n):  # yield every top-level <li> of every outermost <ul>
        for k in n.kids:
            if isinstance(k, str):
                continue
            if k.tag == "ul":
                yield from (x for x in k.kids if not isinstance(x, str) and x.tag == "li")
            else:
                yield from walk(k)

    for li in walk(tree.root):
        raw = clean(text(li))
        m = re.match(r"^([^:|]{1,40}):\s*(.*)$", raw)
        if not m:
            warnings.append(f"unlabeled item ignored: {raw[:60]}")
            continue
        label, value = m.group(1).strip(), m.group(2).strip()
        field = SECTIONS.get(label.lower())
        items = [item(k) for lst in sublists(li) for k in lst.kids if not isinstance(k, str) and k.tag == "li"]
        if field == "_meta":  # "Type: Test Case | Priority: High | Labels: `e2e`, `smoke`"
            for part in raw.split("|"):
                k, _, v = part.partition(":")
                k, v = k.strip().lower(), v.strip().replace("`", "")
                if k in ("type", "tipo"):
                    case["meta"]["type"] = v
                elif k in ("priority", "prioridad"):
                    case["meta"]["priority"] = v
                elif k in ("labels", "etiquetas"):
                    case["meta"]["labels"] = [x.strip() for x in v.split(",") if x.strip()]
        elif field == "steps":
            case["steps"] = [{"n": str(i + 1), "action": t} for i, t in enumerate(items or ([value] if value else []))]
        elif field == "expected":
            case["expected"] = [{"text": t} for t in (items or ([value] if value else []))]
        elif field == "precondition":
            case["precondition"] = clean(f"{value} {' '.join(items)}")
            ref = re.match(r"(?i)^(same as|igual que|mismas? de)\s+([\w-]+)", case["precondition"])
            if ref:
                case["precondition_ref"] = ref.group(2).rstrip(".")
                warnings.append(f"precondition refers to {case['precondition_ref']}: resolve it from that case")
        elif field == "gate":
            case["meta"]["gate"] = value
        elif field == "notes":
            case["notes"] = value
        else:
            extra[label] = value
            warnings.append(f"unknown section kept in 'extra': {label}")

    if extra:
        case["extra"] = extra
    if not case["steps"]:
        warnings.append("no steps found")
    if not case["expected"]:
        warnings.append("no expected results found")
    if jira_priority and case["meta"].get("priority") and case["meta"]["priority"] != jira_priority:
        warnings.append(f"priority conflict: description says {case['meta']['priority']}, Jira field says {jira_priority}")
    case["warnings"] = warnings
    return case


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--key", required=True)
    ap.add_argument("--summary", required=True)
    ap.add_argument("--lang", default="en")
    ap.add_argument("--parent")
    ap.add_argument("--jira-priority")
    a = ap.parse_args()
    json.dump(convert(sys.stdin.read(), a.key, a.summary, a.lang, a.parent, a.jira_priority), sys.stdout,
              indent=2, ensure_ascii=False)
