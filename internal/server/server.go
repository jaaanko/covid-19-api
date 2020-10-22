package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jaaanko/covid-19-api/internal/store"
)

type Server struct {
	store   store.Service
	handler http.Handler
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(store store.Service) *Server {
	s := &Server{store: store}
	router := mux.NewRouter()

	router.HandleFunc("/list/countries", s.GetCountries).Methods("GET")
	router.HandleFunc("/global", s.GetGlobalStats).Methods("GET")
	router.HandleFunc("/summary", s.GetSummary).Methods("GET")
	router.Handle("/timeseries/{countryslug}/{status}", StatusMiddleware(http.HandlerFunc(s.GetTimeSeries))).Methods("GET")
	router.Handle("/timeseries/total/{countryslug}/{status}", StatusMiddleware(http.HandlerFunc(s.GetAggTimeSeries))).Methods("GET")

	s.handler = router
	return s
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.handler)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&errorResponse{Error: err.Error()})
}

func (s *Server) GetCountries(w http.ResponseWriter, r *http.Request) {
	countries, err := s.store.GetCountries()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, countries)
	}
}

func (s *Server) GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	globalStats, err := s.store.GetGlobalStats()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, globalStats)
	}
}

func (s *Server) GetSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.store.GetSummary()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, summary)
	}
}

func (s *Server) GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	timeSeries, err := s.store.GetTimeSeries(vars["countryslug"], vars["status"])

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, timeSeries)
	}
}

func (s *Server) GetAggTimeSeries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	aggTimeSeries, err := s.store.GetAggTimeSeries(vars["countryslug"], vars["status"])

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	} else {
		writeJSONResponse(w, aggTimeSeries)
	}
}

func StatusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if vars["status"] != store.Confirmed && vars["status"] != store.Recoveries && vars["status"] != store.Deaths {
			writeError(
				w,
				http.StatusBadRequest,
				errors.New("Invalid status. Please select from the following: confirmed, recoveries, deaths"),
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}
