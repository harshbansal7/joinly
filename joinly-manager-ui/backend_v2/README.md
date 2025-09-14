# Joinly Manager Backend v2

A high-performance Go-based backend for managing multiple Joinly AI agents using goroutines instead of subprocesses. This is a complete rewrite of the Python-based backend with improved efficiency, scalability, and maintainability.

## 🚀 Features

- **Goroutine-based Agent Management**: Each agent runs in its own goroutine for optimal resource utilization
- **Real-time WebSocket Updates**: Live status updates and event streaming for all agents
- **RESTful API**: Complete API compatibility with the original Python backend
- **Production Ready**: Comprehensive logging, error handling, and configuration management
- **Docker Support**: Containerized deployment with health checks
- **High Performance**: Go's concurrency model provides better performance than Python subprocesses

## 🏗️ Architecture

```
backend_v2/
├── cmd/server/           # Main application entry point
├── internal/
│   ├── api/             # HTTP handlers and routing
│   ├── client/          # Joinly client implementation
│   ├── config/          # Configuration management
│   ├── manager/         # Agent manager with goroutines
│   ├── models/          # Data models
│   └── websocket/       # WebSocket hub for real-time updates
├── Dockerfile           # Docker build configuration
├── docker-compose.yml   # Docker Compose setup
└── go.mod              # Go module dependencies
```

## 📋 Prerequisites

- Go 1.24 or later
- Docker (optional, for containerized deployment)
- Access to Joinly server (typically running on port 8000)

## 🛠️ Installation

### Local Development

1. **Clone and navigate to the backend directory:**
   ```bash
   cd joinly-manager-ui/backend_v2
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   Create a `.env` file in the parent `joinly/` directory or set environment variables:
   ```bash
   export OPENAI_API_KEY=your_openai_key
   export ANTHROPIC_API_KEY=your_anthropic_key
   export ELEVENLABS_API_KEY=your_elevenlabs_key
   ```

4. **Run the server:**
   ```bash
   go run cmd/server/main.go
   ```

### Docker Deployment

1. **Build and run with Docker Compose:**
   ```bash
   docker-compose up --build
   ```

2. **Or build manually:**
   ```bash
   docker build -t joinly-manager .
   docker run -p 8001:8001 joinly-manager
   ```

## ⚙️ Configuration

The application can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | `8001` | Server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json or text) |
| `JOINLY_URL` | `http://localhost:8000/mcp/` | Joinly server URL |
| `MAX_AGENTS` | `10` | Maximum number of concurrent agents |

## 📡 API Endpoints

### Health Check
- **GET** `/` - Health check endpoint

### Agents
- **GET** `/agents` - List all agents
- **POST** `/agents` - Create a new agent
- **GET** `/agents/{agent_id}` - Get agent details
- **DELETE** `/agents/{agent_id}` - Delete an agent
- **POST** `/agents/{agent_id}/start` - Start an agent
- **POST** `/agents/{agent_id}/stop` - Stop an agent
- **GET** `/agents/{agent_id}/logs` - Get agent logs

### Meetings
- **GET** `/meetings` - List all active meetings

### WebSocket
- **WS** `/ws/agents/{agent_id}` - Real-time agent updates

### Utilities
- **GET** `/usage` - Get usage statistics
- **GET** `/ws/stats` - Get WebSocket connection statistics

## 🔌 WebSocket Events

The WebSocket endpoint provides real-time updates for agent events:

```javascript
// Connect to agent updates
const ws = new WebSocket('ws://localhost:8001/ws/agents/agent_123');

// Listen for messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Agent update:', data);
};
```

### Event Types

- `status` - Agent status changes (created, starting, running, stopping, stopped, error)
- `utterance` - Speech utterance events with transcript segments
- `segment` - Individual transcript segment updates

## 📝 Agent Configuration

When creating an agent, use the following configuration structure:

```json
{
  "name": "Meeting Assistant",
  "meeting_url": "https://meet.google.com/abc-defg-hij",
  "llm_provider": "openai",
  "llm_model": "gpt-4o",
  "tts_provider": "kokoro",
  "stt_provider": "whisper",
  "language": "en",
  "prompt_style": "mpc",
  "custom_prompt": null,
  "name_trigger": false,
  "auto_join": true,
  "env_vars": {
    "OPENAI_API_KEY": "your_key_here"
  }
}
```

## 🧪 Testing

### Manual Testing

1. **Start the server:**
   ```bash
   go run cmd/server/main.go
   ```

2. **Create an agent:**
   ```bash
   curl -X POST http://localhost:8001/agents \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Test Agent",
       "meeting_url": "https://example.com/meeting",
       "llm_provider": "openai",
       "llm_model": "gpt-4o"
     }'
   ```

3. **Start the agent:**
   ```bash
   curl -X POST http://localhost:8001/agents/agent_xxx/start
   ```

### Docker Testing

```bash
# Build and run tests
docker-compose up --build

# Check health
curl http://localhost:8001/

# View logs
docker-compose logs -f joinly-manager
```

## 🔍 Monitoring

### Logs

The application provides structured logging with the following levels:
- `DEBUG` - Detailed debugging information
- `INFO` - General information about operations
- `WARN` - Warning messages for potential issues
- `ERROR` - Error messages for failures

### Health Checks

The application includes built-in health checks:
- HTTP endpoint: `GET /`
- Docker health check configured in Dockerfile
- Readiness probes for Kubernetes deployment

### Metrics

Access usage statistics via:
```bash
curl http://localhost:8001/usage
```

## 🚀 Performance Benefits

Compared to the Python subprocess-based approach:

1. **Memory Efficiency**: Goroutines use significantly less memory than Python processes
2. **Startup Speed**: Near-instantaneous agent startup vs. subprocess overhead
3. **Resource Utilization**: Better CPU utilization through Go's lightweight concurrency
4. **Scalability**: Can handle hundreds of concurrent agents on a single machine
5. **Communication**: Direct in-memory communication vs. inter-process communication

## 🛡️ Security

- CORS protection for web frontend
- Input validation for all API endpoints
- Environment variable-based configuration
- Non-root container execution
- Health checks for container orchestration

## 🔧 Development

### Code Structure

- **`cmd/server/`** - Application entry point
- **`internal/api/`** - HTTP handlers and routing
- **`internal/client/`** - Joinly client implementation
- **`internal/config/`** - Configuration management
- **`internal/manager/`** - Agent lifecycle management
- **`internal/models/`** - Data structures
- **`internal/websocket/`** - Real-time communication

### Adding New Features

1. **New API Endpoints**: Add handlers in `internal/api/handlers.go` and routes in `internal/api/router.go`
2. **New Models**: Add to `internal/models/models.go`
3. **New Client Features**: Extend `internal/client/client.go`
4. **Configuration**: Update `internal/config/config.go`

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/manager
```

## 📚 API Documentation

### Create Agent

```http
POST /agents
Content-Type: application/json

{
  "name": "Meeting Assistant",
  "meeting_url": "https://meet.google.com/abc-defg-hij",
  "llm_provider": "openai",
  "llm_model": "gpt-4o",
  "tts_provider": "kokoro",
  "stt_provider": "whisper",
  "language": "en",
  "prompt_style": "mpc",
  "custom_prompt": null,
  "name_trigger": false,
  "auto_join": true,
  "env_vars": {}
}
```

### Response
```json
{
  "id": "agent_abc123",
  "config": { ... },
  "status": "created",
  "created_at": "2024-01-01T12:00:00Z",
  "logs": []
}
```

### WebSocket Connection

```javascript
const ws = new WebSocket('ws://localhost:8001/ws/agents/agent_abc123');

ws.onopen = () => {
  console.log('Connected to agent updates');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Update:', message);
};

ws.onclose = () => {
  console.log('Disconnected from agent updates');
};
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the same license as the original Joinly project.

## 🆘 Troubleshooting

### Common Issues

1. **"Failed to connect to joinly server"**
   - Ensure the Joinly server is running on the configured URL
   - Check network connectivity and firewall settings

2. **"Agent failed to start"**
   - Verify API keys are properly configured
   - Check agent configuration for validity
   - Review logs for specific error messages

3. **"WebSocket connection failed"**
   - Ensure CORS settings allow your frontend origin
   - Check that the agent exists before connecting

### Debug Mode

Enable debug logging for detailed information:
```bash
export LOG_LEVEL=debug
go run cmd/server/main.go
```

### Logs Location

- **Docker**: `docker-compose logs joinly-manager`
- **Local**: Console output with structured JSON logs

## 🔄 Migration from Python Backend

The Go backend is designed to be a drop-in replacement:

1. **API Compatibility**: All endpoints maintain the same interface
2. **Data Models**: Compatible JSON structures for requests/responses
3. **WebSocket Events**: Same event format and types
4. **Configuration**: Environment variable compatibility

### Migration Steps

1. Stop the Python backend
2. Start the Go backend on the same port (8001)
3. Update any direct dependencies if needed
4. The frontend should work without changes

## 📞 Support

For issues and questions:
1. Check the logs for error messages
2. Review the troubleshooting section
3. Create an issue with detailed information about your setup and the problem
