import os, sys, unittest
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import runner  # noqa: E402

CFG = {"allow_writes": True, "hosts": ["127.0.0.1"], "endpoints": ["POST /uploadFile"]}


def env_for(writes):
    with mock.patch.object(runner.subprocess, "run") as run:
        run.return_value = mock.Mock(returncode=0, stdout="", stderr="")
        runner._pytest("/repo", ["x"], "http://127.0.0.1:3000", 10, writes=writes)
        return run.call_args.kwargs["env"]


class WritesComeFromTheTeamsConfig(unittest.TestCase):
    def test_nothing_is_writable_without_a_config(self):
        self.assertEqual(env_for(None)["HYDRA_ALLOW_WRITES"], "0")

    def test_a_config_that_allows_writes_passes_its_hosts_and_endpoints(self):
        env = env_for(CFG)
        self.assertEqual((env["HYDRA_ALLOW_WRITES"], env["HYDRA_WRITE_HOSTS"], env["HYDRA_WRITE_ENDPOINTS"]),
                         ("1", "127.0.0.1", "POST /uploadFile"))

    def test_a_config_that_switches_writes_off_allows_nothing(self):
        self.assertEqual(env_for(dict(CFG, allow_writes=False))["HYDRA_ALLOW_WRITES"], "0")

    def test_collection_and_probes_stay_read_only(self):
        with mock.patch.object(runner.subprocess, "run") as run:
            run.return_value = mock.Mock(returncode=0, stdout="", stderr="")
            runner._pytest("/repo", ["--collect-only"], "http://127.0.0.1:3000", 10)
            self.assertEqual(run.call_args.kwargs["env"]["HYDRA_ALLOW_WRITES"], "0")


if __name__ == "__main__":
    unittest.main()
