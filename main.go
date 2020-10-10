package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

const (
	confirmedCasesSrc    = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_confirmed_global.csv"
	deathsSrc            = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_deaths_global.csv"
	recoveriesSrc        = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_recovered_global.csv"
	usaConfirmedCasesSrc = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_confirmed_US.csv"
	usaDeathsSrc         = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/master/csse_covid_19_data/csse_covid_19_time_series/time_series_covid19_deaths_US.csv"
)

var (
	start time.Time
	db    *sql.DB
)

type TimeSeries struct {
	Province    string    `json:"province"`
	CountryName string    `json:"countryName"`
	CountrySlug string    `json:"countrySlug"`
	Lat         float64   `json:"latitude"`
	Long        float64   `json:"longitude"`
	Date        time.Time `json:"date"`
}

type AllTimeSeries struct {
	TimeSeries
	Summary
}

type ConfirmedTimeSeries struct {
	TimeSeries
	ConfirmedData
}

type DeathsTimeSeries struct {
	TimeSeries
	DeathsData
}

type RecoveriesTimeSeries struct {
	TimeSeries
	RecoveriesData
}

type ConfirmedData struct {
	Confirmed    int64 `json:"confirmed"`
	NewConfirmed int64 `json:"newConfirmed"`
}

type DeathsData struct {
	Deaths    int64 `json:"deaths"`
	NewDeaths int64 `json:"newDeaths"`
}

type RecoveriesData struct {
	Recoveries    int64 `json:"recovered"`
	NewRecoveries int64 `json:"newRecovered"`
}

type TimeSeriesResponse struct {
	TimeSeriesArray []interface{} `json:"timeseries"`
}

type SummaryResponse struct {
	World     Summary          `json:"world"`
	Countries []CountrySummary `json:"countries"`
}

type Summary struct {
	ConfirmedData
	RecoveriesData
	DeathsData
}

type CountrySummary struct {
	Country     string `json:"country"`
	CountrySlug string `json:"countrySlug"`
	Summary
}

func initDb() error {
	var err error
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")

	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/covid19?parseTime=true", dbUser, dbPass))
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	return nil
}

func main() {
	initialTicker := time.NewTicker(1)

	ticker := time.NewTicker(12 * time.Hour)
	done := make(chan (bool))

	err := initDb()

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	fetchData := func() {
		response, _ := http.Get(confirmedCasesSrc)
		response2, _ := http.Get(deathsSrc)
		recoveries, _ := http.Get(recoveriesSrc)

		// usaConfirmed, _ := http.Get(usaConfirmedCasesSrc)
		// usaDeaths, _ := http.Get(usaDeathsSrc)

		var wg sync.WaitGroup
		start = time.Now()

		wg.Add(1)
		go func() {
			defer wg.Done()
			saveConfirmedAndDeaths(response, response2)
			// saveUsaConfirmedAndDeaths(usaConfirmed, usaDeaths)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			saveRecoveries(recoveries)
		}()

		wg.Wait()
		fmt.Printf("Took %s", time.Since(start))
	}

	go func() {
		for {
			select {
			case <-initialTicker.C:
				initialTicker.Stop()
				fetchData()
			case <-ticker.C:
				fetchData()
			case <-done:
				ticker.Stop()
			}
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/world", world).Methods("GET")
	router.HandleFunc("/summary", summary).Methods("GET")
	router.HandleFunc("/timeseries/{countryslug}/{stat}", timeSeries).Methods("GET")
	router.HandleFunc("/timeseries/total/{countryslug}/{stat}", timeSeriesTotal).Methods("GET")

	http.ListenAndServe(":8080", router)
}

func generateCountrySlug(country string) string {
	r := regexp.MustCompile("[^a-zA-Z- ]")
	country = r.ReplaceAllString(country, "")

	r = regexp.MustCompile(" ")
	country = r.ReplaceAllString(country, "-")

	return strings.ToLower(country)
}

func world(w http.ResponseWriter, r *http.Request) {

	row := db.QueryRow(`select cd.confirmed,cd.new_confirmed,cd.deaths,cd.new_deaths,r.recoveries,r.new_recoveries
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

	worldSummaryResponse := new(Summary)

	err := row.Scan(
		&worldSummaryResponse.Confirmed,
		&worldSummaryResponse.NewConfirmed,
		&worldSummaryResponse.Deaths,
		&worldSummaryResponse.NewDeaths,
		&worldSummaryResponse.Recoveries,
		&worldSummaryResponse.NewRecoveries,
	)

	if err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(worldSummaryResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func timeSeriesTotal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		rows *sql.Rows
		err  error
	)

	if vars["stat"] == "confirmed" {
		rows, err = db.Query(`
		select country,country_slug,SUM(confirmed_cases),SUM(new_confirmed),date_recorded 
		from confirmed_and_deaths_time_series where country_slug = ? group by date_recorded
		`, vars["countryslug"])

	} else if vars["stat"] == "deaths" {
		rows, err = db.Query(`
		select country,country_slug,SUM(deaths),SUM(new_deaths),date_recorded 
		from confirmed_and_deaths_time_series where country_slug = ? group by date_recorded
		`, vars["countryslug"])

	} else if vars["stat"] == "recoveries" {
		rows, err = db.Query(`
		select country,country_slug,SUM(recoveries),SUM(new_recoveries),date_recorded 
		from recoveries_time_series where country_slug = ? group by date_recorded
		`, vars["countryslug"])

	} else if vars["stat"] == "all" {
		rows, err = db.Query(`
		select cd.country, cd.country_slug, cd.total_confirmed, cd.new_confirmed, cd.total_deaths, cd.new_deaths, r.total_recoveries, r.new_recoveries, cd.date_recorded
		from (
			select country,country_slug,sum(confirmed_cases) total_confirmed, sum(new_confirmed) new_confirmed, sum(deaths) total_deaths, sum(new_deaths) new_deaths,date_recorded 
			from confirmed_and_deaths_time_series 
			where country_slug = ?
			group by date_recorded
		) cd
		join (
			select sum(recoveries) total_recoveries, sum(new_recoveries) new_recoveries,date_recorded
			from recoveries_time_series where country_slug = ? group by date_recorded
		) r
		on cd.date_recorded = r.date_recorded
		`, vars["countryslug"], vars["countryslug"])
	} else {
		log.Fatal("Not allowed")
	}

	timeSeriesResponse := new(TimeSeriesResponse)

	for rows.Next() {
		var err error

		if vars["stat"] == "confirmed" {
			timeSeries := new(ConfirmedTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Confirmed,
				&timeSeries.NewConfirmed,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
		if vars["stat"] == "recoveries" {
			timeSeries := new(RecoveriesTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Recoveries,
				&timeSeries.NewRecoveries,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
		if vars["stat"] == "deaths" {
			timeSeries := new(DeathsTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Deaths,
				&timeSeries.NewDeaths,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
		if vars["stat"] == "all" {
			timeSeries := new(AllTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Confirmed,
				&timeSeries.NewConfirmed,
				&timeSeries.Deaths,
				&timeSeries.NewDeaths,
				&timeSeries.Recoveries,
				&timeSeries.NewRecoveries,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(timeSeriesResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func timeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		rows *sql.Rows
		err  error
	)

	if vars["stat"] == "confirmed" {
		rows, err = db.Query(`
		select country,country_slug,province,confirmed_cases,new_confirmed,latitude,longitude,date_recorded
		from confirmed_and_deaths_time_series where country_slug = ?
		`, vars["countryslug"])
	}
	if vars["stat"] == "deaths" {
		rows, err = db.Query(`
		select country,country_slug,province,deaths,new_deaths,latitude,longitude,date_recorded
		from confirmed_and_deaths_time_series where country_slug = ?
		`, vars["countryslug"])
	}
	if vars["stat"] == "recoveries" {
		rows, err = db.Query(`
		select country,country_slug,province,recoveries,new_recoveries,latitude,longitude,date_recorded
		from recoveries_time_series where country_slug = ?
		`, vars["countryslug"])
	}

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	timeSeriesResponse := new(TimeSeriesResponse)

	for rows.Next() {
		if vars["stat"] == "confirmed" {
			timeSeries := new(ConfirmedTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Province,
				&timeSeries.Confirmed,
				&timeSeries.NewConfirmed,
				&timeSeries.Lat,
				&timeSeries.Long,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
		if vars["stat"] == "recoveries" {
			timeSeries := new(RecoveriesTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Province,
				&timeSeries.Recoveries,
				&timeSeries.NewRecoveries,
				&timeSeries.Lat,
				&timeSeries.Long,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
		if vars["stat"] == "deaths" {
			timeSeries := new(DeathsTimeSeries)

			err = rows.Scan(
				&timeSeries.CountryName,
				&timeSeries.CountrySlug,
				&timeSeries.Province,
				&timeSeries.Deaths,
				&timeSeries.NewDeaths,
				&timeSeries.Lat,
				&timeSeries.Long,
				&timeSeries.Date,
			)

			if err != nil {
				log.Fatal(err)
			}

			timeSeriesResponse.TimeSeriesArray = append(timeSeriesResponse.TimeSeriesArray, *timeSeries)
		}
	}

	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(timeSeriesResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func summary(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
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
		log.Fatal(err)
	}
	defer rows.Close()

	summaryResponse := new(SummaryResponse)

	for rows.Next() {
		countrySummary := new(CountrySummary)

		err := rows.Scan(
			&countrySummary.Country,
			&countrySummary.CountrySlug,
			&countrySummary.Confirmed,
			&countrySummary.NewConfirmed,
			&countrySummary.Deaths,
			&countrySummary.NewDeaths,
			&countrySummary.Recoveries,
			&countrySummary.NewRecoveries,
		)
		if err != nil {
			log.Fatal(err)
		}

		summaryResponse.World.Confirmed += countrySummary.Confirmed
		summaryResponse.World.NewConfirmed += countrySummary.NewConfirmed
		summaryResponse.World.Deaths += countrySummary.Deaths
		summaryResponse.World.NewDeaths += countrySummary.NewDeaths
		summaryResponse.World.Recoveries += countrySummary.Recoveries
		summaryResponse.World.NewRecoveries += countrySummary.NewRecoveries

		summaryResponse.Countries = append(summaryResponse.Countries, *countrySummary)
	}

	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(summaryResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func saveRecoveries(recoveries *http.Response) {
	defer recoveries.Body.Close()

	recoveriesReader := csv.NewReader(recoveries.Body)
	headers, err := recoveriesReader.Read()
	if err != nil {
		log.Fatal(err)
	}

	dates := headers[4:]

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO recoveries_time_series 
	(province,country,country_slug,latitude,longitude,recoveries,new_recoveries,date_recorded) 
	VALUES (?,?,?,?,?,?,?,?)
	ON DUPLICATE KEY UPDATE recoveries = VALUES(recoveries), new_recoveries = VALUES(new_recoveries)
	`)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for {
		row, err := recoveriesReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		lat, _ := strconv.ParseFloat(row[2], 64)
		long, _ := strconv.ParseFloat(row[3], 64)

		prevRecoveries := 0

		for j := 0; j < len(dates); j++ {
			recoveries, _ := strconv.Atoi(row[j+4])
			date, _ := time.Parse("1/2/06", dates[j])

			_, err = stmt.Exec(row[0], row[1], generateCountrySlug(row[1]), lat, long, recoveries, max(0, recoveries-prevRecoveries), date)
			if err != nil {
				log.Fatal(err)
			}
			prevRecoveries = recoveries
		}

	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done w/ recoveries")

}

// TODO Dont use log.Fatal here
func saveConfirmedAndDeaths(confirmedCasesRes *http.Response, deathsRes *http.Response) {
	defer confirmedCasesRes.Body.Close()
	defer deathsRes.Body.Close()

	confirmedCasesReader := csv.NewReader(confirmedCasesRes.Body)
	deathsReader := csv.NewReader(deathsRes.Body)

	headers, err := confirmedCasesReader.Read()
	if err != nil {
		log.Fatal(err)
	}

	_, err = deathsReader.Read()
	if err != nil {
		log.Fatal(err)
	}

	dates := headers[4:]

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO confirmed_and_deaths_time_series 
	(province,country,country_slug,latitude,longitude,confirmed_cases,new_confirmed,deaths,new_deaths,date_recorded) 
	VALUES (?,?,?,?,?,?,?,?,?,?)
	ON DUPLICATE KEY UPDATE confirmed_cases = VALUES(confirmed_cases), new_confirmed = VALUES(new_confirmed),deaths = VALUES(deaths), new_deaths = VALUES(new_deaths)
	`)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for {
		prevConfirmed := 0
		prevDeaths := 0

		confirmedCasesRow, err := confirmedCasesReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		deathsRow, err := deathsReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		lat, _ := strconv.ParseFloat(confirmedCasesRow[2], 64)
		long, _ := strconv.ParseFloat(confirmedCasesRow[3], 64)

		for j := 0; j < len(dates); j++ {
			deaths, _ := strconv.Atoi(deathsRow[j+4])
			confirmed, _ := strconv.Atoi(confirmedCasesRow[j+4])
			date, _ := time.Parse("1/2/06", dates[j])

			_, err = stmt.Exec(confirmedCasesRow[0], confirmedCasesRow[1], generateCountrySlug(confirmedCasesRow[1]), lat, long, confirmed,
				max(0, confirmed-prevConfirmed), deaths, max(0, deaths-prevDeaths), date)

			if err != nil {
				log.Fatal(err)
			}
			prevConfirmed = confirmed
			prevDeaths = deaths
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done w/ confirmed and deaths")
}

// func saveUsaConfirmedAndDeaths(confirmedCasesRes *http.Response, deathsRes *http.Response) {
// 	defer confirmedCasesRes.Body.Close()
// 	defer deathsRes.Body.Close()

// 	confirmedCasesReader := csv.NewReader(confirmedCasesRes.Body)
// 	deathsReader := csv.NewReader(deathsRes.Body)

// 	headers, err := confirmedCasesReader.Read()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	_, err = deathsReader.Read()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// TODO make date offset a var
// 	dates := headers[11:]

// 	tx, err := db.Begin()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer tx.Rollback()

// 	stmt, err := tx.Prepare(`
// 	INSERT INTO confirmed_and_deaths_time_series
// 	(county,province,country,country_slug,latitude,longitude,confirmed_cases,new_confirmed,deaths,new_deaths,date_recorded)
// 	VALUES (?,?,?,?,?,?,?,?,?,?,?)
// 	ON DUPLICATE KEY UPDATE confirmed_cases = VALUES(confirmed_cases), new_confirmed = VALUES(new_confirmed),deaths = VALUES(deaths), new_deaths = VALUES(new_deaths)
// 	`)

// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer stmt.Close()

// 	for {
// 		prevConfirmed := 0
// 		prevDeaths := 0

// 		confirmedCasesRow, err := confirmedCasesReader.Read()
// 		if err == io.EOF {
// 			break
// 		} else if err != nil {
// 			log.Fatal(err)
// 		}

// 		deathsRow, err := deathsReader.Read()
// 		if err == io.EOF {
// 			break
// 		} else if err != nil {
// 			log.Fatal(err)
// 		}

// 		lat, _ := strconv.ParseFloat(confirmedCasesRow[8], 64)
// 		long, _ := strconv.ParseFloat(confirmedCasesRow[9], 64)

// 		for j := 0; j < len(dates); j++ {
// 			deaths, _ := strconv.Atoi(deathsRow[j+11+1])
// 			confirmed, _ := strconv.Atoi(confirmedCasesRow[j+11])
// 			date, _ := time.Parse("1/2/06", dates[j])

// 			_, err = stmt.Exec(confirmedCasesRow[5], confirmedCasesRow[6], confirmedCasesRow[7], confirmedCasesRow[7], lat, long, confirmed,
// 				max(0, confirmed-prevConfirmed), deaths, max(0, deaths-prevDeaths), date)
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			prevConfirmed = confirmed
// 			prevDeaths = deaths
// 		}
// 	}

// 	if err := tx.Commit(); err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("Done w/ US confirmed and deaths")
// }
