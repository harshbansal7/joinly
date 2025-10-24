package client

import (
	"fmt"
	"sort"
	"time"
)

// utteranceProcessor handles transcript processing and utterance callbacks
type utteranceProcessor struct {
	client *JoinlyClient

	// Utterance callback system
	utteranceCallbacks []func([]map[string]interface{})

	// Processing state
	pendingSegments   []map[string]interface{}
	lastUtteranceTime time.Time
	utteranceDebounce time.Duration
	debounceTimer     *time.Timer

	// Deduplication and tracking
	processedSegments map[string]bool
	utteranceStates   map[string]string

	// Transcript tracking (like original client)
	lastUtteranceStart float64
	lastSegmentStart   float64
}

// newUtteranceProcessor creates a new utterance processor
func newUtteranceProcessor(c *JoinlyClient) *utteranceProcessor {
	return &utteranceProcessor{
		client:            c,
		utteranceDebounce: 2 * time.Second,
		processedSegments: make(map[string]bool),
		utteranceStates:   make(map[string]string),
	}
}

// AddUtteranceCallback adds a callback for utterance events
func (up *utteranceProcessor) AddUtteranceCallback(callback func([]map[string]interface{})) {
	up.utteranceCallbacks = append(up.utteranceCallbacks, callback)
}

// processTranscriptUpdate processes incoming transcript updates
func (up *utteranceProcessor) processTranscriptUpdate(transcript interface{}) {
	transcriptMap, ok := transcript.(map[string]interface{})
	if !ok {
		return
	}

	segments, ok := transcriptMap["segments"].([]interface{})
	if !ok || len(segments) == 0 {
		return
	}

	// Convert segments to the expected format
	var segmentMaps []map[string]interface{}
	for _, segment := range segments {
		if segmentMap, ok := segment.(map[string]interface{}); ok {
			segmentMaps = append(segmentMaps, segmentMap)
		}
	}

	if len(segmentMaps) == 0 {
		return
	}

	up.client.log("debug", fmt.Sprintf("Processing %d transcript segments", len(segmentMaps)))

	// Filter and process new segments
	newSegments := up.filterNewSegments(segmentMaps)
	if len(newSegments) == 0 {
		return
	}

	// Add to pending segments
	up.pendingSegments = append(up.pendingSegments, newSegments...)

	// Reset debounce timer
	up.resetDebounceTimer()
}

// filterNewSegments filters out segments that have already been processed
func (up *utteranceProcessor) filterNewSegments(segments []map[string]interface{}) []map[string]interface{} {
	var newSegments []map[string]interface{}

	for _, segment := range segments {
		segmentHash := up.generateSegmentHash(segment)
		if !up.processedSegments[segmentHash] {
			newSegments = append(newSegments, segment)
			up.processedSegments[segmentHash] = true
		} else {
			up.client.log("debug", fmt.Sprintf("Segment %s already processed", segmentHash))
		}
	}

	return newSegments
}

// generateSegmentHash creates a simple hash for segment deduplication
func (up *utteranceProcessor) generateSegmentHash(segment map[string]interface{}) string {
	// Create a simple hash based on timestamp and text
	timestamp, _ := segment["timestamp"].(float64)
	text, _ := segment["text"].(string)
	speaker, _ := segment["speaker"].(string)

	return fmt.Sprintf("%.2f|%s|%s", timestamp, speaker, text)
}

// resetDebounceTimer resets the debounce timer for utterance processing
func (up *utteranceProcessor) resetDebounceTimer() {
	// Cancel existing timer if running
	if up.debounceTimer != nil {
		up.debounceTimer.Stop()
	}

	up.debounceTimer = time.AfterFunc(up.utteranceDebounce, func() {
		up.processPendingUtterance()
	})
}

// processPendingUtterance processes the pending utterance after debounce
func (up *utteranceProcessor) processPendingUtterance() {
	if len(up.pendingSegments) == 0 {
		return
	}

	// Sort segments by timestamp
	sort.Slice(up.pendingSegments, func(i, j int) bool {
		timestampI, _ := up.pendingSegments[i]["timestamp"].(float64)
		timestampJ, _ := up.pendingSegments[j]["timestamp"].(float64)
		return timestampI < timestampJ
	})

	// Create utterance from pending segments
	utterance := up.pendingSegments

	// Clear pending segments
	up.pendingSegments = nil

	// Update tracking
	up.lastUtteranceTime = time.Now()
	up.updateTranscriptTracking(utterance)

	// Trigger callbacks
	up.triggerUtteranceCallbacks(utterance)

	up.client.log("info", fmt.Sprintf("🎤 Processed utterance with %d segments", len(utterance)))
}

// updateTranscriptTracking updates transcript tracking like the original client
func (up *utteranceProcessor) updateTranscriptTracking(segments []map[string]interface{}) {
	for _, segment := range segments {
		if timestamp, ok := segment["timestamp"].(float64); ok {
			if up.lastUtteranceStart == 0.0 || timestamp < up.lastUtteranceStart {
				up.lastUtteranceStart = timestamp
			}
			if timestamp > up.lastSegmentStart {
				up.lastSegmentStart = timestamp
			}
		}
	}
}

// triggerUtteranceCallbacks calls all registered utterance callbacks
func (up *utteranceProcessor) triggerUtteranceCallbacks(utterance []map[string]interface{}) {
	for _, callback := range up.utteranceCallbacks {
		if callback != nil {
			go func(cb func([]map[string]interface{})) {
				defer func() {
					if r := recover(); r != nil {
						up.client.log("error", fmt.Sprintf("Utterance callback panic: %v", r))
					}
				}()
				cb(utterance)
			}(callback)
		}
	}
}

// ResetTranscriptTracking resets transcript tracking (called when joining new meeting)
func (up *utteranceProcessor) ResetTranscriptTracking() {
	up.lastUtteranceStart = 0.0
	up.lastSegmentStart = 0.0
	up.pendingSegments = nil
	up.processedSegments = make(map[string]bool)
	up.utteranceStates = make(map[string]string)

	// Cancel any pending debounce timer
	if up.debounceTimer != nil {
		up.debounceTimer.Stop()
		up.debounceTimer = nil
	}

	up.client.log("debug", "Transcript tracking reset")
}

// GetUtteranceCallbacks returns the registered callbacks (for testing/debugging)
func (up *utteranceProcessor) GetUtteranceCallbacks() []func([]map[string]interface{}) {
	return up.utteranceCallbacks
}
