package services

import (
	"fmt"
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

	// Bootstrap del primer administrador. Si BootstrapAdminPassword está vacío
	// no se crea ningún usuario automáticamente.
	BootstrapAdminUsername string
	BootstrapAdminPassword string
	BootstrapAdminEmail    string

	// Protección contra fuerza bruta en el login.
	LoginMaxAttempts   int
	LoginWindowMinutes int
	LoginBlockMinutes  int
}

func LoadConfig() (*Config, error) {
	// .env es opcional: en producción se esperan variables de entorno reales.
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:           getEnv("SERVER_PORT", ":8080"),
		DBType:               getEnv("DB_TYPE", "sqlite"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnvInt("DB_PORT", 5432),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "template"),
		DBPath:               getEnv("DB_PATH", filepath.Join("data", "template.db")),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		JWTSecret:            getEnv("JWT_SECRET", ""),
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

		BootstrapAdminUsername: getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		BootstrapAdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", "admin@example.com"),

		LoginMaxAttempts:   getEnvInt("LOGIN_MAX_ATTEMPTS", 5),
		LoginWindowMinutes: getEnvInt("LOGIN_WINDOW_MINUTES", 15),
		LoginBlockMinutes:  getEnvInt("LOGIN_BLOCK_MINUTES", 15),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate falla rápido ante configuraciones inseguras. Es mejor no arrancar
// que arrancar con un JWT predecible.
func (c *Config) Validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET es obligatorio y debe tener al menos 32 caracteres; genera uno con: go run ./cmd/generate_secrets.go")
	}
	if c.CORSOrigin == "" {
		return fmt.Errorf("CORS_ORIGIN es obligatorio")
	}
	if c.DBType != "sqlite" && c.DBType != "postgres" {
		return fmt.Errorf("DB_TYPE no soportado: %q (usa sqlite o postgres)", c.DBType)
	}
	if c.BootstrapAdminPassword != "" {
		if err := ValidatePasswordForUser(c.BootstrapAdminUsername, c.BootstrapAdminPassword); err != nil {
			return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD inválida: %w", err)
		}
	}
	if c.LoginMaxAttempts < 1 {
		c.LoginMaxAttempts = 5
	}
	if c.LoginWindowMinutes < 1 {
		c.LoginWindowMinutes = 15
	}
	if c.LoginBlockMinutes < 1 {
		c.LoginBlockMinutes = 15
	}
	return nil
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
