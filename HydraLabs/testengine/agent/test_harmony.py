import json, os, sys, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import agent  # noqa: E402


class HarmonyText(unittest.TestCase):
    def test_a_call_written_as_text_is_read_as_a_call(self):
        text = '<|channel|>final to=functions.finish<|channel|>commentary<|constrain|>json<|message|>{"summary":"listo"}'
        calls = agent.harmony_calls(text)
        self.assertEqual([c["function"]["name"] for c in calls], ["finish"])
        self.assertEqual(json.loads(calls[0]["function"]["arguments"]), {"summary": "listo"})

    def test_plain_prose_and_broken_json_are_not_calls(self):
        self.assertEqual(agent.harmony_calls("Voy a revisar la página."), [])
        self.assertEqual(agent.harmony_calls('to=functions.finish<|message|>{"summary":'), [])


if __name__ == "__main__":
    unittest.main()
