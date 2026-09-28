package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLookupCityByCEP_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"localidade": "São Paulo"})
	}))
	defer srv.Close()

	os.Setenv("VIACEP_BASE_URL", srv.URL)
	defer os.Unsetenv("VIACEP_BASE_URL")

	city, err := lookupCityByCEP("01001000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city != "São Paulo" {
		t.Fatalf("unexpected city: %s", city)
	}
}

func TestLookupCityByCEP_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"erro": true})
	}))
	defer srv.Close()

	os.Setenv("VIACEP_BASE_URL", srv.URL)
	defer os.Unsetenv("VIACEP_BASE_URL")

	_, err := lookupCityByCEP("99999999")
	if err == nil {
		t.Fatalf("expected error for not found")
	}
}

func TestLookupTemperatureByCity_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"current": map[string]any{"temp_c": 24}})
	}))
	defer srv.Close()

	os.Setenv("WEATHERAPI_BASE_URL", srv.URL)
	os.Setenv("WEATHERAPI_KEY", "key")
	defer os.Unsetenv("WEATHERAPI_BASE_URL")
	defer os.Unsetenv("WEATHERAPI_KEY")

	temp, err := lookupTemperatureByCity("São Paulo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if temp != 24 {
		t.Fatalf("unexpected temp: %v", temp)
	}
}

func TestLookupTemperatureByCity_MissingKey(t *testing.T) {
	os.Unsetenv("WEATHERAPI_KEY")
	os.Setenv("WEATHERAPI_BASE_URL", "http://example.invalid")
	defer os.Unsetenv("WEATHERAPI_BASE_URL")

	_, err := lookupTemperatureByCity("City")
	if err == nil {
		t.Fatalf("expected error when WEATHERAPI_KEY missing")
	}
}

func TestLookupTemperatureByCity_WeatherApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "backend error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	os.Setenv("WEATHERAPI_BASE_URL", srv.URL)
	os.Setenv("WEATHERAPI_KEY", "key")
	defer os.Unsetenv("WEATHERAPI_BASE_URL")
	defer os.Unsetenv("WEATHERAPI_KEY")

	_, err := lookupTemperatureByCity("City")
	if err == nil {
		t.Fatalf("expected error for WeatherAPI 500")
	}
}
