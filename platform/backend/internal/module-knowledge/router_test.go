package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wfagent/internal/base/config"
)

func TestCallForwardsInternalKeyAndCompanyFilter(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/retrieve" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"answer":"ok","sources":[]}`))
	}))
	defer server.Close()
	previous := config.C
	config.C = &config.Config{Knowledge: config.KnowledgeConfig{ServiceBaseURL: server.URL, APIKey: "test-key", Timeout: time.Second}}
	defer func() { config.C = previous }()

	data, err := (&Handler{}).call(context.Background(), "/api/v1/retrieve", QueryRequest{Query: "基金运营", CompanyCode: "beijing_hangke", TopK: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got["company_code"] != "beijing_hangke" || got["top_k"].(float64) != 4 {
		t.Fatalf("unexpected forwarded body: %#v", got)
	}
	if data["answer"] != "ok" {
		t.Fatalf("unexpected response: %#v", data)
	}
}

func TestCallRequiresConfiguration(t *testing.T) {
	previous := config.C
	config.C = &config.Config{}
	defer func() { config.C = previous }()
	if _, err := (&Handler{}).call(context.Background(), "/api/v1/retrieve", QueryRequest{Query: "x", CompanyCode: "beijing_hangke"}); err != errNotConfigured {
		t.Fatalf("error = %v, want %v", err, errNotConfigured)
	}
}
