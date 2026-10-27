import json
import os
import pathlib
import subprocess
import tempfile
import unittest


SCRIPTS = pathlib.Path(__file__).resolve().parent
SETUP = SCRIPTS / "configure-agent-attribution.py"
ASSISTED_BY = SCRIPTS / "assisted-by"


class AttributionSetupTest(unittest.TestCase):
    def test_existing_settings_survive_repeated_setup(self):
        with tempfile.TemporaryDirectory() as directory:
            home = pathlib.Path(directory)
            settings_file = home / ".claude" / "settings.json"
            settings_file.parent.mkdir()
            settings_file.write_text(
                json.dumps(
                    {
                        "autoUpdatesChannel": "stable",
                        "attribution": {"commit": "old", "pr": "old", "other": "keep"},
                    }
                ),
                encoding="utf-8",
            )
            env = {**os.environ, "HOME": str(home), "GIT_CONFIG_NOSYSTEM": "1"}
            for _ in range(2):
                subprocess.run(["python3", str(SETUP), str(ASSISTED_BY)], env=env, check=True)

            settings = json.loads(settings_file.read_text(encoding="utf-8"))
            self.assertEqual(settings["autoUpdatesChannel"], "stable")
            self.assertEqual(settings["attribution"], {"commit": "", "pr": "", "other": "keep"})
            installed = home / ".local" / "bin" / "assisted-by"
            self.assertEqual(installed.read_bytes(), ASSISTED_BY.read_bytes())
            self.assertEqual(installed.stat().st_mode & 0o777, 0o755)
            config = subprocess.check_output(
                ["git", "config", "--global", "--get", "trailer.assisted-by.ifExists"],
                env=env,
                text=True,
            )
            self.assertEqual(config.strip(), "addIfDifferent")

    def test_invalid_settings_stop_setup(self):
        with tempfile.TemporaryDirectory() as directory:
            home = pathlib.Path(directory)
            settings_file = home / ".claude" / "settings.json"
            settings_file.parent.mkdir()
            settings_file.write_text('{"attribution": "bad"}', encoding="utf-8")
            result = subprocess.run(
                ["python3", str(SETUP), str(ASSISTED_BY)],
                env={**os.environ, "HOME": str(home)},
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("Claude attribution settings must be a JSON object", result.stderr)
            self.assertFalse((home / ".local" / "bin" / "assisted-by").exists())


if __name__ == "__main__":
    unittest.main()
