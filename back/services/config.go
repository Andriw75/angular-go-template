package services

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	UseMock          bool
	DBType           string
	DBHost           string
	DBPort           int
	DBUser           string
	DBPassword       string
	DBName           string
	JWTSecret        string
	JWTExpirationMin int
	JWTRenewMin      int
	CookieName       string
	CookieSecure     bool
	CORSOrigin       string
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", ":8080"),
		UseMock:          getEnvBool("USE_MOCK", true),
		DBType:           getEnv("DB_TYPE", "postgres"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnvInt("DB_PORT", 5432),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "mydb"),
		JWTSecret:        getEnv("JWT_SECRET", "mi-secreto-super-seguro-2024"),
		JWTExpirationMin: getEnvInt("JWT_EXPIRATION_MINUTES", 15),
		JWTRenewMin:      getEnvInt("JWT_RENEW_MINUTES", 5),
		CookieName:       getEnv("COOKIE_NAME", "access_token_go"),
		CookieSecure:     getEnvBool("COOKIE_SECURE", false),
		CORSOrigin:       getEnv("CORS_ORIGIN", "http://localhost:4200"),
	}
	return cfg, nil
}

func (c *Config) DBDSN() string {
	switch c.DBType {
	case "postgres":
		return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + itoa(c.DBPort) + "/" + c.DBName
	case "mysql":
		return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + itoa(c.DBPort) + ")/" + c.DBName + "?parseTime=true"
	default:
		return ""
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
