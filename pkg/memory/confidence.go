package memory

import (
	"math"
	"time"
)

// ConfidenceConfig 置信度计算配置。
type ConfidenceConfig struct {
	// DecayHalfLife 置信度衰减半衰期。
	// 记忆经过这个时长后，置信度衰减到原来的50%。
	DecayHalfLife time.Duration

	// MinConfidence 最低置信度阈值。
	// 低于此值的记忆应被剪枝（遗忘）。
	MinConfidence float64

	// CorroborationBoost 每次验证增加的置信度。
	CorroborationBoost float64

	// MaxCorroborationBoost 验证提升的最大累计值。
	MaxCorroborationBoost float64

	// RecencyWeight 最近访问对置信度的权重（0.0-1.0）。
	// 0 表示不考虑访问时间，1 表示完全基于访问时间。
	RecencyWeight float64
}

// DefaultConfidenceConfig 返回默认置信度配置。
func DefaultConfidenceConfig() ConfidenceConfig {
	return ConfidenceConfig{
		DecayHalfLife:          30 * 24 * time.Hour, // 30天
		MinConfidence:          0.20,
		CorroborationBoost:     0.05,
		MaxCorroborationBoost:  0.20,
		RecencyWeight:          0.3,
	}
}

// ConfidenceCalculator 置信度计算器。
type ConfidenceCalculator struct {
	cfg ConfidenceConfig
}

// NewConfidenceCalculator 创建置信度计算器。
func NewConfidenceCalculator(cfg ConfidenceConfig) *ConfidenceCalculator {
	return &ConfidenceCalculator{cfg: cfg}
}

// Calculate 计算记忆的当前置信度。
// 综合考虑：初始置信度、时间衰减、验证次数、访问记录。
func (cc *ConfidenceCalculator) Calculate(p *MemoryProvenance) float64 {
	if p == nil {
		return 0.0
	}

	// 1. 基础置信度（根据来源类型）
	baseConfidence := p.Confidence

	// 2. 时间衰减
	decayFactor := cc.calculateDecayFactor(p)
	decayedConfidence := baseConfidence * decayFactor

	// 3. 验证提升
	corroborationBoost := cc.calculateCorroborationBoost(p)

	// 4. 访问记录影响
	recencyBoost := cc.calculateRecencyBoost(p)

	// 综合计算
	finalConfidence := decayedConfidence + corroborationBoost + recencyBoost

	// 限制在 [0, 1] 范围内
	if finalConfidence < 0 {
		finalConfidence = 0
	}
	if finalConfidence > 1 {
		finalConfidence = 1
	}

	return finalConfidence
}

// calculateDecayFactor 计算时间衰减因子。
// 使用指数衰减模型：decay = 0.5^(age / halfLife)
func (cc *ConfidenceCalculator) calculateDecayFactor(p *MemoryProvenance) float64 {
	if cc.cfg.DecayHalfLife == 0 {
		return 1.0 // 不衰减
	}

	// Bootstrapped 数据不衰减
	if p.SourceType == SourceBootstrapped {
		return 1.0
	}

	age := p.Freshness() // 使用最后更新时间
	if age <= 0 {
		return 1.0
	}

	// 指数衰减: 0.5^(t / halfLife)
	exponent := float64(age) / float64(cc.cfg.DecayHalfLife)
	decay := math.Pow(0.5, exponent)

	return decay
}

// calculateCorroborationBoost 计算验证提升。
func (cc *ConfidenceCalculator) calculateCorroborationBoost(p *MemoryProvenance) float64 {
	if p.CorroborationCount == 0 {
		return 0.0
	}

	boost := cc.cfg.CorroborationBoost * float64(p.CorroborationCount)
	if boost > cc.cfg.MaxCorroborationBoost {
		boost = cc.cfg.MaxCorroborationBoost
	}

	return boost
}

// calculateRecencyBoost 计算访问记录对置信度的提升。
// 最近被访问的记忆更有价值。
func (cc *ConfidenceCalculator) calculateRecencyBoost(p *MemoryProvenance) float64 {
	if cc.cfg.RecencyWeight == 0 || p.LastAccessedAt == nil {
		return 0.0
	}

	timeSinceAccess := time.Since(*p.LastAccessedAt)
	if timeSinceAccess < 0 {
		timeSinceAccess = 0
	}

	// 访问在1天内：完整权重
	// 访问在30天前：无权重
	const maxRecencyPeriod = 30 * 24 * time.Hour
	if timeSinceAccess > maxRecencyPeriod {
		return 0.0
	}

	recencyFactor := 1.0 - (float64(timeSinceAccess) / float64(maxRecencyPeriod))
	boost := cc.cfg.RecencyWeight * recencyFactor * 0.1 // 最多提升10%

	return boost
}

// ShouldPrune 判断记忆是否应被剪枝（遗忘）。
func (cc *ConfidenceCalculator) ShouldPrune(p *MemoryProvenance) bool {
	if p == nil {
		return true
	}

	// Bootstrapped 数据永不剪枝
	if p.SourceType == SourceBootstrapped {
		return false
	}

	// 显式记忆降低剪枝阈值
	threshold := cc.cfg.MinConfidence
	if p.IsExplicit {
		threshold = cc.cfg.MinConfidence * 0.7 // 显式记忆更宽容
	}

	currentConfidence := cc.Calculate(p)
	return currentConfidence < threshold
}

// UpdateConfidence 更新记忆的置信度。
// 这会修改 Provenance 对象的 Confidence 字段。
func (cc *ConfidenceCalculator) UpdateConfidence(p *MemoryProvenance) {
	if p == nil {
		return
	}

	newConfidence := cc.Calculate(p)
	if newConfidence != p.Confidence {
		p.Confidence = newConfidence
		p.UpdatedAt = time.Now()
	}
}

// ScoreByRelevance 综合计算记忆的相关性得分。
// 结合语义相似度和置信度。
func (cc *ConfidenceCalculator) ScoreByRelevance(semanticScore float64, p *MemoryProvenance) float64 {
	if p == nil {
		return semanticScore
	}

	confidence := cc.Calculate(p)

	// 相关性得分 = 语义相似度 × 置信度
	// 这样可以过滤掉高相似但低置信的结果
	relevanceScore := semanticScore * confidence

	return relevanceScore
}

// ConfidenceTier 将置信度分级。
type ConfidenceTier string

const (
	TierVeryHigh ConfidenceTier = "very_high" // > 0.9
	TierHigh     ConfidenceTier = "high"      // 0.7 - 0.9
	TierMedium   ConfidenceTier = "medium"    // 0.5 - 0.7
	TierLow      ConfidenceTier = "low"       // 0.3 - 0.5
	TierVeryLow  ConfidenceTier = "very_low"  // < 0.3
)

// GetConfidenceTier 返回置信度分级。
func GetConfidenceTier(confidence float64) ConfidenceTier {
	switch {
	case confidence > 0.9:
		return TierVeryHigh
	case confidence > 0.7:
		return TierHigh
	case confidence > 0.5:
		return TierMedium
	case confidence > 0.3:
		return TierLow
	default:
		return TierVeryLow
	}
}
