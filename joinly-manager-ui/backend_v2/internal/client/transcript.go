package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// utteranceUpdate processes transcript updates for utterances with enhanced consolidation
func (c *JoinlyClient) utteranceUpdate(transcript interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isJoined {
		return
	}

	transcriptMap, ok := transcript.(map[string]interface{})
	if !ok {
		return
	}

	segments, ok := transcriptMap["segments"].([]interface{})
	if !ok || len(segments) == 0 {
		return
	}

	// Process segments for both participant utterances and assistant responses
	participantSegments := []map[string]interface{}{}
	assistantSegments := []map[string]interface{}{}
	latestStart := c.lastUtteranceStart
	newParticipantAdded := false

	for _, segment := range segments {
		segmentMap, ok := segment.(map[string]interface{})
		if !ok {
			continue
		}

		startVal, ok := segmentMap["start"].(float64)
		if !ok {
			continue
		}

		// Get segment details for processing
		text, _ := segmentMap["text"].(string)

		if c.isAgentSpeaker(segmentMap) {
			// Check if we've already processed this assistant segment (by text content)
			if c.hasProcessedSegment(text) {
				continue
			}
			// This is an assistant response - add to assistant segments for processing
			assistantSegments = append(assistantSegments, segmentMap)
			// Mark this segment as processed immediately
			c.markSegmentProcessed(text)
		} else {
			// For participant segments, use start time filtering AND avoid re-adding the same ones between polls
			if startVal <= c.lastSegmentStart { // already queued before
				continue
			}
			participantSegments = append(participantSegments, segmentMap)
			newParticipantAdded = true
			if startVal > latestStart {
				latestStart = startVal
			}
		}
	}

	// Handle assistant segments - process them silently (no logs unless errors)
	if len(assistantSegments) > 0 {
		go c.handleAssistantSegments(assistantSegments)
	}

	// Handle participant segments (for traditional STT+LLM flow)
	if len(participantSegments) > 0 {
		// Add new segments to pending buffer
		c.pendingSegments = append(c.pendingSegments, participantSegments...)
		// Mark these as queued so we don't re-add on next poll before debounce fires
		if newParticipantAdded {
			if latestStart > c.lastSegmentStart {
				c.lastSegmentStart = latestStart
			}
		}
		c.lastUtteranceTime = time.Now()

		// Check name trigger for any pending segments
		shouldTrigger := !c.config.NameTrigger
		if c.config.NameTrigger {
			for _, segment := range c.pendingSegments {
				if text, ok := segment["text"].(string); ok {
					if c.nameInText(text) {
						shouldTrigger = true
						break
					}
				}
			}
		}

		if shouldTrigger {
			// Reset or start the debounce timer
			if c.debounceTimer != nil {
				c.debounceTimer.Stop()
			}

			c.debounceTimer = time.AfterFunc(c.utteranceDebounce, func() {
				c.processConsolidatedUtterance(latestStart)
			})
		}
	}
}

// handleAssistantSegments processes assistant response segments but does NOT speak them again
func (c *JoinlyClient) handleAssistantSegments(segments []map[string]interface{}) {
	for _, segment := range segments {
		if text, ok := segment["text"].(string); ok && strings.TrimSpace(text) != "" {
			// Mark as processed so we don't try to speak these again
			c.markSegmentProcessed(text)
		}
	}
}

// processConsolidatedUtterance processes all pending segments as a complete utterance
func (c *JoinlyClient) processConsolidatedUtterance(latestStart float64) {
	c.mu.Lock()
	if len(c.pendingSegments) == 0 {
		c.mu.Unlock()
		return
	}

	// Update last utterance BEFORE calling callbacks
	c.lastUtteranceStart = latestStart

	// Compact and merge segments for better coherence
	compactedSegments := c.compactSegments(c.pendingSegments)

	// Prepare text and speaker for tracking
	combinedText := ""
	if len(compactedSegments) > 0 {
		if t, ok := compactedSegments[0]["text"].(string); ok {
			combinedText = t
		}
	}

	// Compute utterance hash and set lifecycle state to received (if new)
	uttHash := c.hashText(combinedText)
	if _, exists := c.utteranceStates[uttHash]; !exists {
		c.utteranceStates[uttHash] = "received"
	}

	// Call utterance callbacks with all consolidated segments (non-blocking). Manager will handle LLM+TTS.
	for _, callback := range c.utteranceCallbacks {
		go callback(compactedSegments)
	}

	// Clear pending segments after processing
	c.pendingSegments = make([]map[string]interface{}, 0)
	c.mu.Unlock()

	// Do NOT generate response or speak here to avoid duplicate TTS. The manager owns LLM+TTS.
}

// compactSegments consolidates adjacent segments from the same speaker into coherent utterances
func (c *JoinlyClient) compactSegments(segments []map[string]interface{}) []map[string]interface{} {
	if len(segments) == 0 {
		return segments
	}

	if len(segments) == 1 {
		return segments
	}

	// Sort segments by start time to ensure proper order
	sort.Slice(segments, func(i, j int) bool {
		startI, okI := segments[i]["start"].(float64)
		startJ, okJ := segments[j]["start"].(float64)
		if !okI || !okJ {
			return false
		}
		return startI < startJ
	})

	compacted := make([]map[string]interface{}, 0)
	current := make(map[string]interface{})

	// Deep copy first segment as starting point
	for k, v := range segments[0] {
		current[k] = v
	}

	for i := 1; i < len(segments); i++ {
		segment := segments[i]

		currentSpeaker := ""
		if speaker, ok := current["speaker"].(string); ok {
			currentSpeaker = speaker
		}

		segmentSpeaker := ""
		if speaker, ok := segment["speaker"].(string); ok {
			segmentSpeaker = speaker
		}

		currentEnd, currentEndOk := current["end"].(float64)
		segmentStart, segmentStartOk := segment["start"].(float64)
		segmentEnd, segmentEndOk := segment["end"].(float64)

		// Check if segments can be merged (same speaker, minimal gap)
		canMerge := currentSpeaker == segmentSpeaker &&
			currentEndOk && segmentStartOk && segmentEndOk &&
			(segmentStart-currentEnd) <= 2.0 // Max 2 second gap to merge

		if canMerge {
			// Merge segments: concatenate text and extend time range
			currentText := ""
			if text, ok := current["text"].(string); ok {
				currentText = text
			}

			segmentText := ""
			if text, ok := segment["text"].(string); ok {
				segmentText = text
			}

			// Concatenate text with space if both exist
			if currentText != "" && segmentText != "" {
				current["text"] = currentText + " " + segmentText
			} else if segmentText != "" {
				current["text"] = segmentText
			}

			// Extend the end time
			current["end"] = segmentEnd
		} else {
			// Cannot merge, add current to result and start new segment
			compacted = append(compacted, current)

			// Start new segment
			current = make(map[string]interface{})
			for k, v := range segment {
				current[k] = v
			}
		}
	}

	// Add the last segment
	compacted = append(compacted, current)

	c.log("debug", fmt.Sprintf("Compacted %d segments into %d segments", len(segments), len(compacted)))

	return compacted
}

// isAgentSpeaker checks if the speaker is the agent itself using role field
func (c *JoinlyClient) isAgentSpeaker(segment map[string]interface{}) bool {
	// First check the role field (most reliable) - no debug logs to reduce noise
	if roleVal, ok := segment["role"].(string); ok {
		if roleVal == "assistant" {
			return true
		}
	}

	// WORKAROUND: Assistant responses may have role='participant' but speaker='Assistant'
	// Check if speaker is 'Assistant' and text contains assistant response format
	if speakerVal, ok := segment["speaker"].(string); ok {
		speaker := speakerVal

		// Check if this is an assistant response (speaker='Assistant' with response text)
		if speaker == "Assistant" {
			if textVal, ok := segment["text"].(string); ok {
				text := textVal
				// Assistant responses often contain "[Heard: ...]" prefix
				if strings.Contains(text, "[Heard:") || strings.Contains(text, "That's great") {
					return true
				}
			}
		}

		// Check if speaker matches agent's name (case-insensitive)
		if c.config.Name != "" && speaker != "" && speaker != "Participant" {
			lowerSpeaker := strings.ToLower(speaker)
			lowerAgentName := strings.ToLower(c.config.Name)
			return lowerSpeaker == lowerAgentName
		}
	}

	return false
}

// nameInText checks if the agent's name is mentioned in the text
func (c *JoinlyClient) nameInText(text string) bool {
	if c.config.Name == "" {
		return true // If no name set, always respond
	}
	lowerText := strings.ToLower(text)
	lowerName := strings.ToLower(c.config.Name)
	return strings.Contains(lowerText, lowerName)
}

// hasProcessedSegment checks if we've already processed this assistant segment text (normalized)
func (c *JoinlyClient) hasProcessedSegment(text string) bool {
	n := strings.ToLower(strings.TrimSpace(text))
	return c.processedSegments[n]
}

// markSegmentProcessed marks an assistant segment text as processed to prevent repetition (normalized)
func (c *JoinlyClient) markSegmentProcessed(text string) {
	n := strings.ToLower(strings.TrimSpace(text))
	c.processedSegments[n] = true
	if len(c.processedSegments) > 100 {
		c.processedSegments = map[string]bool{n: true}
	}
}

// hashText returns a stable hash for utterance deduplication
func (c *JoinlyClient) hashText(text string) string {
	clean := strings.TrimSpace(text)
	sum := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(sum[:])
}
