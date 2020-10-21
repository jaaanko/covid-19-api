package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaaanko/covid-19-api/internal/server"
	"github.com/jaaanko/covid-19-api/internal/store"
	"github.com/jaaanko/covid-19-api/internal/store/storetest"
)

var (
	testCountry1      = store.Country{Name: "Test Country 1", Slug: "test-country-1"}
	testCountry2      = store.Country{Name: "Test Country 2", Slug: "test-country-2"}
	testLocation1     = store.Location{Country: testCountry1, Latitude: 88.644, Longitude: 99.999}
	testLocation2     = store.Location{Country: testCountry2, Latitude: 76.66, Longitude: 100.00}
	testCountry1Stats = store.CovidStats{
		Confirmed:     23,
		NewConfirmed:  1,
		Recoveries:    33,
		NewRecoveries: 2,
		Deaths:        0,
		NewDeaths:     0,
	}
	testCountry2Stats = store.CovidStats{
		Confirmed:     88,
		NewConfirmed:  13,
		Recoveries:    44,
		NewRecoveries: 22,
		Deaths:        0,
		NewDeaths:     0,
	}
	testGlobalStats = store.CovidStats{
		Confirmed:     50,
		NewConfirmed:  7,
		Recoveries:    44,
		NewRecoveries: 24,
		Deaths:        0,
		NewDeaths:     0,
	}
)

func TestGetCountries(t *testing.T) {
	st := &storetest.StubStore{
		Countries: []store.Country{
			testCountry1,
			testCountry2,
		},
	}
	server := server.New(st)

	req, err := http.NewRequest(http.MethodGet, "/list/countries", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetCountries)
	handler.ServeHTTP(res, req)

	// Test status code
	if expectedCode, got := http.StatusOK, res.Code; got != expectedCode {
		t.Errorf("Wrong status code returned: got %v want %v", got, expectedCode)
	}

	// Test body
	countries := []store.Country{}

	err = json.Unmarshal(res.Body.Bytes(), &countries)
	if err != nil {
		t.Fatal(err)
	}

	if expectedName, receivedName := testCountry2.Name, countries[1].Name; receivedName != expectedName {
		t.Errorf("Wrong name returned: got %v want %v", receivedName, expectedName)
	}
}

func TestGetGlobalStats(t *testing.T) {
	st := &storetest.StubStore{
		GlobalStats: testGlobalStats,
	}

	server := server.New(st)

	req, err := http.NewRequest(http.MethodGet, "/global", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetGlobalStats)
	handler.ServeHTTP(res, req)

	// Test status code
	if expectedCode, got := http.StatusOK, res.Code; got != expectedCode {
		t.Errorf("Wrong status code returned: got %v want %v", got, expectedCode)
	}

	// Test body
	expectedConfirmed := testGlobalStats.Confirmed
	expectedRecoveries := testGlobalStats.Recoveries
	expectedDeaths := testGlobalStats.Deaths

	globalStats := store.CovidStats{}

	err = json.Unmarshal(res.Body.Bytes(), &globalStats)
	if err != nil {
		t.Fatal(err)
	}

	if receivedConfirmed := globalStats.Confirmed; receivedConfirmed != expectedConfirmed {
		t.Errorf("Wrong amount of confirmed cases returned: got %v want %v", receivedConfirmed, expectedConfirmed)
	}
	if receivedRecoveries := globalStats.Recoveries; receivedRecoveries != expectedRecoveries {
		t.Errorf("Wrong amount of recoveries returned: got %v want %v", receivedRecoveries, expectedRecoveries)
	}
	if receivedDeaths := globalStats.Deaths; receivedDeaths != expectedDeaths {
		t.Errorf("Wrong amount of deaths returned: got %v want %v", receivedDeaths, expectedDeaths)
	}
}

func TestGetSummary(t *testing.T) {
	locationStatsList := []store.LocationStats{
		store.LocationStats{
			Location:   testLocation1,
			CovidStats: testCountry1Stats,
		},
		store.LocationStats{
			Location:   testLocation2,
			CovidStats: testCountry2Stats,
		},
	}

	st := &storetest.StubStore{
		Summary: store.Summary{
			CovidStats:        testGlobalStats,
			LocationStatsList: locationStatsList,
		},
	}

	server := server.New(st)

	req, err := http.NewRequest(http.MethodGet, "/summary", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetSummary)
	handler.ServeHTTP(res, req)

	// Test status code
	if expectedCode, got := http.StatusOK, res.Code; got != expectedCode {
		t.Errorf("Wrong status code returned: got %v want %v", got, expectedCode)
	}

	// Test Body
	expectedConfirmed := testGlobalStats.Confirmed
	expectedNumOfLocations := len(locationStatsList)
	summary := store.Summary{}

	err = json.Unmarshal(res.Body.Bytes(), &summary)
	if err != nil {
		t.Fatal(err)
	}

	if receivedConfirmed := summary.Confirmed; receivedConfirmed != expectedConfirmed {
		t.Errorf("Wrong amount of global confirmed cases returned: got %v want %v", receivedConfirmed, expectedConfirmed)
	}

	if receivedNumOfLocations := len(summary.LocationStatsList); receivedNumOfLocations != expectedNumOfLocations {
		t.Errorf("Wrong amount of locations returned: got %v want %v", receivedNumOfLocations, expectedNumOfLocations)
	}
}

func TestGetTimeSeriesWithInvalidStatus(t *testing.T) {
	st := &storetest.StubStore{}
	s := server.New(st)

	req, err := http.NewRequest(http.MethodGet, "/timeseries/country-slug/invalidstatus", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handler := server.StatusMiddleware(http.HandlerFunc(s.GetTimeSeries))
	handler.ServeHTTP(res, req)

	// Test status code
	if expectedCode, got := http.StatusBadRequest, res.Code; got != expectedCode {
		t.Errorf("Wrong status code returned: got %v want %v", got, expectedCode)
	}
}

func TestGetAggTimeSeriesWithInvalidStatus(t *testing.T) {
	st := &storetest.StubStore{}
	s := server.New(st)

	req, err := http.NewRequest(http.MethodGet, "/timeseries/total/country-slug/invalidstatus", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handler := server.StatusMiddleware(http.HandlerFunc(s.GetAggTimeSeries))
	handler.ServeHTTP(res, req)

	// Test status code
	if expectedCode, got := http.StatusBadRequest, res.Code; got != expectedCode {
		t.Errorf("Wrong status code returned: got %v want %v", got, expectedCode)
	}
}
