package httpadapter

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

func evaluationFailure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEval), errors.Is(err, domain.ErrInvalidInspector):
		httpx.Failure(c, http.StatusBadRequest, "ADMIN_EVALUATION_INVALID")
	case errors.Is(err, domain.ErrEvalNotFound):
		httpx.Failure(c, http.StatusNotFound, "ADMIN_EVALUATION_NOT_FOUND")
	case errors.Is(err, domain.ErrEvalConflict):
		httpx.Failure(c, http.StatusConflict, "ADMIN_EVALUATION_CONFLICT")
	case errors.Is(err, application.ErrUnverifiedStaff):
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
	case errors.Is(err, application.ErrAuditUnavailable):
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
	default:
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_EVALUATIONS_UNAVAILABLE")
	}
}

func parseEvalFilter(c *gin.Context, now time.Time) (domain.EvalFilter, error) {
	f := domain.NewEvalFilter(now)
	query := c.Request.URL.Query()
	for key, vals := range query {
		switch key {
		case "from", "to", "pageSize", "status", "cursor":
		default:
			return f, domain.ErrInvalidEval
		}
		if len(vals) != 1 || vals[0] == "" {
			return f, domain.ErrInvalidEval
		}
	}
	from, hasFrom := query["from"]
	to, hasTo := query["to"]
	if hasFrom != hasTo {
		return f, domain.ErrInvalidEval
	}
	if hasFrom {
		var err error
		f.From, err = time.Parse(time.RFC3339, from[0])
		if err != nil {
			return f, domain.ErrInvalidEval
		}
		f.To, err = time.Parse(time.RFC3339, to[0])
		if err != nil {
			return f, domain.ErrInvalidEval
		}
	}
	if f.To.After(now.Add(time.Second)) {
		return f, domain.ErrInvalidEval
	}
	if raw := query.Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return f, domain.ErrInvalidEval
		}
		f.Limit = n
	}
	if raw := query.Get("cursor"); raw != "" {
		value, err := domain.DecodeInspectorCursor(raw)
		if err != nil {
			return f, domain.ErrInvalidEval
		}
		f.Cursor = value
	}
	f.Status = query.Get("status")
	return f, f.Validate()
}

func (h *Handler) Evaluations(c *gin.Context) {
	p, requestID, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	f, err := parseEvalFilter(c, time.Now().UTC())
	if err != nil {
		evaluationFailure(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	page, err := h.service.ListEvaluations(ctx, p, requestID, f)
	if err != nil {
		evaluationFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) EvaluationDetail(c *gin.Context) {
	p, requestID, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		evaluationFailure(c, domain.ErrInvalidEval)
		return
	}
	id, err := uuid.Parse(c.Param("runId"))
	if err != nil || id == uuid.Nil {
		evaluationFailure(c, domain.ErrInvalidEval)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	detail, err := h.service.GetEvaluation(ctx, p, requestID, id)
	if err != nil {
		evaluationFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, detail)
}

// EvaluationImport accepts *only* the raw, bounded WP-15 eval_report.v1 JSON
// file. X-Eval-SHA256 must match the exact bytes sent by the caller. The hash
// is integrity evidence, not signature/attestation of a runner or reviewer.
// Staff CSRF+Origin checks are enforced by the enclosing RequireStaff group.
func (h *Handler) EvaluationImport(c *gin.Context) {
	p, requestID, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	query := c.Request.URL.Query()
	var baseline *uuid.UUID
	for key, vals := range query {
		if key != "baselineRunId" || len(vals) != 1 || vals[0] == "" {
			evaluationFailure(c, domain.ErrInvalidEval)
			return
		}
	}
	if raw := query.Get("baselineRunId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil || id.String() != raw {
			evaluationFailure(c, domain.ErrInvalidEval)
			return
		}
		baseline = &id
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		httpx.Failure(c, http.StatusUnsupportedMediaType, "ADMIN_EVALUATION_CONTENT_TYPE")
		return
	}
	if c.Request.ContentLength > domain.MaxEvalReportBytes {
		httpx.Failure(c, http.StatusRequestEntityTooLarge, "ADMIN_EVALUATION_TOO_LARGE")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, domain.MaxEvalReportBytes+1))
	if err != nil || len(raw) == 0 {
		evaluationFailure(c, domain.ErrInvalidEval)
		return
	}
	if len(raw) > domain.MaxEvalReportBytes {
		httpx.Failure(c, http.StatusRequestEntityTooLarge, "ADMIN_EVALUATION_TOO_LARGE")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	run, created, err := h.service.ImportEvaluation(ctx, p, requestID, raw, c.GetHeader("X-Eval-SHA256"), baseline)
	if err != nil {
		evaluationFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	if created {
		c.JSON(http.StatusCreated, run)
	} else {
		c.JSON(http.StatusOK, run)
	}
}
