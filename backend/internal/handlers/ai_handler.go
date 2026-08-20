package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"intellisearch/internal/contracts"
	"intellisearch/internal/middleware"
	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
	"intellisearch/internal/services"
)

// aiRunner is the pipeline contract the pool executes; services.AIService
// satisfies it and tests substitute a fake.
type aiRunner interface {
	Answer(ctx context.Context, input services.AskInput) (services.AskResult, error)
	// SuggestFollowUps is trigger-driven follow-up question composition for an
	// existing session (runs only when the user asks for it — token-saving).
	SuggestFollowUps(ctx context.Context, userID *uuid.UUID, sessionID uuid.UUID) ([]string, error)
	// GenerateMiniApp composes a complete mini app (HTML/CSS/JS) from a prompt,
	// attributed to the user so generation counts against their daily quota.
	GenerateMiniApp(ctx context.Context, userID uuid.UUID, prompt string) (services.MiniAppDraft, error)
}

// AIHandler is the single entry point for all AI work. It owns a worker pool
// and a bounded queue whose knobs come from ai_queue_config (reloaded with a
// short TTL so Owner Control Panel changes apply without redeploying).
// visitorCookie is the httpOnly guest token: it survives localStorage clears,
// so a visitor who deletes the frontend-stored token is still recognized and
// their single AI allowance stays consumed.
const visitorCookie = "visitor_id"

type AIHandler struct {
	service   aiRunner
	queueCfg  *repositories.QueueConfigRepository
	users     *repositories.UserRepository
	usageLogs *repositories.UsageLogRepository
	anonymous *repositories.AnonymousUsageRepository
	limiter   services.Limiter
	auth      *services.AuthService

	jobs      chan job
	workers   atomic.Int64
	inflight  atomic.Int64
	rejected  atomic.Int64
	stopOnce  sync.Once
	stop      chan struct{}
	wg        sync.WaitGroup
	cfgMu     sync.Mutex
	cfg       entities.AIQueueConfig
	cfgLoaded bool
	cfgAt     time.Time
}

// job is one unit of AI work on the bounded queue. run is a closure capturing
// the pipeline inputs so one generic queue can serve asks, URL asks, and
// mini-app generation through the same worker pool.
type job struct {
	ctx     context.Context
	timeout time.Duration
	run     func(ctx context.Context) (any, error)
	resp    chan jobResult
}

type jobResult struct {
	value any
	err   error
}

func NewAIHandler(service aiRunner, queueCfg *repositories.QueueConfigRepository, users *repositories.UserRepository, usageLogs *repositories.UsageLogRepository, anonymous *repositories.AnonymousUsageRepository, limiter services.Limiter, auth *services.AuthService) *AIHandler {
	handler := &AIHandler{service: service, queueCfg: queueCfg, users: users, usageLogs: usageLogs, anonymous: anonymous, limiter: limiter, auth: auth, stop: make(chan struct{})}
	// The seed guarantees sane defaults; if the first read fails, currentConfig
	// returns a fallback so the pool still starts.
	config, _ := handler.currentConfig()
	if config.MaxQueueSize < 1 {
		config.MaxQueueSize = 1
	}
	handler.jobs = make(chan job, config.MaxQueueSize)
	handler.ensureWorkers(config.MaxConcurrent)
	return handler
}

// currentConfig returns the queue knobs, cached for five seconds so admin edits
// take effect without restarting.
func (h *AIHandler) currentConfig() (entities.AIQueueConfig, error) {
	h.cfgMu.Lock()
	defer h.cfgMu.Unlock()
	if h.cfgLoaded && time.Since(h.cfgAt) < 5*time.Second {
		return h.cfg, nil
	}
	config, err := h.queueCfg.Get()
	if err != nil {
		logrus.WithError(err).Error("ai queue config read failed; using fallback")
		if !h.cfgLoaded {
			return entities.AIQueueConfig{MaxConcurrent: 4, MaxQueueSize: 20, RequestTimeoutMS: 60000, PerUserRateLimit: 10}, err
		}
		return h.cfg, err
	}
	h.cfg, h.cfgLoaded, h.cfgAt = config, true, time.Now()
	return config, nil
}

// ensureWorkers grows the pool up to target concurrent workers. The pool is
// grow-only: shrinking max_concurrent applies to new jobs via the queue-depth
// gate, while already-running workers finish their current job.
func (h *AIHandler) ensureWorkers(target int) {
	for h.workers.Load() < int64(target) {
		h.wg.Add(1)
		h.workers.Add(1)
		go h.worker()
	}
}

func (h *AIHandler) worker() {
	defer h.wg.Done()
	for {
		select {
		case <-h.stop:
			return
		case current := <-h.jobs:
			h.inflight.Add(1)
			ctx, cancel := context.WithTimeout(current.ctx, current.timeout)
			value, err := current.run(ctx)
			cancel()
			current.resp <- jobResult{value: value, err: err}
			h.inflight.Add(-1)
		}
	}
}

// Stop drains the pool (for tests and graceful shutdown).
func (h *AIHandler) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
		h.wg.Wait()
	})
}

// Metrics mirrors current queue health for the admin AI-statistics endpoint.
type Metrics struct {
	QueueDepth    int   `json:"queueDepth"`
	InFlight      int64 `json:"inFlight"`
	Rejected      int64 `json:"rejected"`
	MaxConcurrent int   `json:"maxConcurrent"`
}

func (h *AIHandler) Metrics() Metrics {
	config, _ := h.currentConfig()
	return Metrics{QueueDepth: len(h.jobs), InFlight: h.inflight.Load(), Rejected: h.rejected.Load(), MaxConcurrent: config.MaxConcurrent}
}

// QueueMetrics implements services.QueueMetricsProvider so the admin AI-stats
// panel can render live queue health without handlers importing services twice.
func (h *AIHandler) QueueMetrics() services.QueueHealth {
	metrics := h.Metrics()
	return services.QueueHealth{QueueDepth: metrics.QueueDepth, InFlight: metrics.InFlight, Rejected: metrics.Rejected, MaxConcurrent: metrics.MaxConcurrent}
}

func (h *AIHandler) Ask(c *gin.Context) {
	var request struct {
		Query     string                `json:"query"`
		SessionID *uuid.UUID            `json:"sessionId"`
		Location  *services.GeoLocation `json:"location"`
		Mode      string                `json:"mode"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.AISY01004, "Ask a question first.", "parse ask request", err)
		return
	}
	userID, err := h.optionalUser(c)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "unauthorized ask caller", err)
		return
	}
	var location *services.GeoLocation
	if request.Location != nil && services.ValidateGeoLocation(*request.Location) {
		location = request.Location
	} else if request.Location != nil {
		logrus.WithFields(logrus.Fields{"latitude": request.Location.Latitude, "longitude": request.Location.Longitude}).Warn("invalid device location ignored")
	}
	// Ask modes: "search" returns raw web results (no LLM), everything else
	// falls back to the full "enhanced" pipeline for backward compatibility.
	mode := services.ModeEnhanced
	if request.Mode == services.ModeSearch {
		mode = services.ModeSearch
	}
	// Anonymous guests get exactly one AI search (mode=search is free — it runs
	// no LLM). The gate issues the visitor token on the first ask and rejects
	// repeat use, backed by a per-IP claim so clearing cookies/storage cannot
	// reset the allowance.
	var visitorID *uuid.UUID
	if userID == nil && mode == services.ModeEnhanced {
		visitorID, err = h.gateAnonymous(c)
		if err != nil {
			h.respond(c, services.AskResult{}, err)
			return
		}
	}
	config, _ := h.currentConfig()
	result, err := h.enqueue(c.Request.Context(), services.AskInput{Query: request.Query, SessionID: request.SessionID, UserID: userID, IP: c.ClientIP(), Location: location, Mode: mode}, config.PerUserRateLimit, time.Minute)
	h.settleAnonymousClaim(visitorID, err, &result)
	h.respond(c, result, err)
}

func (h *AIHandler) AskURL(c *gin.Context) {
	var request struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, contracts.AISY03003, "Submit a valid URL.", "parse ask-url request", err)
		return
	}
	userID, err := h.optionalUser(c)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "unauthorized ask-url caller", err)
		return
	}
	// URL submissions are rate-limited much more strictly (5 per hour), and for
	// anonymous guests they also consume the single AI-usage allowance.
	var visitorID *uuid.UUID
	if userID == nil {
		visitorID, err = h.gateAnonymous(c)
		if err != nil {
			h.respond(c, services.AskResult{}, err)
			return
		}
	}
	result, err := h.enqueue(c.Request.Context(), services.AskInput{Query: "Summarize this page", URL: request.URL, UserID: userID, IP: c.ClientIP()}, 5, time.Hour)
	h.settleAnonymousClaim(visitorID, err, &result)
	h.respond(c, result, err)
}

// SessionSuggestions serves trigger-driven follow-up question suggestions for
// an existing conversation (the AI Summary tab's "Suggest follow-up questions"
// button). It is a lightweight, synchronous LLM call — not an ask-pipeline job —
// so no tokens are spent automatically: the client only requests it on tap. The
// suggestions endpoint is rate-limited more strictly than asks.
func (h *AIHandler) SessionSuggestions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, contracts.SESS01001, "That conversation could not be found.", "parse session id", err)
		return
	}
	userID, err := h.optionalUser(c)
	if err != nil {
		middleware.RespondError(c, http.StatusUnauthorized, contracts.AUTH01002, "Your session is invalid or has expired.", "unauthorized suggestions caller", err)
		return
	}
	key := c.ClientIP()
	if key == "" {
		key = "anonymous"
	}
	if userID != nil {
		key = userID.String()
	}
	allowed, err := h.limiter.Allow(c.Request.Context(), "suggest", key, 3, time.Minute)
	if err != nil {
		logrus.WithError(err).WithField("scope", "suggest").Error("rate limiter unavailable; allowing request")
	} else if !allowed {
		h.rejected.Add(1)
		middleware.RespondError(c, http.StatusTooManyRequests, contracts.AISY02002, services.SanitizedErrorMessage(services.ErrRateLimited), "suggestions rate limited", services.ErrRateLimited)
		return
	}
	suggestions, err := h.service.SuggestFollowUps(c.Request.Context(), userID, id)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSessionNotFound):
			middleware.RespondError(c, http.StatusNotFound, contracts.SESS01001, "That conversation could not be found.", "load session", err)
		case errors.Is(err, services.ErrSessionForbidden):
			middleware.RespondError(c, http.StatusForbidden, contracts.SESS01002, "You don't have access to that conversation.", "session access denied", err)
		default:
			code, status := h.errorResponse(err)
			middleware.RespondError(c, status, code, services.SanitizedErrorMessage(err), "suggestions generation failed", err)
		}
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(gin.H{"suggestions": suggestions}))
}

// enqueue applies per-user rate limiting and the daily quota, then submits the
// ask job to the bounded queue. Overflow is rejected with the friendly busy
// error. It wraps enqueueJob so asks and mini-app generation share the same
// rate-limit/quota/queue gate.
func (h *AIHandler) enqueue(ctx context.Context, input services.AskInput, limitMax int, window time.Duration) (services.AskResult, error) {
	if services.SanitizeQuery(input.Query) == "" {
		return services.AskResult{}, services.ErrInvalidQuery
	}
	key := input.IP
	if key == "" {
		key = "anonymous"
	}
	value, err := h.enqueueJob(ctx, key, limitMax, window, input.UserID, func(c context.Context) (any, error) {
		return h.service.Answer(c, input)
	})
	if err != nil {
		return services.AskResult{}, err
	}
	return value.(services.AskResult), nil
}

// GenerateMiniApp runs an AI mini-app generation job through the same pool,
// queue, rate limit, and daily quota gate as asks. Generation is signed-in
// only (a quota is useless to anonymous callers), so the user id is required.
func (h *AIHandler) GenerateMiniApp(ctx context.Context, userID uuid.UUID, prompt string) (services.MiniAppDraft, error) {
	value, err := h.enqueueJob(ctx, userID.String(), 5, time.Minute, &userID, func(c context.Context) (any, error) {
		return h.service.GenerateMiniApp(c, userID, prompt)
	})
	if err != nil {
		return services.MiniAppDraft{}, err
	}
	return value.(services.MiniAppDraft), nil
}

// enqueueJob is the shared gate for every AI job type: Redis sliding-window
// rate limit, the user's daily question quota, and the bounded queue. The run
// closure carries the actual pipeline work.
func (h *AIHandler) enqueueJob(ctx context.Context, rlKey string, limitMax int, window time.Duration, userID *uuid.UUID, run func(ctx context.Context) (any, error)) (any, error) {
	allowed, err := h.limiter.Allow(ctx, "ask", rlKey, limitMax, window)
	if err != nil {
		logrus.WithError(err).WithField("scope", "ask").Error("rate limiter unavailable; allowing request")
	} else if !allowed {
		h.rejected.Add(1)
		return nil, services.ErrRateLimited
	}
	if userID != nil {
		if err := h.checkDailyQuota(ctx, *userID); err != nil {
			return nil, err
		}
	}
	config, _ := h.currentConfig()
	if len(h.jobs) >= config.MaxQueueSize {
		h.rejected.Add(1)
		return nil, services.ErrQueueFull
	}
	h.ensureWorkers(config.MaxConcurrent)
	current := job{ctx: ctx, timeout: time.Duration(config.RequestTimeoutMS) * time.Millisecond, run: run, resp: make(chan jobResult, 1)}
	select {
	case h.jobs <- current:
	default:
		h.rejected.Add(1)
		return nil, services.ErrQueueFull
	}
	select {
	case done := <-current.resp:
		return done.value, done.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// settleAnonymousClaim finalizes a gated ask: on success the issued visitor
// token is attached to the response so the frontend can persist it; on failure
// the claim is released so the guest's single allowance is only consumed by a
// successful AI usage (a provider outage must not burn it).
func (h *AIHandler) settleAnonymousClaim(visitorID *uuid.UUID, err error, result *services.AskResult) {
	if visitorID == nil || h.anonymous == nil {
		return
	}
	if err != nil {
		if releaseErr := h.anonymous.Release(*visitorID); releaseErr != nil {
			logrus.WithError(releaseErr).WithField("visitorID", *visitorID).Error("anonymous usage release failed")
		}
		return
	}
	result.VisitorID = visitorID
}

// gateAnonymous enforces the one-AI-search allowance for anonymous callers and
// returns the visitor token to attach to the successful response. It is
// DB-backed (independent of Redis), so it stays effective even when the rate
// limiter is degraded. Identity is layered:
//  1. the httpOnly visitor cookie / X-Visitor-ID header — repeat use by the
//     same browser is rejected even after localStorage is cleared;
//  2. a unique per-IP claim — clearing cookies AND storage still cannot reuse
//     an IP that already used its allowance;
//  3. trusted-proxy configuration (see config) prevents forging X-Forwarded-For
//     to fake a fresh IP.
func (h *AIHandler) gateAnonymous(c *gin.Context) (*uuid.UUID, error) {
	if h.anonymous == nil {
		return nil, nil // repo not wired (tests); feature disabled
	}
	raw := c.GetHeader("X-Visitor-ID")
	if raw == "" {
		raw, _ = c.Cookie(visitorCookie)
	}
	visitorID := uuid.Nil
	if parsed, err := uuid.Parse(raw); err == nil {
		visitorID = parsed
	}
	if visitorID == uuid.Nil {
		visitorID = uuid.New()
	}
	if _, err := h.anonymous.ByVisitorID(visitorID); err == nil {
		// This visitor token already used its single allowance.
		return nil, services.ErrAnonymousLimit
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.WithError(err).WithField("visitorID", visitorID).Error("anonymous usage lookup failed")
		return nil, err
	}
	ip := c.ClientIP()
	if ip == "" {
		ip = "unknown"
	}
	winner, err := h.anonymous.Claim(visitorID, repositories.HashIP(ip))
	if err != nil {
		logrus.WithError(err).WithField("ip", repositories.HashIP(ip)).Error("anonymous usage claim failed")
		return nil, err
	}
	if winner.VisitorID != visitorID {
		// The IP was already claimed by another visitor.
		return nil, services.ErrAnonymousLimit
	}
	// Issue the identity cookie (httpOnly, 1 year) so the allowance sticks even
	// if the frontend-stored token is cleared.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(visitorCookie, visitorID.String(), 31536000, "/", "", false, true)
	return &visitorID, nil
}

func (h *AIHandler) checkDailyQuota(ctx context.Context, userID uuid.UUID) error {
	user, err := h.users.ByID(userID)
	if err != nil || user.AIDailyQuota <= 0 {
		return nil // no personal quota configured (0 = unlimited)
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	count, err := h.usageLogs.CountSince(userID, start)
	if err != nil {
		logrus.WithError(err).WithField("userID", userID).Error("daily quota check failed; allowing request")
		return nil
	}
	if count >= int64(user.AIDailyQuota) {
		return services.ErrQuotaExceeded
	}
	return nil
}

// optionalUser resolves the caller when a Bearer token is present; anonymous
// callers (no header) return nil. An invalid token is a hard 401.
func (h *AIHandler) optionalUser(c *gin.Context) (*uuid.UUID, error) {
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if raw == c.GetHeader("Authorization") || raw == "" {
		return nil, nil
	}
	claims, err := h.auth.Parse(raw)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (h *AIHandler) respond(c *gin.Context, result services.AskResult, err error) {
	if err != nil {
		code, status := h.errorResponse(err)
		middleware.RespondError(c, status, code, services.SanitizedErrorMessage(err), "ask job failed", err)
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(result))
}

func (h *AIHandler) errorResponse(err error) (string, int) {
	code := services.CodeForError(err)
	switch code {
	case "AISY02001", "AISY02002", "AISY02003", "AISY02004":
		return code, http.StatusTooManyRequests
	case "AISY01004", "AISY03003":
		return code, http.StatusBadRequest
	case "AISY03002":
		return code, http.StatusForbidden
	case "AISY01002":
		return code, http.StatusGatewayTimeout
	case "AISY01003", "AISY03004", "MINI02001":
		return code, http.StatusBadGateway
	case "AISY01001":
		return code, http.StatusServiceUnavailable
	default:
		return code, http.StatusInternalServerError
	}
}
