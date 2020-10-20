package store

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type mySql struct {
	db *sql.DB
}

func NewMySql(dataSourceName string) (Service, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	return mySql{db}, db.Ping()
}

func (m mySql) GetDbInstance() (*sql.DB, error) {
	return m.db, m.db.Ping()
}

func (m mySql) GetCountries() ([]Country, error) {
	rows, err := m.db.Query(`
	select country,country_slug from recoveries_time_series group by country_slug
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countryList := []Country{}

	for rows.Next() {
		country := new(Country)
		err := rows.Scan(&country.Name, &country.Slug)

		if err != nil {
			return nil, err
		}

		countryList = append(countryList, *country)
	}

	return countryList, rows.Err()
}

func (m mySql) GetGlobalStats() (*CovidStats, error) {
	row := m.db.QueryRow(`select cd.confirmed,cd.new_confirmed,cd.deaths,cd.new_deaths,r.recoveries,r.new_recoveries
	from (
		select SUM(confirmed_cases) confirmed, SUM(new_confirmed) new_confirmed, SUM(deaths) deaths, SUM(new_deaths) new_deaths
		from confirmed_and_deaths_time_series
		where date_recorded = (
			select MAX(date_recorded) from confirmed_and_deaths_time_series where country_slug = "France"
		) 
	) cd
	join (
		select sum(recoveries) recoveries, sum(new_recoveries) new_recoveries
		from recoveries_time_series
		where date_recorded = (
			select MAX(date_recorded) from recoveries_time_series where country_slug = "France"
		) 
	) r
	`)

	globalStats := new(CovidStats)

	err := row.Scan(
		&globalStats.Confirmed,
		&globalStats.NewConfirmed,
		&globalStats.Deaths,
		&globalStats.NewDeaths,
		&globalStats.Recoveries,
		&globalStats.NewRecoveries,
	)

	return globalStats, err
}

func (m mySql) GetSummary() (*Summary, error) {
	rows, err := m.db.Query(`
	select cd.country, cd.country_slug, cd.total_confirmed, cd.new_confirmed, cd.total_deaths, cd.new_deaths, r.total_recoveries, r.new_recoveries
	from (
		select country,country_slug,sum(confirmed_cases) total_confirmed, sum(new_confirmed) new_confirmed, sum(deaths) total_deaths, sum(new_deaths) new_deaths
		from confirmed_and_deaths_time_series
		where date_recorded = (
			select MAX(date_recorded) from confirmed_and_deaths_time_series where country_slug = "France"
		)
		group by country_slug
	) cd
	join (
		select country_slug, sum(recoveries) total_recoveries, sum(new_recoveries) new_recoveries
		from recoveries_time_series
		where date_recorded = (
			select MAX(date_recorded) from recoveries_time_series where country_slug = "France"
		) 
		group by country_slug	
	) r
	on cd.country_slug = r.country_slug
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := new(Summary)

	for rows.Next() {
		locationStats := LocationStats{}

		err := rows.Scan(
			&locationStats.Country.Name,
			&locationStats.Country.Slug,
			&locationStats.Confirmed,
			&locationStats.NewConfirmed,
			&locationStats.Recoveries,
			&locationStats.NewRecoveries,
			&locationStats.Deaths,
			&locationStats.NewDeaths,
		)

		if err != nil {
			return nil, err
		}

		summary.Confirmed += locationStats.Confirmed
		summary.NewConfirmed += locationStats.NewConfirmed
		summary.Recoveries += locationStats.Recoveries
		summary.NewRecoveries += locationStats.NewRecoveries
		summary.Deaths += locationStats.Deaths
		summary.NewDeaths += locationStats.NewDeaths

		summary.LocationStatsList = append(summary.LocationStatsList, locationStats)

	}
	return summary, rows.Err()
}

func (m mySql) GetTimeSeries(countrySlug string, status string) (*TimeSeries, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if status == "confirmed" {
		rows, err = m.db.Query(`
		select country,country_slug,province,confirmed_cases,new_confirmed,latitude,longitude,date_recorded
		from confirmed_and_deaths_time_series where country_slug = ?
		`, countrySlug)
	}
	if status == "recoveries" {
		rows, err = m.db.Query(`
		select country,country_slug,province,recoveries,new_recoveries,latitude,longitude,date_recorded
		from recoveries_time_series where country_slug = ?
		`, countrySlug)
	}
	if status == "deaths" {
		rows, err = m.db.Query(`
		select country,country_slug,province,deaths,new_deaths,latitude,longitude,date_recorded
		from confirmed_and_deaths_time_series where country_slug = ?
		`, countrySlug)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeSeries := new(TimeSeries)

	for rows.Next() {
		dataPoint := TimeSeriesDataPoint{}

		err = rows.Scan(
			&dataPoint.Country.Name,
			&dataPoint.Country.Slug,
			&dataPoint.Province,
			&dataPoint.Amount,
			&dataPoint.New,
			&dataPoint.Latitude,
			&dataPoint.Longitude,
			&dataPoint.Date,
		)

		dataPoint.Status = status

		if err != nil {
			return nil, err
		}

		timeSeries.DataPoints = append(timeSeries.DataPoints, dataPoint)
	}

	return timeSeries, rows.Err()
}

func (m mySql) GetAggTimeSeries(countrySlug string, status string) (*TimeSeries, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if status == "confirmed" {
		rows, err = m.db.Query(`
		select country,country_slug,SUM(confirmed_cases),SUM(new_confirmed),date_recorded 
		from confirmed_and_deaths_time_series where country_slug = ? group by date_recorded
		`, countrySlug)

	}
	if status == "deaths" {
		rows, err = m.db.Query(`
		select country,country_slug,SUM(deaths),SUM(new_deaths),date_recorded 
		from confirmed_and_deaths_time_series where country_slug = ? group by date_recorded
		`, countrySlug)

	}
	if status == "recoveries" {
		rows, err = m.db.Query(`
		select country,country_slug,SUM(recoveries),SUM(new_recoveries),date_recorded 
		from recoveries_time_series where country_slug = ? group by date_recorded
		`, countrySlug)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeSeries := new(TimeSeries)

	for rows.Next() {
		dataPoint := TimeSeriesDataPoint{}

		err = rows.Scan(
			&dataPoint.Country.Name,
			&dataPoint.Country.Slug,
			&dataPoint.Amount,
			&dataPoint.New,
			&dataPoint.Date,
		)

		dataPoint.Status = status

		if err != nil {
			return nil, err
		}

		timeSeries.DataPoints = append(timeSeries.DataPoints, dataPoint)
	}

	return timeSeries, rows.Err()
}

func (m mySql) Close() error {
	return m.db.Close()
}
