"""AUTH-05 static wiring + no-network JDK smoke + realm installer regression tests."""
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
MODULE = ROOT / "infra/keycloak/sms-spi"

spec = importlib.util.spec_from_file_location("sms_flow", ROOT / "scripts/auth/configure_user_sms_flow.py")
sms_flow = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sms_flow)

try:
    import yaml
except ImportError:
    yaml = None


class FakeAdmin:
    def __init__(self, provider=True, client_disabled=True):
        self.provider = provider
        self.client_disabled = client_disabled
        self.flows = []
        self.executions = []
        self.realm = {"browserFlow": "browser", "loginTheme": "keycloak"}
        self.writes = []

    def call(self, method, path, payload=None):
        if method in ("PUT", "POST"):
            self.writes.append((method, path, payload))
        if path.startswith("/clients"):
            return [{"enabled": not self.client_disabled}]
        if path == "/authentication/authenticator-providers":
            return [{"id": "theia-sms-otp"}] if self.provider else []
        if path == "/authentication/flows":
            if method == "POST":
                self.flows.append(payload)
                return None
            return self.flows
        if path == "/authentication/flows/theia-users-sms/executions":
            if method == "PUT":
                self.executions[0]["requirement"] = payload["requirement"]
                return None
            return self.executions
        if path == "/authentication/flows/theia-users-sms/executions/execution":
            self.executions.append({"id": "execution-1", "providerId": payload["provider"], "requirement": "DISABLED"})
            return None
        if path == "":
            if method == "PUT":
                self.realm.update(payload)
                return None
            return self.realm
        raise AssertionError((method, path))


class SmsWiring(unittest.TestCase):
    def test_disabled_user_client_and_unchanged_staff_realm(self):
        users = json.loads((ROOT / "infra/keycloak/realms/theia-users-realm.json").read_text())
        staff = json.loads((ROOT / "infra/keycloak/realms/theia-staff-realm.json").read_text())
        self.assertFalse(users["clients"][0]["enabled"])
        self.assertEqual(staff["realm"], "theia-staff")
        self.assertTrue(staff["clients"][0]["enabled"])
        self.assertFalse(users["registrationAllowed"])
        self.assertFalse(users["resetPasswordAllowed"])

    @unittest.skipIf(yaml is None, "PyYAML unavailable")
    def test_docker_overlay_requires_secret_and_no_new_public_port(self):
        overlay = yaml.safe_load((ROOT / "compose.auth-sms.yaml").read_text())
        kc = overlay["services"]["keycloak"]
        self.assertNotIn("ports", kc)
        self.assertNotIn("command", kc)
        self.assertIn("HMAC_KEY_BASE64", repr(kc["environment"]))
        self.assertEqual(kc["build"]["dockerfile"], "infra/keycloak/Dockerfile.sms")
        self.assertNotIn("theia-staff", repr(kc))
        self.assertIn("quay.io/keycloak/keycloak:26.8.2", (ROOT / "infra/keycloak/Dockerfile.sms").read_text())

    def test_service_loader_and_local_private_mock_guard(self):
        loader = MODULE / "src/main/resources/META-INF/services/org.keycloak.authentication.AuthenticatorFactory"
        self.assertEqual(loader.read_text().strip(), "ir.theia.keycloak.sms.SmsOtpAuthenticatorFactory")
        config = (MODULE / "src/main/java/ir/theia/keycloak/sms/SmsConfiguration.java").read_text()
        self.assertIn('"local".equals(System.getenv("THEIA_SMS_ENV"))', config)
        self.assertIn('"http://auth.localhost:8180"', config)
        self.assertIn('"kavenegar".equals(mode)', config)
        self.assertIn('"disabled"', config)

    def test_sms_hmac_key_created_once_with_owner_only_env(self):
        key_path = ROOT / "scripts/auth/ensure_local_sms_key.py"
        spec_key = importlib.util.spec_from_file_location("sms_key", key_path)
        sms_key = importlib.util.module_from_spec(spec_key)
        spec_key.loader.exec_module(sms_key)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / ".env.auth"
            path.write_text("KEYCLOAK_DB_PASSWORD=keep-this-unchanged\n")
            path.chmod(0o600)
            self.assertTrue(sms_key.ensure_key(path))
            first = path.read_text()
            self.assertIn("KEYCLOAK_DB_PASSWORD=keep-this-unchanged", first)
            self.assertEqual(first.count("THEIA_SMS_HMAC_KEY_BASE64="), 1)
            self.assertFalse(sms_key.ensure_key(path))
            self.assertEqual(path.read_text(), first)
            path.chmod(0o644)
            with self.assertRaises(ValueError):
                sms_key.ensure_key(path)

    def test_installer_idempotent_and_client_never_enabled(self):
        fake = FakeAdmin()
        sms_flow.configure(fake)
        self.assertEqual(fake.realm["browserFlow"], "theia-users-sms")
        self.assertEqual(fake.executions[0]["requirement"], "REQUIRED")
        self.assertEqual(len(fake.writes), 4)
        sms_flow.configure(fake)
        self.assertEqual(len(fake.writes), 4)
        self.assertNotIn('"enabled": true', repr(fake.writes).lower())

    def test_installer_rejects_missing_provider_or_enabled_client_or_extra_execution(self):
        for fake in (FakeAdmin(provider=False), FakeAdmin(client_disabled=False)):
            with self.assertRaises(RuntimeError):
                sms_flow.configure(fake)
        fake = FakeAdmin()
        fake.flows = [{"alias": "theia-users-sms"}]
        fake.executions = [{"id": "1", "providerId": "auth-username-password-form", "requirement": "REQUIRED"}]
        with self.assertRaises(RuntimeError):
            sms_flow.configure(fake)
        self.assertFalse(fake.writes)

    @unittest.skipUnless(shutil.which("javac") and shutil.which("java"), "JDK not installed")
    def test_pure_java_otp_smoke_without_maven_or_keycloak(self):
        classes = MODULE / "src/main/java/ir/theia/keycloak/sms"
        names = ["PhoneNumber", "SmsSender", "OtpCrypto", "AtomicStore", "ChallengeService", "MockSmsSender", "KavenegarSmsSender", "SmsConfiguration"]
        source = r'''
import ir.theia.keycloak.sms.*;
import java.util.*;
import java.util.concurrent.*;
import java.time.*;
public class Smoke {
    static class Store implements AtomicStore {
        final ConcurrentHashMap<String,Map<String,String>> data = new ConcurrentHashMap<>();
        public boolean reserve(String k,long t) { return data.putIfAbsent(k,Map.of())==null; }
        public void put(String k,long t,Map<String,String> m) { data.put(k,Map.copyOf(m)); }
        public Map<String,String> get(String k) { return data.get(k); }
        public Map<String,String> consume(String k) { return data.remove(k); }
    }
    static class Sender implements SmsSender {
        String code;
        public void send(String p,String c) { code=c; }
    }
    public static void main(String[] args) throws Exception {
        byte[] secret = new byte[32]; new java.security.SecureRandom().nextBytes(secret);
        OtpCrypto crypto = new OtpCrypto(Base64.getEncoder().encodeToString(secret));
        Store store = new Store(); Sender sender = new Sender();
        var now = Clock.fixed(Instant.parse("2026-11-19T16:00:00Z"), ZoneOffset.UTC);
        var svc = new ChallengeService(store,sender,crypto,now);
        var item = svc.send("۰۹۱۲۳۴۵۶۷۸۹", "127.0.0.1", null);
        if(!item.phone().equals("+989123456789")) throw new AssertionError("phone");
        if(store.data.toString().contains(sender.code)) throw new AssertionError("plaintext OTP stored");
        if(store.data.toString().contains(item.phone())) throw new AssertionError("plaintext phone stored");
        try {svc.send(item.phone(),"127.0.0.1",null);throw new AssertionError("cooldown");}
        catch(ChallengeService.RateLimited expected) {}
        if(svc.verify(item.id(), item.phone(), sender.code) != ChallengeService.Verified.SUCCESS) throw new AssertionError("verify");
        if(svc.verify(item.id(), item.phone(), sender.code) == ChallengeService.Verified.SUCCESS) throw new AssertionError("replay");
        System.out.println("AUTH05_PURE_SMOKE_OK");
    }
}'''
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp)
            (path / "Smoke.java").write_text(source)
            result = subprocess.run(["javac", "-encoding", "UTF-8", "-d", str(path)] +
                                    [str(classes / (n + ".java")) for n in names] +
                                    [str(path / "Smoke.java")], capture_output=True, text=True, timeout=25)
            self.assertEqual(result.returncode, 0, result.stderr[-1200:])
            result = subprocess.run(["java", "-cp", str(path), "Smoke"], capture_output=True, text=True, timeout=15)
            self.assertEqual(result.returncode, 0, result.stderr[-1200:])
            self.assertIn("AUTH05_PURE_SMOKE_OK", result.stdout)


if __name__ == "__main__": unittest.main()
