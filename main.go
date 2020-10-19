package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jaaanko/covid-19-api/internal/store"
)

var (
	start time.Time
	st    store.Service
)

func main() {
	initialTicker := time.NewTicker(1)

	ticker := time.NewTicker(12 * time.Hour)
	done := make(chan (bool))

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")

	var err error

	st, err = store.NewMySql(fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/covid19?parseTime=true", dbUser, dbPass))

	if err != nil {
		log.Fatal(err)
	}

	db, err := st.GetDbInstance()

	if err != nil {
		log.Fatal(err)
	}

	dataCollector, err := store.NewJhuCsseDataCollector(db)
	if err != nil {
		log.Fatal(err)
	}

	fetchData := func() {
		var wg sync.WaitGroup
		start = time.Now()

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := dataCollector.UpdateConfirmedAndDeaths()
			if err != nil {
				log.Fatal(err)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := dataCollector.UpdateRecoveries()
			if err != nil {
				log.Fatal(err)
			}
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

	router.HandleFunc("/list/countries", countriesList).Methods("GET")
	router.HandleFunc("/world", world).Methods("GET")
	router.HandleFunc("/summary", summary).Methods("GET")
	router.HandleFunc("/timeseries/{countryslug}/{stat}", timeSeries).Methods("GET")
	router.HandleFunc("/timeseries/total/{countryslug}/{stat}", timeSeriesTotal).Methods("GET")

	http.ListenAndServe(":8080", router)
}

func countriesList(w http.ResponseWriter, r *http.Request) {

	countries, err := st.GetCountries()

	if err != nil {
		log.Fatal(err)
	}

	countryListResponse := struct {
		Countries []store.Country `json:"countries"`
	}{
		countries,
	}

	response, _ := json.Marshal(countryListResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func world(w http.ResponseWriter, r *http.Request) {

	worldSummaryResponse, err := st.GetGlobalStats()
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

	aggTimeSeriesResponse, err := st.GetAggTimeSeries(vars["countryslug"], vars["stat"])
	if err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(aggTimeSeriesResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func timeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if vars["stat"] != store.Confirmed && vars["stat"] != store.Recoveries && vars["stat"] != store.Deaths {
		log.Fatal("Invalid status")
	}

	timeSeriesResponse, err := st.GetTimeSeries(vars["countryslug"], vars["stat"])
	if err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(timeSeriesResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}

func summary(w http.ResponseWriter, r *http.Request) {
	summaryResponse, err := st.GetSummary()

	if err != nil {
		log.Fatal(err)
	}

	response, _ := json.Marshal(summaryResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(response)
}
