package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateGeoLocation(t *testing.T) {
	if !ValidateGeoLocation(GeoLocation{Latitude: -6.2, Longitude: 106.8}) {
		t.Fatal("expected valid coordinates")
	}
	if ValidateGeoLocation(GeoLocation{Latitude: 91, Longitude: 0}) {
		t.Fatal("expected invalid latitude")
	}
	if ValidateGeoLocation(GeoLocation{Latitude: 0, Longitude: 0}) {
		t.Fatal("expected null island rejected")
	}
}

func TestNeedsLocationContext(t *testing.T) {
	if !NeedsLocationContext("What supermarket is near my place?") {
		t.Fatal("expected location intent")
	}
	if NeedsLocationContext("What is inflation?") {
		t.Fatal("expected no location intent")
	}
}

func TestEnrichQueryWithLocation(t *testing.T) {
	got := EnrichQueryWithLocation("supermarket near my place", "Jakarta, Indonesia")
	if got != "supermarket near Jakarta, Indonesia" {
		t.Fatalf("unexpected rewrite %q", got)
	}
	got = EnrichQueryWithLocation("best coffee shops", "Tokyo, Japan")
	if got != "best coffee shops near Tokyo, Japan" {
		t.Fatalf("unexpected append %q", got)
	}
}

func TestBuildLocationContextWithoutLocation(t *testing.T) {
	search, note, label := BuildLocationContext(t.Context(), nil, "stores near me", nil)
	if search != "stores near me" {
		t.Fatalf("unexpected search query %q", search)
	}
	if note == "" {
		t.Fatal("expected missing-location note")
	}
	if label != "" {
		t.Fatalf("expected no label without location, got %q", label)
	}
}

func TestGeocodeParsesNominatimResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"display_name":"St Mary's Hospital, London","lat":"51.5183","lon":"-0.1721"},{"display_name":"Second Place","lat":"40.7128","lon":"-74.0060"}]`))
	}))
	t.Cleanup(server.Close)
	geo := NewGeoService(server.URL, "test/1.0", 2000)
	points, err := geo.Geocode(t.Context(), "hospital near me", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].Label != "St Mary's Hospital, London" || points[0].Latitude != 51.5183 || points[0].Longitude != -0.1721 {
		t.Fatalf("unexpected points %#v", points)
	}
}

func TestGeocodeSkipsMalformedCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"display_name":"Bad","lat":"not-a-number","lon":"0"},{"display_name":"Good","lat":"1.5","lon":"2.5"}]`))
	}))
	t.Cleanup(server.Close)
	geo := NewGeoService(server.URL, "test/1.0", 2000)
	points, err := geo.Geocode(t.Context(), "query", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].Label != "Good" {
		t.Fatalf("expected only the valid point, got %#v", points)
	}
}

func TestHaversineKM(t *testing.T) {
	// Paris to London is ~343 km by great-circle.
	distance := haversineKM(GeoLocation{Latitude: 48.8566, Longitude: 2.3522}, GeoLocation{Latitude: 51.5074, Longitude: -0.1278})
	if distance < 330 || distance > 360 {
		t.Fatalf("unexpected Paris-London distance %.1f km", distance)
	}
	if haversineKM(GeoLocation{Latitude: 1, Longitude: 2}, GeoLocation{Latitude: 1, Longitude: 2}) != 0 {
		t.Fatal("expected zero distance for identical points")
	}
}
