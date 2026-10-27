package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// 本文件集中加载所有 .env 配置，并暴露全局只读配置结构体 config.C。
var C *Config

type Config struct {
	App       AppConfig
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	CORS      CORSConfig
	File      FileConfig
	Task      TaskConfig
	Log       LogConfig
	RateLimit RateLimitConfig
	Audit     AuditConfig
	AI        AIConfig
	Knowledge KnowledgeConfig
	Microsoft MicrosoftConfig
}

type AppConfig struct {
	Name    string
	Env     string
	Version string
}

type ServerConfig struct {
	Host string
	Port int
}

func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Enabled         bool
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	DSN             string
}

func (d DatabaseConfig) BuildDSN() string {
	if d.DSN != "" {
		return d.DSN
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		d.Host,
		d.User,
		d.Password,
		d.Name,
		d.Port,
		d.SSLMode,
	)
}

type RedisConfig struct {
	Enabled      bool
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Issuer        string
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

type FileConfig struct {
	StorageDriver string
	UploadDir     string
	MaxUploadMB   int64
	AllowedTypes  []string
	MinIOEndpoint string
	MinIOBucket   string
	MinIOAccessID string
	MinIOSecret   string
	MinIOUseSSL   bool
}

type TaskConfig struct {
	WorkerCount int
	RetryLimit  int
}

type LogConfig struct {
	Level string
}

type RateLimitConfig struct {
	Enabled bool
	Limit   int
	Window  time.Duration
}

type AuditConfig struct {
	LogFile     string
	MaxSizeMB   int64
	MaxBackups  int
	PersistToDB bool
}

type AIConfig struct {
	ServiceBaseURL string
	Timeout        time.Duration
}

type KnowledgeConfig struct {
	ServiceBaseURL string
	APIKey         string
	Timeout        time.Duration
}

type MicrosoftConfig struct {
	TenantID           string
	ClientID           string
	ClientSecret       string
	RedirectURI        string
	GraphScopes        []string
	TokenEncryptionKey string
	FrontendURL        string
}

func Load(root string) (*Config, error) {
	env, err := loadEnvMap(root)
	if err != nil {
		return nil, err
	}

	defaultAuditPersistToDB := false
	if strings.EqualFold(get(env, "APP_ENV", "dev"), "prod") || strings.EqualFold(get(env, "APP_ENV", "dev"), "production") {
		defaultAuditPersistToDB = true
	}

	cfg := &Config{
		App: AppConfig{
			Name:    get(env, "APP_NAME", "WFAgent"),
			Env:     get(env, "APP_ENV", "dev"),
			Version: get(env, "APP_VERSION", "0.1.0"),
		},
		Server: ServerConfig{
			Host: get(env, "HTTP_HOST", "0.0.0.0"),
			Port: getInt(env, "HTTP_PORT", 8080),
		},
		Database: DatabaseConfig{
			Enabled:         getBool(env, "DB_ENABLED", false),
			Host:            get(env, "POSTGRES_HOST", "127.0.0.1"),
			Port:            getInt(env, "POSTGRES_PORT", 5432),
			User:            get(env, "POSTGRES_USER", "postgres"),
			Password:        get(env, "POSTGRES_PASSWORD", ""),
			Name:            get(env, "POSTGRES_DATABASE", "wfagent_db"),
			SSLMode:         get(env, "POSTGRES_SSLMODE", "disable"),
			MaxIdleConns:    getInt(env, "POSTGRES_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getInt(env, "POSTGRES_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getDuration(env, "POSTGRES_CONN_MAX_LIFETIME", time.Hour),
			DSN:             get(env, "POSTGRES_DSN", ""),
		},
		Redis: RedisConfig{
			Enabled:      getBool(env, "REDIS_ENABLED", true),
			Host:         get(env, "REDIS_HOST", "127.0.0.1"),
			Port:         getInt(env, "REDIS_PORT", 6379),
			Password:     get(env, "REDIS_PASSWORD", ""),
			DB:           getInt(env, "REDIS_DB", 0),
			PoolSize:     getInt(env, "REDIS_POOL_SIZE", 10),
			MinIdleConns: getInt(env, "REDIS_MIN_IDLE_CONNS", 2),
			DialTimeout:  getDuration(env, "REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getDuration(env, "REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getDuration(env, "REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		JWT: JWTConfig{
			AccessSecret:  get(env, "JWT_ACCESS_SECRET", "change-me-access-secret"),
			RefreshSecret: get(env, "JWT_REFRESH_SECRET", "change-me-refresh-secret"),
			AccessTTL:     getDuration(env, "JWT_ACCESS_TTL", 2*time.Hour),
			RefreshTTL:    getDuration(env, "JWT_REFRESH_TTL", 7*24*time.Hour),
			Issuer:        get(env, "JWT_ISSUER", "wfagent"),
		},
		CORS: CORSConfig{
			AllowOrigins:     splitCSV(get(env, "CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001,http://localhost:5173,http://127.0.0.1:5173")),
			AllowMethods:     splitCSV(get(env, "CORS_ALLOW_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")),
			AllowHeaders:     splitCSV(get(env, "CORS_ALLOW_HEADERS", "Authorization,Content-Type,X-Request-ID")),
			AllowCredentials: getBool(env, "CORS_ALLOW_CREDENTIALS", true),
		},
		File: FileConfig{
			StorageDriver: get(env, "FILE_STORAGE_DRIVER", "minio"),
			UploadDir:     get(env, "FILE_UPLOAD_DIR", "storage/uploads"),
			MaxUploadMB:   int64(getInt(env, "FILE_MAX_UPLOAD_MB", 50)),
			AllowedTypes:  splitCSV(get(env, "FILE_ALLOWED_TYPES", ".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.png,.jpg,.jpeg,.txt,.csv,.zip")),
			MinIOEndpoint: get(env, "MINIO_ENDPOINT", "127.0.0.1:9000"),
			MinIOBucket:   get(env, "MINIO_BUCKET", "wfagent"),
			MinIOAccessID: get(env, "MINIO_ACCESS_KEY", "minioadmin"),
			MinIOSecret:   get(env, "MINIO_SECRET_KEY", "minioadmin"),
			MinIOUseSSL:   getBool(env, "MINIO_USE_SSL", false),
		},
		Task: TaskConfig{
			WorkerCount: getInt(env, "TASK_WORKER_COUNT", 2),
			RetryLimit:  getInt(env, "TASK_RETRY_LIMIT", 3),
		},
		Log: LogConfig{
			Level: get(env, "LOG_LEVEL", "info"),
		},
		RateLimit: RateLimitConfig{
			Enabled: getBool(env, "RATE_LIMIT_ENABLED", true),
			Limit:   getInt(env, "RATE_LIMIT_PER_MINUTE", 100),
			Window:  getDuration(env, "RATE_LIMIT_WINDOW", time.Minute),
		},
		Audit: AuditConfig{
			LogFile:     get(env, "AUDIT_LOG_FILE", "logs/audit.log"),
			MaxSizeMB:   int64(getInt(env, "AUDIT_LOG_MAX_SIZE_MB", 20)),
			MaxBackups:  getInt(env, "AUDIT_LOG_MAX_BACKUPS", 5),
			PersistToDB: getBool(env, "AUDIT_PERSIST_TO_DB", defaultAuditPersistToDB),
		},
		AI: AIConfig{
			ServiceBaseURL: get(env, "AI_SERVICE_BASE_URL", "http://127.0.0.1:9100"),
			Timeout:        getDuration(env, "AI_SERVICE_TIMEOUT", 60*time.Second),
		},
		Knowledge: KnowledgeConfig{
			ServiceBaseURL: get(env, "KNOWLEDGE_SERVICE_BASE_URL", "http://127.0.0.1:8201"),
			APIKey:         get(env, "KNOWLEDGE_SERVICE_API_KEY", ""),
			Timeout:        getDuration(env, "KNOWLEDGE_SERVICE_TIMEOUT", 30*time.Second),
		},
		Microsoft: MicrosoftConfig{
			TenantID:           get(env, "MICROSOFT_TENANT_ID", "common"),
			ClientID:           get(env, "MICROSOFT_CLIENT_ID", ""),
			ClientSecret:       get(env, "MICROSOFT_CLIENT_SECRET", ""),
			RedirectURI:        get(env, "MICROSOFT_REDIRECT_URI", "http://localhost:8080/v1/integrations/microsoft/callback"),
			GraphScopes:        splitCSV(get(env, "MICROSOFT_GRAPH_SCOPES", "openid,profile,offline_access,User.Read,Mail.ReadWrite,Mail.Send")),
			TokenEncryptionKey: get(env, "MICROSOFT_TOKEN_ENCRYPTION_KEY", ""),
			FrontendURL:        get(env, "APP_FRONTEND_URL", "http://localhost:3000"),
		},
	}

	if err := validateProductionCORS(cfg); err != nil {
		return nil, err
	}
	if err := validateBaseSecurity(cfg); err != nil {
		return nil, err
	}

	C = cfg
	return cfg, nil
}

func Watch(ctx context.Context, root string, interval time.Duration, onChange func(*Config)) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	last, err := configFingerprint(root)
	if err != nil {
		slog.Warn("config watch disabled", "error", err)
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next, err := configFingerprint(root)
			if err != nil {
				slog.Warn("config watch fingerprint failed", "error", err)
				continue
			}
			if reflect.DeepEqual(last, next) {
				continue
			}
			cfg, err := Load(root)
			if err != nil {
				slog.Warn("config hot reload failed", "error", err)
				continue
			}
			last = next
			slog.Info("config hot reloaded", "env", cfg.App.Env)
			if onChange != nil {
				onChange(cfg)
			}
		}
	}
}

func validateProductionCORS(cfg *Config) error {
	if cfg.App.Env != "prod" && cfg.App.Env != "production" {
		return nil
	}
	for _, origin := range cfg.CORS.AllowOrigins {
		if origin == "*" || strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") || strings.Contains(origin, "*") {
			return fmt.Errorf("CORS_ALLOW_ORIGINS must use explicit production domains when APP_ENV=%s: %s", cfg.App.Env, origin)
		}
	}
	return nil
}

func validateBaseSecurity(cfg *Config) error {
	if cfg.File.StorageDriver != "minio" {
		return fmt.Errorf("FILE_STORAGE_DRIVER must be minio")
	}
	if cfg.App.Env != "prod" && cfg.App.Env != "production" {
		return nil
	}
	for _, secret := range []string{cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret} {
		if len(secret) < 32 || strings.Contains(secret, "please-run-make-init-secret") || strings.Contains(secret, "change-me") {
			return fmt.Errorf("JWT secrets must be securely generated and at least 32 characters when APP_ENV=%s", cfg.App.Env)
		}
	}
	return nil
}

func loadEnvMap(root string) (map[string]string, error) {
	if root == "" {
		root = "."
	}
	env := map[string]string{}
	yamlFiles, err := DiscoverYAMLFiles(root)
	if err != nil {
		return nil, err
	}
	for _, file := range yamlFiles {
		values, err := parseYAMLFile(file)
		if err != nil {
			return nil, err
		}
		for key, value := range values {
			env[key] = value
		}
	}
	files, err := DiscoverEnvFiles(root)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		values, err := parseEnvFile(file)
		if err != nil {
			return nil, err
		}
		for key, value := range values {
			env[key] = value
		}
	}
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			env[key] = value
		}
	}
	return env, nil
}

func DiscoverEnvFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, ".env") || name == ".env.example" {
			continue
		}
		files = append(files, filepath.Join(root, name))
	}
	sort.Strings(files)
	return files, nil
}

func DiscoverYAMLFiles(root string) ([]string, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "dev"
	}
	candidates := []string{
		filepath.Join(root, "config.yaml"),
		filepath.Join(root, "config.yml"),
		filepath.Join(root, "configs", "config.yaml"),
		filepath.Join(root, "configs", "config.yml"),
		filepath.Join(root, "configs", "config."+appEnv+".yaml"),
		filepath.Join(root, "configs", "config."+appEnv+".yml"),
	}
	files := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			files = append(files, candidate)
		}
	}
	sort.Strings(files)
	return files, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, nil
}

func parseYAMLFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	if err := yaml.Unmarshal(content, &values); err != nil {
		return nil, err
	}
	flat := map[string]string{}
	flattenYAML("", values, flat)
	return flat, nil
}

func flattenYAML(prefix string, values map[string]any, out map[string]string) {
	for key, value := range values {
		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		if prefix != "" {
			envKey = prefix + "_" + envKey
		}
		switch typed := value.(type) {
		case map[string]any:
			flattenYAML(envKey, typed, out)
		case []any:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				parts = append(parts, fmt.Sprint(item))
			}
			out[envKey] = strings.Join(parts, ",")
		default:
			out[envKey] = fmt.Sprint(typed)
		}
	}
}

func configFingerprint(root string) (map[string]time.Time, error) {
	envFiles, err := DiscoverEnvFiles(root)
	if err != nil {
		return nil, err
	}
	yamlFiles, err := DiscoverYAMLFiles(root)
	if err != nil {
		return nil, err
	}
	files := append(envFiles, yamlFiles...)
	out := make(map[string]time.Time, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		out[file] = info.ModTime()
	}
	return out, nil
}

func get(env map[string]string, key string, fallback string) string {
	if value, ok := env[key]; ok && value != "" {
		return value
	}
	return fallback
}

func getInt(env map[string]string, key string, fallback int) int {
	raw := get(env, key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getBool(env map[string]string, key string, fallback bool) bool {
	raw := strings.ToLower(get(env, key, ""))
	if raw == "" {
		return fallback
	}
	return raw == "true" || raw == "1" || raw == "yes" || raw == "y"
}

func getDuration(env map[string]string, key string, fallback time.Duration) time.Duration {
	raw := get(env, key, "")
	if raw == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
