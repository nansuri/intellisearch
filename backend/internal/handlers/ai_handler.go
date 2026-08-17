package handlers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

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
}

// AIHandler is the single entry point for all AI work. It owns a worker pool
// and a bounded queue whose knobs come from ai_queue_config (reloaded with a
// short TTL so Owner Control Panel changes apply without redeploying).
type AIHandler struct {
	service   aiRunner
	queueCfg  *repositories.QueueConfigRepository
	users     *repositories.UserRepository
	usageLogs *repositories.UsageLogRepository
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

type job struct {
	ctx     context.Context
	input   services.AskInput
	timeout time.Duration
	resp    chan jobResult
}

type jobResult struct {
	result services.AskResult
	err    error
}

func NewAIHandler(service aiRunner, queueCfg *repositories.QueueConfigRepository, users *repositories.UserRepository, usageLogs *repositories.UsageLogRepository, limiter services.Limiter, auth *services.AuthService) *AIHandler {
	handler := &AIHandler{service: service, queueCfg: queueCfg, users: users, usageLogs: usageLogs, limiter: limiter, auth: auth, stop: make(chan struct{})}
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
			result, err := h.service.Answer(ctx, current.input)
			cancel()
			current.resp <- jobResult{result: result, err: err}
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
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.AISY01004, "Ask a question first."))
		return
	}
	userID, err := h.optionalUser(c)
	if err != nil {
		middleware.JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01002, "Your session is invalid or has expired."))
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
	config, _ := h.currentConfig()
	result, err := h.enqueue(c.Request.Context(), services.AskInput{Query: request.Query, SessionID: request.SessionID, UserID: userID, IP: c.ClientIP(), Location: location, Mode: mode}, config.PerUserRateLimit, time.Minute)
	h.respond(c, result, err)
}

func (h *AIHandler) AskURL(c *gin.Context) {
	var request struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		middleware.JSON(c, http.StatusBadRequest, contracts.Fail(contracts.AISY03003, "Submit a valid URL."))
		return
	}
	userID, err := h.optionalUser(c)
	if err != nil {
		middleware.JSON(c, http.StatusUnauthorized, contracts.Fail(contracts.AUTH01002, "Your session is invalid or has expired."))
		return
	}
	// URL submissions are rate-limited much more strictly (5 per hour).
	result, err := h.enqueue(c.Request.Context(), services.AskInput{Query: "Summarize this page", URL: request.URL, UserID: userID, IP: c.ClientIP()}, 5, time.Hour)
	h.respond(c, result, err)
}

// enqueue applies per-user rate limiting and the daily quota, then submits the
// job to the bounded queue. Overflow is rejected with the friendly busy error.
func (h *AIHandler) enqueue(ctx context.Context, input services.AskInput, limitMax int, window time.Duration) (services.AskResult, error) {
	if services.SanitizeQuery(input.Query) == "" {
		return services.AskResult{}, services.ErrInvalidQuery
	}
	key := input.IP
	if key == "" {
		key = "anonymous"
	}
	allowed, err := h.limiter.Allow(ctx, "ask", key, limitMax, window)
	if err != nil {
		logrus.WithError(err).WithField("scope", "ask").Error("rate limiter unavailable; allowing request")
	} else if !allowed {
		h.rejected.Add(1)
		return services.AskResult{}, services.ErrRateLimited
	}
	if input.UserID != nil {
		if err := h.checkDailyQuota(ctx, *input.UserID); err != nil {
			return services.AskResult{}, err
		}
	}
	config, _ := h.currentConfig()
	if len(h.jobs) >= config.MaxQueueSize {
		h.rejected.Add(1)
		return services.AskResult{}, services.ErrQueueFull
	}
	h.ensureWorkers(config.MaxConcurrent)
	current := job{ctx: ctx, input: input, timeout: time.Duration(config.RequestTimeoutMS) * time.Millisecond, resp: make(chan jobResult, 1)}
	select {
	case h.jobs <- current:
	default:
		h.rejected.Add(1)
		return services.AskResult{}, services.ErrQueueFull
	}
	select {
	case done := <-current.resp:
		return done.result, done.err
	case <-ctx.Done():
		return services.AskResult{}, ctx.Err()
	}
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
		logrus.WithError(err).WithField("errorCode", code).Error("ask job failed")
		middleware.JSON(c, status, contracts.Fail(code, services.SanitizedErrorMessage(err)))
		return
	}
	middleware.JSON(c, http.StatusOK, contracts.OK(result))
}

func (h *AIHandler) errorResponse(err error) (string, int) {
	code := services.CodeForError(err)
	switch code {
	case "AISY02001", "AISY02002", "AISY02003":
		return code, http.StatusTooManyRequests
	case "AISY01004", "AISY03003":
		return code, http.StatusBadRequest
	case "AISY03002":
		return code, http.StatusForbidden
	case "AISY01002":
		return code, http.StatusGatewayTimeout
	case "AISY01003", "AISY03004":
		return code, http.StatusBadGateway
	case "AISY01001":
		return code, http.StatusServiceUnavailable
	default:
		return code, http.StatusInternalServerError
	}
}
