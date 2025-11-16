package config

import "os"

type Config struct {
	Port               string
	DB_DSN             string
	JWTSecret          string
	CORSTrustedOrigins string
	SMTPHost           string
	SMTPPort           string
	SMTPFrom           string
	SMTPUsername       string // Add this
	SMTPPassword       string // Add this
}

// LoadConfig loads environment variables into a Config struct
func LoadConfig() Config {
	return Config{
		Port:               getEnv("PORT", "8081"),
		DB_DSN:             getEnv("DB_DSN", "postgres://user:password@postgres/mydb?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "supersecretjwtkey"),
		CORSTrustedOrigins: getEnv("CORS_TRUSTED_ORIGINS", "http://localhost:8080,http://localhost:3000,http://localhost:53589,http://localhost:*,http://127.0.0.1:*"),
		SMTPHost:           getEnv("SMTP_HOST", "mailpit"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPFrom:           getEnv("SMTP_FROM", "noreply@example.com"),
		SMTPUsername:       getEnv("SMTP_USERNAME", ""), // Add this
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""), // Add this
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}