package store

import "testing"

func TestGenerateCountrySlug(t *testing.T) {
	input := "United Kingdom"
	got := generateCountrySlug(input)
	want := "united-kingdom"

	if got != want {
		t.Errorf("generateCountrySlug(%s) = %s; want %s", input, got, want)
	}

	input = "Korea"
	got = generateCountrySlug(input)
	want = "korea"

	if got != want {
		t.Errorf("generateCountrySlug(%s) = %s; want %s", input, got, want)
	}

	input = "Saint Kitts and Nevis"
	got = generateCountrySlug(input)
	want = "saint-kitts-and-nevis"

	if got != want {
		t.Errorf("generateCountrySlug(%s) = %s; want %s", input, got, want)
	}

	input = "^&%(test %country*[]"
	got = generateCountrySlug(input)
	want = "test-country"

	if got != want {
		t.Errorf("generateCountrySlug(%s) = %s; want %s", input, got, want)
	}
}

func TestMax(t *testing.T) {
	got := max(1, 100)
	want := 100

	if got != want {
		t.Errorf("max(1,100) = %d; want %d", got, want)
	}

	got = max(-77, 0)
	want = 0
	if got != want {
		t.Errorf("max(-77,0) = %d; want %d", got, want)
	}
}
