package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

type response struct {
	Results []result `json:"results"`
	Status  string   `json:"status"`
}

type result struct {
	Geometry geometry `json:"geometry"`
	Address  string   `json:"formatted_address"`
}

type geometry struct {
	Location     location `json:"location"`
	LocationType string   `json:"location_type"`
}

type location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type serviceParamName string

const (
	serviceParamNameLatLng  serviceParamName = "latlng"
	serviceParamNameAddress serviceParamName = "address"
)

var bindAddress = DefaultENV("BIND_ADDRESS", ":5000")
var googleAPIKey = DefaultENV("GOOGLE_API_KEY", "")

func DefaultENV(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func geoServiceUrl(apiKey string, paramName serviceParamName, paramValue string) string {
	// paramName: address, latlng
	return fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?%s=%s&key=%s", paramName, paramValue, apiKey)
}

func callExternalService(serviceName serviceParamName, param string) []byte {
	url := geoServiceUrl(googleAPIKey, serviceName, param)
	log.Println("About to call: ", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	return body
}

func MockService() []byte {
	resp, err := ioutil.ReadFile("data/LA.json")
	if err != nil {
		fmt.Println(err)
	}
	return resp
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", HomeHandler)
	router.HandleFunc("/geocode/{address}", GeocodeHandler)
	router.HandleFunc("/geoloc/{lat},{lng}", GeolocHandler)

	router.Use(LoggingMiddleware)

	srv := &http.Server{
		Handler:      router,
		Addr:         bindAddress,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	go func() {
		log.Println("Listening on ", bindAddress)
		err := srv.ListenAndServe()
		if err != nil {
			log.Println("Error on ListenAndServe: ", err)
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)
	sig := <-sigChan
	log.Println("FRecieved terminate, graceful shutdown ", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Printf("\t%s \t %s \n", r.Method, r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is up and running!\n"))
}

func GeocodeHandler(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	raw := MockService() // callExternalService(serviceParamNameAddress, vars["address"])
	var resp response
	err := json.Unmarshal(raw, &resp)
	if err != nil {
		log.Println("Error on Unmarshal1: ", err)
	}

	w.WriteHeader(http.StatusOK)
	out, err := json.Marshal(resp.Results[0])
	w.Write([]byte(out))
}

func GeolocHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s\n", vars)))
}
