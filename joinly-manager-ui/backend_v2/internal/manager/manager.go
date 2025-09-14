package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client"
	"joinly-manager/internal/config"
	"joinly-manager/internal/models"
	"joinly-manager/internal/websocket"
)

// AgentManager manages multiple Joinly clients
type AgentManager struct {
	config              *config.Config
	clients             map[string]*client.JoinlyClient
	agents              map[string]*models.Agent
	meetings            map[string]*models.MeetingInfo
	analysts            map[string]*client.AnalystAgent // Analyst agents for analysis mode
	wsHub               *websocket.Hub
	running             bool
	startTime           time.Time
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	agentContexts       map[string]context.CancelFunc
	logBuffers          map[string][]models.LogEntry
	logBufferSize       int
	utteranceTasks      map[string]context.CancelFunc // Track active utterance processing tasks
	conversationHistory map[string][]models.ConversationEntry
}

// NewAgentManager creates a new agent manager
func NewAgentManager(cfg *config.Config) *AgentManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &AgentManager{
		config:              cfg,
		clients:             make(map[string]*client.JoinlyClient),
		agents:              make(map[string]*models.Agent),
		meetings:            make(map[string]*models.MeetingInfo),
		analysts:            make(map[string]*client.AnalystAgent),
		wsHub:               websocket.NewHub(),
		running:             false,
		startTime:           time.Now(),
		ctx:                 ctx,
		cancel:              cancel,
		agentContexts:       make(map[string]context.CancelFunc),
		logBuffers:          make(map[string][]models.LogEntry),
		logBufferSize:       1000,
		utteranceTasks:      make(map[string]context.CancelFunc),
		conversationHistory: make(map[string][]models.ConversationEntry),
	}
}

// Start starts the agent manager
func (m *AgentManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("agent manager already running")
	}

	logrus.Info("Starting agent manager")
	m.running = true
	m.startTime = time.Now()

	// Start WebSocket hub
	m.wsHub.Start()

	logrus.Info("Agent manager started successfully")
	return nil
}

// Stop stops the agent manager and all agents
func (m *AgentManager) Stop() error {
	m.mu.Lock()

	if !m.running {
		m.mu.Unlock()
		return nil
	}

	logrus.Info("Stopping agent manager")
	m.running = false
	m.cancel()

	// Cancel all active utterance processing tasks
	for agentID, cancelFunc := range m.utteranceTasks {
		logrus.Debugf("Cancelling utterance task for agent %s", agentID)
		cancelFunc()
	}
	m.utteranceTasks = make(map[string]context.CancelFunc)

	// Stop WebSocket hub
	m.wsHub.Stop()

	m.mu.Unlock() // Release lock before waiting

	// Wait for all agents to stop
	m.wg.Wait()

	logrus.Info("Agent manager stopped successfully")
	return nil
}

// GetAnalystAgent gets an analyst agent by ID
func (m *AgentManager) GetAnalystAgent(agentID string) *client.AnalystAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.analysts[agentID]
}
