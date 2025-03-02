package main

import (
	"fmt"
	l "log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/pact-foundation/pact-go/v2/log"
	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/pact-foundation/pact-go/v2/provider"
	"github.com/pact-foundation/pact-go/v2/version"
	"github.com/stretchr/testify/assert"
)

var dir, _ = os.Getwd()
var pactDir = fmt.Sprintf("%s/pacts", dir)

var requestFilterCalled = false
var stateHandlerCalled = false

func TestV3HTTPProvider(t *testing.T) {
	log.SetLogLevel("ERROR")
	version.CheckVersion()

	// Start provider API in the background
	go startServer()

	verifier := provider.NewVerifier()

	// Authorization middleware
	// This is your chance to modify the request before it hits your provider
	// NOTE: this should be used very carefully, as it has the potential to
	// _change_ the contract
	f := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.Println("[DEBUG] HOOK request filter")
			requestFilterCalled = true
			r.Header.Add("Authorization", "Bearer 1234-dynamic-value")
			next.ServeHTTP(w, r)
		})
	}

	// Verify the Provider with local Pact Files

	if os.Getenv("SKIP_PUBLISH") != "true" {
		err := verifier.VerifyProvider(t, provider.VerifyRequest{
			ProviderBaseURL: "http://127.0.0.1:8111",
			Provider:        "V3Provider",
			ProviderVersion: os.Getenv("APP_SHA"),
			BrokerURL:       os.Getenv("PACT_BROKER_BASE_URL"),
			ConsumerVersionSelectors: []provider.Selector{
				&provider.ConsumerVersionSelector{
					Tag: "master",
				},
				&provider.ConsumerVersionSelector{
					Tag: "prod",
				},
			},
			PublishVerificationResults: true,
			RequestFilter:              f,
			BeforeEach: func() error {
				l.Println("[DEBUG] HOOK before each")
				return nil
			},
			AfterEach: func() error {
				l.Println("[DEBUG] HOOK after each")
				return nil
			},
			StateHandlers: models.StateHandlers{
				"User foo exists": func(setup bool, s models.ProviderState) (models.ProviderStateResponse, error) {
					stateHandlerCalled = true

					if setup {
						l.Println("[DEBUG] HOOK calling user foo exists state handler", s)
					} else {
						l.Println("[DEBUG] HOOK teardown the 'User foo exists' state")
					}

					// ... do something, such as create "foo" in the database

					// Optionally (if there are generators in the pact) return provider state values to be used in the verification
					return models.ProviderStateResponse{"uuid": "1234"}, nil
				},
			},
			DisableColoredOutput: true,
		})
		assert.NoError(t, err)
		assert.True(t, requestFilterCalled)
		assert.True(t, stateHandlerCalled)
	} else {
		err := verifier.VerifyProvider(t, provider.VerifyRequest{
			ProviderBaseURL: "http://127.0.0.1:8111",
			Provider:        "V3Provider",
			PactFiles: []string{
				filepath.ToSlash(fmt.Sprintf("%s/pact-demo-ui-pact-demo-be.json", pactDir)),
			},
			RequestFilter: f,
			BeforeEach: func() error {
				l.Println("[DEBUG] HOOK before each")
				return nil
			},
			AfterEach: func() error {
				l.Println("[DEBUG] HOOK after each")
				return nil
			},
			StateHandlers: models.StateHandlers{
				"User foo exists": func(setup bool, s models.ProviderState) (models.ProviderStateResponse, error) {
					stateHandlerCalled = true

					if setup {
						l.Println("[DEBUG] HOOK calling user foo exists state handler", s)
					} else {
						l.Println("[DEBUG] HOOK teardown the 'User foo exists' state")
					}

					// ... do something, such as create "foo" in the database

					// Optionally (if there are generators in the pact) return provider state values to be used in the verification
					return models.ProviderStateResponse{"uuid": "1234"}, nil
				},
			},
			DisableColoredOutput: true,
		})
		assert.NoError(t, err)
		assert.True(t, requestFilterCalled)
		// assert.True(t, stateHandlerCalled)
	}

}

func startServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/users/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, `
			{
				"accountBalance": 123.76,
				"datetime": "2020-01-01",
				"equality": "a thing",
				"id": 12,
				"itemsMin": [
					"thereshouldbe3ofthese",
					"thereshouldbe3ofthese",
					"thereshouldbe3ofthese"
				],
				"itemsMinMax": [
					27,
					27,
					27,
					27,
					27
				],
				"lastName": "Sampson",
				"name": "Billy",
				"superstring": "foo",
				"arrayContaining": [
					"string",
					1,
					{
						"foo": "bar"
					}
				]
			}`,
		)
	})

	l.Fatal(http.ListenAndServe("127.0.0.1:8111", mux))
}

type User struct {
	ID       int    `json:"id" pact:"example=27"`
	Name     string `json:"name" pact:"example=billy"`
	LastName string `json:"lastName" pact:"example=Sampson"`
	Date     string `json:"datetime" pact:"example=2020-01-01'T'08:00:45,format=yyyy-MM-dd'T'HH:mm:ss,generator=datetime"`
}
