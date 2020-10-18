package store

import "time"

type CaseStatus string

const (
	Confirmed  CaseStatus = "Confirmed"
	Recoveries            = "Recoveries"
	Deaths                = "Deaths"
)

type Country struct {
	Name string `json:"countryName"`
	Slug string `json:"countrySlug"`
}

type Location struct {
	Country
	Province  string   `json:"province"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type CovidStats struct {
	Confirmed     int64 `json:"confirmed"`
	NewConfirmed  int64 `json:"newConfirmed"`
	Recoveries    int64 `json:"recovered"`
	NewRecoveries int64 `json:"newRecovered"`
	Deaths        int64 `json:"deaths"`
	NewDeaths     int64 `json:"newDeaths"`
}

type LocationStats struct {
	Location
	CovidStats
}

type Summary struct {
	CovidStats
	LocationStatsList []LocationStats `json:"countries"`
}

type TimeSeriesDataPoint struct {
	Location
	Amount int64
	New    int64
	Status CaseStatus
	Date   time.Time `json:"date"`
}

type TimeSeries struct {
	DataPoints []TimeSeriesDataPoint `json:"timeSeries"`
}

type Service interface {
	GetCountries() ([]Country, error)
	GetGlobalStats() (*CovidStats, error)
	GetSummary() (*Summary, error)
	GetTimeSeries(countrySlug string, status CaseStatus) (*TimeSeries, error)
	GetAggTimeSeries(countrySlug string, status CaseStatus) (*TimeSeries, error)
}
