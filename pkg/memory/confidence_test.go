package memory

import (
	"math"
	"testing"
	"time"
)

func TestConfidenceCalculator_Calculate(t *testing.T) {
	cfg := DefaultConfidenceConfig()
	cfg.DecayHalfLife = 30 * 24 * time.Hour // 30 days
	calc := NewConfidenceCalculator(cfg)

	tests := []struct {
		name           string
		setupProvenance func() *MemoryProvenance
		wantMin        float64
		wantMax        float64
	}{
		{
			name: "fresh user input",
			setupProvenance: func() *MemoryProvenance {
				return NewProvenance(SourceUserInput, "session-1")
			},
			wantMin: 0.69, // 0.70 base, might have tiny decay
			wantMax: 0.71,
		},
		{
			name: "old user input",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceUserInput, "session-1")
				// Simulate 30 days old (half-life)
				p.UpdatedAt = time.Now().Add(-30 * 24 * time.Hour)
				return p
			},
			wantMin: 0.34, // 0.70 * 0.5 (half-life decay)
			wantMax: 0.36,
		},
		{
			name: "bootstrapped data never decays",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceBootstrapped, "crm-1")
				p.UpdatedAt = time.Now().Add(-365 * 24 * time.Hour) // 1 year old
				return p
			},
			wantMin: 0.94,
			wantMax: 0.96,
		},
		{
			name: "corroborated memory",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceUserInput, "session-1")
				p.Corroborate("session-2")
				p.Corroborate("session-3")
				return p
			},
			wantMin: 0.79, // 0.70 + 0.10 (corroboration boost) + 0.10 (recency boost)
			wantMax: 0.95,
		},
		{
			name: "recently accessed memory",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceUserInput, "session-1")
				now := time.Now()
				p.LastAccessedAt = &now
				return p
			},
			wantMin: 0.69,
			wantMax: 0.80, // Base + recency boost
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setupProvenance()
			confidence := calc.Calculate(p)

			if confidence < tt.wantMin || confidence > tt.wantMax {
				t.Errorf("Calculate() = %v, want between %v and %v", confidence, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestConfidenceCalculator_calculateDecayFactor(t *testing.T) {
	cfg := DefaultConfidenceConfig()
	cfg.DecayHalfLife = 30 * 24 * time.Hour
	calc := NewConfidenceCalculator(cfg)

	tests := []struct {
		name      string
		age       time.Duration
		wantDecay float64
		tolerance float64
	}{
		{
			name:      "no decay for fresh memory",
			age:       0,
			wantDecay: 1.0,
			tolerance: 0.01,
		},
		{
			name:      "half decay at half-life",
			age:       30 * 24 * time.Hour,
			wantDecay: 0.5,
			tolerance: 0.01,
		},
		{
			name:      "quarter decay at 2x half-life",
			age:       60 * 24 * time.Hour,
			wantDecay: 0.25,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvenance(SourceUserInput, "session-1")
			p.UpdatedAt = time.Now().Add(-tt.age)

			decay := calc.calculateDecayFactor(p)

			if math.Abs(decay-tt.wantDecay) > tt.tolerance {
				t.Errorf("calculateDecayFactor() = %v, want %v (±%v)", decay, tt.wantDecay, tt.tolerance)
			}
		})
	}

	// Test bootstrapped data doesn't decay
	t.Run("bootstrapped never decays", func(t *testing.T) {
		p := NewProvenance(SourceBootstrapped, "crm-1")
		p.UpdatedAt = time.Now().Add(-365 * 24 * time.Hour) // 1 year old

		decay := calc.calculateDecayFactor(p)

		if decay != 1.0 {
			t.Errorf("Bootstrapped data should not decay, got %v", decay)
		}
	})
}

func TestConfidenceCalculator_ShouldPrune(t *testing.T) {
	cfg := DefaultConfidenceConfig()
	cfg.MinConfidence = 0.20
	cfg.DecayHalfLife = 30 * 24 * time.Hour
	calc := NewConfidenceCalculator(cfg)

	tests := []struct {
		name            string
		setupProvenance func() *MemoryProvenance
		wantPrune       bool
	}{
		{
			name: "fresh memory should not prune",
			setupProvenance: func() *MemoryProvenance {
				return NewProvenance(SourceUserInput, "session-1")
			},
			wantPrune: false,
		},
		{
			name: "very old memory should prune",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceUserInput, "session-1")
				// Make it very old (5x half-life = 3.125% confidence)
				p.UpdatedAt = time.Now().Add(-150 * 24 * time.Hour)
				return p
			},
			wantPrune: true,
		},
		{
			name: "bootstrapped never prunes",
			setupProvenance: func() *MemoryProvenance {
				p := NewProvenance(SourceBootstrapped, "crm-1")
				p.UpdatedAt = time.Now().Add(-1000 * 24 * time.Hour)
				return p
			},
			wantPrune: false,
		},
		{
			name: "explicit memory has lower prune threshold",
			setupProvenance: func() *MemoryProvenance {
				p := NewExplicitProvenance(SourceUserInput, "session-1")
				// Make it old enough that implicit would prune but explicit won't
				// 45 days is about 1.5 half-lives, giving confidence ~0.28
				// With explicit threshold of 0.14 (0.20 * 0.7), it won't prune
				p.UpdatedAt = time.Now().Add(-45 * 24 * time.Hour)
				return p
			},
			wantPrune: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setupProvenance()
			shouldPrune := calc.ShouldPrune(p)

			if shouldPrune != tt.wantPrune {
				confidence := calc.Calculate(p)
				t.Errorf("ShouldPrune() = %v, want %v (confidence: %v)", shouldPrune, tt.wantPrune, confidence)
			}
		})
	}
}

func TestConfidenceCalculator_ScoreByRelevance(t *testing.T) {
	calc := NewConfidenceCalculator(DefaultConfidenceConfig())

	tests := []struct {
		name           string
		semanticScore  float64
		confidence     float64
		wantMin        float64
		wantMax        float64
	}{
		{
			name:           "high semantic, high confidence",
			semanticScore:  0.9,
			confidence:     0.9,
			wantMin:        0.80,
			wantMax:        0.82,
		},
		{
			name:           "high semantic, low confidence",
			semanticScore:  0.9,
			confidence:     0.3,
			wantMin:        0.26,
			wantMax:        0.28,
		},
		{
			name:           "low semantic, high confidence",
			semanticScore:  0.3,
			confidence:     0.9,
			wantMin:        0.26,
			wantMax:        0.28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvenance(SourceUserInput, "session-1")
			p.Confidence = tt.confidence

			score := calc.ScoreByRelevance(tt.semanticScore, p)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("ScoreByRelevance() = %v, want between %v and %v", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetConfidenceTier(t *testing.T) {
	tests := []struct {
		confidence float64
		want       ConfidenceTier
	}{
		{0.95, TierVeryHigh},
		{0.85, TierHigh},
		{0.65, TierMedium},
		{0.45, TierLow},
		{0.15, TierVeryLow},
		{1.0, TierVeryHigh},
		{0.0, TierVeryLow},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			tier := GetConfidenceTier(tt.confidence)
			if tier != tt.want {
				t.Errorf("GetConfidenceTier(%v) = %v, want %v", tt.confidence, tier, tt.want)
			}
		})
	}
}

func TestConfidenceCalculator_UpdateConfidence(t *testing.T) {
	cfg := DefaultConfidenceConfig()
	cfg.DecayHalfLife = 30 * 24 * time.Hour
	calc := NewConfidenceCalculator(cfg)

	p := NewProvenance(SourceUserInput, "session-1")
	initialConfidence := p.Confidence

	// Simulate aging
	p.UpdatedAt = time.Now().Add(-30 * 24 * time.Hour)

	calc.UpdateConfidence(p)

	// Confidence should have decayed
	if p.Confidence >= initialConfidence {
		t.Errorf("Confidence should have decayed, was %v now %v", initialConfidence, p.Confidence)
	}

	// UpdatedAt should be refreshed
	if time.Since(p.UpdatedAt) > time.Second {
		t.Error("UpdatedAt should be recent after UpdateConfidence")
	}
}
