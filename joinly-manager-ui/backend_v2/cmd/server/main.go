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
	"joinly-manager/internal/manager"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	if err := config.SetupLogging(&cfg.Logging); err != nil {
		logrus.Fatalf("Failed to setup logging: %v", err)
	}

	logrus.Info("Starting Joinly Manager Backend v2")

	// Create agent manager
	agentManager := manager.NewAgentManager(cfg)

	// Start agent manager
	if err := agentManager.Start(); err != nil {
		logrus.Fatalf("Failed to start agent manager: %v", err)
	}

	// Setup router
	router := api.SetupRouter(cfg, agentManager)

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
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}
