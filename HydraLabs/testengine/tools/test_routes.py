import random, unittest

import routes


def page(url, *links):
    return {"url": url, "elements": [{"href": h, "locator": f"role=link[name=\"{h}\"]", "stable": True} for h in links]}


class RoutesTest(unittest.TestCase):
    def test_every_shortest_route_is_kept_and_the_chosen_one_is_one_of_them(self):
        crawl = {"baseUrl": "http://h/", "pages": [page("http://h/", "/a", "/c"), page("http://h/a", "/b"),
                                                   page("http://h/c", "/b"), page("http://h/b")]}
        r = routes.routes(crawl, "X", rng=random.Random(3))
        b = next(x for x in r["routes"] if x["url"] == "http://h/b")
        self.assertEqual(b["clicks"], 2)
        self.assertEqual(sorted(h[0]["to"] for h in [[x for x in route] for route in b["routes"]]), ["http://h/a", "http://h/c"])
        self.assertIn(b["hops"], b["routes"])

    def test_the_cap_limits_how_many_are_kept(self):
        crawl = {"baseUrl": "http://h/", "pages": [page("http://h/", "/a", "/c"), page("http://h/a", "/b"),
                                                   page("http://h/c", "/b"), page("http://h/b")]}
        r = routes.routes(crawl, "X", max_routes=1, rng=random.Random(3))
        self.assertEqual(len(next(x for x in r["routes"] if x["url"] == "http://h/b")["routes"]), 1)

    def test_the_same_seed_picks_the_same_route(self):
        crawl = {"baseUrl": "http://h/", "pages": [page("http://h/", "/a", "/c"), page("http://h/a", "/b"),
                                                   page("http://h/c", "/b"), page("http://h/b")]}
        picks = {str(next(x for x in routes.routes(crawl, "X", rng=random.Random(7))["routes"] if x["url"] == "http://h/b")["hops"])
                 for _ in range(5)}
        self.assertEqual(len(picks), 1)


if __name__ == "__main__":
    unittest.main()
