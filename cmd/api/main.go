package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/jsonlog"
	"github.com/araromirichard/internal/mailer"
	"github.com/araromirichard/internal/uploader"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

// version declares a constant holding the application version number
const version = "0.1.0"

// config struct defines the configuration for the application
type config struct {
	port    int    // port declares the port number the API server should listen on
	env     string // env declares the environment the application is running in
	db      db
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	cloudinary struct {
		cloudName string
		apiKey    string
		apiSecret string
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type db struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

// application struct defines the application object
type application struct {
	config   config
	logger   *jsonlog.Logger
	models   data.Models
	uploader *uploader.ImageUploaderService
	mailer   mailer.Mailer
	wg       sync.WaitGroup
}

func main() {
	var cfg config

	// Load environment variables if not in production
	if os.Getenv("FLY_APP_NAME") == "" {
		if err := godotenv.Load(); err != nil {
			panic("Error loading .env file")
		}
	}

	// Parse command-line flags
	flag.IntVar(&cfg.port, "port", 8080, "API server port")
	flag.StringVar(&cfg.env, "env", os.Getenv("ENV_PLATFORM"), "Environment (development|staging|production)")

	// Admin initialization flags
	var adminFirstName, adminLastName, adminEmail, adminPassword string
	flag.StringVar(&adminFirstName, "admin-first-name", "Ivywhiz", "First name for the default admin user")
	flag.StringVar(&adminLastName, "admin-last-name", "Admin", "Last name for the default admin user")
	flag.StringVar(&adminEmail, "admin-email", "ivywhizadmin@mailinator.com", "Email for the default admin user")
	flag.StringVar(&adminPassword, "admin-password", "IvywhizPass123", "Password for the default admin user")

	// Database configuration
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:OCuG1gml5uvgF2U@ivywhizdb.flycast:5432/postgres?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// Rate limiter configuration
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate Limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate Limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable Rate Limiter")

	// Cloudinary configuration
	flag.StringVar(&cfg.cloudinary.cloudName, "cloudinary-cloud-name", os.Getenv("CLOUDINARY_CLOUD_NAME"), "Cloudinary cloud name")
	flag.StringVar(&cfg.cloudinary.apiKey, "cloudinary-api-key", os.Getenv("CLOUDINARY_API_KEY"), "Cloudinary API key")
	flag.StringVar(&cfg.cloudinary.apiSecret, "cloudinary-api-secret", os.Getenv("CLOUDINARY_API_SECRET"), "Cloudinary API secret")

	// SMTP configuration
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 587, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USER"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASS"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "IVYWHIZ <no-reply@Ivywhiz.krobotechnologies.com>", "SMTP sender")

	// CORS configuration
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	// Parse flags
	flag.Parse()

	// Initialize logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Open DB connection
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("DB connection established", nil)

	// Initialize application
	app := &application{
		config:   cfg,
		logger:   logger,
		models:   data.NewModels(db),
		uploader: uploader.New(cfg.cloudinary.cloudName, cfg.cloudinary.apiKey, cfg.cloudinary.apiSecret),
		mailer:   mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	// Initialize the admin user
	if err := app.initAdminUser(adminFirstName, adminLastName, adminEmail, adminPassword); err != nil {
		logger.PrintFatal(err, nil)
	}

	// Start the server
	logger.PrintFatal(app.serve(), nil)
}

// db connection
func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	// Try pinging the database to ensure a connection was established
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
