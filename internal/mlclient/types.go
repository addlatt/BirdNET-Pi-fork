package mlclient

// Status represents the ML service status response.
type Status struct {
	BirdNET BirdNETStatus `json:"birdnet"`
	VAD     VADStatus     `json:"vad"`
	LLM     LLMStatus     `json:"llm"`
}

// BirdNETStatus represents BirdNET model status.
type BirdNETStatus struct {
	Loaded      bool   `json:"loaded"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

// VADStatus represents VAD model status (Part 2).
type VADStatus struct {
	Enabled bool `json:"enabled"`
}

// LLMStatus represents LLM model status (Part 2).
type LLMStatus struct {
	Enabled bool `json:"enabled"`
	Loaded  bool `json:"loaded"`
}

// MemoryStats represents memory usage from the ML service.
type MemoryStats struct {
	BirdNET uint64 `json:"birdnet"`
	VAD     uint64 `json:"vad"`
	LLM     uint64 `json:"llm"`
	Total   uint64 `json:"total"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// Part 2 types (defined now, used later)

// VADRequest represents a VAD check request.
type VADRequest struct {
	AudioPath string `json:"audio_path"`
}

// VADResult represents a VAD check result.
type VADResult struct {
	HasSpeech       bool    `json:"has_speech"`
	SpeechScore     float64 `json:"speech_score"`
	SpeechDuration  float64 `json:"speech_duration"`
	ShouldSkip      bool    `json:"should_skip"`
}

// LLMRequest represents an LLM query request.
type LLMRequest struct {
	Question    string  `json:"question"`
	Context     string  `json:"context,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// LLMResponse represents an LLM query response.
type LLMResponse struct {
	Answer     string `json:"answer"`
	TokensUsed int    `json:"tokens_used"`
}
