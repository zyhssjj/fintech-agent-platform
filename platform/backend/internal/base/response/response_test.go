package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponseEncodeDecode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.GET("/ok", func(c *gin.Context) {
		Success(c, gin.H{"name": "wfagent"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != OK.Code || body.Message != OK.Message || body.RequestID != "req-test" {
		t.Fatalf("unexpected body: %#v", body)
	}
	data, ok := body.Data.(map[string]any)
	if !ok || data["name"] != "wfagent" {
		t.Fatalf("unexpected data: %#v", body.Data)
	}
}
