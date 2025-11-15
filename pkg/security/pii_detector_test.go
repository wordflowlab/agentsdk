package security

import (
	"context"
	"testing"
)

func TestRegexPIIDetector_Detect(t *testing.T) {
	detector := NewRegexPIIDetector()
	ctx := context.Background()

	tests := []struct {
		name      string
		text      string
		wantTypes []PIIType
		wantCount int
	}{
		{
			name:      "email address",
			text:      "Contact me at john.doe@example.com for details",
			wantTypes: []PIIType{PIIEmail},
			wantCount: 1,
		},
		{
			name:      "chinese phone",
			text:      "我的手机号是13812345678",
			wantTypes: []PIIType{PIIChinesePhone},
			wantCount: 1,
		},
		{
			name:      "credit card",
			text:      "My card number is 4532-1488-0343-6464",
			wantTypes: []PIIType{PIICreditCard},
			wantCount: 1,
		},
		{
			name:      "chinese ID",
			text:      "身份证号：110101199001011237",
			wantTypes: []PIIType{PIIChineseID},
			wantCount: 1,
		},
		{
			name:      "multiple PII",
			text:      "Email: alice@test.com, Phone: 13900001111",
			wantTypes: []PIIType{PIIEmail, PIIChinesePhone},
			wantCount: 2,
		},
		{
			name:      "no PII",
			text:      "This is a normal text without any sensitive information",
			wantTypes: nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := detector.Detect(ctx, tt.text)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			if len(matches) != tt.wantCount {
				t.Errorf("Detect() found %d matches, want %d", len(matches), tt.wantCount)
			}

			// 验证类型
			if tt.wantTypes != nil {
				typeMap := make(map[PIIType]bool)
				for _, match := range matches {
					typeMap[match.Type] = true
				}

				for _, expectedType := range tt.wantTypes {
					if !typeMap[expectedType] {
						t.Errorf("Expected PII type %v not found", expectedType)
					}
				}
			}
		})
	}
}

func TestRegexPIIDetector_DetectTypes(t *testing.T) {
	detector := NewRegexPIIDetector()
	ctx := context.Background()

	text := "Email: user@example.com, Phone: 13812345678, Card: 4532148803436464"

	// 只检测邮箱
	matches, err := detector.DetectTypes(ctx, text, PIIEmail)
	if err != nil {
		t.Fatalf("DetectTypes() error = %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("DetectTypes(PIIEmail) found %d matches, want 1", len(matches))
	}

	if len(matches) > 0 && matches[0].Type != PIIEmail {
		t.Errorf("DetectTypes() type = %v, want %v", matches[0].Type, PIIEmail)
	}

	// 检测多种类型
	matches, err = detector.DetectTypes(ctx, text, PIIEmail, PIIChinesePhone)
	if err != nil {
		t.Fatalf("DetectTypes() error = %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("DetectTypes(Email, Phone) found %d matches, want 2", len(matches))
	}
}

func TestRegexPIIDetector_ContainsPII(t *testing.T) {
	detector := NewRegexPIIDetector()
	ctx := context.Background()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "contains PII",
			text: "Contact: alice@example.com",
			want: true,
		},
		{
			name: "no PII",
			text: "Hello world",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detector.ContainsPII(ctx, tt.text)
			if err != nil {
				t.Fatalf("ContainsPII() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("ContainsPII() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzePII(t *testing.T) {
	detector := NewRegexPIIDetector()
	ctx := context.Background()

	text := "Email: high-risk@test.com, Phone: 13812345678, Card: 4532148803436464"

	result, err := AnalyzePII(ctx, text, detector)
	if err != nil {
		t.Fatalf("AnalyzePII() error = %v", err)
	}

	if !result.HasPII {
		t.Error("AnalyzePII() HasPII = false, want true")
	}

	if result.TotalMatches != 3 {
		t.Errorf("AnalyzePII() TotalMatches = %d, want 3", result.TotalMatches)
	}

	if result.HighestRisk != SensitivityHigh {
		t.Errorf("AnalyzePII() HighestRisk = %v, want %v", result.HighestRisk, SensitivityHigh)
	}
}

func TestValidateChinesePhone(t *testing.T) {
	tests := []struct {
		phone string
		valid bool
	}{
		{"13812345678", true},
		{"13912345678", true},
		{"18612345678", true},
		{"12345678901", false}, // 不是1开头的有效前缀
		{"1381234567", false},  // 太短
		{"138123456789", false}, // 太长
	}

	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			got := validateChinesePhone(tt.phone)
			if got != tt.valid {
				t.Errorf("validateChinesePhone(%s) = %v, want %v", tt.phone, got, tt.valid)
			}
		})
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		number string
		valid  bool
	}{
		{"4532148803436464", true},  // Valid Visa
		{"5425233430109903", true},  // Valid MasterCard
		{"374245455400126", true},   // Valid Amex
		{"4532148803436468", false}, // Invalid (wrong check digit)
		{"1234567890123456", false}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			got := validateLuhn(tt.number)
			if got != tt.valid {
				t.Errorf("validateLuhn(%s) = %v, want %v", tt.number, got, tt.valid)
			}
		})
	}
}

func TestValidateChineseID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"110101199001011237", true},  // 有效的身份证号
		{"110101199001011234", false}, // 无效的校验码
		// 注意：这里使用假的身份证号用于测试
		// 真实的身份证号验证需要正确的校验码
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := validateChineseID(tt.id)
			if got != tt.valid {
				t.Errorf("validateChineseID(%s) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

func TestFilterMatchesByContext(t *testing.T) {
	matches := []PIIMatch{
		{
			Type:       PIIEmail,
			Value:      "user@company.com",
			Confidence: 0.9,
		},
		{
			Type:       PIIChinesePhone,
			Value:      "13812345678",
			Confidence: 0.6,
		},
		{
			Type:       PIICreditCard,
			Value:      "4532148803436464",
			Confidence: 0.95,
		},
	}

	// 测试置信度过滤
	ctx := &PIIContext{
		MinConfidence: 0.8,
	}

	filtered := FilterMatchesByContext(matches, ctx)
	if len(filtered) != 2 {
		t.Errorf("FilterMatchesByContext() returned %d matches, want 2 (confidence >= 0.8)", len(filtered))
	}

	// 测试白名单
	ctx = &PIIContext{
		AllowedTypes: []PIIType{PIIEmail},
	}

	filtered = FilterMatchesByContext(matches, ctx)
	if len(filtered) != 2 {
		t.Errorf("FilterMatchesByContext() returned %d matches, want 2 (excluding email)", len(filtered))
	}

	// 测试忽略模式
	ctx = &PIIContext{
		IgnorePatterns: []string{"@company.com"},
	}

	filtered = FilterMatchesByContext(matches, ctx)
	if len(filtered) != 2 {
		t.Errorf("FilterMatchesByContext() returned %d matches, want 2 (excluding company emails)", len(filtered))
	}
}

func TestGetSensitivityLevel(t *testing.T) {
	tests := []struct {
		piiType  PIIType
		expected PIISensitivityLevel
	}{
		{PIICreditCard, SensitivityHigh},
		{PIIChineseID, SensitivityHigh},
		{PIIChinesePhone, SensitivityMedium},
		{PIIEmail, SensitivityLow},
	}

	for _, tt := range tests {
		t.Run(string(tt.piiType), func(t *testing.T) {
			level := GetSensitivityLevel(tt.piiType)
			if level != tt.expected {
				t.Errorf("GetSensitivityLevel(%v) = %v, want %v", tt.piiType, level, tt.expected)
			}
		})
	}
}
