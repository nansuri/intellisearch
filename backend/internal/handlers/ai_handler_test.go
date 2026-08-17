package handlers

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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
	if err := db.AutoMigrate(&entities.User{}, &entities.AIQueueConfig{}, &entities.UsageLog{}, &entities.ChatSession{}, &entities.Message{}, &entities.SearchResult{}); err != nil {
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
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), allowLimiter{}, nil)
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
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), denyLimiter{}, nil)
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
	handler := NewAIHandler(&fakeRunner{}, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), allowLimiter{}, nil)
	defer handler.Stop()
	_, err := handler.enqueue(context.Background(), services.AskInput{Query: "today's question", UserID: &userID, IP: "ip"}, 10, time.Minute)
	if !errors.Is(err, services.ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestEnqueueSuccess(t *testing.T) {
	db := handlerTestDB(t)
	seedQueueConfig(t, db, entities.AIQueueConfig{MaxConcurrent: 2, MaxQueueSize: 5, RequestTimeoutMS: 60000, PerUserRateLimit: 10})
	runner := &fakeRunner{result: services.AskResult{SessionID: uuid.New(), Answer: "an answer"}}
	handler := NewAIHandler(runner, repositories.NewQueueConfigRepository(db), repositories.NewUserRepository(db), repositories.NewUsageLogRepository(db), allowLimiter{}, nil)
	defer handler.Stop()
	result, err := handler.enqueue(context.Background(), services.AskInput{Query: "hello", IP: "1.2.3.4"}, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if result.Answer != "an answer" || runner.calls.Load() != 1 {
		t.Fatalf("unexpected result %#v, calls %d", result, runner.calls.Load())
	}
}
