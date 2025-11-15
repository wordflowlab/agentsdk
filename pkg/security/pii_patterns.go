package security

import (
	"regexp"
	"strings"
)

// PIIType 定义 PII 的类型。
type PIIType string

const (
	PIIEmail          PIIType = "email"
	PIIPhone          PIIType = "phone"
	PIICreditCard     PIIType = "credit_card"
	PIISSNus          PIIType = "ssn_us"           // 美国社会安全号
	PIIChineseID      PIIType = "chinese_id"       // 中国身份证
	PIIChinesePhone   PIIType = "chinese_phone"    // 中国手机号
	PIIIPAddress      PIIType = "ip_address"
	PIIPassport       PIIType = "passport"
	PIIBankAccount    PIIType = "bank_account"
	PIIDateOfBirth    PIIType = "date_of_birth"
	PIIAddress        PIIType = "address"
	PIIName           PIIType = "name"              // 需要 LLM 检测
	PIICustom         PIIType = "custom"
)

// PIIPattern 定义一个 PII 检测模式。
type PIIPattern struct {
	Type        PIIType
	Description string
	Regex       *regexp.Regexp
	Validator   func(string) bool // 可选的额外验证函数
}

// PIIPatternRegistry PII 模式注册表。
var PIIPatternRegistry = []PIIPattern{
	// 邮箱地址
	{
		Type:        PIIEmail,
		Description: "Email address",
		Regex:       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	},

	// 中国手机号（11位，1开头）- 放在美国电话之前以避免误匹配
	{
		Type:        PIIChinesePhone,
		Description: "Chinese mobile phone number",
		Regex:       regexp.MustCompile(`\b1[3-9]\d{9}\b`),
		Validator:   validateChinesePhone,
	},

	// 美国电话号码
	{
		Type:        PIIPhone,
		Description: "US phone number",
		Regex:       regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})\b`),
	},

	// 信用卡号（支持主流卡组织，支持破折号/空格分隔符）
	{
		Type:        PIICreditCard,
		Description: "Credit card number",
		Regex:       regexp.MustCompile(`\b(?:4[0-9]{3}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}|5[1-5][0-9]{2}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}|3[47][0-9]{2}[-\s]?[0-9]{6}[-\s]?[0-9]{5})\b`),
		Validator:   validateLuhn, // Luhn 算法验证
	},

	// 美国社会安全号 (SSN)
	// 注意：Go 的 regexp 不支持负向前瞻，所以使用简化模式
	{
		Type:        PIISSNus,
		Description: "US Social Security Number",
		Regex:       regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
		Validator:   validateSSN,
	},

	// 中国身份证号（18位）
	{
		Type:        PIIChineseID,
		Description: "Chinese ID card number",
		Regex:       regexp.MustCompile(`\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[0-9Xx]\b`),
		Validator:   validateChineseID,
	},

	// IPv4 地址
	{
		Type:        PIIIPAddress,
		Description: "IPv4 address",
		Regex:       regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`),
	},

	// 护照号（通用格式）
	{
		Type:        PIIPassport,
		Description: "Passport number (generic)",
		Regex:       regexp.MustCompile(`\b[A-Z]{1,2}[0-9]{6,9}\b`),
	},

	// 日期格式（可能是出生日期）
	{
		Type:        PIIDateOfBirth,
		Description: "Date (potential date of birth)",
		Regex:       regexp.MustCompile(`\b(19|20)\d{2}[-/](0[1-9]|1[0-2])[-/](0[1-9]|[12]\d|3[01])\b`),
	},
}

// validateSSN 验证美国社会安全号的有效性。
func validateSSN(ssn string) bool {
	// 移除破折号
	clean := strings.ReplaceAll(ssn, "-", "")

	if len(clean) != 9 {
		return false
	}

	// 验证不是全0
	if clean == "000000000" {
		return false
	}

	// 获取区域号（前3位）
	area := clean[0:3]

	// 无效的区域号
	invalidAreas := map[string]bool{
		"000": true, "666": true,
	}

	if invalidAreas[area] {
		return false
	}

	// 900-999 系列保留给 IRS
	if area[0] == '9' {
		return false
	}

	// 验证组号（中间2位）不为 00
	group := clean[3:5]
	if group == "00" {
		return false
	}

	// 验证序列号（最后4位）不为 0000
	serial := clean[5:9]
	if serial == "0000" {
		return false
	}

	return true
}

// validateChinesePhone 验证中国手机号的有效性。
func validateChinesePhone(phone string) bool {
	// 基本验证：11位，1开头
	if len(phone) != 11 {
		return false
	}

	// 验证运营商号段
	prefix := phone[0:3]
	validPrefixes := map[string]bool{
		// 中国移动
		"134": true, "135": true, "136": true, "137": true, "138": true, "139": true,
		"147": true, "148": true, "150": true, "151": true, "152": true, "157": true,
		"158": true, "159": true, "172": true, "178": true, "182": true, "183": true,
		"184": true, "187": true, "188": true, "198": true,
		// 中国联通
		"130": true, "131": true, "132": true, "145": true, "146": true, "155": true,
		"156": true, "166": true, "171": true, "175": true, "176": true, "185": true,
		"186": true, "196": true,
		// 中国电信
		"133": true, "149": true, "153": true, "173": true, "174": true, "177": true,
		"180": true, "181": true, "189": true, "191": true, "193": true, "199": true,
	}

	return validPrefixes[prefix]
}

// validateLuhn 使用 Luhn 算法验证信用卡号。
func validateLuhn(number string) bool {
	// 移除所有非数字字符
	var digits []int
	for _, r := range number {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	// Luhn 算法
	sum := 0
	alternate := false

	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// validateChineseID 验证中国身份证号的有效性。
func validateChineseID(id string) bool {
	if len(id) != 18 {
		return false
	}

	// 验证校验码
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []rune{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(id[i] - '0')
		sum += digit * weights[i]
	}

	checkCodeIndex := sum % 11
	expectedCheckCode := checkCodes[checkCodeIndex]
	actualCheckCode := rune(id[17])

	if actualCheckCode == 'x' {
		actualCheckCode = 'X'
	}

	return actualCheckCode == expectedCheckCode
}

// GetPatternsByType 按类型获取 PII 模式。
func GetPatternsByType(types ...PIIType) []PIIPattern {
	if len(types) == 0 {
		return PIIPatternRegistry
	}

	typeMap := make(map[PIIType]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	var patterns []PIIPattern
	for _, pattern := range PIIPatternRegistry {
		if typeMap[pattern.Type] {
			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

// AddCustomPattern 添加自定义 PII 模式。
func AddCustomPattern(pattern PIIPattern) {
	PIIPatternRegistry = append(PIIPatternRegistry, pattern)
}

// PIISensitivityLevel PII 敏感度级别。
type PIISensitivityLevel int

const (
	SensitivityLow    PIISensitivityLevel = 1 // 低敏感（如邮箱）
	SensitivityMedium PIISensitivityLevel = 2 // 中等敏感（如电话号码）
	SensitivityHigh   PIISensitivityLevel = 3 // 高敏感（如身份证、信用卡）
)

// GetSensitivityLevel 返回 PII 类型的敏感度级别。
func GetSensitivityLevel(piiType PIIType) PIISensitivityLevel {
	highSensitivity := map[PIIType]bool{
		PIICreditCard:  true,
		PIISSNus:       true,
		PIIChineseID:   true,
		PIIPassport:    true,
		PIIBankAccount: true,
	}

	mediumSensitivity := map[PIIType]bool{
		PIIPhone:        true,
		PIIChinesePhone: true,
		PIIDateOfBirth:  true,
		PIIAddress:      true,
	}

	if highSensitivity[piiType] {
		return SensitivityHigh
	}
	if mediumSensitivity[piiType] {
		return SensitivityMedium
	}
	return SensitivityLow
}
