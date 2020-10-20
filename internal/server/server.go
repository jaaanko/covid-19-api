package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jaaanko/covid-19-api/internal/store"
)

type Server struct {
	store   store.Service
	handler http.Handler
}

func New(store store.Service) *Server {
	s := &Server{store: store}
	router := mux.NewRouter()

	router.HandleFunc("/list/countries", s.getCountriesList).Methods("GET")
	router.HandleFunc("/global", s.getGlobalStats).Methods("GET")
	router.HandleFunc("/summary", s.getSummary).Methods("GET")
	router.Handle("/timeseries/{countryslug}/{status}", statusMiddleware(http.HandlerFunc(s.getTimeSeries))).Methods("GET")
	router.Handle("/timeseries/total/{countryslug}/{status}", statusMiddleware(http.HandlerFunc(s.getAggTimeSeries))).Methods("GET")

	s.handler = router
	return s
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.handler)
}

func writeJSONResponse(w http.ResponseWriter, p interface{}, statusCode int, err error) {
	var payload interface{}

	if err != nil {
		payload = map[string]string{"error": err.Error()}
	} else {
		payload = p
	}

	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(payload)
}

func (s *Server) getCountriesList(w http.ResponseWriter, r *http.Request) {
	countries, err := s.store.GetCountries()

	if err != nil {
		writeJSONResponse(w, nil, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, map[string][]store.Country{"countries": countries}, http.StatusOK, nil)
	}
}

func (s *Server) getGlobalStats(w http.ResponseWriter, r *http.Request) {
	globalStats, err := s.store.GetGlobalStats()

	if err != nil {
		writeJSONResponse(w, nil, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, globalStats, http.StatusOK, nil)
	}
}

func (s *Server) getSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.store.GetSummary()

	if err != nil {
		writeJSONResponse(w, nil, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, summary, http.StatusOK, nil)
	}
}

func (s *Server) getTimeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	timeSeries, err := s.store.GetTimeSeries(vars["countryslug"], vars["status"])

	if err != nil {
		writeJSONResponse(w, nil, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, timeSeries, http.StatusOK, nil)
	}
}

func (s *Server) getAggTimeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	aggTimeSeries, err := s.store.GetAggTimeSeries(vars["countryslug"], vars["status"])
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		writeJSONResponse(w, nil, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, aggTimeSeries, http.StatusOK, nil)
	}
}

func statusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if vars["status"] != store.Confirmed && vars["status"] != store.Recoveries && vars["status"] != store.Deaths {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
