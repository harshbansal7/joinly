# DealSense Advanced Features Documentation

This document describes the new AI-powered features for startup investment analysis in DealSense.

## Overview

DealSense now includes three major feature sets built on Google Cloud technologies:

1. **Document Management & Processing** - Upload and process pitch decks with OCR
2. **Intelligent Chatbot** - RAG-based Q&A over meeting transcripts and documents
3. **Multi-Faceted Startup Analysis** - Comprehensive investment analysis combining meeting data and pitch decks

## 1. Document Management

### Features

- **Drag-and-drop upload** for pitch decks (PDF, DOCX, PPTX)
- **Automatic OCR processing** using Google Document AI
- **Visual element extraction** - detects images, charts, diagrams in pitch decks
- **Page-based chunking** optimized for visual-heavy presentations
- **Real-time processing status** tracking
- **Semantic search** across document contents using Vertex AI embeddings

### Usage

1. Navigate to agent details page
2. Click "Documents" tab
3. Drag and drop pitch deck or click to browse
4. Wait for processing (typically 30-60 seconds for a 10-page deck)
5. Document status will change from "processing" to "processed"

### Technical Details

- **Storage**: Google Cloud Storage
- **Processing**: Document AI with OCR processor
- **Max file size**: 50MB
- **Supported formats**: PDF, DOCX, PPTX
- **Embedding model**: Vertex AI text-embedding-004 (768 dimensions)

### API Endpoints

```typescript
// Upload document
POST /agents/:agent_id/documents
Content-Type: multipart/form-data
Body: { file: File }

// List documents
GET /agents/:agent_id/documents

// Delete document
DELETE /documents/:document_id

// Search documents
POST /agents/:agent_id/documents/search
Body: { query: string, top_k: number }
```

## 2. Intelligent Chatbot

### Features

- **Multi-source RAG** - queries both meeting transcripts and uploaded documents
- **Context-aware responses** - shows which sources were used
- **Session persistence** - maintains conversation history
- **Real-time query processing** - typically <2 seconds
- **Suggested questions** - pre-configured prompts for common queries

### Usage

1. Navigate to agent details page
2. Click "Chat" tab
3. Type your question or click a suggested question
4. View AI-generated response with source attribution
5. Continue conversation naturally

### Example Queries

```
"What were the main action items from the meeting?"
"Summarize the business model from the pitch deck"
"What concerns were raised about the financial projections?"
"Who are the key stakeholders and what are their roles?"
"Compare what was said in the meeting vs what's in the deck"
```

### Technical Details

- **Retrieval**: Semantic search with cosine similarity
- **Context window**: Top 5 most relevant chunks (configurable)
- **LLM**: Uses agent's configured LLM (typically Gemini Pro)
- **Response time**: Averages 1-3 seconds
- **Sources**: Combines meeting transcript segments and document chunks

### API Endpoints

```typescript
// Send chat query
POST /agents/:agent_id/chat
Body: {
  query: string,
  session_id?: string,
  document_id?: string,  // Optional: limit to specific document
  top_k?: number
}

// Get chat history
GET /agents/:agent_id/chat/:session_id
```

## 3. Multi-Faceted Startup Analysis

### Analysis Dimensions

DealSense provides comprehensive startup evaluation across five key dimensions:

#### 1. Pitch Analysis (Pitch Deck Quality)
- **What it evaluates**:
  - Clarity of problem statement and solution
  - Market size and opportunity presentation
  - Business model articulation
  - Visual design and storytelling quality
  - Completeness of key sections
  
- **Scoring criteria**:
  - 80-100: Exceptional pitch, clear value proposition
  - 65-79: Solid pitch with minor gaps
  - 50-64: Adequate but needs improvement
  - <50: Significant issues in pitch quality

#### 2. Founder Reliability
- **What it evaluates**:
  - Consistency between pitch deck claims and meeting discussion
  - Detection of contradictions or vague answers
  - Domain expertise and track record
  - Communication clarity and confidence
  - Team composition and gaps
  
- **Red flags detected**:
  - Exaggerated claims
  - Defensive behavior
  - Inconsistent financial projections
  - Lack of domain knowledge

#### 3. Market Opportunity
- **What it evaluates**:
  - Market size realism (TAM, SAM, SOM)
  - Growth trends and dynamics
  - Target customer definition
  - Pain point severity
  - Competition intensity
  - Market timing
  
- **Opportunities identified**:
  - Underserved market segments
  - First-mover advantages
  - Favorable regulatory environment
  - Technology tailwinds

#### 4. Financial Viability
- **What it evaluates**:
  - Revenue model sustainability
  - Pricing strategy
  - Unit economics (CAC, LTV, margins)
  - Burn rate and runway
  - Financial projection realism
  - Path to profitability
  
- **Analysis includes**:
  - Break-even timeline assessment
  - Capital efficiency metrics
  - Risk factors identification

#### 5. Competitive Landscape
- **What it evaluates**:
  - Direct and indirect competitors
  - Competitive advantages and differentiation
  - Barriers to entry
  - Technology moats
  - Network effects potential
  - Threat of disruption
  
- **Strategic assessment**:
  - Sustainable competitive position
  - Switching costs
  - Winner-take-all dynamics

### Overall Investment Recommendation

Based on the combined analysis score:

| Score Range | Recommendation | Interpretation |
|-------------|---------------|----------------|
| 80-100 | **STRONG INVEST** | Exceptional promise across all dimensions |
| 65-79 | **CONSIDER** | Solid fundamentals, requires due diligence |
| 50-64 | **CAUTIOUS** | Significant concerns, high risk |
| <50 | **PASS** | Too many red flags |

### Usage

1. Upload at least one processed document (pitch deck)
2. Optionally conduct a meeting with the agent active
3. Navigate to "Startup" tab
4. Click "Run Analysis"
5. Wait 30-60 seconds for AI analysis to complete
6. Review overall score and detailed section breakdowns

### Analysis Output

For each dimension, you'll receive:

- **Score** (0-100)
- **Executive summary**
- **Key findings** (positive observations)
- **Red flags** (concerns and risks)
- **Opportunities** (potential upsides)
- **Recommendations** (actionable next steps)

### Example Analysis Output

```json
{
  "overall_score": 73.5,
  "recommendation": "CONSIDER",
  "pitch_analysis": {
    "score": 82,
    "summary": "Strong pitch deck with clear value proposition...",
    "key_findings": [
      "Excellent problem-solution fit articulation",
      "Compelling market size data with credible sources"
    ],
    "red_flags": [
      "Go-to-market strategy lacks specificity"
    ],
    "opportunities": [
      "Underserved SMB segment represents major opportunity"
    ],
    "recommendations": [
      "Develop detailed customer acquisition roadmap"
    ]
  },
  // ... other dimensions
}
```

### Technical Details

- **Processing time**: 30-90 seconds depending on document size
- **LLM**: Gemini Pro with specialized prompts for each dimension
- **Context**: Combines full document text + meeting transcript
- **Storage**: Analysis results stored in PostgreSQL for historical tracking
- **Grounding**: Uses Google Search grounding for fact verification

### API Endpoints

```typescript
// Run startup analysis
POST /agents/:agent_id/analyze
Body: {
  document_ids: string[],
  meeting_id?: string,
  analysis_types?: string[]  // Optional: specify which analyses to run
}

// Get latest analysis
GET /agents/:agent_id/analysis/startup
```

## Architecture

### Backend Stack

```
┌─────────────────────────────────────────────────────────────┐
│                     DealSense Backend                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────────┐  │
│  │  Document   │  │  Chatbot    │  │  Startup Analyzer  │  │
│  │  Service    │  │  Service    │  │                    │  │
│  └──────┬──────┘  └──────┬──────┘  └─────────┬──────────┘  │
│         │                 │                    │             │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌─────────▼──────────┐  │
│  │  Storage    │  │  Embedding  │  │  LLM Provider      │  │
│  │  Client     │  │  Service    │  │  (Gemini)          │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────────────┘  │
│         │                 │                                  │
└─────────┼─────────────────┼──────────────────────────────────┘
          │                 │
          ▼                 ▼
┌─────────────────────────────────────────────────────────────┐
│                  Google Cloud Platform                       │
├─────────────────────────────────────────────────────────────┤
│  Cloud Storage  │  Document AI  │  Vertex AI Embeddings    │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Document Upload**:
   ```
   User → Frontend → Backend → GCS
                            ↓
                      Document AI Processing
                            ↓
                      Text Extraction + OCR
                            ↓
                      Chunk Generation
                            ↓
                      Vertex AI Embeddings
                            ↓
                      PostgreSQL Storage
   ```

2. **Chat Query**:
   ```
   User Query → Backend → Embedding Generation
                       ↓
                 Similarity Search (Documents + Transcripts)
                       ↓
                 Context Retrieval (Top K)
                       ↓
                 LLM Prompt Construction
                       ↓
                 Gemini API Call
                       ↓
                 Response to User
   ```

3. **Startup Analysis**:
   ```
   Trigger Analysis → Gather Context (Docs + Meetings)
                            ↓
                   Run Parallel Analyses
                   (5 dimensions × specialized prompts)
                            ↓
                   Score Aggregation
                            ↓
                   Executive Summary Generation
                            ↓
                   Store Results + Display
   ```

## Database Schema

### New Tables

```sql
-- Documents
CREATE TABLE documents (
  id UUID PRIMARY KEY,
  agent_id UUID NOT NULL,
  name VARCHAR NOT NULL,
  file_type VARCHAR NOT NULL,
  storage_path VARCHAR NOT NULL,
  status VARCHAR NOT NULL,
  extracted_text TEXT,
  page_count INT,
  metadata JSONB,
  created_at TIMESTAMP,
  processed_at TIMESTAMP
);

-- Document Embeddings
CREATE TABLE document_embeddings (
  id UUID PRIMARY KEY,
  document_id UUID REFERENCES documents(id),
  chunk_index INT NOT NULL,
  chunk_text TEXT NOT NULL,
  embedding JSONB NOT NULL,  -- Vector stored as JSON
  chunk_metadata JSONB,
  created_at TIMESTAMP
);

-- Chat Messages
CREATE TABLE chat_messages (
  id UUID PRIMARY KEY,
  agent_id UUID NOT NULL,
  session_id VARCHAR NOT NULL,
  role VARCHAR NOT NULL,  -- 'user' | 'assistant'
  content TEXT NOT NULL,
  context_chunks JSONB,
  created_at TIMESTAMP
);

-- Startup Analysis
CREATE TABLE startup_analyses (
  id UUID PRIMARY KEY,
  agent_id UUID NOT NULL,
  analysis_type VARCHAR NOT NULL,
  score FLOAT,
  summary TEXT,
  key_findings JSONB,
  red_flags JSONB,
  opportunities JSONB,
  recommendations JSONB,
  generated_at TIMESTAMP
);
```

## Configuration

### Environment Variables

```bash
# Google Cloud credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# Or set inline in config.yaml
```

### Config.yaml

```yaml
google:
  api_key: "YOUR_GEMINI_API_KEY"
  project_id: "your-gcp-project-id"
  
  storage:
    bucket_name: "dealsense-documents"
    use_default_credentials: true
  
  document_ai:
    location: "us"
    processor_id: "YOUR_PROCESSOR_ID"
    use_default_credentials: true
  
  vertex_ai:
    location: "us-central1"
    embedding_model: "text-embedding-004"
    use_default_credentials: true
```

## Best Practices

### 1. Document Preparation

- **Use high-quality PDFs** with clear text (not scanned images)
- **Optimize file sizes** - compress images while maintaining readability
- **Include relevant content only** - remove appendices not needed for analysis
- **Standard formats work best** - typical pitch deck layouts process better

### 2. Meeting Conduct

- **Start agent before meeting** to capture full transcript
- **Speak clearly** for accurate transcription
- **Reference the pitch deck** during discussion for better cross-analysis
- **Cover all key topics** - team, market, product, financials, competition

### 3. Analysis Optimization

- **Upload documents first** before running analysis
- **Wait for processing to complete** before analyzing
- **Include meeting data** for founder reliability analysis
- **Review all dimensions** - don't rely solely on overall score
- **Consider context** - analysis is decision support, not a decision

## Troubleshooting

### Documents not processing

- Check file format (PDF, DOCX, PPTX only)
- Verify file size < 50MB
- Ensure Document AI processor is active
- Check backend logs for processing errors

### Chat not finding relevant context

- Ensure documents are fully processed (status = 'processed')
- Try more specific queries
- Check that embeddings were generated successfully
- Verify agent has meeting transcript data

### Analysis scores seem off

- Review individual dimension summaries for context
- Check if all relevant documents were included
- Verify meeting transcript captured key discussions
- Consider running analysis again with updated data

## Future Enhancements

Potential additions for future versions:

1. **Multi-document comparison** - compare multiple startups side-by-side
2. **Historical tracking** - track analysis changes over time
3. **Custom analysis dimensions** - user-defined evaluation criteria
4. **Export capabilities** - PDF reports, CSV data exports
5. **Collaboration features** - shared notes, team comments
6. **Integration with CRMs** - Salesforce, HubSpot connectors

## Support

For questions or issues:
- Refer to SETUP_GOOGLE_CLOUD.md for configuration help
- Check backend logs: `backend_v2/logs/dealsense.log`
- Review API documentation in code comments
- Open GitHub issues for bugs or feature requests

