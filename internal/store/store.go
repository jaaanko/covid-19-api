package store

import (
	"database/sql"
	"time"
)

const (
	Confirmed  string = "confirmed"
	Recoveries        = "recoveries"
	Deaths            = "deaths"
)

type Country struct {
	Name string `json:"countryName"`
	Slug string `json:"countrySlug"`
}

type Location struct {
	Country
	Province  string  `json:"province"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
	Amount int64     `json:"amount"`
	New    int64     `json:"new"`
	Status string    `json:"status"`
	Date   time.Time `json:"date"`
}

type TimeSeries struct {
	DataPoints []TimeSeriesDataPoint `json:"timeSeries"`
}

type Service interface {
	GetCountries() ([]Country, error)
	GetGlobalStats() (*CovidStats, error)
	GetSummary() (*Summary, error)
	GetTimeSeries(countrySlug string, status string) (*TimeSeries, error)
	GetAggTimeSeries(countrySlug string, status string) (*TimeSeries, error)
	GetDbInstance() (*sql.DB, error)
	Close() error
}
