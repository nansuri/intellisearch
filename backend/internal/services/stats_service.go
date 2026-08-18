package services

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"intellisearch/internal/repositories"
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
	QuestionsToday       int64          `json:"questionsToday"`
	QuestionsWeek        int64          `json:"questionsWeek"`
	ActiveUsersWeek      int64          `json:"activeUsersWeek"`
	FailedSince          int64          `json:"failed"`
	RegisteredUsers      int64          `json:"registeredUsers"`
	AnonymousVisitors    int64          `json:"anonymousVisitors"`
	RegisterPageVisitors int64          `json:"registerPageVisitors"`
	TopQueries           []TopQuery     `json:"topQueries"`
	PerUserUsage         []PerUserUsage `json:"perUserUsage"`
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
	TotalCompleted   int64                 `json:"totalCompleted"`
	TotalFailed      int64                 `json:"totalFailed"`
	SuccessRate      float64               `json:"successRate"`
	TotalInputTokens int64                 `json:"totalInputTokens"`
	TotalOutputTokens int64                `json:"totalOutputTokens"`
	TokensPerSec     float64               `json:"tokensPerSec"`
	Errors           []ErrorGroupView      `json:"errors"`
	Latency          LatencyStats          `json:"latency"`
	Providers        []ProviderPerformance `json:"providers"`
	Queue            QueueHealth           `json:"queue"`
}

type TrendPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type Trends struct {
	Daily  []TrendPoint `json:"daily"`
	Weekly []TrendPoint `json:"weekly"`
}

// WordCount is one aggregated search term with its frequency. Terms are
// tokenized from queries, so the admin sees word-level trends without any raw
// query text.
type WordCount struct {
	Word  string `json:"word"`
	Count int64  `json:"count"`
}

// WordTrendBucket is the top terms for one time bucket (a day or a week).
type WordTrendBucket struct {
	Label string      `json:"label"`
	Top   []WordCount `json:"top"`
}

// TrendingWords is the word-level trend view for the control panel: per-bucket
// top terms (for a chart) plus the overall top terms for the window. Every
// term is aggregated and lowercased — never a verbatim query.
type TrendingWords struct {
	Window  string            `json:"window"`
	Buckets []WordTrendBucket `json:"buckets"`
	Overall []WordCount       `json:"overall"`
}

type StatsService struct {
	usage         *repositories.UsageLogRepository
	users         *repositories.UserRepository
	providers     *repositories.ProviderRepository
	queue         QueueMetricsProvider
	anonymous     *repositories.AnonymousUsageRepository
	registerVisit *repositories.RegisterVisitRepository
}

func NewStatsService(usage *repositories.UsageLogRepository, users *repositories.UserRepository, providers *repositories.ProviderRepository, queue QueueMetricsProvider, anonymous *repositories.AnonymousUsageRepository, registerVisit *repositories.RegisterVisitRepository) *StatsService {
	return &StatsService{usage: usage, users: users, providers: providers, queue: queue, anonymous: anonymous, registerVisit: registerVisit}
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
	registeredUsers, err := s.userCount()
	if err != nil {
		return UserStats{}, err
	}
	anonymousVisitors, err := s.anonymousCount()
	if err != nil {
		return UserStats{}, err
	}
	registerPageVisitors, err := s.registerVisitCount()
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
		// Privacy: the control panel never sees verbatim queries — only a masked
		// placeholder so counts stay meaningful without revealing user input.
		topQueries = append(topQueries, TopQuery{Query: MaskQuery(row.Query), Count: row.Count})
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
		QuestionsToday:       questionsToday,
		QuestionsWeek:        questionsWeek,
		ActiveUsersWeek:      activeWeek,
		FailedSince:          failed,
		RegisteredUsers:      registeredUsers,
		AnonymousVisitors:    anonymousVisitors,
		RegisterPageVisitors: registerPageVisitors,
		TopQueries:           topQueries,
		PerUserUsage:         perUserUsage,
	}, nil
}

// userCount returns the total registered accounts; nil-safe for tests that
// construct the service without a user repository.
func (s *StatsService) userCount() (int64, error) {
	if s.users == nil {
		return 0, nil
	}
	return s.users.Count()
}

// anonymousCount returns the total unique anonymous AI visitors; nil-safe.
func (s *StatsService) anonymousCount() (int64, error) {
	if s.anonymous == nil {
		return 0, nil
	}
	return s.anonymous.Count()
}

// registerVisitCount returns the total unique register-page visitors; nil-safe.
func (s *StatsService) registerVisitCount() (int64, error) {
	if s.registerVisit == nil {
		return 0, nil
	}
	return s.registerVisit.Count()
}

// VisitorMetric is one visitor dimension: an all-time total plus the trailing
// 7-day/8-week trend so the Unique visitors page can chart activity.
type VisitorMetric struct {
	Total  int64        `json:"total"`
	Daily  []TrendPoint `json:"daily"`
	Weekly []TrendPoint `json:"weekly"`
}

// VisitorStats is the "Unique user / visitor" control-panel summary: how many
// accounts exist, how many are actually using the AI service, how many unique
// anonymous visitors used it, and how many unique visitors opened the register
// page — each with daily/weekly trends for the charts.
type VisitorStats struct {
	RegisteredUsers    VisitorMetric `json:"registeredUsers"`
	ActiveUsers        int64         `json:"activeUsers"`
	ActiveUsers7d      int64         `json:"activeUsers7d"`
	AnonymousVisitors  VisitorMetric `json:"anonymousVisitors"`
	RegisterPageVisits VisitorMetric `json:"registerPageVisits"`
}

// VisitorStats aggregates registered, active, anonymous, and register-page
// visitor numbers over the same trailing window used by the Trends charts.
func (s *StatsService) VisitorStats() (VisitorStats, error) {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -7*8)

	registeredTimes, err := s.userTimes(start)
	if err != nil {
		return VisitorStats{}, err
	}
	anonymousTimes, err := s.anonymousTimes(start)
	if err != nil {
		return VisitorStats{}, err
	}
	visitTimes, err := s.registerVisitTimes(start)
	if err != nil {
		return VisitorStats{}, err
	}
	registeredTotal, err := s.userCount()
	if err != nil {
		return VisitorStats{}, err
	}
	anonymousTotal, err := s.anonymousCount()
	if err != nil {
		return VisitorStats{}, err
	}
	visitTotal, err := s.registerVisitCount()
	if err != nil {
		return VisitorStats{}, err
	}
	activeTotal, err := s.usage.ActiveUsersSince(time.Time{})
	if err != nil {
		return VisitorStats{}, err
	}
	active7d, err := s.usage.ActiveUsersSince(now.AddDate(0, 0, -7))
	if err != nil {
		return VisitorStats{}, err
	}

	return VisitorStats{
		RegisteredUsers:    visitorMetric(registeredTotal, registeredTimes, now),
		ActiveUsers:        activeTotal,
		ActiveUsers7d:      active7d,
		AnonymousVisitors:  visitorMetric(anonymousTotal, anonymousTimes, now),
		RegisterPageVisits: visitorMetric(visitTotal, visitTimes, now),
	}, nil
}

// visitorMetric combines a total with its bucketed daily/weekly trend.
func visitorMetric(total int64, times []time.Time, now time.Time) VisitorMetric {
	daily, weekly := bucketTrends(times, now)
	return VisitorMetric{Total: total, Daily: daily, Weekly: weekly}
}

// userTimes returns new-account created_at timestamps since start (nil-safe).
func (s *StatsService) userTimes(start time.Time) ([]time.Time, error) {
	if s.users == nil {
		return nil, nil
	}
	return s.users.CreatedSince(start)
}

// anonymousTimes returns anonymous-usage claim times since start (nil-safe).
func (s *StatsService) anonymousTimes(start time.Time) ([]time.Time, error) {
	if s.anonymous == nil {
		return nil, nil
	}
	return s.anonymous.CreatedSince(start)
}

// registerVisitTimes returns register-page visit times since start (nil-safe).
func (s *StatsService) registerVisitTimes(start time.Time) ([]time.Time, error) {
	if s.registerVisit == nil {
		return nil, nil
	}
	return s.registerVisit.CreatedSince(start)
}

// AIStats reports success/failure, error groups, latency percentiles, per-provider
// performance, token usage (input/output) and generation speed, and live queue health.
func (s *StatsService) AIStats(filterType string) (AIStats, error) {
	now := time.Now().UTC()
	weekAgo := now.AddDate(0, 0, -7)
	stats := AIStats{Queue: s.queueMetrics()}

	latencies, err := s.usage.Latencies(weekAgo)
	if err != nil {
		return stats, err
	}
	stats.Latency = latencyStats(latencies)

	tokenUsage, err := s.usage.TokenUsage(weekAgo)
	if err != nil {
		return stats, err
	}
	stats.TotalInputTokens = tokenUsage.InputTokens
	stats.TotalOutputTokens = tokenUsage.OutputTokens
	if tokenUsage.GenerateMS > 0 {
		stats.TokensPerSec = float64(tokenUsage.OutputTokens) * 1000 / float64(tokenUsage.GenerateMS)
	}

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
	daily, weekly := bucketTrends(times, now)
	return Trends{Daily: daily, Weekly: weekly}, nil
}

// bucketTrends buckets a set of creation times into the trailing 7 UTC days
// and 8 Monday-based weeks. Shared by the question trends and the visitor
// metrics so every chart in the control panel uses identical buckets.
func bucketTrends(times []time.Time, now time.Time) ([]TrendPoint, []TrendPoint) {
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
	return daily, weekly
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

// MaskQuery hides all but the first rune of a query so the control panel can
// show "what people searched" (with counts) without revealing the actual query
// text. Very short inputs are fully hidden to avoid leaking even the prefix.
func MaskQuery(query string) string {
	runes := []rune(strings.TrimSpace(query))
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= 2 {
		return "•"
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

// TrendingWords tokenizes stored queries into lowercase terms and aggregates
// them per time bucket (daily over 7 days, or weekly over 8 weeks), plus the
// overall top terms for the window. Raw queries never leave this method — the
// API only returns aggregated word counts.
func (s *StatsService) TrendingWords(window string) (TrendingWords, error) {
	now := time.Now().UTC()
	isWeekly := window == "weekly"
	if !isWeekly {
		window = "daily"
	}
	start := now.AddDate(0, 0, -7*8)
	logs, err := s.usage.QueriesSince(start)
	if err != nil {
		return TrendingWords{}, err
	}
	count := 7
	if isWeekly {
		count = 8
	}
	buckets := make([]WordTrendBucket, 0, count)
	for i := 0; i < count; i++ {
		bucketStart, bucketEnd := bucketBounds(now, i, isWeekly)
		counts := map[string]int64{}
		for _, log := range logs {
			if !log.CreatedAt.Before(bucketStart) && log.CreatedAt.Before(bucketEnd) {
				for _, word := range tokenizeWords(log.Query) {
					counts[word]++
				}
			}
		}
		buckets = append(buckets, WordTrendBucket{Label: bucketStart.Format("2006-01-02"), Top: topWords(counts, 8)})
	}
	overall := map[string]int64{}
	for _, log := range logs {
		for _, word := range tokenizeWords(log.Query) {
			overall[word]++
		}
	}
	return TrendingWords{Window: window, Buckets: buckets, Overall: topWords(overall, 10)}, nil
}

// bucketBounds returns the half-open [start, end) window for a bucket index
// (0 = oldest). Daily buckets are UTC days; weekly buckets are Monday-based
// weeks, matching the Trends() bucketing.
func bucketBounds(now time.Time, index int, weekly bool) (time.Time, time.Time) {
	if weekly {
		week := weekStartUTC(now)
		start := week.AddDate(0, 0, -7*(7-index))
		return start, start.AddDate(0, 0, 7)
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := day.AddDate(0, 0, index-6)
	return start, start.AddDate(0, 0, 1)
}

// topWords returns the highest-count terms, ties broken by word for stability.
func topWords(counts map[string]int64, limit int) []WordCount {
	if limit < 1 {
		limit = 10
	}
	words := make([]string, 0, len(counts))
	for word := range counts {
		words = append(words, word)
	}
	sort.Slice(words, func(i, j int) bool {
		if counts[words[i]] != counts[words[j]] {
			return counts[words[i]] > counts[words[j]]
		}
		return words[i] < words[j]
	})
	result := make([]WordCount, 0, min(len(words), limit))
	for _, word := range words {
		if len(result) == limit {
			break
		}
		result = append(result, WordCount{Word: word, Count: counts[word]})
	}
	return result
}

// stopwords are common English terms filtered from the word trends so the
// graph shows meaningful keywords instead of filler. Keep the list compact;
// any word shorter than 3 runes is also dropped.
var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "was": true, "were": true,
	"with": true, "from": true, "what": true, "when": true, "where": true, "which": true,
	"that": true, "this": true, "your": true, "you": true, "how": true, "why": true,
	"who": true, "can": true, "does": true, "did": true, "not": true, "but": true,
	"all": true, "any": true, "has": true, "have": true, "its": true, "our": true,
	"about": true, "into": true, "than": true, "then": true, "them": true, "they": true,
	"will": true, "would": true, "could": true, "should": true, "please": true,
	"tell": true, "give": true, "find": true, "show": true, "need": true, "want": true,
}

// tokenizeWords splits a query into lowercase terms, dropping stopwords and
// words shorter than three runes.
func tokenizeWords(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := map[string]bool{}
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if len([]rune(field)) < 3 || stopwords[field] {
			continue
		}
		if !seen[field] {
			seen[field] = true
			result = append(result, field)
		}
	}
	return result
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