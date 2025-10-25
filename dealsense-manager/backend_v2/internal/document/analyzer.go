package document

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client"
	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/database"
)

// StartupAnalyzer performs multi-faceted analysis of startup pitch decks and meetings
type StartupAnalyzer struct {
	db          *database.Database
	docService  *Service
	llmProvider llm.LLMProvider
}

// NewStartupAnalyzer creates a new startup analyzer
func NewStartupAnalyzer(db *database.Database, docService *Service, llmProvider llm.LLMProvider) *StartupAnalyzer {
	return &StartupAnalyzer{
		db:          db,
		docService:  docService,
		llmProvider: llmProvider,
	}
}

// AnalysisRequest represents a startup analysis request
type AnalysisRequest struct {
	AgentID       uuid.UUID   `json:"agent_id"`
	DocumentIDs   []uuid.UUID `json:"document_ids"` // Pitch decks to analyze
	MeetingID     *uuid.UUID  `json:"meeting_id,omitempty"`
	AnalysisTypes []string    `json:"analysis_types"` // pitch_analysis, founder_reliability, market_opportunity, etc.
}

// AnalysisResult represents the comprehensive analysis result
type AnalysisResult struct {
	AgentID          uuid.UUID                  `json:"agent_id"`
	OverallScore     float64                    `json:"overall_score"`
	AnalysisSections map[string]AnalysisSection `json:"analysis_sections"`
	Summary          string                     `json:"summary"`
	GeneratedAt      time.Time                  `json:"generated_at"`
}

// AnalysisSection represents a specific analysis dimension
type AnalysisSection struct {
	Type            string   `json:"type"`
	Score           float64  `json:"score"`
	Summary         string   `json:"summary"`
	KeyFindings     []string `json:"key_findings"`
	RedFlags        []string `json:"red_flags"`
	Opportunities   []string `json:"opportunities"`
	Recommendations []string `json:"recommendations"`
}

// AnalyzeStartup performs comprehensive startup analysis
func (a *StartupAnalyzer) AnalyzeStartup(req AnalysisRequest) (*AnalysisResult, error) {
	logrus.Infof("Starting startup analysis for agent %s", req.AgentID.String())

	if len(req.AnalysisTypes) == 0 {
		// Default analysis types
		req.AnalysisTypes = []string{
			"pitch_analysis",
			"founder_reliability",
			"market_opportunity",
			"financial_viability",
			"competitive_landscape",
		}
	}

	result := &AnalysisResult{
		AgentID:          req.AgentID,
		AnalysisSections: make(map[string]AnalysisSection),
		GeneratedAt:      time.Now(),
	}

	// Gather all context: documents + meeting data
	context, err := a.gatherAnalysisContext(req)
	if err != nil {
		return nil, fmt.Errorf("failed to gather context: %w", err)
	}

	// Perform each type of analysis
	totalScore := 0.0
	for _, analysisType := range req.AnalysisTypes {
		section, err := a.performAnalysis(analysisType, context)
		if err != nil {
			logrus.Warnf("Failed to perform %s analysis: %v", analysisType, err)
			continue
		}
		result.AnalysisSections[analysisType] = *section
		totalScore += section.Score
	}

	// Calculate overall score
	if len(result.AnalysisSections) > 0 {
		result.OverallScore = totalScore / float64(len(result.AnalysisSections))
	}

	// Generate executive summary
	result.Summary = a.generateExecutiveSummary(result)

	// Store in database
	a.storeAnalysis(req, result)

	logrus.Infof("Startup analysis completed with overall score: %.2f", result.OverallScore)
	return result, nil
}

// AnalysisContext holds all context for analysis
type AnalysisContext struct {
	Documents         []DocumentContext
	MeetingTranscript string
	MeetingAnalysis   *client.AnalysisData
	Participants      []string
}

// DocumentContext represents a processed document with its content
type DocumentContext struct {
	ID       uuid.UUID
	Name     string
	Text     string
	Metadata map[string]interface{}
	Images   int
	Tables   int
}

// gatherAnalysisContext collects all relevant data
func (a *StartupAnalyzer) gatherAnalysisContext(req AnalysisRequest) (*AnalysisContext, error) {
	context := &AnalysisContext{
		Documents: []DocumentContext{},
	}

	// Gather document data
	for _, docID := range req.DocumentIDs {
		doc, err := a.docService.GetDocument(docID)
		if err != nil {
			logrus.Warnf("Failed to get document %s: %v", docID.String(), err)
			continue
		}

		if doc.Status != "processed" {
			logrus.Warnf("Document %s not processed yet", docID.String())
			continue
		}

		// Parse metadata
		var metadata map[string]interface{}
		if doc.Metadata != "" {
			json.Unmarshal([]byte(doc.Metadata), &metadata)
		}

		context.Documents = append(context.Documents, DocumentContext{
			ID:       doc.ID,
			Name:     doc.Name,
			Text:     doc.ExtractedText,
			Metadata: metadata,
			Images:   getIntFromMetadata(metadata, "total_images"),
			Tables:   getIntFromMetadata(metadata, "total_tables"),
		})
	}

	// Gather meeting data if available
	if req.MeetingID != nil {
		var transcripts []database.TranscriptSegment
		a.db.Where("agent_id = ?", req.AgentID).Order("timestamp ASC").Find(&transcripts)

		var transcriptBuilder strings.Builder
		for _, seg := range transcripts {
			speaker := "Unknown"
			if seg.Speaker != nil {
				speaker = *seg.Speaker
			}
			transcriptBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", speaker, seg.Text))
		}
		context.MeetingTranscript = transcriptBuilder.String()
	}

	return context, nil
}

// performAnalysis executes a specific type of analysis
func (a *StartupAnalyzer) performAnalysis(analysisType string, context *AnalysisContext) (*AnalysisSection, error) {
	prompt := a.buildAnalysisPrompt(analysisType, context)

	response, err := a.llmProvider.Call(prompt)
	if err != nil {
		return nil, err
	}

	// Parse LLM response into structured analysis
	section, err := a.parseAnalysisResponse(response, analysisType)
	if err != nil {
		return nil, err
	}

	return section, nil
}

// buildAnalysisPrompt creates specialized prompts for each analysis type
func (a *StartupAnalyzer) buildAnalysisPrompt(analysisType string, context *AnalysisContext) string {
	var prompt strings.Builder

	// Common context
	prompt.WriteString("You are an expert startup investor and analyst. Analyze the following startup based on the provided pitch deck and meeting transcript.\n\n")

	// Add document context
	if len(context.Documents) > 0 {
		prompt.WriteString("PITCH DECK CONTENT:\n")
		prompt.WriteString("---\n")
		for _, doc := range context.Documents {
			prompt.WriteString(fmt.Sprintf("\nDocument: %s\n", doc.Name))
			prompt.WriteString(fmt.Sprintf("Pages: %d, Images: %d, Tables: %d\n",
				len(strings.Split(doc.Text, "\n"))/50, doc.Images, doc.Tables))
			// Include excerpt or full text based on length
			if len(doc.Text) > 5000 {
				prompt.WriteString(doc.Text[:5000] + "...\n")
			} else {
				prompt.WriteString(doc.Text + "\n")
			}
		}
		prompt.WriteString("---\n\n")
	}

	// Add meeting context
	if context.MeetingTranscript != "" {
		prompt.WriteString("MEETING TRANSCRIPT:\n")
		prompt.WriteString("---\n")
		// Include recent transcript excerpt
		if len(context.MeetingTranscript) > 3000 {
			prompt.WriteString(context.MeetingTranscript[:3000] + "...\n")
		} else {
			prompt.WriteString(context.MeetingTranscript + "\n")
		}
		prompt.WriteString("---\n\n")
	}

	// Type-specific analysis instructions
	switch analysisType {
	case "pitch_analysis":
		prompt.WriteString(a.getPitchAnalysisInstructions())
	case "founder_reliability":
		prompt.WriteString(a.getFounderReliabilityInstructions())
	case "market_opportunity":
		prompt.WriteString(a.getMarketOpportunityInstructions())
	case "financial_viability":
		prompt.WriteString(a.getFinancialViabilityInstructions())
	case "competitive_landscape":
		prompt.WriteString(a.getCompetitiveLandscapeInstructions())
	default:
		prompt.WriteString("Provide a comprehensive analysis of this startup.\n")
	}

	// Response format
	prompt.WriteString("\n\nProvide your analysis in the following JSON format:\n")
	prompt.WriteString("```json\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"score\": 75.5,  // Score out of 100\n")
	prompt.WriteString("  \"summary\": \"Brief summary of findings\",\n")
	prompt.WriteString("  \"key_findings\": [\"finding1\", \"finding2\"],\n")
	prompt.WriteString("  \"red_flags\": [\"concern1\", \"concern2\"],\n")
	prompt.WriteString("  \"opportunities\": [\"opportunity1\", \"opportunity2\"],\n")
	prompt.WriteString("  \"recommendations\": [\"recommendation1\", \"recommendation2\"]\n")
	prompt.WriteString("}\n")
	prompt.WriteString("```\n")

	return prompt.String()
}

// Analysis instruction methods
func (a *StartupAnalyzer) getPitchAnalysisInstructions() string {
	return `PITCH DECK ANALYSIS:
Evaluate the quality and effectiveness of the pitch deck:
- Clarity of problem statement and solution
- Market size and opportunity presentation
- Business model clarity
- Product/service demonstration
- Visual design and storytelling
- Completeness of key sections (team, traction, financials)
- Identify any missing critical information
- Assess overall persuasiveness`
}

func (a *StartupAnalyzer) getFounderReliabilityInstructions() string {
	return `FOUNDER RELIABILITY ANALYSIS:
Assess the founders' credibility and capability:
- Cross-reference claims made in pitch deck vs meeting discussion
- Identify any inconsistencies or contradictions
- Evaluate domain expertise and track record
- Assess communication clarity and confidence
- Look for red flags: vague answers, exaggerations, defensive behavior
- Evaluate team composition and gaps
- Assess founder-market fit`
}

func (a *StartupAnalyzer) getMarketOpportunityInstructions() string {
	return `MARKET OPPORTUNITY ANALYSIS:
Evaluate the market potential:
- Market size (TAM, SAM, SOM) - are estimates realistic?
- Market growth trends and dynamics
- Target customer definition
- Pain point severity and urgency
- Competition intensity
- Market timing and readiness
- Regulatory considerations
- Scalability potential`
}

func (a *StartupAnalyzer) getFinancialViabilityInstructions() string {
	return `FINANCIAL VIABILITY ANALYSIS:
Assess the financial aspects:
- Revenue model clarity and sustainability
- Pricing strategy reasonableness
- Unit economics (CAC, LTV, margins)
- Burn rate and runway
- Financial projections realism
- Path to profitability
- Funding requirements justification
- Risk factors and mitigation`
}

func (a *StartupAnalyzer) getCompetitiveLandscapeInstructions() string {
	return `COMPETITIVE LANDSCAPE ANALYSIS:
Evaluate competitive positioning:
- Direct and indirect competitors identified
- Competitive advantages and differentiation
- Barriers to entry
- Switching costs
- Technology moats
- Network effects potential
- Threat of disruption
- Sustainable competitive position`
}

// parseAnalysisResponse parses LLM response into structured analysis
func (a *StartupAnalyzer) parseAnalysisResponse(response string, analysisType string) (*AnalysisSection, error) {
	// Extract JSON from response
	jsonStr := extractJSONFromMarkdown(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var parsed struct {
		Score           float64  `json:"score"`
		Summary         string   `json:"summary"`
		KeyFindings     []string `json:"key_findings"`
		RedFlags        []string `json:"red_flags"`
		Opportunities   []string `json:"opportunities"`
		Recommendations []string `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &AnalysisSection{
		Type:            analysisType,
		Score:           parsed.Score,
		Summary:         parsed.Summary,
		KeyFindings:     parsed.KeyFindings,
		RedFlags:        parsed.RedFlags,
		Opportunities:   parsed.Opportunities,
		Recommendations: parsed.Recommendations,
	}, nil
}

// generateExecutiveSummary creates an overall executive summary
func (a *StartupAnalyzer) generateExecutiveSummary(result *AnalysisResult) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Overall Investment Score: %.1f/100\n\n", result.OverallScore))

	if result.OverallScore >= 80 {
		summary.WriteString("Recommendation: STRONG INVEST - This startup shows exceptional promise across multiple dimensions.\n\n")
	} else if result.OverallScore >= 65 {
		summary.WriteString("Recommendation: CONSIDER - This startup has solid fundamentals but requires further due diligence.\n\n")
	} else if result.OverallScore >= 50 {
		summary.WriteString("Recommendation: CAUTIOUS - Significant concerns exist. Proceed only with risk mitigation strategies.\n\n")
	} else {
		summary.WriteString("Recommendation: PASS - Too many red flags for current investment.\n\n")
	}

	// Highlight top strengths and concerns
	allRedFlags := []string{}
	allOpportunities := []string{}

	for _, section := range result.AnalysisSections {
		allRedFlags = append(allRedFlags, section.RedFlags...)
		allOpportunities = append(allOpportunities, section.Opportunities...)
	}

	if len(allOpportunities) > 0 {
		summary.WriteString("Key Strengths:\n")
		for i, opp := range allOpportunities {
			if i >= 3 {
				break
			}
			summary.WriteString(fmt.Sprintf("• %s\n", opp))
		}
		summary.WriteString("\n")
	}

	if len(allRedFlags) > 0 {
		summary.WriteString("Key Concerns:\n")
		for i, flag := range allRedFlags {
			if i >= 3 {
				break
			}
			summary.WriteString(fmt.Sprintf("• %s\n", flag))
		}
	}

	return summary.String()
}

// storeAnalysis saves the analysis to the database
func (a *StartupAnalyzer) storeAnalysis(req AnalysisRequest, result *AnalysisResult) error {
	for analysisType, section := range result.AnalysisSections {
		keyFindingsJSON, _ := json.Marshal(section.KeyFindings)
		redFlagsJSON, _ := json.Marshal(section.RedFlags)
		opportunitiesJSON, _ := json.Marshal(section.Opportunities)
		recommendationsJSON, _ := json.Marshal(section.Recommendations)
		documentIDsJSON, _ := json.Marshal(req.DocumentIDs)

		analysis := &database.StartupAnalysis{
			AgentID:         req.AgentID,
			MeetingID:       req.MeetingID,
			AnalysisType:    analysisType,
			Score:           section.Score,
			Summary:         section.Summary,
			KeyFindings:     string(keyFindingsJSON),
			RedFlags:        string(redFlagsJSON),
			Opportunities:   string(opportunitiesJSON),
			Recommendations: string(recommendationsJSON),
			DocumentIDs:     string(documentIDsJSON),
			GeneratedAt:     result.GeneratedAt,
			ModelUsed:       "gemini-2.5-flash-lite", // Could be dynamic
		}

		if err := a.db.Create(analysis).Error; err != nil {
			logrus.Errorf("Failed to store analysis: %v", err)
		}
	}

	return nil
}

// GetLatestAnalysis retrieves the latest analysis for an agent
func (a *StartupAnalyzer) GetLatestAnalysis(agentID uuid.UUID) (*AnalysisResult, error) {
	var analyses []database.StartupAnalysis
	err := a.db.
		Where("agent_id = ?", agentID).
		Order("generated_at DESC").
		Limit(10). // Get latest analyses of each type
		Find(&analyses).Error

	if err != nil {
		return nil, err
	}

	if len(analyses) == 0 {
		return nil, fmt.Errorf("no analyses found for agent")
	}

	// Group by analysis type (get most recent of each)
	result := &AnalysisResult{
		AgentID:          agentID,
		AnalysisSections: make(map[string]AnalysisSection),
		GeneratedAt:      analyses[0].GeneratedAt,
	}

	seen := make(map[string]bool)
	totalScore := 0.0

	for _, dbAnalysis := range analyses {
		if seen[dbAnalysis.AnalysisType] {
			continue
		}
		seen[dbAnalysis.AnalysisType] = true

		var keyFindings, redFlags, opportunities, recommendations []string
		json.Unmarshal([]byte(dbAnalysis.KeyFindings), &keyFindings)
		json.Unmarshal([]byte(dbAnalysis.RedFlags), &redFlags)
		json.Unmarshal([]byte(dbAnalysis.Opportunities), &opportunities)
		json.Unmarshal([]byte(dbAnalysis.Recommendations), &recommendations)

		section := AnalysisSection{
			Type:            dbAnalysis.AnalysisType,
			Score:           dbAnalysis.Score,
			Summary:         dbAnalysis.Summary,
			KeyFindings:     keyFindings,
			RedFlags:        redFlags,
			Opportunities:   opportunities,
			Recommendations: recommendations,
		}

		result.AnalysisSections[dbAnalysis.AnalysisType] = section
		totalScore += dbAnalysis.Score
	}

	if len(result.AnalysisSections) > 0 {
		result.OverallScore = totalScore / float64(len(result.AnalysisSections))
	}

	result.Summary = a.generateExecutiveSummary(result)

	return result, nil
}

// Helper functions
func extractJSONFromMarkdown(text string) string {
	start := strings.Index(text, "```json")
	if start == -1 {
		start = strings.Index(text, "```")
	}
	if start == -1 {
		return ""
	}

	start = strings.Index(text[start:], "\n") + start + 1
	end := strings.Index(text[start:], "```")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(text[start : start+end])
}

func getIntFromMetadata(metadata map[string]interface{}, key string) int {
	if val, ok := metadata[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Try to parse string as int
			var result int
			fmt.Sscanf(v, "%d", &result)
			return result
		}
	}
	return 0
}
