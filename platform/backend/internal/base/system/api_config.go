package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/response"
)

type apiConfigRequest struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	TimeoutMS string `json:"timeout_ms"`
}

func (h *Handler) GetAPIConfig(c *gin.Context) {
	values, err := readAIEnv()
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, gin.H{
		"provider":           values["AI_PROVIDER"],
		"base_url":           values["AI_BASE_URL"],
		"model":              values["AI_MODEL"],
		"timeout_ms":         values["AI_TIMEOUT_MS"],
		"api_key_configured": strings.TrimSpace(values["AI_API_KEY"]) != "",
	})
}

func (h *Handler) UpdateAPIConfig(c *gin.Context) {
	var req apiConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if strings.TrimSpace(req.BaseURL) == "" || strings.TrimSpace(req.Model) == "" {
		response.Error(c, response.ErrValidation, "AI Base URL and model are required")
		return
	}
	values, err := readAIEnv()
	if err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	values["AI_PROVIDER"] = strings.TrimSpace(req.Provider)
	values["AI_BASE_URL"] = strings.TrimSpace(req.BaseURL)
	values["AI_MODEL"] = strings.TrimSpace(req.Model)
	values["AI_TIMEOUT_MS"] = strings.TrimSpace(req.TimeoutMS)
	if strings.TrimSpace(req.APIKey) != "" {
		values["AI_API_KEY"] = strings.TrimSpace(req.APIKey)
	}
	if err := writeAIEnv(values); err != nil {
		response.Error(c, response.ErrInternal, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "saved", "restart_required": true})
}

func aiEnvPath() string {
	if _, err := os.Stat(filepath.Join("ai", ".env")); err == nil {
		return filepath.Join("ai", ".env")
	}
	return filepath.Join("..", "ai", ".env")
}

func readAIEnv() (map[string]string, error) {
	path := aiEnvPath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && strings.HasPrefix(key, "AI_") {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values, nil
}

func writeAIEnv(values map[string]string) error {
	path := aiEnvPath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	keys := []string{"AI_PROVIDER", "AI_BASE_URL", "AI_API_KEY", "AI_MODEL", "AI_TIMEOUT_MS"}
	remaining := make(map[string]string, len(values))
	for _, key := range keys {
		remaining[key] = values[key]
	}
	lines := make([]string, 0)
	for _, line := range strings.Split(strings.TrimRight(string(data), "\r\n"), "\n") {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok {
			if value, managed := remaining[key]; managed {
				lines = append(lines, fmt.Sprintf("%s=%s", key, value))
				delete(remaining, key)
				continue
			}
		}
		lines = append(lines, line)
	}
	for _, key := range keys {
		if value, ok := remaining[key]; ok {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
