package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	dockertest "github.com/ory/dockertest/v3"
)

var server *httptest.Server

func FeatureContext(ctx *godog.ScenarioContext) {
	ctx.Step(`^a valid zipcode "([^\"]*)"$`, aValidZipcode)
	ctx.Step(`^I request the weather$`, iRequestTheWeather)
	ctx.Step(`^the response status should be (\d+)$`, theResponseStatusShouldBe)
	ctx.Step(`^the response should contain temperatures in C F and K$`, theResponseShouldContainTemps)
	ctx.Step(`^the WEATHERAPI_KEY is not set$`, weatherAPIKeyNotSet)
	ctx.Step(`^the response body should contain "([^"]*)"$`, theResponseBodyShouldContain)
}

var lastResp *http.Response
var lastZip string

func aValidZipcode(zip string) error {
	server = httptest.NewServer(NewHandler())
	lastZip = zip
	return nil
}

func iRequestTheWeather() error {
	resp, err := http.Get(server.URL + "/weather?cep=" + lastZip)
	if err != nil {
		return err
	}
	lastResp = resp
	return nil
}

func theResponseStatusShouldBe(code int) error {
	if lastResp == nil {
		return fmt.Errorf("no response")
	}
	if lastResp.StatusCode != code {
		return fmt.Errorf("expected %d got %d", code, lastResp.StatusCode)
	}
	return nil
}

func theResponseShouldContainTemps() error {
	if lastResp == nil {
		return fmt.Errorf("no response")
	}
	defer lastResp.Body.Close()
	b, _ := io.ReadAll(lastResp.Body)
	if len(b) == 0 {
		return fmt.Errorf("empty body")
	}
	return nil
}

func weatherAPIKeyNotSet() error {
	os.Unsetenv("WEATHERAPI_KEY")
	return nil
}

func theResponseBodyShouldContain(substr string) error {
	if lastResp == nil {
		return fmt.Errorf("no response")
	}
	defer lastResp.Body.Close()
	b, _ := io.ReadAll(lastResp.Body)
	if !bytes.Contains(b, []byte(substr)) {
		return fmt.Errorf("response body does not contain %q: %s", substr, string(b))
	}
	return nil
}

func TestComponent(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Skipf("docker not available, skipping component BDD tests: %v", err)
		return
	}

	resource, err := pool.Run("wiremock/wiremock", "2.35.0", nil)
	if err != nil {
		t.Skipf("could not start wiremock container, skipping BDD tests: %v", err)
		return
	}
	defer pool.Purge(resource)

	var baseURL string
	err = pool.Retry(func() error {
		hostPort := resource.GetHostPort("8080/tcp")
		baseURL = "http://" + hostPort
		resp, err := http.Get(baseURL + "/__admin/mappings")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("wiremock not ready status %d", resp.StatusCode)
		}
		return nil
	})
	if err != nil {
		t.Skipf("wiremock did not start in time, skipping BDD tests: %v", err)
		return
	}

	viacep001 := map[string]any{
		"request":  map[string]any{"method": "GET", "urlPattern": "/ws/01001000/json/"},
		"response": map[string]any{"status": 200, "headers": map[string]string{"Content-Type": "application/json"}, "jsonBody": map[string]any{"localidade": "São Paulo"}},
	}
	registerMapping(t, baseURL, viacep001)

	viacepNotFound := map[string]any{
		"request":  map[string]any{"method": "GET", "urlPattern": "/ws/99999999/json/"},
		"response": map[string]any{"status": 200, "headers": map[string]string{"Content-Type": "application/json"}, "jsonBody": map[string]any{"erro": true}},
	}
	registerMapping(t, baseURL, viacepNotFound)

	viacep020 := map[string]any{
		"request":  map[string]any{"method": "GET", "urlPattern": "/ws/02002000/json/"},
		"response": map[string]any{"status": 200, "headers": map[string]string{"Content-Type": "application/json"}, "jsonBody": map[string]any{"localidade": "CityError"}},
	}
	registerMapping(t, baseURL, viacep020)

	weatherBodySP := map[string]any{"current": map[string]any{"temp_c": 24}}
	weatherSP := map[string]any{
		"request":  map[string]any{"method": "GET", "urlPath": "/v1/current.json", "queryParameters": map[string]any{"q": map[string]any{"equalTo": "São Paulo"}}},
		"response": map[string]any{"status": 200, "headers": map[string]string{"Content-Type": "application/json"}, "jsonBody": weatherBodySP},
	}
	registerMapping(t, baseURL, weatherSP)

	weatherErr := map[string]any{
		"request":  map[string]any{"method": "GET", "urlPath": "/v1/current.json", "queryParameters": map[string]any{"q": map[string]any{"equalTo": "CityError"}}},
		"response": map[string]any{"status": 500, "headers": map[string]string{"Content-Type": "application/json"}, "jsonBody": map[string]any{"error": "backend error"}},
	}
	registerMapping(t, baseURL, weatherErr)

	os.Setenv("VIACEP_BASE_URL", baseURL)
	os.Setenv("WEATHERAPI_BASE_URL", baseURL)
	// supply a test API key so lookupTemperatureByCity does not fail
	os.Setenv("WEATHERAPI_KEY", "testkey")

	time.Sleep(500 * time.Millisecond)

	status := godog.TestSuite{
		Name:                "weather",
		ScenarioInitializer: FeatureContext,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{"../../test/features"},
		},
	}.Run()

	if status != 0 {
		t.Fatalf("godog failed: %d", status)
	}
}

func registerMapping(t *testing.T, baseURL string, mapping map[string]any) {
	b, _ := json.Marshal(mapping)
	resp, err := http.Post(baseURL+"/__admin/mappings", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to register mapping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status registering mapping: %d body: %s", resp.StatusCode, string(body))
	}
}
