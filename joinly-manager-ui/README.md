# Joinly Manager UI

A production-grade web dashboard for managing multiple joinly.ai agents in video meetings.

## Overview

This dashboard provides a user-friendly interface to:
- Spawn multiple joinly clients to different meetings
- Monitor active agents and their status
- Manage agent configurations
- View real-time transcripts and logs
- Control meeting interactions

## Architecture

```
joinly-manager-ui/
├── backend/          # FastAPI backend for client management
├── frontend/         # React/Next.js dashboard
└── docs/            # Documentation and deployment guides
```

## Features

- **Multi-Agent Management**: Start/stop multiple agents simultaneously
- **Real-time Monitoring**: Live status updates and transcripts
- **Configuration Management**: Save and reuse agent configurations
- **Meeting Control**: Join, leave, mute, and chat controls
- **Log Management**: View and export agent logs
- **Responsive Design**: Works on desktop and mobile devices

## Technology Stack

- **Frontend**: React with TypeScript, Tailwind CSS
- **Backend**: FastAPI with Python
- **Real-time**: WebSocket for live updates
- **State Management**: Zustand for client-side state
- **UI Components**: Shadcn/ui for consistent design

## Prerequisites

1. **joinly.ai server running on localhost:8080**
   ```bash
   # In your joinly directory
   docker run --env-file .env ghcr.io/joinly-ai/joinly:latest --server
   ```

2. **Python 3.12+** for the backend
3. **Node.js 18+** for the frontend

## Quick Start

### Backend Setup
```bash
cd backend
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
python main.py
```

### Frontend Setup
```bash
cd frontend
npm install
npm run dev
```

### Access the Dashboard
Open http://localhost:3000

## Environment Variables

### Backend (.env in backend directory)
```bash
# Backend configuration
JOINLY_MANAGER_HOST=0.0.0.0
JOINLY_MANAGER_PORT=8001

# Joinly server connection
JOINLY_SERVER_URL=http://localhost:8080
```

### Frontend (.env.local in frontend directory)
```bash
# API endpoint
NEXT_PUBLIC_API_URL=http://localhost:8001
```

### Agent API Keys (passed when creating agents)
```bash
# LLM Keys
OPENAI_API_KEY=your_openai_key
ANTHROPIC_API_KEY=your_anthropic_key

# TTS Keys
ELEVENLABS_API_KEY=your_elevenlabs_key
DEEPGRAM_API_KEY=your_deepgram_key
```

## Usage

1. **Start the joinly server** on localhost:8080
2. **Launch the backend**: `cd backend && python main.py`
3. **Launch the frontend**: `cd frontend && npm run dev`
4. **Create agents** through the dashboard
5. **Monitor and manage** your agents in real-time

## API Documentation

### REST Endpoints
- `GET /agents` - List all agents
- `POST /agents` - Create new agent
- `GET /agents/{id}` - Get agent details
- `POST /agents/{id}/start` - Start agent
- `POST /agents/{id}/stop` - Stop agent
- `GET /agents/{id}/logs` - Get agent logs

### WebSocket
- `ws://localhost:8001/ws/agents/{id}` - Real-time agent updates

## Development

### Backend
```bash
cd backend
pip install -r requirements.txt
python main.py  # Auto-reloads on changes
```

### Frontend
```bash
cd frontend
npm install
npm run dev     # Auto-reloads on changes
npm run build   # Production build
```

## Deployment

### Docker Deployment
```dockerfile
# Backend
FROM python:3.12-slim
WORKDIR /app
COPY backend/requirements.txt .
RUN pip install -r requirements.txt
COPY backend/ .
EXPOSE 8001
CMD ["python", "main.py"]

# Frontend
FROM node:18-alpine
WORKDIR /app
COPY frontend/package*.json .
RUN npm ci
COPY frontend/ .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

### Production Considerations
- Use reverse proxy (nginx) for production
- Set up proper CORS configuration
- Configure SSL/TLS
- Use environment-specific configurations
- Set up monitoring and logging

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
