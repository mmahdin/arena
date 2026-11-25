"""Static regression checks for the synthetic-only PROD-01 data boundary."""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]
MIGRATION = ROOT / "apps/api/db/migrations/0035_product_events.sql"
PRESERVATION = MIGRATION.with_name("0036_product_events_preservation.sql")
GO = ROOT / "apps/api/internal/platform/productevents/events.go"


class ProductEventsContractTests(unittest.TestCase):
    def test_mssql_append_only_schema_and_aggregate_view(self):
        migration = MIGRATION.read_text()
        self.assertIn("CREATE TABLE dbo.product_events", migration)
        self.assertIn("FOREIGN KEY (user_id) REFERENCES dbo.users(id)", migration)
        self.assertIn("UNIQUE(event_name,subject_id)", migration)
        self.assertIn("datetime2(3)", migration)
        self.assertIn("SYSUTCDATETIME()", migration)
        self.assertIn("CREATE VIEW dbo.v_product_daily_events_synthetic", migration)
        self.assertIn("u.is_test=1", migration)
        self.assertIn("CREATE TRIGGER dbo.product_events_reject_real_users", migration)
        self.assertIn("u.is_test<>1", migration)
        self.assertIn("product_events_preserve_delete_guard", PRESERVATION.read_text())
        view = migration.split("CREATE VIEW dbo.v_product_daily_events_synthetic", 1)[1]
        self.assertNotRegex(view, r"SELECT\s+[^\n]*(?:user_id|subject_id)")
        self.assertNotRegex(migration, r"\b(?:text|content|prompt|response|properties_json|phone|email)\s+(?:nvarchar|varchar)")
        for forbidden in (r"\bGO\b",r"\bBEGIN\s+TRAN\b",r"\bON\s+DELETE\s+CASCADE\b"):
            self.assertNotRegex(migration, forbidden)

    def test_fixed_catalog_and_safe_server_sql(self):
        source = GO.read_text()
        self.assertIn("switch n", source)
        self.assertIn("u.is_test=1", source)
        self.assertIn("u.deleted_at IS NULL", source)
        self.assertIn("UPDLOCK,HOLDLOCK", source)
        self.assertNotIn("properties map", source)
        self.assertNotIn("map[string]interface{}", source)
        self.assertIn("*sql.Tx", source)

    def test_every_event_write_uses_business_transaction(self):
        paths = {
            "auth/adapters/mssql/repository.go": "UserRegistered",
            "user/adapters/mssql/progress.go": "OnboardingStarted",
            "user/adapters/mssql/repository.go": "OnboardingCompleted",
            "conversation/adapters/mssql/repository.go": "ConversationStarted",
            "conversation/adapters/mssql/lifecycle.go": "AITurnAdmitted",
            "bookmark/adapters/mssql/repository.go": "BookmarkSaved",
        }
        for rel, event in paths.items():
            with self.subTest(rel=rel):
                data = (ROOT/"apps/api/internal/modules"/rel).read_text()
                self.assertIn("productevents.Record(ctx, tx", data)
                self.assertIn("productevents."+event, data)
        lifecycle=(ROOT/"apps/api/internal/modules/conversation/adapters/mssql/lifecycle.go").read_text()
        self.assertIn("productevents.AITurnCompleted", lifecycle)
        self.assertIn("productevents.AITurnFailed", lifecycle)


if __name__ == "__main__":
    unittest.main()
