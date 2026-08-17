package services

import "testing"

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
	search, note := BuildLocationContext(t.Context(), nil, "stores near me", nil)
	if search != "stores near me" {
		t.Fatalf("unexpected search query %q", search)
	}
	if note == "" {
		t.Fatal("expected missing-location note")
	}
}
