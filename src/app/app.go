package app

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type WeatherResp struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

var cepRegex = regexp.MustCompile(`^[0-9]{8}$`)

// NewHandler returns the HTTP handler for the application.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/weather", weatherHandler)
	return mux
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	cep := r.URL.Query().Get("cep")
	if !cepRegex.MatchString(cep) {
		http.Error(w, "invalid zipcode", 422)
		return
	}

	city, err := lookupCityByCEP(cep)
	if err != nil {
		http.Error(w, "can not find zipcode", 404)
		return
	}

	tempC, err := lookupTemperatureByCity(city)
	if err != nil {
		// provide more informative error when API key is missing
		if err.Error() == "missing WEATHERAPI_KEY" {
			http.Error(w, "missing WEATHERAPI_KEY", 500)
			return
		}
		http.Error(w, "can not get weather", 500)
		return
	}

	resp := WeatherResp{
		TempC: round(tempC, 2),
		TempF: round(cToF(tempC), 2),
		TempK: round(cToK(tempC), 2),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func lookupCityByCEP(cep string) (string, error) {
	base := os.Getenv("VIACEP_BASE_URL")
	if base == "" {
		base = "https://viacep.com.br"
	}
	url := base + "/ws/" + cep + "/json/"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("viacep status: %d for cep=%s", resp.StatusCode, cep)
		return "", errors.New("viacep error")
	}

	var data struct {
		Localidade string `json:"localidade"`
		Erro       bool   `json:"erro"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Erro || data.Localidade == "" {
		log.Printf("viacep: not found or empty localidade for cep=%s", cep)
		return "", errors.New("not found")
	}
	return data.Localidade, nil
}

func lookupTemperatureByCity(city string) (float64, error) {
	apiKey := os.Getenv("WEATHERAPI_KEY")
	if apiKey == "" {
		return 0, errors.New("missing WEATHERAPI_KEY")
	}

	base := os.Getenv("WEATHERAPI_BASE_URL")
	if base == "" {
		base = "http://api.weatherapi.com"
	}
	q := url.QueryEscape(city)
	url := base + "/v1/current.json?key=" + apiKey + "&q=" + q + "&aqi=no"
	client := &http.Client{Timeout: 7 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var b []byte
		b, _ = io.ReadAll(resp.Body)
		log.Printf("weatherapi status=%d url=%s body=%s", resp.StatusCode, url, string(b))
		return 0, errors.New("weatherapi error")
	}

	var data struct {
		Current struct {
			TempC float64 `json:"temp_c"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data.Current.TempC, nil
}

// conversion helpers
func cToF(c float64) float64 { return c*1.8 + 32 }
func cToK(c float64) float64 { return c + 273 }

func round(v float64, prec int) float64 {
	shift := mathPow(10, prec)
	return float64(int(v*shift+0.5)) / shift
}

func mathPow(a float64, b int) float64 {
	res := 1.0
	for i := 0; i < b; i++ {
		res *= a
	}
	return res
}
