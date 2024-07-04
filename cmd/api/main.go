package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/jsonlog"
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
}

type db struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

// application struct defines the application object
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	wg     sync.WaitGroup
}

func main() {
	var cfg config // cfg will hold the configuration
	// Load variables from .env file
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Parse command line flags into the cfg config struct
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	// Set database configuration fields
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("IVYWHIZ_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// Parse the command line flags
	flag.Parse()

	// Initialize a new jsonlog.Logger which writes any *at or above* the
	// INFO severity level messages to the standard out stream
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Call the open db function
	db, err := openDB(cfg)
	if err != nil {
		// Use the PrintFatal() method to write a log entry containing the error at the
		// FATAL level and exit. We have no additional properties to include in the log
		// entry, so we pass nil as the second parameter.
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("DB connection established", nil)

	// Initialize an instance of the application
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

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

	// Create a context with a 5 sec delay
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the database
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
