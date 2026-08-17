package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"intellisearch/internal/services"
)

type allowLimiter struct{}
type denyLimiter struct{}

func (allowLimiter) Allow(context.Context, string, string, int, time.Duration) (bool, error) { return true, nil }
func (denyLimiter) Allow(context.Context, string, string, int, time.Duration) (bool, error)  { return false, nil }

type fakeRunner struct {
	release chan struct{}
	calls   atomic.Int64
	result  services.AskResult
	err     error
}

func (r *fakeRunner) Answer(ctx context.Context, input services.AskInput) (services.AskResult, error) {
	r.calls.Add(1)
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return services.AskResult{}, ctx.Err()
		}
	}
	if r.err != nil {
		return services.AskResult{}, r.err
	}
	return r.result, nil
}

func handlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:ai-handler-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&entities.User{}, &entities.AIQueueConfig{}, &entities.UsageLog{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}, &entities.ImageResult{}, &entities.AnonymousUsage{}, &entities.Note{}, &entities.MapPoint{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedQueueConfig(t *testing.T, db *gorm.DB, config entities.AIQueueConfig) {
	t.Helper()
	config.ID = 1
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueQueueOverflow(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 0, MaxQueueSize: 1, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowLimiter{}, nil)
	defer handler.Stop()

	// With zero workers and a queue of one, the first job sits queued forever
	// (no worker to pick it up), so the second must overflow.
	go func() {
		_, _ = handler.enqueue(context.Background(), services.AskInput{Query: "first"}, 10, time.Minute)
	}()
	time.Sleep(50 * time.Millisecond)
	_, err := handler.enqueue(context.Background(), services.AskInput{Query: "second"}, 10, time.Minute)
	if !errors.Is(err, services.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
	if handler.Metrics().Rejected != 1 {
		t.Fatalf("expected 1 rejected request, got %d", handler.Metrics().Rejected)
	}
}

func TestEnqueueRateLimited(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 1, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), denyLimiter{}, nil)
	defer handler.Stop()
	_, err := handler.enqueue(context.Background(), services.AskInput{Query: "hello", IP: "1.2.3.4"}, 10, time.Minute)
	if !errors.Is(err, services.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestEnqueueDailyQuotaExceeded(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 1, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	userID := uuid.New()
	if err := db.Create(&entities.User{ID: userID, Name: "User", Email: "u@example.com", PasswordHash: "x", Role: entities.RoleGeneralUser, Status: entities.StatusActive, AIDailyQuota: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entities.UsageLog{ID: 1, UserID: &userID, Query: "earlier", Status: entities.MessageStatusCompleted, CreatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowLimiter{}, nil)
	defer handler.Stop()
	_, err := handler.enqueue(context.Background(), services.AskInput{Query: "today's question", UserID: &userID, IP: "ip"}, 10, time.Minute)
	if !errors.Is(err, services.ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
}

// anonymousAsk performs one HTTP ask against the handler and returns the
// recorder plus the issued visitorId (empty on rejection).
func anonymousAsk(t *testing.T, handler *AIHandler, method, path, ip, body, visitor string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":1234"
	if visitor != "" {
		req.Header.Set("X-Visitor-ID", visitor)
	}
	c.Request = req
	switch method + " " + path {
	case "POST /api/v1/ask/url":
		handler.AskURL(c)
	default:
		handler.Ask(c)
	}
	var envelope struct {
		Data struct {
			VisitorID string `json:"visitorId"`
		} `json:"data"`
		ErrorCode string `json:"errorCode"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	return w, envelope.Data.VisitorID
}

func newAnonymousTestHandler(t *testing.T) (*AIHandler, *gorm.DB) {
	t.Helper()
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 2, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	runner := &fakeRunner{result: services.AskResult{SessionID: uuid.New(), Answer: "ok"}}
	handler := NewAIHandler(runner, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowLimiter{}, nil)
	t.Cleanup(handler.Stop)
	return handler, db
}

func TestAskAnonymousFirstAskIssuesVisitorToken(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	w, visitor := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.10", `{"query":"hello"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if visitor == "" {
		t.Fatal("expected a visitorId to be issued on the first anonymous ask")
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "visitor_id=") {
		t.Fatalf("expected visitor cookie to be set, got %q", w.Header().Get("Set-Cookie"))
	}
}

func TestAskAnonymousSecondAskRejected(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	visitor := uuid.New().String()
	first, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.11", `{"query":"hello"}`, visitor)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first ask to succeed, got %d", first.Code)
	}
	second, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.11", `{"query":"again"}`, visitor)
	if second.Code != http.StatusTooManyRequests || !strings.Contains(second.Body.String(), "AISY02004") {
		t.Fatalf("expected 429 AISY02004 for repeat visitor, got %d: %s", second.Code, second.Body.String())
	}
}

func TestAskAnonymousClearedTokenStillBlockedByIP(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	first, visitor := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.12", `{"query":"hello"}`, "")
	if first.Code != http.StatusOK || visitor == "" {
		t.Fatalf("expected first ask to succeed with a visitor, got %d", first.Code)
	}
	// Same IP, token cleared (no X-Visitor-ID): the per-IP claim still blocks.
	second, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.12", `{"query":"again"}`, "")
	if second.Code != http.StatusTooManyRequests || !strings.Contains(second.Body.String(), "AISY02004") {
		t.Fatalf("expected 429 AISY02004 for reused IP, got %d: %s", second.Code, second.Body.String())
	}
}

func TestAskAnonymousDifferentIPGetsFreshAllowance(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	first, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.13", `{"query":"hello"}`, "")
	if first.Code != http.StatusOK {
		t.Fatalf("expected first ask to succeed, got %d", first.Code)
	}
	second, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.14", `{"query":"hello again"}`, "")
	if second.Code != http.StatusOK {
		t.Fatalf("a fresh IP should get its own allowance, got %d: %s", second.Code, second.Body.String())
	}
}

func TestAskAnonymousSearchModeDoesNotConsumeAllowance(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	search, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.15", `{"query":"raw results","mode":"search"}`, "")
	if search.Code != http.StatusOK {
		t.Fatalf("search mode should always be allowed, got %d", search.Code)
	}
	enhanced, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.15", `{"query":"ai answer"}`, "")
	if enhanced.Code != http.StatusOK {
		t.Fatalf("search mode must not consume the AI allowance, got %d: %s", enhanced.Code, enhanced.Body.String())
	}
	blocked, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.15", `{"query":"one more"}`, "")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("second AI ask should be blocked, got %d", blocked.Code)
	}
}

func TestAskAnonymousFailureReleasesAllowance(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 2, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	runner := &fakeRunner{result: services.AskResult{SessionID: uuid.New(), Answer: "ok"}, err: services.ErrAIUnavailable}
	handler := NewAIHandler(runner, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowLimiter{}, nil)
	t.Cleanup(handler.Stop)
	// The first ask fails (provider down): the allowance must NOT be burned.
	first, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.30", `{"query":"will fail"}`, "")
	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for provider down, got %d", first.Code)
	}
	// Provider recovers: the same IP is allowed again (claim was released).
	runner.err = nil
	second, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.30", `{"query":"now ok"}`, "")
	if second.Code != http.StatusOK {
		t.Fatalf("failed asks must not consume the allowance, got %d: %s", second.Code, second.Body.String())
	}
	// The successful ask consumed it: a third ask is now blocked.
	third, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask", "203.0.113.30", `{"query":"blocked"}`, "")
	if third.Code != http.StatusTooManyRequests || !strings.Contains(third.Body.String(), "AISY02004") {
		t.Fatalf("expected 429 AISY02004 after the successful ask, got %d", third.Code)
	}
}

func TestAskURLAnonymousConsumesAllowance(t *testing.T) {
	handler, _ := newAnonymousTestHandler(t)
	first, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask/url", "203.0.113.16", `{"url":"https://example.com/a"}`, "")
	if first.Code != http.StatusOK {
		t.Fatalf("expected first URL ask to succeed, got %d: %s", first.Code, first.Body.String())
	}
	second, _ := anonymousAsk(t, handler, "POST", "/api/v1/ask/url", "203.0.113.16", `{"url":"https://example.com/b"}`, "")
	if second.Code != http.StatusTooManyRequests || !strings.Contains(second.Body.String(), "AISY02004") {
		t.Fatalf("expected 429 AISY02004 for second URL ask, got %d: %s", second.Code, second.Body.String())
	}
}

func TestEnqueueSuccess(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 2, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	runner := &fakeRunner{result: services.AskResult{SessionID: uuid.New(), Answer: "an answer"}}
	handler := NewAIHandler(runner, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), repositories.NewAnonymousUsageRepository(db), allowLimiter{}, nil)
	defer handler.Stop()
	result, err := handler.enqueue(context.Background(), services.AskInput{Query: "hello", IP: "1.2.3.4"}, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "an answer" || runner.calls.Load() != 1 {
		t.Fatalf("unexpected result %#v, calls %d", result, runner.calls.Load())
	}
}
