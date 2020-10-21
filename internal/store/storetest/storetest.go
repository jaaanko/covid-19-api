package storetest

import (
	"database/sql"

	"github.com/jaaanko/covid-19-api/internal/store"
)

type StubStore struct {
	Countries     []store.Country
	GlobalStats   store.CovidStats
	Summary       store.Summary
	TimeSeries    store.TimeSeries
	AggTimeSeries store.TimeSeries
}

func (s *StubStore) GetCountries() ([]store.Country, error) {
	return s.Countries, nil
}

func (s *StubStore) GetGlobalStats() (*store.CovidStats, error) {
	return &s.GlobalStats, nil
}

func (s *StubStore) GetSummary() (*store.Summary, error) {
	return &s.Summary, nil
}

func (s *StubStore) GetTimeSeries(countrySlug string, status string) (*store.TimeSeries, error) {
	return &s.TimeSeries, nil
}

func (s *StubStore) GetAggTimeSeries(countrySlug string, status string) (*store.TimeSeries, error) {
	return &s.AggTimeSeries, nil
}

func (s *StubStore) GetDbInstance() (*sql.DB, error) {
	return nil, nil
}

func (s *StubStore) Close() error {
	return nil
}
