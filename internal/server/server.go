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

type Route struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

func New(store store.Service) *Server {
	s := &Server{store: store}
	router := mux.NewRouter()

	router.HandleFunc("/", s.Default).Methods("GET")
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

func (s *Server) Default(w http.ResponseWriter, r *http.Request) {
	routes := []Route{
		Route{
			Path: "/list/countries",
			Description: "Returns a list of countries with their name and slug. " +
				"Please use the country slug when requesting for data for a specific country.",
		},
		Route{
			Path:        "/global",
			Description: "Returns the number of confirmed cases, recoveries, and deaths globally.",
		},
		Route{
			Path:        "/summary",
			Description: "Returns the number of confirmed cases, recoveries, and deaths both globally and per country.",
		},
		Route{
			Path: "/timeseries/{countryslug}/{status}",
			Description: "Returns the history of either confirmed cases, recoveries, and deaths " +
				"of the specified country and each of its provinces " +
				"starting from Jan. 22, 2020. " +
				"{countryslug} must be a valid country slug from '/list/countries'. " +
				"{status} must be one of the following: [confirmed, recoveries, deaths].",
		},
		Route{
			Path: "/timeseries/total/{countryslug}/{status}",
			Description: "Returns the history of either confirmed cases, recoveries, and deaths " +
				"of the specified country starting from Jan. 22, 2020. " +
				"Unlike '/timeseries/{countryslug}/{status}', this route does not return a country's provinces ." +
				"Instead, the data is all summed up. " +
				"{countryslug} must be a valid country slug from '/list/countries'. " +
				"{status} must be one of the following: [confirmed, recoveries, deaths].",
		},
	}

	writeJSONResponse(w, routes)
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
