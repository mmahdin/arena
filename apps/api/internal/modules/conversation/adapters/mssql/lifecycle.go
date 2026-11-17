package mssql

// This adapter implements the AI consumer's storage contracts. The composition
// root injects the AI-owned SQL participant; sql.Tx never crosses an application port.
import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"golang.org/x/text/unicode/norm"
)

// StateParticipant is an infrastructure-only UoW seam. Implementations must use
// the supplied transaction exclusively and perform no network/provider calls.
type StateParticipant interface {
	Admit(context.Context, *sql.Tx, c.TurnRequest, *c.Admission) error
	Apply(context.Context, *sql.Tx, c.Admission, c.PreparedTurn) error
	Acquire(context.Context, *sql.Tx, c.ID, c.ID, time.Duration) (ports.LeaseHandle, error)
	Check(context.Context, *sql.Tx, ports.LeaseHandle) error
	Release(context.Context, *sql.Tx, ports.LeaseHandle) error
	Renew(context.Context, *sql.Tx, ports.LeaseHandle, time.Duration) (ports.LeaseHandle, error)
}

type TurnStore struct {
	db    *sql.DB
	state StateParticipant
	ttl   time.Duration
}

func NewTurnStore(db *sql.DB, state StateParticipant, ttl time.Duration) *TurnStore {
	if db == nil || state == nil || ttl < time.Second || ttl > time.Hour {
		panic("invalid turn persistence configuration")
	}
	return &TurnStore{db: db, state: state, ttl: ttl}
}

var _ ports.ConversationStore = (*TurnStore)(nil)
var _ ports.TurnFinalizer = (*TurnStore)(nil)
var _ ports.TurnSerialiser = (*TurnStore)(nil)
var _ ports.TurnRecovery = (*TurnStore)(nil)

func failure(code c.Code) error { return &c.Error{Code: code, Step: "persistence"} }
func (s *TurnStore) begin(ctx context.Context, user c.ID) (*sql.Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	// Every lifecycle mutation locks the same durable owner row first. No
	// inference runs while this short database lock is held.
	var id c.ID
	err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1`, user).Scan(&id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			err = httpx.ErrNotFound
		}
		return nil, err
	}
	return tx, nil
}
func (s *TurnStore) ValidateOwner(ctx context.Context, user, conv c.ID) error {
	var id c.ID
	err := s.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.conversations WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, conv, user).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	return err
}
func normalize(req c.TurnRequest) (c.TurnRequest, [32]byte, error) {
	var zero [32]byte
	if !c.ValidID(req.UserID) || !c.ValidID(req.ConversationID) || !c.ValidID(req.ClientTurnID) || !utf8.ValidString(req.Text) {
		return req, zero, failure(c.ErrValidation)
	}
	req.Text = norm.NFC.String(req.Text)
	if strings.TrimSpace(req.Text) == "" || len(utf16.Encode([]rune(req.Text))) > 20000 || len(req.Locale) > 24 || len(req.TZ) > 100 {
		return req, zero, failure(c.ErrValidation)
	}
	if req.Origin == "" {
		req.Origin = c.OriginUser
	}
	if req.Origin != c.OriginUser || req.WakeUp != nil {
		return req, zero, failure(c.ErrValidation)
	}
	if req.Locale == "" {
		req.Locale = "fa"
	}
	if req.TZ == "" {
		req.TZ = "UTC"
	}
	if req.Channel == "" {
		req.Channel = "app_chat"
	}
	if _, err := time.LoadLocation(req.TZ); err != nil {
		return req, zero, failure(c.ErrValidation)
	}
	// Client timestamp is transport metadata; changing it on retry is harmless.
	hashed := req
	hashed.ClientTS = nil
	b, err := json.Marshal(hashed)
	return req, sha256.Sum256(b), err
}
func (s *TurnStore) AdmitUserMessage(ctx context.Context, req c.TurnRequest) (c.Admission, error) {
	req, hash, err := normalize(req)
	if err != nil {
		return c.Admission{}, err
	}
	tx, err := s.begin(ctx, req.UserID)
	if err != nil {
		return c.Admission{}, err
	}
	defer tx.Rollback()
	a := c.Admission{UserID: req.UserID, ConversationID: req.ConversationID}
	var owner c.ID
	if err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.conversations WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, req.ConversationID, req.UserID).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = httpx.ErrNotFound
		}
		return a, err
	}
	var stored []byte
	var conv c.ID
	var status string
	var version int
	err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id),CONVERT(varchar(36),conversation_id),input_hash,status,lifecycle_version,CONVERT(varchar(36),user_message_id) FROM dbo.turns WHERE user_id=@p1 AND client_turn_id=@p2`, req.UserID, req.ClientTurnID).Scan(&a.TurnID, &conv, &stored, &status, &version, &a.UserMessageID)
	if err == nil {
		if conv != req.ConversationID || version != 1 || string(stored) != string(hash[:]) {
			return a, failure(c.ErrDuplicate)
		}
		a.DuplicateOf = &a.TurnID
	} else if errors.Is(err, sql.ErrNoRows) {
		metaReq := req
		metaReq.Text = ""
		metadata, e := json.Marshal(metaReq)
		if e != nil {
			return a, e
		}
		a.TurnID = c.NewID()
		a.UserMessageID = c.NewID()
		status = "queued"
		if req.ReplyToID != nil {
			var found int
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1 AND conversation_id=@p2`, *req.ReplyToID, req.ConversationID).Scan(&found); err != nil {
				return a, err
			}
			if found != 1 {
				return a, httpx.ErrNotFound
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT dbo.turns(id,conversation_id,user_id,client_turn_id,input_hash,status,user_message_id,lifecycle_version,request_meta_json) VALUES(@p1,@p2,@p3,@p4,@p5,N'queued',@p6,1,@p7)`, a.TurnID, req.ConversationID, req.UserID, req.ClientTurnID, hash[:], a.UserMessageID, string(metadata))
		if err != nil {
			return a, err
		}
		_, err = tx.ExecContext(ctx, `INSERT dbo.messages(id,conversation_id,turn_id,role,content,lang) VALUES(@p1,@p2,@p3,N'user',@p4,@p5)`, a.UserMessageID, req.ConversationID, a.TurnID, req.Text, req.Locale)
		if err != nil {
			return a, err
		}
	} else {
		return a, err
	}
	if status == "queued" || status == "running" {
		// Admission retries do not receive another worker's live fencing token.
		lease, e := s.acquire(ctx, tx, req.UserID, a.TurnID)
		if e == nil {
			a.LeaseToken = lease.Token
		} else {
			var ce *c.Error
			if !errors.As(e, &ce) || ce.Code != c.ErrQueued {
				return a, e
			}
			a.Queued = true
		}
		if err = s.state.Admit(ctx, tx, req, &a); err != nil {
			return a, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE dbo.conversations SET updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2`, req.ConversationID, req.UserID); err != nil {
		return a, err
	}
	if err = tx.Commit(); err != nil {
		return c.Admission{}, err
	}
	return a, nil
}
func (s *TurnStore) acquire(ctx context.Context, tx *sql.Tx, user, turn c.ID) (ports.LeaseHandle, error) {
	var first c.ID
	err := tx.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),id) FROM dbo.turns WHERE user_id=@p1 AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=turns.conversation_id AND v.user_id=turns.user_id AND v.deleted_at IS NULL) AND lifecycle_version=1 AND status IN(N'queued',N'running') ORDER BY admission_order`, user).Scan(&first)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ports.LeaseHandle{}, err
	}
	if errors.Is(err, sql.ErrNoRows) || first != turn {
		return ports.LeaseHandle{}, failure(c.ErrQueued)
	}
	h, err := s.state.Acquire(ctx, tx, user, turn, s.ttl)
	if err != nil {
		return h, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.turns SET status=N'running',attempt_count=attempt_count+1,error_code=NULL WHERE id=@p1 AND user_id=@p2`, turn, user)
	return h, err
}
func (s *TurnStore) Acquire(ctx context.Context, user, turn c.ID) (ports.LeaseHandle, error) {
	tx, err := s.begin(ctx, user)
	if err != nil {
		return ports.LeaseHandle{}, err
	}
	defer tx.Rollback()
	h, err := s.acquire(ctx, tx, user, turn)
	if err != nil {
		return h, err
	}
	return h, tx.Commit()
}
func (s *TurnStore) Release(ctx context.Context, h ports.LeaseHandle) error {
	tx, err := s.begin(ctx, h.UserID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.state.Release(ctx, tx, h); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *TurnStore) Renew(ctx context.Context, h ports.LeaseHandle) (ports.LeaseHandle, error) {
	tx, err := s.begin(ctx, h.UserID)
	if err != nil {
		return h, err
	}
	defer tx.Rollback()
	h, err = s.state.Renew(ctx, tx, h, s.ttl)
	if err != nil {
		return h, err
	}
	return h, tx.Commit()
}
func handle(a c.Admission) ports.LeaseHandle {
	return ports.LeaseHandle{UserID: a.UserID, TurnID: a.TurnID, Token: a.LeaseToken}
}
func (s *TurnStore) check(ctx context.Context, tx *sql.Tx, a c.Admission) error {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns WHERE id=@p1 AND user_id=@p2 AND conversation_id=@p3 AND user_message_id=@p4 AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=turns.conversation_id AND v.user_id=turns.user_id AND v.deleted_at IS NULL) AND lifecycle_version=1 AND status=N'running'`, a.TurnID, a.UserID, a.ConversationID, a.UserMessageID).Scan(&n)
	if err != nil {
		return err
	}
	if n != 1 {
		return failure(c.ErrStateConflict)
	}
	return s.state.Check(ctx, tx, handle(a))
}
func (s *TurnStore) Complete(ctx context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	if err := ctx.Err(); err != nil {
		return c.TurnResult{}, err
	}
	if p.TurnID != a.TurnID || p.UserMessageID != a.UserMessageID || p.State.TurnID != a.TurnID || p.State.UserID != a.UserID || !p.Route.Valid() || !p.Objective.Valid() || strings.TrimSpace(p.Text) == "" || !utf8.ValidString(p.Text) || len(utf16.Encode([]rune(p.Text))) > 20000 {
		return c.TurnResult{}, failure(c.ErrValidation)
	}
	tx, err := s.begin(ctx, a.UserID)
	if err != nil {
		return c.TurnResult{}, err
	}
	defer tx.Rollback()
	// Completion retry after a lost acknowledgement replays the stored result.
	r, err := getResult(ctx, tx, `t.id=@p1 AND t.user_id=@p2 AND t.conversation_id=@p3`, a.TurnID, a.UserID, a.ConversationID)
	if err != nil {
		return r, err
	}
	if r.Status == c.TurnCompleted {
		if r.UserMessageID != a.UserMessageID {
			return c.TurnResult{}, failure(c.ErrStateConflict)
		}
		return r, tx.Commit()
	}
	if err = s.check(ctx, tx, a); err != nil {
		return c.TurnResult{}, err
	}
	id := c.NewID()
	r = c.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, AssistantMessageID: &id, Text: norm.NFC.String(p.Text), Status: c.TurnCompleted, Route: p.Route, Degraded: p.Degraded, UI: p.UI, Safety: p.Safety}
	// The replay metadata deliberately excludes text: messages remain its only store.
	meta := r
	meta.Text = ""
	raw, err := json.Marshal(meta)
	if err != nil {
		return c.TurnResult{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT dbo.messages(id,conversation_id,turn_id,role,content) VALUES(@p1,@p2,@p3,N'assistant',@p4)`, id, a.ConversationID, a.TurnID, r.Text)
	if err != nil {
		return c.TurnResult{}, err
	}
	if p.State.Session.PendingQuestion != nil {
		question := *p.State.Session.PendingQuestion
		question.MessageID = id
		p.State.Session.PendingQuestion = &question
	}
	if err = s.state.Apply(ctx, tx, a, p); err != nil {
		return c.TurnResult{}, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.turns SET status=N'completed',assistant_message_id=@p3,completed_at=SYSUTCDATETIME(),result_meta_json=@p4 WHERE id=@p1 AND user_id=@p2;
 UPDATE dbo.conversations SET updated_at=SYSUTCDATETIME() WHERE id=@p5 AND user_id=@p2`, a.TurnID, a.UserID, id, string(raw), a.ConversationID)
	if err != nil {
		return c.TurnResult{}, err
	}
	// Check expiry again after state writes; an expired worker cannot publish.
	if err = s.state.Release(ctx, tx, handle(a)); err != nil {
		return c.TurnResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return c.TurnResult{}, err
	}
	return r, nil
}
func (s *TurnStore) Fail(ctx context.Context, a c.Admission, code c.Code) error {
	return s.finish(ctx, a, code, c.TurnFailed)
}
func (s *TurnStore) Cancel(ctx context.Context, a c.Admission) error {
	return s.finish(ctx, a, c.ErrInternal, c.TurnCancelled)
}
func (s *TurnStore) finish(ctx context.Context, a c.Admission, code c.Code, status c.TurnStatus) error {
	if !code.Valid() {
		return failure(c.ErrValidation)
	}
	tx, err := s.begin(ctx, a.UserID)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var storedStatus string
	var storedCode sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status,error_code FROM dbo.turns WHERE id=@p1 AND user_id=@p2 AND conversation_id=@p3 AND user_message_id=@p4 AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=turns.conversation_id AND v.user_id=turns.user_id AND v.deleted_at IS NULL) AND lifecycle_version=1`, a.TurnID, a.UserID, a.ConversationID, a.UserMessageID).Scan(&storedStatus, &storedCode)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if storedStatus == string(status) && storedCode.String == string(code) {
		return tx.Commit()
	}
	if err = s.check(ctx, tx, a); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.turns SET status=@p3,error_code=@p4 WHERE id=@p1 AND user_id=@p2`, a.TurnID, a.UserID, string(status), string(code))
	if err != nil {
		return err
	}
	if err = s.state.Release(ctx, tx, handle(a)); err != nil {
		return err
	}
	return tx.Commit()
}

// Retry is an explicit authorized recovery operation. Ordinary duplicate
// admission never silently retries a failed/cancelled turn.
func (s *TurnStore) Retry(ctx context.Context, user, client c.ID) error {
	tx, err := s.begin(ctx, user)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, `UPDATE dbo.turns SET status=N'queued',error_code=NULL WHERE user_id=@p1 AND client_turn_id=@p2 AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=turns.conversation_id AND v.user_id=turns.user_id AND v.deleted_at IS NULL) AND lifecycle_version=1 AND status IN(N'failed',N'cancelled')`, user, client)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return failure(c.ErrStateConflict)
	}
	return tx.Commit()
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getResult(ctx context.Context, q queryer, predicate string, args ...any) (c.TurnResult, error) {
	var r c.TurnResult
	var uid, aid uuid.NullUUID
	var text, meta sql.NullString
	err := q.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),t.id),CONVERT(varchar(36),t.user_message_id),CONVERT(varchar(36),t.assistant_message_id),t.status,m.content,t.result_meta_json FROM dbo.turns t LEFT JOIN dbo.messages m ON m.id=t.assistant_message_id AND m.turn_id=t.id AND m.deleted_at IS NULL JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id AND v.deleted_at IS NULL WHERE NOT EXISTS(SELECT 1 FROM dbo.messages hidden WHERE hidden.turn_id=t.id AND hidden.deleted_at IS NOT NULL) AND `+predicate, args...).Scan(&r.TurnID, &uid, &aid, &r.Status, &text, &meta)
	if errors.Is(err, sql.ErrNoRows) {
		return r, httpx.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if meta.Valid {
		if err = json.Unmarshal([]byte(meta.String), &r); err != nil {
			return c.TurnResult{}, err
		}
	}
	r.UserMessageID = uid.UUID
	if aid.Valid {
		r.AssistantMessageID = &aid.UUID
	}
	r.Text = text.String
	return r, nil
}
func (s *TurnStore) GetTurn(ctx context.Context, user, client c.ID) (c.TurnResult, error) {
	return getResult(ctx, s.db, `t.user_id=@p1 AND t.client_turn_id=@p2 AND t.lifecycle_version=1`, user, client)
}
func (s *TurnStore) Recent(ctx context.Context, user, conv c.ID, limit int) ([]c.Message, error) {
	if err := s.ValidateOwner(ctx, user, conv); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 200 {
		return nil, failure(c.ErrValidation)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),CONVERT(varchar(36),turn_id),role,content,created_at,lang FROM (SELECT TOP(@p3) m.id,m.turn_id,m.role,m.content,m.created_at,m.lang,m.message_order FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.id=@p1 AND v.user_id=@p2 AND v.deleted_at IS NULL AND m.deleted_at IS NULL ORDER BY m.message_order DESC) r ORDER BY message_order`, conv, user, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []c.Message{}
	for rows.Next() {
		m := c.Message{UserID: user, ConvID: conv}
		var lang sql.NullString
		if err = rows.Scan(&m.ID, &m.TurnID, &m.Role, &m.Content, &m.CreatedAt, &lang); err != nil {
			return nil, err
		}
		m.Lang = lang.String
		out = append(out, m)
	}
	return out, rows.Err()
}

// Next returns the oldest recoverable request, including its original metadata.
// It does not acquire a lease; call AdmitUserMessage to compete for processing.
func (s *TurnStore) Next(ctx context.Context, user c.ID) (c.TurnRequest, error) {
	var r c.TurnRequest
	var raw, text string
	err := s.db.QueryRowContext(ctx, `SELECT TOP(1) t.request_meta_json,m.content FROM dbo.turns t JOIN dbo.messages m ON m.id=t.user_message_id AND m.turn_id=t.id WHERE t.user_id=@p1 AND t.lifecycle_version=1 AND t.status IN(N'queued',N'running') AND m.deleted_at IS NULL AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=t.conversation_id AND v.user_id=t.user_id AND v.deleted_at IS NULL) ORDER BY t.admission_order`, user).Scan(&raw, &text)
	if errors.Is(err, sql.ErrNoRows) {
		return r, httpx.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal([]byte(raw), &r); err != nil {
		return r, err
	}
	r.Text = text
	return r, nil
}

// CancelQueued allows the owner to withdraw a queued message without waiting
// for another turn's lease. It preserves the raw message and terminal status.
func (s *TurnStore) CancelQueued(ctx context.Context, user, client c.ID) error {
	tx, err := s.begin(ctx, user)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE dbo.turns SET status=N'cancelled',error_code=NULL WHERE user_id=@p1 AND client_turn_id=@p2 AND EXISTS(SELECT 1 FROM dbo.conversations v WHERE v.id=turns.conversation_id AND v.user_id=turns.user_id AND v.deleted_at IS NULL) AND lifecycle_version=1 AND status=N'queued'`, user, client)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return failure(c.ErrStateConflict)
	}
	return tx.Commit()
}
