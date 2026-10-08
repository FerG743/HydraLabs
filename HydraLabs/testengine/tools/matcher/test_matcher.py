import os, sys, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import matcher  # noqa: E402

NAMES = {"a_flujo", "b_flujo"}


class Decide(unittest.TestCase):
    def test_agreement_is_accepted(self):
        self.assertEqual(matcher.decide("a_flujo", "a_flujo", NAMES), ("accept", "a_flujo"))

    def test_disagreement_goes_to_review_with_primarys_pick(self):
        self.assertEqual(matcher.decide("a_flujo", "b_flujo", NAMES), ("review", "a_flujo"))

    def test_abstention_is_never_agreement(self):  # two "none" must not be auto-accepted
        self.assertEqual(matcher.decide("none", "none", NAMES), ("review", None))

    def test_primary_abstains_secondary_valid(self):
        self.assertEqual(matcher.decide("none", "b_flujo", NAMES), ("review", "b_flujo"))

    def test_invented_name_is_never_accepted(self):
        self.assertEqual(matcher.decide("zzz_flujo", "zzz_flujo", NAMES), ("review", None))


if __name__ == "__main__":
    unittest.main()
