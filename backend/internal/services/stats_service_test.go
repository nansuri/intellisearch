package services

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"intellisearch/internal/models/entities"
	"intellisearch/internal/repositories"
)

func TestMaskQuery(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"ai", "•"},
		{"best ramen tokyo", "b***************"},
		{"  padded query  ", "p***********"},
		{"Crypto news today", "C****************"},
	}
	for _, tc := range cases {
		if got := MaskQuery(tc.in); got != tc.want {
			t.Errorf("MaskQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUserStatsMasksTopQueries(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()
	usage := repositories.NewUsageLogRepository(db)
	uid := uuid.New()
	for i, q := range []string{"best ramen in tokyo", "best ramen in tokyo", "cheapest flights"} {
		if err := usage.Create(&entities.UsageLog{ID: uint64(i + 1), UserID: &uid, Query: q, Status: "completed", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	stats := NewStatsService(usage, repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil)
	result, err := stats.UserStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TopQueries) != 2 {
		t.Fatalf("expected 2 top queries, got %d", len(result.TopQueries))
	}
	for _, row := range result.TopQueries {
		if strings.Contains(row.Query, "ramen") || strings.Contains(row.Query, "flights") {
			t.Fatalf("raw query leaked into admin stats: %q", row.Query)
		}
		if row.Query == "" || strings.Contains(row.Query, " ") {
			t.Fatalf("unexpected masked query %q", row.Query)
		}
	}
	if result.TopQueries[0].Count != 2 {
		t.Fatalf("expected count 2 for the repeated query, got %d", result.TopQueries[0].Count)
	}
}

func TestTrendingWordsAggregatesTermsPerBucket(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()
	usage := repositories.NewUsageLogRepository(db)
	uid := uuid.New()
	// Two queries today sharing the term "tokyo"; one yesterday with a term
	// that must not pollute today's bucket. created_at of "yesterday" lands in
	// a different daily bucket.
	yesterday := now.AddDate(0, 0, -1)
	for i, q := range []string{"best ramen tokyo", "tokyo nightlife guide", "how to cook pasta"} {
		created := now
		if i == 2 {
			created = yesterday
		}
		if err := usage.Create(&entities.UsageLog{ID: uint64(i + 1), UserID: &uid, Query: q, Status: "completed", CreatedAt: created}); err != nil {
			t.Fatal(err)
		}
	}
	stats := NewStatsService(usage, repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil)
	result, err := stats.TrendingWords("daily")
	if err != nil {
		t.Fatal(err)
	}
	if result.Window != "daily" || len(result.Buckets) != 7 {
		t.Fatalf("unexpected window/buckets: %s/%d", result.Window, len(result.Buckets))
	}
	// Today's bucket (index 6) has "tokyo" twice — the top term.
	today := result.Buckets[6]
	if len(today.Top) == 0 || today.Top[0].Word != "tokyo" || today.Top[0].Count != 2 {
		t.Fatalf("expected tokyo (2) at top of today's bucket, got %+v", today.Top)
	}
	// The stopword "how" and short word "to" are filtered; "pasta" lands in
	// yesterday's bucket only.
	for _, bucket := range result.Buckets {
		for _, term := range bucket.Top {
			if term.Word == "how" || term.Word == "to" || term.Word == "pasta" && bucket.Label == today.Label {
				t.Fatalf("unexpected term %q in bucket %s", term.Word, bucket.Label)
			}
		}
	}
	// Overall counts: tokyo 2, ramen 1, nightlife 1, guide 1, cook 1, pasta 1.
	overall := map[string]int64{}
	for _, term := range result.Overall {
		overall[term.Word] = term.Count
	}
	if overall["tokyo"] != 2 || overall["pasta"] != 1 {
		t.Fatalf("unexpected overall terms: %+v", overall)
	}
}

func TestTrendingWordsWeeklyWindow(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().UTC()
	usage := repositories.NewUsageLogRepository(db)
	uid := uuid.New()
	for i, q := range []string{"best ramen tokyo", "tokyo nightlife"} {
		if err := usage.Create(&entities.UsageLog{ID: uint64(i + 1), UserID: &uid, Query: q, Status: "completed", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	stats := NewStatsService(usage, repositories.NewUserRepository(db), repositories.NewProviderRepository(db), nil)
	result, err := stats.TrendingWords("weekly")
	if err != nil {
		t.Fatal(err)
	}
	if result.Window != "weekly" || len(result.Buckets) != 8 {
		t.Fatalf("unexpected window/buckets: %s/%d", result.Window, len(result.Buckets))
	}
	if result.Buckets[7].Top[0].Word != "tokyo" || result.Buckets[7].Top[0].Count != 2 {
		t.Fatalf("expected tokyo (2) in current week, got %+v", result.Buckets[7].Top)
	}
}
