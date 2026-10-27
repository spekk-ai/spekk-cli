import os
import pathlib
import subprocess
import tempfile
import unittest


ASSISTED_BY = pathlib.Path(__file__).resolve().parent / "assisted-by"
SESSION_VARIABLES = (
    "CODEX_SESSION_ID",
    "CODEX_HOME",
    "CLAUDECODE",
    "CLAUDE_CODE_SESSION_ID",
    "OPENCODE_PID",
    "OPENCODE_CONFIG_DIR",
)


class AssistedByTest(unittest.TestCase):
    def run_helper(self, home: pathlib.Path, cwd: pathlib.Path, **variables: str):
        env = os.environ.copy()
        for name in SESSION_VARIABLES:
            env.pop(name, None)
        env.update(
            {
                "HOME": str(home),
                "GIT_CONFIG_NOSYSTEM": "1",
                "GIT_CONFIG_GLOBAL": os.devnull,
                **variables,
            }
        )
        return subprocess.run([str(ASSISTED_BY)], cwd=cwd, env=env, capture_output=True, text=True)

    def test_missing_codex_record_does_not_use_config_model(self):
        with tempfile.TemporaryDirectory() as directory:
            home = pathlib.Path(directory)
            codex_home = home / ".codex"
            codex_home.mkdir()
            (codex_home / "config.toml").write_text('model = "wrong-model"\n', encoding="utf-8")

            result = self.run_helper(home, home, CODEX_SESSION_ID="missing-session")

            self.assertEqual(result.returncode, 1)
            self.assertEqual(result.stdout, "")
            self.assertIn("no Codex rollout for session", result.stderr)

    def test_multiple_claude_transcripts_are_ambiguous(self):
        with tempfile.TemporaryDirectory() as directory:
            home = pathlib.Path(directory)
            project = home / "project"
            project.mkdir()
            project_key = str(project).replace("/", "-")
            transcripts = home / ".claude" / "projects" / project_key
            transcripts.mkdir(parents=True)
            for name in ("first", "second"):
                (transcripts / f"{name}.jsonl").write_text(
                    '{"type":"assistant","message":{"model":"wrong-model"}}\n',
                    encoding="utf-8",
                )

            result = self.run_helper(home, project, CLAUDECODE="1")

            self.assertEqual(result.returncode, 1)
            self.assertEqual(result.stdout, "")
            self.assertIn("multiple claude-code transcripts", result.stderr)


if __name__ == "__main__":
    unittest.main()
