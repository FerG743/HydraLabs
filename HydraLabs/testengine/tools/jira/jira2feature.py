#!/usr/bin/env python3
"""One step for orchestrators: Jira issue JSON on stdin -> {"case", "feature", "warnings"} on stdout.
Taking JSON (not flags) means issue text never goes through a shell.

    stdin: {"key": "DWQ-130", "summary": "...", "html": "<ul>...</ul>", "lang": "en",
            "parent": "DWQ-129", "jira_priority": "Medium"}
"""
import json, os, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gherkin_gen import generate  # noqa: E402
from jira2case import convert  # noqa: E402

if __name__ == "__main__":
    i = json.load(sys.stdin)
    case = convert(i.get("html") or "", i["key"], i["summary"], i.get("lang", "en"), i.get("parent"), i.get("jira_priority"))
    json.dump({"case": case, "feature": generate(case), "warnings": case["warnings"]}, sys.stdout, ensure_ascii=False)
