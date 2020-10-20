package store

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseUrl = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/"

var (
	confirmedSrc  = fmt.Sprintf("%stime_series_covid19_confirmed_global.csv", baseUrl)
	deathsSrc     = fmt.Sprintf("%stime_series_covid19_deaths_global.csv", baseUrl)
	recoveriesSrc = fmt.Sprintf("%stime_series_covid19_recovered_global.csv", baseUrl)
)

type jhuCsseDataCollector struct {
	db *sql.DB
}

func NewJhuCsseDataCollector(db *sql.DB) (jhuCsseDataCollector, error) {
	return jhuCsseDataCollector{db}, db.Ping()
}

func (jhu jhuCsseDataCollector) UpdateConfirmedAndDeaths() error {
	confirmedResponse, err := http.Get(confirmedSrc)
	if err != nil {
		return err
	}
	defer confirmedResponse.Body.Close()

	deathsResponse, err := http.Get(deathsSrc)
	if err != nil {
		return err
	}
	defer deathsResponse.Body.Close()

	confirmedReader := csv.NewReader(confirmedResponse.Body)
	deathsReader := csv.NewReader(deathsResponse.Body)

	headers, err := confirmedReader.Read()
	if err != nil {
		return err
	}

	_, err = deathsReader.Read()
	if err != nil {
		return err
	}

	dates := headers[4:]

	tx, err := jhu.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO confirmed_and_deaths_time_series 
	(province,country,country_slug,latitude,longitude,confirmed_cases,new_confirmed,deaths,new_deaths,date_recorded) 
	VALUES (?,?,?,?,?,?,?,?,?,?)
	ON DUPLICATE KEY UPDATE confirmed_cases = VALUES(confirmed_cases), new_confirmed = VALUES(new_confirmed),deaths = VALUES(deaths), new_deaths = VALUES(new_deaths)
	`)

	if err != nil {
		return err
	}
	defer stmt.Close()

	for {
		prevConfirmed := 0
		prevDeaths := 0

		confirmedRow, err := confirmedReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		deathsRow, err := deathsReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		lat, err := strconv.ParseFloat(confirmedRow[2], 64)
		if err != nil {
			return err
		}

		long, err := strconv.ParseFloat(confirmedRow[3], 64)
		if err != nil {
			return err
		}

		for j := 0; j < len(dates); j++ {
			deaths, err := strconv.Atoi(deathsRow[j+4])
			if err != nil {
				return err
			}

			confirmed, err := strconv.Atoi(confirmedRow[j+4])
			if err != nil {
				return err
			}

			date, err := time.Parse("1/2/06", dates[j])
			if err != nil {
				return err
			}

			_, err = stmt.Exec(confirmedRow[0], confirmedRow[1], generateCountrySlug(confirmedRow[1]), lat, long, confirmed,
				max(0, confirmed-prevConfirmed), deaths, max(0, deaths-prevDeaths), date)

			if err != nil {
				return err
			}

			prevConfirmed = confirmed
			prevDeaths = deaths
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("Done updating confirmed and deaths")
	return nil
}

func (jhu jhuCsseDataCollector) UpdateRecoveries() error {
	recoveriesResponse, err := http.Get(recoveriesSrc)

	if err != nil {
		return err
	}
	defer recoveriesResponse.Body.Close()

	recoveriesReader := csv.NewReader(recoveriesResponse.Body)

	headers, err := recoveriesReader.Read()
	if err != nil {
		return err
	}

	dates := headers[4:]

	tx, err := jhu.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO recoveries_time_series 
	(province,country,country_slug,latitude,longitude,recoveries,new_recoveries,date_recorded) 
	VALUES (?,?,?,?,?,?,?,?)
	ON DUPLICATE KEY UPDATE recoveries = VALUES(recoveries), new_recoveries = VALUES(new_recoveries)
	`)

	if err != nil {
		return err
	}
	defer stmt.Close()

	for {
		row, err := recoveriesReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		lat, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			return err
		}

		long, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			return err
		}

		prevRecoveries := 0

		for j := 0; j < len(dates); j++ {
			recoveries, err := strconv.Atoi(row[j+4])
			if err != nil {
				return err
			}
			date, err := time.Parse("1/2/06", dates[j])
			if err != nil {
				return err
			}

			_, err = stmt.Exec(row[0], row[1], generateCountrySlug(row[1]), lat, long, recoveries, max(0, recoveries-prevRecoveries), date)
			if err != nil {
				return err
			}

			prevRecoveries = recoveries
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("Done updating recoveries")
	return nil
}

func generateCountrySlug(country string) string {
	r := regexp.MustCompile("[^a-zA-Z- ]")
	country = r.ReplaceAllString(country, "")

	r = regexp.MustCompile(" ")
	country = r.ReplaceAllString(country, "-")

	return strings.ToLower(country)
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
