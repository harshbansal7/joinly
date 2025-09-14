package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"joinly-manager/internal/config"
	"joinly-manager/internal/manager"
)

// SetupRouter sets up the Gin router with all routes
func SetupRouter(cfg *config.Config, agentManager *manager.AgentManager) *gin.Engine {
	// Set Gin mode
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.CORS.AllowedOrigins,
		AllowMethods:     cfg.Server.CORS.AllowedMethods,
		AllowHeaders:     cfg.Server.CORS.AllowedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Create handler
	handler := NewHandler(agentManager)

	// Health check
	router.GET("/", handler.HealthCheck)

	// Agent routes
	agents := router.Group("/agents")
	{
		agents.GET("", handler.ListAgents)
		agents.POST("", handler.CreateAgent)
		agents.GET("/:agent_id", handler.GetAgent)
		agents.DELETE("/:agent_id", handler.DeleteAgent)
		agents.POST("/:agent_id/start", handler.StartAgent)
		agents.POST("/:agent_id/stop", handler.StopAgent)
		agents.POST("/:agent_id/join-meeting", handler.JoinMeeting)
		agents.GET("/:agent_id/logs", handler.GetAgentLogs)
		agents.GET("/:agent_id/analysis", handler.GetAgentAnalysis)
		agents.GET("/:agent_id/analysis/formatted", handler.GetAgentAnalysisFormatted)
	}

	// WebSocket routes
	router.GET("/ws/agents/:agent_id", handler.WebSocketAgent)
	router.GET("/ws/session", handler.WebSocketSession)

	// Meeting routes
	router.GET("/meetings", handler.ListMeetings)

	// Additional utility routes
	router.GET("/usage", handler.GetUsageStats)
	router.GET("/ws/stats", handler.GetWebSocketStats)

	return router
}
