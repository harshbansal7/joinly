package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/manager"
	"joinly-manager/internal/models"
)

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	agentManager *manager.AgentManager
}

// NewHandler creates a new handler instance
func NewHandler(agentManager *manager.AgentManager) *Handler {
	return &Handler{
		agentManager: agentManager,
	}
}

// HealthCheck handles the root endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Joinly Manager API is running",
	})
}

// ListAgents handles GET /agents
func (h *Handler) ListAgents(c *gin.Context) {
	agents := h.agentManager.ListAgents()
	c.JSON(http.StatusOK, agents)
}

// CreateAgent handles POST /agents
func (h *Handler) CreateAgent(c *gin.Context) {
	var config models.AgentConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values if not provided
	val := 1.0
	config.UtteranceTailSeconds = &val

	agent, err := h.agentManager.CreateAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send response first
	c.JSON(http.StatusCreated, agent)

	// Auto-start if enabled (after response is sent to prevent deadlock)
	if config.AutoJoin {
		go func() {
			// Small delay to ensure response is sent
			time.Sleep(500 * time.Millisecond)
			if err := h.agentManager.StartAgent(agent.ID); err != nil {
				// Log error but don't affect the creation response
				logrus.Errorf("Failed to auto-start agent %s: %v", agent.ID, err)
			}
		}()
	}
}

// GetAgent handles GET /agents/{agent_id}
func (h *Handler) GetAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	agent, exists := h.agentManager.GetAgent(agentID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent handles DELETE /agents/{agent_id}
func (h *Handler) DeleteAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.DeleteAgent(agentID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted successfully"})
}

// StartAgent handles POST /agents/{agent_id}/start
func (h *Handler) StartAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.StartAgent(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent started successfully"})
}

// StopAgent handles POST /agents/{agent_id}/stop
func (h *Handler) StopAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.StopAgent(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent stopped successfully"})
}

// JoinMeeting handles POST /agents/{agent_id}/join-meeting
func (h *Handler) JoinMeeting(c *gin.Context) {
	agentID := c.Param("agent_id")

	if err := h.agentManager.JoinMeeting(agentID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found or not running" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "agent not connected" {
			statusCode = http.StatusBadRequest
		} else if err.Error() == "agent already joined meeting" {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Join meeting initiated"})
}

// GetAgentLogs handles GET /agents/{agent_id}/logs
func (h *Handler) GetAgentLogs(c *gin.Context) {
	agentID := c.Param("agent_id")

	lines := 100 // default
	if linesStr := c.Query("lines"); linesStr != "" {
		if parsedLines, err := strconv.Atoi(linesStr); err == nil && parsedLines > 0 {
			lines = parsedLines
		}
	}

	logs, err := h.agentManager.GetAgentLogs(agentID, lines)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// WebSocketAgent handles WebSocket connections for agents
func (h *Handler) WebSocketAgent(c *gin.Context) {
	agentID := c.Param("agent_id")

	// Check if agent exists
	if _, exists := h.agentManager.GetAgent(agentID); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	wsHub := h.agentManager.GetWebSocketHub()
	wsHub.ServeWs(c, agentID)
}

// WebSocketSession handles WebSocket connections for entire user session
func (h *Handler) WebSocketSession(c *gin.Context) {
	wsHub := h.agentManager.GetWebSocketHub()
	wsHub.ServeSessionWs(c)
}

// ListMeetings handles GET /meetings
func (h *Handler) ListMeetings(c *gin.Context) {
	meetings := h.agentManager.ListMeetings()
	c.JSON(http.StatusOK, meetings)
}

// GetUsageStats handles GET /usage (additional endpoint for usage statistics)
func (h *Handler) GetUsageStats(c *gin.Context) {
	stats := h.agentManager.GetUsageStats()
	c.JSON(http.StatusOK, stats)
}

// GetWebSocketStats handles GET /ws/stats (additional endpoint for WebSocket stats)
func (h *Handler) GetWebSocketStats(c *gin.Context) {
	wsHub := h.agentManager.GetWebSocketHub()
	c.JSON(http.StatusOK, gin.H{
		"total_clients":    wsHub.GetClientCount(),
		"agents_monitored": len(h.agentManager.ListAgents()),
	})
}
