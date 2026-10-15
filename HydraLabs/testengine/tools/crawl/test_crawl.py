"""best_locator: a data-testid that carries a record id (ULID, UUID, db number) is not trusted as stable."""
import os, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from crawl import best_locator  # noqa: E402


def b(**k):
    return best_locator({"testid": "", "id": "", "role": "", "name": "", "tag": "button", "type": "", **k})


def test_clean_testid_wins():
    assert b(testid="nav-sign-in") == ('[data-testid="nav-sign-in"]', "testid", True)
    assert b(testid="footer-2024")[2] is True  # a year is not a record id


def test_generated_testid_falls_back_to_role_and_name():
    assert b(testid="category-01M4YH6982NP1CKJVFTV5Y6VNB", role="checkbox", name="Drill") == \
        ('role=checkbox[name="Drill"]', "role+name", True)


def test_generated_testid_alone_is_used_but_flagged_weak():
    assert b(testid="product-01M4YH6982NP1CKJVFTV5Y6VNB")[1:] == ("testid", False)
    assert b(testid="row-3fa85f64-5717-4562-b3fc-2c963f66afa6")[2] is False
