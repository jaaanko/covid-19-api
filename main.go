package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jaaanko/covid-19-api/internal/server"
	"github.com/jaaanko/covid-19-api/internal/store"
)

func main() {
	initialTicker := time.NewTicker(1)
	interval := 12 * time.Hour

	ticker := time.NewTicker(interval)
	done := make(chan (bool))

	dbUser := os.Getenv("COVID19_DB_USER")
	dbPass := os.Getenv("COVID19_DB_PASS")
	dbHost := os.Getenv("COVID19_DB_HOST")
	dbPort := os.Getenv("COVID19_DB_PORT")
	dbName := os.Getenv("COVID19_DB_NAME")

	st, err := store.NewMySql(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName))
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

	updateData := func() {
		go func() {
			err := dataCollector.UpdateConfirmedAndDeaths()
			if err != nil {
				log.Fatal(err)
			}
		}()

		go func() {
			err := dataCollector.UpdateRecoveries()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	go func() {
		for {
			select {
			case <-initialTicker.C:
				initialTicker.Stop()
				updateData()
			case <-ticker.C:
				updateData()
			case <-done:
				ticker.Stop()
			}
		}
	}()

	s := server.New(st)
	serverPort := os.Getenv("COVID19_SERVER_PORT")

	err = s.Run(fmt.Sprintf(":%s", serverPort))

	if err != nil {
		log.Fatal(err)
	}
}
