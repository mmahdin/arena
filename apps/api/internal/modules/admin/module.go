// Package admin contains the Back-Office application boundary. It uses the
// shared Auth module's pre-provisioned staff identity and session guard.
package admin

import (
	"database/sql"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/adapters/http"
	adminmssql "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// Inventory is optional for legacy composition tests; normal boot passes the
// one-time verified, metadata-only observation of deployed/candidate bundles.
func NewModule(db *sql.DB, inventory ...domain.BundleInventory) *httpadapter.Handler {
	service := application.NewWithEvaluationRepositories(adminmssql.NewAuditRepository(db), adminmssql.NewOverviewRepository(db), adminmssql.NewInspectorRepository(db), adminmssql.NewEvaluationRepository(db))
	service.SetTraceReferenceStore(adminmssql.NewTraceReferenceReader(db))
	if len(inventory) > 0 {
		service.SetBundleInventory(inventory[0])
	} else {
		service.SetBundleInventory(domain.BundleInventory{Scope: "not_configured"})
	}
	return httpadapter.New(service)
}
