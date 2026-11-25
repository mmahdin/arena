"""AUTH-04 structure and safety contract tests (real SQL/Keycloak need Docker)."""
from pathlib import Path
import json
import unittest

ROOT = Path(__file__).resolve().parents[2]


class StaffAuthenticationTests(unittest.TestCase):
    def test_staff_realm_is_private_and_password_only(self):
        realm = json.loads((ROOT / "infra/keycloak/realms/theia-staff-realm.json").read_text())
        self.assertTrue(realm["enabled"])
        self.assertFalse(realm["registrationAllowed"])
        self.assertFalse(realm["resetPasswordAllowed"])
        self.assertTrue(realm["bruteForceProtected"])
        self.assertIn("length(14)", realm["passwordPolicy"])
        client = realm["clients"][0]
        self.assertTrue(client["enabled"])
        self.assertFalse(client["directAccessGrantsEnabled"])
        self.assertFalse(client["serviceAccountsEnabled"])

    def test_local_provision_command_and_lifecycle(self):
        source = (ROOT / "apps/api/cmd/staffctl/main.go").read_text()
        self.assertIn('staffctl is limited to APP_ENV=local', source)
        self.assertIn('mutations require explicit -confirm', source)
        self.assertNotIn('password :=', source)
        repo = (ROOT / "apps/api/internal/modules/auth/adapters/mssql/staff.go").read_text()
        self.assertIn('Isolation: sql.LevelSerializable', repo)
        self.assertIn('UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()', repo)
        self.assertIn('auth.ErrInactive', repo)
        self.assertIn('auth_audit_events', repo)

    def test_staff_audit_migration_and_docker_binary(self):
        sql = (ROOT / "apps/api/db/migrations/0025_staff_provisioning_audit.sql").read_text()
        self.assertIn('staff_deactivated', sql)
        self.assertIn('staff_reactivated', sql)
        self.assertIn('target_staff_id', sql)
        dockerfile = (ROOT / "Dockerfile").read_text()
        self.assertIn('go build -trimpath -o /out/staffctl', dockerfile)
        self.assertIn('COPY --from=build /out/staffctl /app/staffctl', dockerfile)


if __name__ == '__main__':
    unittest.main()
