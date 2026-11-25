"""The make api .env contract, tested without starting the database/server."""
import importlib.util
from concurrent.futures import ThreadPoolExecutor
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import unittest

MODULE = Path(__file__).resolve().parents[1] / "run_local.py"
spec = importlib.util.spec_from_file_location("run_local", MODULE)
run_local = importlib.util.module_from_spec(spec)
spec.loader.exec_module(run_local)


class LocalEnvironmentTests(unittest.TestCase):
    def test_missing_file_is_allowed(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(run_local.local_env(Path(tmp), {"LOG_LEVEL": "debug"})["LOG_LEVEL"], "debug")

    def test_dotenv_debug_and_exported_precedence(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / ".env").write_text('LOG_LEVEL=debug\nexport MODEL_NAME="safe-model"\n# no effect\n', encoding="utf-8")
            self.assertEqual(run_local.local_env(root, {})["LOG_LEVEL"], "debug")
            self.assertEqual(run_local.local_env(root, {})["MODEL_NAME"], "safe-model")
            self.assertEqual(run_local.local_env(root, {"LOG_LEVEL": "warn"})["LOG_LEVEL"], "warn")

    def test_no_shell_evaluation(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / ".env").write_text('LOG_LEVEL=debug # comment\nMALFORMED\nVALUE=$(echo unsafe)\n', encoding="utf-8")
            env = run_local.local_env(root, {})
            self.assertEqual(env["LOG_LEVEL"], "debug")
            self.assertEqual(env["VALUE"], "$(echo unsafe)")

    def test_auth_bff_credentials_fill_only_missing_values_when_enabled(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / ".env").write_text(
                "AUTH_BFF_ENABLED=true\nTHEIA_USERS_BFF_CLIENT_SECRET=dotenv-user\n", encoding="utf-8",
            )
            (root / ".env.auth").write_text(
                "AUTH_BFF_KEY_BASE64=auth-file-key\nTHEIA_USERS_BFF_CLIENT_SECRET=auth-file-user\n"
                "THEIA_STAFF_BFF_CLIENT_SECRET=auth-file-staff\nKEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD=private\n"
                "AUTH_USER_LOGIN_ENABLED=true\n", encoding="utf-8",
            )
            env = run_local.local_env(root, {})
            self.assertEqual(env["AUTH_BFF_KEY_BASE64"], "auth-file-key")
            self.assertEqual(env["THEIA_USERS_BFF_CLIENT_SECRET"], "dotenv-user")
            self.assertEqual(env["THEIA_STAFF_BFF_CLIENT_SECRET"], "auth-file-staff")
            self.assertNotIn("KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD", env)
            self.assertNotIn("AUTH_USER_LOGIN_ENABLED", env)
            self.assertEqual(
                run_local.local_env(root, {"AUTH_BFF_KEY_BASE64": "exported"})["AUTH_BFF_KEY_BASE64"],
                "exported",
            )
            self.assertNotIn("AUTH_BFF_KEY_BASE64", run_local.local_env(root, {"AUTH_BFF_ENABLED": "false"}))


class EnvironmentSyncTests(unittest.TestCase):
    def test_existing_values_comments_empty_keys_and_bytes_are_preserved(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            original = b'\xef\xbb\xbf# developer settings\r\nexport SECRET="private-value"\r\nEMPTY=\r\nCUSTOM=local'
            (root / ".env").write_bytes(original)
            (root / ".env.example").write_text(
                'SECRET=placeholder\nEMPTY=new-default\nNEW="literal $(echo unsafe)"\n',
                encoding="utf-8",
            )
            self.assertEqual(run_local.sync_env(root), ["NEW"])
            self.assertTrue((root / ".env").read_bytes().startswith(original + b"\r\n"))
            self.assertEqual(run_local.local_env(root, {}), {
                "SECRET": "private-value", "EMPTY": "", "CUSTOM": "local",
                "NEW": "literal $(echo unsafe)",
            })

    def test_existing_file_mode_and_idempotency(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            dotenv = root / ".env"
            dotenv.write_text("KEEP=local\n", encoding="utf-8")
            dotenv.chmod(0o640)
            (root / ".env.example").write_text("KEEP=default\nNEW=one\n", encoding="utf-8")
            self.assertEqual(run_local.sync_env(root), ["NEW"])
            before = dotenv.read_bytes(), dotenv.stat().st_mtime_ns
            self.assertEqual(run_local.sync_env(root), [])
            self.assertEqual((dotenv.read_bytes(), dotenv.stat().st_mtime_ns), before)
            self.assertEqual(stat.S_IMODE(dotenv.stat().st_mode), 0o640)

    def test_clean_checkout_copies_comments_and_creates_private_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            template = b"# Configure once\nSECRET=placeholder\nNEW=default\n"
            (root / ".env.example").write_bytes(template)
            self.assertEqual(run_local.sync_env(root), ["SECRET", "NEW"])
            self.assertEqual((root / ".env").read_bytes(), template)
            self.assertEqual(stat.S_IMODE((root / ".env").stat().st_mode), 0o600)

    def test_duplicate_template_keys_are_appended_once(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / ".env").write_text("KEEP=local\n", encoding="utf-8")
            (root / ".env.example").write_text("NEW=first\nNEW=second\n", encoding="utf-8")
            self.assertEqual(run_local.sync_env(root), ["NEW"])
            self.assertEqual((root / ".env").read_text().count("NEW="), 1)
            self.assertEqual(run_local.local_env(root, {})["NEW"], "first")

    def test_concurrent_launches_do_not_duplicate_or_lose_keys(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / ".env").write_text("KEEP=local\n", encoding="utf-8")
            (root / ".env.example").write_text("KEEP=default\nNEW=one\nOTHER=two\n", encoding="utf-8")
            with ThreadPoolExecutor(max_workers=8) as pool:
                results = list(pool.map(lambda _: run_local.sync_env(root), range(8)))
            self.assertEqual(sum(len(keys) for keys in results), 2)
            self.assertEqual((root / ".env").read_text().count("NEW="), 1)
            self.assertEqual(run_local.local_env(root, {}), {"KEEP": "local", "NEW": "one", "OTHER": "two"})

    def test_invalid_utf8_does_not_modify_existing_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            original = b"SECRET=private\xff\n"
            (root / ".env").write_bytes(original)
            (root / ".env.example").write_text("NEW=default\n", encoding="utf-8")
            with self.assertRaises(UnicodeError):
                run_local.sync_env(root)
            self.assertEqual((root / ".env").read_bytes(), original)


class LauncherTests(unittest.TestCase):
    """Exercise the real Makefile/launcher with a stand-in for Go; no server or DB."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / "scripts").mkdir()
        (self.root / "apps/api").mkdir(parents=True)
        (self.root / "bin").mkdir()
        shutil.copyfile(MODULE, self.root / "scripts/run_local.py")
        (self.root / ".env").write_text("KEEP=developer\nAI_LOCAL_INSPECTOR=false\n", encoding="utf-8")
        (self.root / ".env.example").write_text(
            "KEEP=default\nNEW=literal-$(echo unsafe)\nAI_LOCAL_INSPECTOR=false\n", encoding="utf-8",
        )
        fake_go = self.root / "bin/go"
        fake_go.write_text(
            f"#!{sys.executable}\n"
            "import json, os, sys\n"
            "print(json.dumps({'args': sys.argv[1:], 'cwd': os.getcwd(), "
            "'KEEP': os.environ['KEEP'], 'NEW': os.environ['NEW'], "
            "'inspector': os.environ.get('AI_LOCAL_INSPECTOR')}))\n",
            encoding="utf-8",
        )
        fake_go.chmod(0o700)
        self.env = os.environ.copy()
        for key in ("KEEP", "NEW", "AI_LOCAL_INSPECTOR", "MAKEFLAGS", "MFLAGS", "MAKELEVEL"):
            self.env.pop(key, None)
        self.env["PATH"] = str(self.root / "bin") + os.pathsep + self.env["PATH"]

    def launch(self, *command):
        result = subprocess.run(command, cwd=self.root, env=self.env, text=True, capture_output=True, check=True)
        return result, json.loads(result.stdout)

    def test_make_api_syncs_then_starts_api_with_inspector(self):
        result, launched = self.launch("make", "-s", "-f", str(MODULE.parents[1] / "Makefile"), "api")
        self.assertEqual(launched, {
            "args": ["run", "./cmd/api"], "cwd": str(self.root / "apps/api"),
            "KEEP": "developer", "NEW": "literal-$(echo unsafe)", "inspector": "true",
        })
        self.assertIn("NEW", result.stderr)
        self.assertNotIn("literal-", result.stderr)
        self.assertIn("AI_LOCAL_INSPECTOR=false\n", (self.root / ".env").read_text())

    def test_make_api_honors_exported_disable_and_value_overrides(self):
        self.env.update(AI_LOCAL_INSPECTOR="false", KEEP="shell-override")
        _, launched = self.launch("make", "-s", "-f", str(MODULE.parents[1] / "Makefile"), "api")
        self.assertEqual(launched["inspector"], "false")
        self.assertEqual(launched["KEEP"], "shell-override")
        self.assertIn("KEEP=developer\n", (self.root / ".env").read_text())

    def test_direct_command_and_migrate_use_dotenv_inspector_setting(self):
        _, direct = self.launch(sys.executable, "scripts/run_local.py", "go", "run", "./cmd/api")
        self.assertEqual(direct["inspector"], "false")
        _, migrate = self.launch("make", "-s", "-f", str(MODULE.parents[1] / "Makefile"), "migrate")
        self.assertEqual(migrate["inspector"], "false")
        self.assertEqual(migrate["args"], ["run", "./cmd/migrate"])

    def test_make_env_sync_does_not_require_go_or_api_directory(self):
        shutil.rmtree(self.root / "apps")
        shutil.rmtree(self.root / "bin")
        result = subprocess.run(
            ["make", "-s", "-f", str(MODULE.parents[1] / "Makefile"), "env-sync"],
            cwd=self.root, env=self.env, text=True, capture_output=True, check=True,
        )
        self.assertEqual(result.stdout, "")
        self.assertEqual(run_local.local_env(self.root, {})["NEW"], "literal-$(echo unsafe)")


if __name__ == "__main__":
    unittest.main()
