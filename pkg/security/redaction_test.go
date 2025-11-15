package security

import (
	"context"
	"strings"
	"testing"
)

func TestMaskStrategy_Redact(t *testing.T) {
	strategy := NewMaskStrategy()

	tests := []struct {
		name  string
		match PIIMatch
		want  string
	}{
		{
			name: "chinese phone",
			match: PIIMatch{
				Type:  PIIChinesePhone,
				Value: "13812345678",
			},
			want: "138****5678",
		},
		{
			name: "email",
			match: PIIMatch{
				Type:  PIIEmail,
				Value: "john.doe@example.com",
			},
			want: "j*******@example.com",
		},
		{
			name: "credit card",
			match: PIIMatch{
				Type:  PIICreditCard,
				Value: "4532148803436464",
			},
			want: "4532********6464",
		},
		{
			name: "chinese ID",
			match: PIIMatch{
				Type:  PIIChineseID,
				Value: "110101199001011237",
			},
			want: "110101********1237",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.Redact(tt.match)
			if got != tt.want {
				t.Errorf("MaskStrategy.Redact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplaceStrategy_Redact(t *testing.T) {
	strategy := NewReplaceStrategy()

	tests := []struct {
		name  string
		match PIIMatch
		want  string
	}{
		{
			name: "email",
			match: PIIMatch{
				Type:  PIIEmail,
				Value: "user@example.com",
			},
			want: "[EMAIL]",
		},
		{
			name: "phone",
			match: PIIMatch{
				Type:  PIIChinesePhone,
				Value: "13812345678",
			},
			want: "[CHINESE_PHONE]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.Redact(tt.match)
			if got != tt.want {
				t.Errorf("ReplaceStrategy.Redact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashStrategy_Redact(t *testing.T) {
	strategy := NewHashStrategy()

	match := PIIMatch{
		Type:  PIIChinesePhone,
		Value: "13812345678",
	}

	result := strategy.Redact(match)

	// 验证格式
	if !strings.HasPrefix(result, "[HASH:") || !strings.HasSuffix(result, "...]") {
		t.Errorf("HashStrategy.Redact() = %v, want format [HASH:xxx...]", result)
	}

	// 验证一致性（相同输入应产生相同哈希）
	result2 := strategy.Redact(match)
	if result != result2 {
		t.Error("HashStrategy.Redact() should be deterministic")
	}
}

func TestAdaptiveStrategy_Redact(t *testing.T) {
	strategy := NewAdaptiveStrategy()

	tests := []struct {
		name     string
		match    PIIMatch
		wantType string // 期望使用的策略类型（部分匹配）
	}{
		{
			name: "high sensitivity uses replace",
			match: PIIMatch{
				Type:     PIICreditCard,
				Value:    "4532148803436464",
				Severity: SensitivityHigh,
			},
			wantType: "[", // Replace strategy 使用 [TYPE]
		},
		{
			name: "low sensitivity uses mask",
			match: PIIMatch{
				Type:     PIIEmail,
				Value:    "user@example.com",
				Severity: SensitivityLow,
			},
			wantType: "@", // Mask strategy 保留 @
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.Redact(tt.match)
			if !strings.Contains(got, tt.wantType) {
				t.Errorf("AdaptiveStrategy.Redact() = %v, should contain %v", got, tt.wantType)
			}
		})
	}
}

func TestRedactor_Redact(t *testing.T) {
	detector := NewRegexPIIDetector()
	strategy := NewMaskStrategy()
	redactor := NewRedactor(detector, strategy)
	ctx := context.Background()

	tests := []struct {
		name       string
		text       string
		wantRedact bool // 是否应该被脱敏
	}{
		{
			name:       "redact email",
			text:       "Contact me at john.doe@example.com",
			wantRedact: true,
		},
		{
			name:       "redact phone",
			text:       "我的手机号是13812345678",
			wantRedact: true,
		},
		{
			name:       "no PII",
			text:       "This is a normal text",
			wantRedact: false,
		},
		{
			name:       "multiple PII",
			text:       "Email: alice@test.com, Phone: 13900001111",
			wantRedact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, err := redactor.Redact(ctx, tt.text)
			if err != nil {
				t.Fatalf("Redactor.Redact() error = %v", err)
			}

			if tt.wantRedact {
				// 应该被脱敏（与原文不同）
				if redacted == tt.text {
					t.Error("Text should be redacted but wasn't")
				}

				// 检查是否包含掩码字符
				if !strings.Contains(redacted, "*") {
					t.Errorf("Redacted text should contain mask character: %v", redacted)
				}
			} else {
				// 不应该被脱敏（与原文相同）
				if redacted != tt.text {
					t.Errorf("Text should not be redacted but was: %v -> %v", tt.text, redacted)
				}
			}
		})
	}
}

func TestRedactor_RedactWithReport(t *testing.T) {
	detector := NewRegexPIIDetector()
	strategy := NewMaskStrategy()
	redactor := NewRedactor(detector, strategy)
	ctx := context.Background()

	text := "Email: john@example.com, Phone: 13812345678"

	redacted, report, err := redactor.RedactWithReport(ctx, text)
	if err != nil {
		t.Fatalf("RedactWithReport() error = %v", err)
	}

	// 验证报告
	if report.TotalMatches != 2 {
		t.Errorf("Report.TotalMatches = %d, want 2", report.TotalMatches)
	}

	if report.RedactedCharacters == 0 {
		t.Error("Report.RedactedCharacters should be > 0")
	}

	if len(report.MatchesByType) != 2 {
		t.Errorf("Report.MatchesByType length = %d, want 2", len(report.MatchesByType))
	}

	// 验证脱敏结果
	if redacted == text {
		t.Error("Text should be redacted")
	}

	// 验证原文中的 PII 不在脱敏后的文本中
	if strings.Contains(redacted, "john@example.com") {
		t.Error("Original email should be redacted")
	}

	if strings.Contains(redacted, "13812345678") {
		t.Error("Original phone should be redacted")
	}
}

func TestRedactor_MultipleOccurrences(t *testing.T) {
	detector := NewRegexPIIDetector()
	strategy := NewReplaceStrategy()
	redactor := NewRedactor(detector, strategy)
	ctx := context.Background()

	text := "First email: alice@test.com, second email: bob@test.com"

	redacted, report, err := redactor.RedactWithReport(ctx, text)
	if err != nil {
		t.Fatalf("RedactWithReport() error = %v", err)
	}

	if report.TotalMatches != 2 {
		t.Errorf("Should find 2 email addresses, found %d", report.TotalMatches)
	}

	// 两个邮箱都应该被脱敏
	if strings.Contains(redacted, "alice@test.com") || strings.Contains(redacted, "bob@test.com") {
		t.Errorf("Both emails should be redacted: %v", redacted)
	}
}

func TestRedactor_PreserveTextStructure(t *testing.T) {
	detector := NewRegexPIIDetector()
	strategy := NewMaskStrategy()
	redactor := NewRedactor(detector, strategy)
	ctx := context.Background()

	text := "User info: Name=Alice, Email=alice@example.com, Phone=13812345678, Status=Active"

	redacted, err := redactor.Redact(ctx, text)
	if err != nil {
		t.Fatalf("Redactor.Redact() error = %v", err)
	}

	// 验证文本结构保留
	if !strings.Contains(redacted, "User info:") {
		t.Error("Non-PII parts should be preserved")
	}

	if !strings.Contains(redacted, "Name=Alice") {
		t.Error("Non-PII parts should be preserved")
	}

	if !strings.Contains(redacted, "Status=Active") {
		t.Error("Non-PII parts should be preserved")
	}
}

func TestNoOpStrategy(t *testing.T) {
	strategy := &NoOpStrategy{}

	match := PIIMatch{
		Type:  PIIEmail,
		Value: "test@example.com",
	}

	result := strategy.Redact(match)

	if result != match.Value {
		t.Errorf("NoOpStrategy should not modify value, got %v want %v", result, match.Value)
	}
}
