package services

import (
	"intellisearch/internal/repositories"
	"fmt"
	"sort"
	"time"
)

// QueueMetricsProvider lets the stats panel read live queue health without the
// services layer depending on handlers (the AI handler implements it).
type QueueMetricsProvider interface {
	QueueMetrics() QueueHealth
}

// QueueHealth mirrors the AI handler's runtime queue counters.
type QueueHealth struct {
	QueueDepth    int   `json:"queueDepth"`
	InFlight      int64 `json:"inFlight"`
	Rejected      int64 `json:"rejected"`
	MaxConcurrent int   `json:"maxConcurrent"`
}

type TopQuery struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

type PerUserUsage struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Count  int64  `json:"count"`
}

type UserStats struct {
	QuestionsToday  int64           `json:"questionsToday"`
	QuestionsWeek   int64           `json:"questionsWeek"`
	ActiveUsersWeek int64           `json:"activeUsersWeek"`
	FailedSince     int64           `json:"failed"`
	TopQueries      []TopQuery      `json:"topQueries"`
	PerUserUsage    []PerUserUsage  `json:"perUserUsage"`
}

type ProviderPerformance struct {
	ProviderID string  `json:"providerId"`
	Name       string  `json:"name"`
	Model      string  `json:"model"`
	Successes  int64   `json:"successes"`
	Total      int64   `json:"total"`
	Rate       float64 `json:"rate"`
}

type ErrorGroupView struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Count        int64  `json:"count"`
	LastSeen     string `json:"lastSeen"`
}

type LatencyStats struct {
	AverageMS int `json:"averageMs"`
	P50       int `json:"p50"`
	P95       int `json:"p95"`
	P99       int `json:"p99"`
}

type AIStats struct {
	TotalCompleted int64                  `json:"totalCompleted"`
	TotalFailed    int64                  `json:"totalFailed"`
	SuccessRate    float64                `json:"successRate"`
	Errors         []ErrorGroupView       `json:"errors"`
	Latency        LatencyStats           `json:"latency"`
	Providers      []ProviderPerformance  `json:"providers"`
	Queue          QueueHealth            `json:"queue"`
}

type TrendPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type Trends struct {
	Daily  []TrendPoint `json:"daily"`
	Weekly []TrendPoint `json:"weekly"`
}

type StatsService struct {
	usage     *repositories.UsageLogRepository
	users     *repositories.UserRepository
	providers *repositories.ProviderRepository
	queue     QueueMetricsProvider
}

func NewStatsService(usage *repositories.UsageLogRepository, users *repositories.UserRepository, providers *repositories.ProviderRepository, queue QueueMetricsProvider) *StatsService {
	return &StatsService{usage: usage, users: users, providers: providers, queue: queue}
}

// UserStats aggregates usage over the current UTC day and the trailing 7 days.
func (s *StatsService) UserStats() (UserStats, error) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekAgo := today.AddDate(0, 0, -7)

	questionsToday, err := s.usage.CountBetween(today, now)
	if err != nil {
		return UserStats{}, err
	}
	questionsWeek, err := s.usage.CountBetween(weekAgo, now)
	if err != nil {
		return UserStats{}, err
	}
	activeWeek, err := s.usage.ActiveUsersSince(weekAgo)
	if err != nil {
		return UserStats{}, err
	}
	failed, err := s.usage.CountFailedSince(today)
	if err != nil {
		return UserStats{}, err
	}
	top, err := s.usage.TopQueries(10)
	if err != nil {
		return UserStats{}, err
	}
	perUser, err := s.usage.PerUserUsage()
	if err != nil {
		return UserStats{}, err
	}
	topQueries := make([]TopQuery, 0, len(top))
	for _, row := range top {
		topQueries = append(topQueries, TopQuery{Query: row.Query, Count: row.Count})
	}
	perUserUsage := make([]PerUserUsage, 0, len(perUser))
	for _, row := range perUser {
		user, err := s.users.ByID(row.UserID)
		if err != nil {
			continue
		}
		perUserUsage = append(perUserUsage, PerUserUsage{UserID: row.UserID.String(), Name: user.Name, Email: user.Email, Count: row.Count})
	}
	return UserStats{
		QuestionsToday:  questionsToday,
		QuestionsWeek:   questionsWeek,
		ActiveUsersWeek: activeWeek,
		FailedSince:     failed,
		TopQueries:      topQueries,
		PerUserUsage:    perUserUsage,
	}, nil
}

// AIStats reports success/failure, error groups, latency percentiles, per-provider
// performance, and live queue health.
func (s *StatsService) AIStats(filterType string) (AIStats, error) {
	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7)
	stats := AIStats{Queue: s.queueMetrics()}

	latencies, err := s.usage.Latencies(weekAgo)
	if err != nil {
		return stats, err
	}
	stats.Latency = latencyStats(latencies)

	errors, err := s.usage.ErrorGroups(filterType, 25)
	if err != nil {
		return stats, err
	}
	stats.Errors = make([]ErrorGroupView, 0, len(errors))
	for _, group := range errors {
		stats.Errors = append(stats.Errors, ErrorGroupView{ErrorCode: group.ErrorCode, ErrorMessage: group.ErrorMessage, Count: group.Count, LastSeen: group.LastSeen})
	}

	providerStats, err := s.usage.ProviderPerformance()
	if err != nil {
		return stats, err
	}
	providers, err := s.providers.List()
	if err != nil {
		return stats, err
	}
	byID := make(map[string]repositories.ProviderStats, len(providerStats))
	for _, p := range providerStats {
		byID[p.ProviderID.String()] = p
	}
	stats.Providers = make([]ProviderPerformance, 0, len(providers))
	for _, provider := range providers {
		entry := ProviderPerformance{ProviderID: provider.ID.String(), Name: provider.Name, Model: provider.Model}
		if p, ok := byID[provider.ID.String()]; ok {
			entry.Successes = p.Successes
			entry.Total = p.Total
			if p.Total > 0 {
				entry.Rate = float64(p.Successes) / float64(p.Total) * 100
			}
		}
		stats.TotalCompleted += entry.Successes
		stats.TotalFailed += entry.Total - entry.Successes
		stats.Providers = append(stats.Providers, entry)
	}
	if stats.TotalCompleted+stats.TotalFailed > 0 {
		stats.SuccessRate = float64(stats.TotalCompleted) / float64(stats.TotalCompleted+stats.TotalFailed) * 100
	}
	return stats, nil
}

func (s *StatsService) queueMetrics() QueueHealth {
	if s.queue == nil {
		return QueueHealth{}
	}
	return s.queue.QueueMetrics()
}

// Trends buckets usage logs into the trailing 7 days and 8 weeks (labelled
// "YYYY-MM-DD" / "YYYY-WW") for the statistics panel's charts.
func (s *StatsService) Trends() (Trends, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -7*8)
	times, err := s.usage.CreatedSince(start)
	if err != nil {
		return Trends{}, err
	}
	daily := make([]TrendPoint, 0, 7)
	weekly := make([]TrendPoint, 0, 8)
	for day := 0; day < 7; day++ {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, day-6)
		daily = append(daily, TrendPoint{Label: dayStart.Format("2006-01-02"), Count: countWithin(times, dayStart, dayStart.AddDate(0, 0, 1))})
	}
	for week := 0; week < 8; week++ {
		weekStart := weekStartUTC(now).AddDate(0, 0, -7*(7-week))
		label := weekStart.Format("2006") + "-W" + fmt.Sprintf("%02d", weekNumber(weekStart))
		weekly = append(weekly, TrendPoint{Label: label, Count: countWithin(times, weekStart, weekStart.AddDate(0, 0, 7))})
	}
	return Trends{Daily: daily, Weekly: weekly}, nil
}

func countWithin(times []time.Time, start, end time.Time) int64 {
	var count int64
	for _, t := range times {
		if !t.Before(start) && t.Before(end) {
			count++
		}
	}
	return count
}

// weekStartUTC returns the Monday of the current UTC week.
func weekStartUTC(now time.Time) time.Time {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(day.Weekday()) + 6) % 7 // Monday = 0
	return day.AddDate(0, 0, -offset)
}

// weekNumber computes the ISO-like week number for a Monday-based week.
func weekNumber(day time.Time) int {
	weekDay := int(day.Weekday())
	yearStart := time.Date(day.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	dayOfYear := day.YearDay()
	weekdayOfJan1 := (int(yearStart.Weekday()) + 6) % 7
	week := (dayOfYear + weekdayOfJan1 - 1) / 7
	if weekDay == 0 { // Sunday -> belongs to the week that started the prior Monday
		week--
	}
	if week < 1 {
		week = 1
	}
	return week
}

// latencyStats computes average and percentile latency from a sorted slice.
func latencyStats(values []int) LatencyStats {
	if len(values) == 0 {
		return LatencyStats{}
	}
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)
	total := 0
	for _, v := range values {
		total += v
	}
	return LatencyStats{
		AverageMS: total / len(values),
		P50:       percentile(sorted, 0.50),
		P95:       percentile(sorted, 0.95),
		P99:       percentile(sorted, 0.99),
	}
}

func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	index := int(p * float64(len(sorted)-1))
	return sorted[index]
}