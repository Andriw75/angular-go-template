package services

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort           string
	DBType               string
	DBHost               string
	DBPort               int
	DBUser               string
	DBPassword           string
	DBName               string
	DBPath               string
	DBSSLMode            string
	JWTSecret            string
	JWTExpirationMin     int
	JWTRenewMin          int
	CookieName           string
	CookieSecure         bool
	CORSOrigin           string
	JWTStoreEnabled      bool
	JWTLoadUserOnRenew   bool
	PublicDir            string
	XAccelRedirectPrefix string
	ReportSyncTimeout    int
	JobTTLMinutes        int
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	cfg := &Config{
		ServerPort:           getEnv("SERVER_PORT", ":8080"),
		DBType:               getEnv("DB_TYPE", "sqlite"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnvInt("DB_PORT", 5432),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", "postgres"),
		DBName:               getEnv("DB_NAME", "template"),
		DBPath:               getEnv("DB_PATH", filepath.Join("data", "template.db")),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		JWTSecret:            getEnv("JWT_SECRET", "REDACTED_DEFAULT_JWT_SECRET"),
		JWTExpirationMin:     getEnvInt("JWT_EXPIRATION_MINUTES", 15),
		JWTRenewMin:          getEnvInt("JWT_RENEW_MINUTES", 5),
		CookieName:           getEnv("COOKIE_NAME", "access_token_go"),
		CookieSecure:         getEnvBool("COOKIE_SECURE", false),
		CORSOrigin:           getEnv("CORS_ORIGIN", "http://localhost:4200"),
		JWTStoreEnabled:      getEnvBool("JWT_STORE_ENABLED", true),
		JWTLoadUserOnRenew:   getEnvBool("JWT_LOAD_USER_ON_RENEW", true),
		PublicDir:            getEnv("PUBLIC_DIR", ""),
		XAccelRedirectPrefix: getEnv("X_ACCEL_REDIRECT_PREFIX", ""),
		ReportSyncTimeout:    getEnvInt("REPORT_SYNC_TIMEOUT_SECONDS", 60),
		JobTTLMinutes:        getEnvInt("JOB_TTL_MINUTES", 30),
	}
	return cfg, nil
}

// PublicPath es la carpeta pública servida por nginx.
// La variable de entorno llega hasta "template"; la subruta "public" es fija
// (a futuro: private, user, etc.).
func (c *Config) PublicPath() string {
	return filepath.Join(c.PublicDir, "public")
}

// ImagesDir es la carpeta privada de imágenes de entidades. nginx no la sirve;
// la API valida el estado de la entidad antes de entregar cada archivo.
func (c *Config) ImagesDir() string {
	return c.PrivateImagesDir()
}

// PublicImagesDir es la carpeta que nginx sirve para imágenes públicas.
func (c *Config) PublicImagesDir() string {
	return filepath.Join(c.PublicPath(), "imagenes")
}

// PrivateImagesDir es la carpeta de imágenes restringidas; solo la sirve Go.
func (c *Config) PrivateImagesDir() string {
	return filepath.Join(c.FilesDir(), "imagenes")
}

// PublicFilesDir es la carpeta de archivos públicos que sirve nginx.
func (c *Config) PublicFilesDir() string {
	return filepath.Join(c.PublicPath(), "archivos")
}

// PrivateFilesDir es la carpeta de archivos restringidos; solo la sirve Go.
func (c *Config) PrivateFilesDir() string {
	return c.FilesDir()
}

// DocumentosDir es la carpeta privada de documentos generados.
func (c *Config) DocumentosDir() string {
	return filepath.Join(c.FilesDir(), "documentos")
}

// FilesDir es la carpeta privada de archivos. Vive dentro de PUBLIC_DIR para
// mantener todo el proyecto junto, pero nginx la bloquea y solo la sirve Go.
func (c *Config) FilesDir() string {
	return filepath.Join(c.PublicDir, "archivos")
}

func (c *Config) DBDSN() string {
	switch c.DBType {
	case "postgres":
		return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + itoa(c.DBPort) + "/" + c.DBName + "?sslmode=" + c.DBSSLMode
	case "sqlite":
		return c.DBPath
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
