package store

import "time"

type Location struct {
	CountryName string  `json:"country"`
	CountrySlug string  `json:"slug"`
	Province    string  `json:"province"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type CovidStats struct {
	Confirmed     int64 `json:"confirmed"`
	NewConfirmed  int64 `json:"newConfirmed"`
	Recoveries    int64 `json:"recovered"`
	NewRecoveries int64 `json:"newRecovered"`
	Deaths        int64 `json:"deaths"`
	NewDeaths     int64 `json:"newDeaths"`
}

type LocationCovidStats struct {
	Location
	CovidStats
}

type GlobalSummary struct {
	CovidStats
	ListOfLocationCovidStats []LocationCovidStats `json:"countries"`
}

type TimeSeriesDataPoint struct {
	LocationCovidStats
	Date time.Time `json:"date"`
}

type TimeSeries struct {
	DataPoints []TimeSeriesDataPoint `json:"timeseries"`
}
