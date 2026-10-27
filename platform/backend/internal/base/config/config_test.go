package config

import "testing"

func TestValidateBaseSecurity(t *testing.T) {
	validSecret := "0123456789abcdef0123456789abcdef"

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "rejects non MinIO storage",
			config:  Config{App: AppConfig{Env: "dev"}, File: FileConfig{StorageDriver: "local"}},
			wantErr: true,
		},
		{
			name:   "allows MinIO in development",
			config: Config{App: AppConfig{Env: "dev"}, File: FileConfig{StorageDriver: "minio"}},
		},
		{
			name: "rejects production placeholder secret",
			config: Config{
				App:  AppConfig{Env: "prod"},
				File: FileConfig{StorageDriver: "minio"},
				JWT:  JWTConfig{AccessSecret: "please-run-make-init-secret", RefreshSecret: "please-run-make-init-secret"},
			},
			wantErr: true,
		},
		{
			name: "allows securely configured production secrets",
			config: Config{
				App:  AppConfig{Env: "prod"},
				File: FileConfig{StorageDriver: "minio"},
				JWT:  JWTConfig{AccessSecret: validSecret, RefreshSecret: validSecret},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateBaseSecurity(&test.config)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateBaseSecurity() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}
