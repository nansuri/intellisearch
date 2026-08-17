package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GeoLocation is an optional device position supplied by the browser.
type GeoLocation struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
}

var locationIntentPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bnear my place\b`),
	regexp.MustCompile(`(?i)\bnear me\b`),
	regexp.MustCompile(`(?i)\baround here\b`),
	regexp.MustCompile(`(?i)\bnearby\b`),
	regexp.MustCompile(`(?i)\bclose to me\b`),
	regexp.MustCompile(`(?i)\bin my area\b`),
	regexp.MustCompile(`(?i)\baround my location\b`),
	regexp.MustCompile(`(?i)\bnear my location\b`),
	regexp.MustCompile(`(?i)\bwhere i am\b`),
	regexp.MustCompile(`(?i)\bmy location\b`),
}

// ValidateGeoLocation checks that coordinates are within real-world bounds.
func ValidateGeoLocation(location GeoLocation) bool {
	if location.Latitude < -90 || location.Latitude > 90 {
		return false
	}
	if location.Longitude < -180 || location.Longitude > 180 {
		return false
	}
	return !(location.Latitude == 0 && location.Longitude == 0)
}

// NeedsLocationContext reports whether the question likely depends on the user's position.
func NeedsLocationContext(query string) bool {
	for _, pattern := range locationIntentPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}

// EnrichQueryWithLocation rewrites location-relative phrasing for web search.
func EnrichQueryWithLocation(query, placeLabel string) string {
	placeLabel = strings.TrimSpace(placeLabel)
	if placeLabel == "" {
		return query
	}
	rewritten := query
	for _, pattern := range locationIntentPatterns {
		rewritten = pattern.ReplaceAllStringFunc(rewritten, func(_ string) string {
			return "near " + placeLabel
		})
	}
	if strings.EqualFold(strings.TrimSpace(rewritten), strings.TrimSpace(query)) {
		return strings.TrimSpace(query) + " near " + placeLabel
	}
	return strings.TrimSpace(rewritten)
}

// GeocodePoint is one geocoded place with coordinates and a display label.
type GeocodePoint struct {
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GeoService reverse-geocodes coordinates into a human-readable place label
// and geocodes place queries into coordinates (both via Nominatim).
type GeoService struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

func NewGeoService(baseURL, userAgent string, timeoutMS int) *GeoService {
	if timeoutMS < 1 {
		timeoutMS = 5000
	}
	if userAgent == "" {
		userAgent = "Intellisearch/1.0"
	}
	return &GeoService{
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutMS) * time.Millisecond,
		},
	}
}

// ReverseGeocode resolves coordinates to a short city/region label for search.
func (g *GeoService) ReverseGeocode(ctx context.Context, location GeoLocation) (string, error) {
	if g == nil {
		return "", fmt.Errorf("geo service unavailable")
	}
	requestURL := fmt.Sprintf("%s/reverse?lat=%f&lon=%f&format=json&zoom=12&addressdetails=1", g.baseURL, location.Latitude, location.Longitude)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Accept", "application/json")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reverse geocode status: %d", response.StatusCode)
	}

	var payload struct {
		DisplayName string `json:"display_name"`
		Address     struct {
			City        string `json:"city"`
			Town        string `json:"town"`
			Village     string `json:"village"`
			Suburb      string `json:"suburb"`
			State       string `json:"state"`
			Country     string `json:"country"`
			CountryCode string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}

	locality := firstNonEmpty(payload.Address.City, payload.Address.Town, payload.Address.Village, payload.Address.Suburb)
	if locality != "" && payload.Address.Country != "" {
		return locality + ", " + payload.Address.Country, nil
	}
	if payload.DisplayName != "" {
		parts := strings.Split(payload.DisplayName, ",")
		if len(parts) > 3 {
			return strings.TrimSpace(strings.Join(parts[:3], ",")), nil
		}
		return strings.TrimSpace(payload.DisplayName), nil
	}
	return fmt.Sprintf("%.4f, %.4f", location.Latitude, location.Longitude), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Geocode resolves a free-text place query to up to limit coordinates via the
// Nominatim search endpoint. Best-effort by design: callers treat an error or
// empty result as "no coordinates" rather than failing the whole ask.
func (g *GeoService) Geocode(ctx context.Context, query string, limit int) ([]GeocodePoint, error) {
	if g == nil {
		return nil, fmt.Errorf("geo service unavailable")
	}
	if limit < 1 {
		limit = 1
	}
	requestURL := fmt.Sprintf("%s/search?q=%s&format=json&limit=%d&addressdetails=0", g.baseURL, url.QueryEscape(strings.TrimSpace(query)), limit)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Accept", "application/json")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocode status: %d", response.StatusCode)
	}

	var payload []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	points := make([]GeocodePoint, 0, len(payload))
	for _, item := range payload {
		latitude, latErr := strconv.ParseFloat(item.Lat, 64)
		longitude, lonErr := strconv.ParseFloat(item.Lon, 64)
		if latErr != nil || lonErr != nil {
			continue
		}
		points = append(points, GeocodePoint{Label: item.DisplayName, Latitude: latitude, Longitude: longitude})
		if len(points) >= limit {
			break
		}
	}
	return points, nil
}

// haversineKM returns the great-circle distance between two coordinates in
// kilometers, used to keep geocoded markers plausibly near the user.
func haversineKM(a, b GeoLocation) float64 {
	const earthRadiusKM = 6371.0
	toRad := func(degrees float64) float64 { return degrees * math.Pi / 180 }
	dLat := toRad(b.Latitude - a.Latitude)
	dLon := toRad(b.Longitude - a.Longitude)
	lat1 := toRad(a.Latitude)
	lat2 := toRad(b.Latitude)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKM * math.Asin(math.Sqrt(h))
}

// BuildLocationContext returns search-query and LLM notes derived from device
// location, plus the human-readable place label for the user's coordinates
// (used as the map center label). The label is "" when no valid location was
// shared.
func BuildLocationContext(ctx context.Context, geo *GeoService, query string, location *GeoLocation) (searchQuery string, llmNote string, placeLabel string) {
	searchQuery = query
	if location == nil || !ValidateGeoLocation(*location) {
		if NeedsLocationContext(query) {
			llmNote = "The user asked about nearby places but did not share their device location. Answer generally and suggest enabling location for better local results."
		}
		return searchQuery, llmNote, ""
	}

	if geo != nil {
		if label, err := geo.ReverseGeocode(ctx, *location); err == nil {
			placeLabel = label
		}
	}
	if placeLabel == "" {
		placeLabel = fmt.Sprintf("%.4f, %.4f", location.Latitude, location.Longitude)
	}

	if NeedsLocationContext(query) {
		searchQuery = EnrichQueryWithLocation(query, placeLabel)
		llmNote = fmt.Sprintf("The user's approximate location is %s. Prefer local, relevant results for this area when answering.", placeLabel)
	}
	return searchQuery, llmNote, placeLabel
}
