package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"joinly-manager/internal/api"
	"joinly-manager/internal/config"
	"joinly-manager/internal/database"
	"joinly-manager/internal/services"
)

func main() {
	// in main.go or a new file
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		resp, err := http.Get("http://localhost:8001/health")
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Set global configuration
	config.SetGlobalConfig(cfg)

	// Setup logging
	if err := config.SetupLogging(&cfg.Logging); err != nil {
		logrus.Fatalf("Failed to setup logging: %v", err)
		os.Exit(1)
	}

	logrus.Info("Starting DealSense Backend v2")

	// Initialize database
	dbConfig := &database.Config{
		Host:                        cfg.Database.Host,
		Port:                        cfg.Database.Port,
		User:                        cfg.Database.User,
		Password:                    cfg.Database.Password,
		DBName:                      cfg.Database.DBName,
		SSLMode:                     cfg.Database.SSLMode,
		DeleteExistingDataOnStartup: cfg.Database.DeleteExistingDataOnStartup,
	}

	db, err := database.NewDatabase(dbConfig)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
		os.Exit(1)
	}

	// Run migrations
	if err := db.AutoMigrate(); err != nil {
		logrus.Fatalf("Failed to run database migrations: %v", err)
		os.Exit(1)
	}

	// Create agent manager
	agentManager := services.NewAgentManager(db)

	// Start agent manager
	if err := agentManager.Start(); err != nil {
		logrus.Fatalf("Failed to start agent manager: %v", err)
		os.Exit(1)
	}

	// Setup router
	router := api.SetupRouter(cfg, agentManager, db)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop agent manager first
	if err := agentManager.Stop(); err != nil {
		logrus.Errorf("Failed to stop agent manager: %v", err)
		os.Exit(1)
	}

	// Close database connection
	if err := db.Close(); err != nil {
		logrus.Errorf("Failed to close database connection: %v", err)
		os.Exit(1)
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	logrus.Info("Server exited")
}
