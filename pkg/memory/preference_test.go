package memory

import (
	"context"
	"testing"
	"time"

	agentext "github.com/wordflowlab/agentsdk/pkg/context"
)

func TestNewPreferenceManager(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	manager := NewPreferenceManager(config)

	if manager == nil {
		t.Fatal("NewPreferenceManager returned nil")
	}

	if len(manager.preferences) != 0 {
		t.Errorf("new manager should have 0 preferences, got %d", len(manager.preferences))
	}
}

func TestPreferenceManager_AddPreference(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}

	err := manager.AddPreference(context.Background(), pref)
	if err != nil {
		t.Fatalf("AddPreference failed: %v", err)
	}

	if pref.ID == "" {
		t.Error("ID should be generated")
	}

	if pref.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestPreferenceManager_GetPreference(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加偏好
	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref)

	// 获取偏好
	retrieved, err := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if err != nil {
		t.Fatalf("GetPreference failed: %v", err)
	}

	if retrieved.Value != "dark" {
		t.Errorf("Value = %s, want 'dark'", retrieved.Value)
	}

	if retrieved.AccessCount != 1 {
		t.Errorf("AccessCount = %d, want 1", retrieved.AccessCount)
	}
}

func TestPreferenceManager_GetPreference_NotFound(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	_, err := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if err == nil {
		t.Error("should return error for non-existent preference")
	}
}

func TestPreferenceManager_ListPreferences(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加多个偏好
	prefs := []*Preference{
		{UserID: "user-1", Category: CategoryUI, Key: "theme", Value: "dark", Strength: 0.8, Confidence: 0.9},
		{UserID: "user-1", Category: CategoryUI, Key: "font", Value: "monospace", Strength: 0.7, Confidence: 0.8},
		{UserID: "user-1", Category: CategoryLanguage, Key: "language", Value: "zh", Strength: 0.9, Confidence: 1.0},
	}

	for _, pref := range prefs {
		manager.AddPreference(context.Background(), pref)
	}

	// 列出所有偏好
	all, err := manager.ListPreferences(context.Background(), "user-1", "")
	if err != nil {
		t.Fatalf("ListPreferences failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("ListPreferences returned %d preferences, want 3", len(all))
	}

	// 列出特定类别的偏好
	uiPrefs, err := manager.ListPreferences(context.Background(), "user-1", CategoryUI)
	if err != nil {
		t.Fatalf("ListPreferences failed: %v", err)
	}

	if len(uiPrefs) != 2 {
		t.Errorf("ListPreferences returned %d UI preferences, want 2", len(uiPrefs))
	}
}

func TestPreferenceManager_UpdatePreference(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加偏好
	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref)

	// 更新偏好
	err := manager.UpdatePreference(context.Background(), "user-1", CategoryUI, "theme", "light")
	if err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	// 验证更新
	updated, _ := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if updated.Value != "light" {
		t.Errorf("Value = %s, want 'light'", updated.Value)
	}
}

func TestPreferenceManager_DeletePreference(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加偏好
	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref)

	// 删除偏好
	err := manager.DeletePreference(context.Background(), "user-1", CategoryUI, "theme")
	if err != nil {
		t.Fatalf("DeletePreference failed: %v", err)
	}

	// 验证已删除
	_, err = manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if err == nil {
		t.Error("preference should be deleted")
	}
}

func TestPreferenceManager_ConflictKeepLatest(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	config.ConflictStrategy = ConflictKeepLatest
	manager := NewPreferenceManager(config)

	// 添加第一个偏好
	pref1 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref1)

	// 添加冲突的偏好（相同 category 和 key）
	pref2 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "light",
		Strength:   0.6,
		Confidence: 0.7,
	}
	manager.AddPreference(context.Background(), pref2)

	// 验证保留了最新的值
	result, _ := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if result.Value != "light" {
		t.Errorf("Value = %s, want 'light'", result.Value)
	}
}

func TestPreferenceManager_ConflictKeepStronger(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	config.ConflictStrategy = ConflictKeepStronger
	manager := NewPreferenceManager(config)

	// 添加第一个偏好
	pref1 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref1)

	// 添加强度更低的冲突偏好
	pref2 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "light",
		Strength:   0.6,
		Confidence: 0.7,
	}
	manager.AddPreference(context.Background(), pref2)

	// 验证保留了强度更高的值
	result, _ := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if result.Value != "dark" {
		t.Errorf("Value = %s, want 'dark'", result.Value)
	}
}

func TestPreferenceManager_ConflictMerge(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	config.ConflictStrategy = ConflictMerge
	manager := NewPreferenceManager(config)

	// 添加第一个偏好
	pref1 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.6,
		Confidence: 0.8,
	}
	manager.AddPreference(context.Background(), pref1)

	// 添加冲突的偏好
	pref2 := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref2)

	// 验证强度被合并
	result, _ := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if result.Strength != 0.7 { // (0.6 + 0.8) / 2
		t.Errorf("Strength = %.2f, want 0.70", result.Strength)
	}

	if result.AccessCount != 1 {
		t.Errorf("AccessCount = %d, want 1", result.AccessCount)
	}
}

func TestPreferenceManager_GetTopPreferences(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加多个偏好（不同强度）
	prefs := []*Preference{
		{UserID: "user-1", Category: CategoryUI, Key: "theme", Value: "dark", Strength: 0.9, Confidence: 0.9},
		{UserID: "user-1", Category: CategoryUI, Key: "font", Value: "mono", Strength: 0.5, Confidence: 0.8},
		{UserID: "user-1", Category: CategoryLanguage, Key: "lang", Value: "zh", Strength: 0.8, Confidence: 1.0},
	}

	for _, pref := range prefs {
		manager.AddPreference(context.Background(), pref)
	}

	// 获取 TOP 2
	top, err := manager.GetTopPreferences(context.Background(), "user-1", 2)
	if err != nil {
		t.Fatalf("GetTopPreferences failed: %v", err)
	}

	if len(top) != 2 {
		t.Errorf("GetTopPreferences returned %d preferences, want 2", len(top))
	}

	// 验证排序
	if top[0].Strength < top[1].Strength {
		t.Error("preferences should be sorted by strength (descending)")
	}

	// 验证第一个是强度最高的
	if top[0].Key != "theme" {
		t.Errorf("top[0].Key = %s, want 'theme'", top[0].Key)
	}
}

func TestPreferenceManager_ApplyDecay(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	config.StrengthDecay = 0.5 // 每天衰减 50%
	config.MinStrength = 0.3
	manager := NewPreferenceManager(config)

	// 添加偏好
	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.8,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref)

	// 手动设置更新时间为 1 天前
	manager.mu.Lock()
	pref.UpdatedAt = time.Now().Add(-24 * time.Hour)
	manager.mu.Unlock()

	// 应用衰减
	removed := manager.ApplyDecay(context.Background())

	// 验证强度衰减
	updated, _ := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	expectedStrength := 0.8 * 0.5 // 衰减 50%

	if updated.Strength < expectedStrength-0.01 || updated.Strength > expectedStrength+0.01 {
		t.Errorf("Strength = %.2f, want ~%.2f", updated.Strength, expectedStrength)
	}

	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

func TestPreferenceManager_ApplyDecay_RemoveWeak(t *testing.T) {
	config := DefaultPreferenceManagerConfig()
	config.StrengthDecay = 0.9 // 每天衰减 90%
	config.MinStrength = 0.3
	manager := NewPreferenceManager(config)

	// 添加弱偏好
	pref := &Preference{
		UserID:     "user-1",
		Category:   CategoryUI,
		Key:        "theme",
		Value:      "dark",
		Strength:   0.5,
		Confidence: 0.9,
	}
	manager.AddPreference(context.Background(), pref)

	// 手动设置更新时间为 1 天前
	manager.mu.Lock()
	pref.UpdatedAt = time.Now().Add(-24 * time.Hour)
	manager.mu.Unlock()

	// 应用衰减
	removed := manager.ApplyDecay(context.Background())

	// 验证被删除（0.5 * 0.1 = 0.05 < 0.3）
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	_, err := manager.GetPreference(context.Background(), "user-1", CategoryUI, "theme")
	if err == nil {
		t.Error("weak preference should be removed")
	}
}

func TestPreferenceManager_GetStats(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())

	// 添加多个偏好
	prefs := []*Preference{
		{UserID: "user-1", Category: CategoryUI, Key: "theme", Value: "dark", Strength: 0.9, Confidence: 0.9},
		{UserID: "user-1", Category: CategoryUI, Key: "font", Value: "mono", Strength: 0.5, Confidence: 0.8},
		{UserID: "user-1", Category: CategoryLanguage, Key: "lang", Value: "zh", Strength: 0.3, Confidence: 1.0},
	}

	for _, pref := range prefs {
		manager.AddPreference(context.Background(), pref)
	}

	stats := manager.GetStats("user-1")

	if stats.TotalPreferences != 3 {
		t.Errorf("TotalPreferences = %d, want 3", stats.TotalPreferences)
	}

	if stats.CategoryDistribution[CategoryUI] != 2 {
		t.Errorf("UI preferences = %d, want 2", stats.CategoryDistribution[CategoryUI])
	}

	if stats.HighStrengthCount != 1 {
		t.Errorf("HighStrengthCount = %d, want 1", stats.HighStrengthCount)
	}

	if stats.MediumStrengthCount != 1 {
		t.Errorf("MediumStrengthCount = %d, want 1", stats.MediumStrengthCount)
	}

	if stats.LowStrengthCount != 1 {
		t.Errorf("LowStrengthCount = %d, want 1", stats.LowStrengthCount)
	}

	expectedAverage := (0.9 + 0.5 + 0.3) / 3
	if stats.AverageStrength < expectedAverage-0.01 || stats.AverageStrength > expectedAverage+0.01 {
		t.Errorf("AverageStrength = %.2f, want ~%.2f", stats.AverageStrength, expectedAverage)
	}
}

func TestPreferenceExtractor_ExtractFromMessage(t *testing.T) {
	manager := NewPreferenceManager(DefaultPreferenceManagerConfig())
	extractor := NewPreferenceExtractor(manager)

	tests := []struct {
		name            string
		message         agentext.Message
		wantPreferences int
		checkFunc       func(t *testing.T, prefs []*Preference)
	}{
		{
			name: "extract dark mode preference",
			message: agentext.Message{
				Role:    "user",
				Content: "I prefer dark mode",
			},
			wantPreferences: 1,
			checkFunc: func(t *testing.T, prefs []*Preference) {
				if prefs[0].Category != CategoryUI {
					t.Errorf("Category = %s, want %s", prefs[0].Category, CategoryUI)
				}
				if prefs[0].Key != "theme" {
					t.Errorf("Key = %s, want 'theme'", prefs[0].Key)
				}
				if prefs[0].Value != "dark" {
					t.Errorf("Value = %s, want 'dark'", prefs[0].Value)
				}
			},
		},
		{
			name: "extract language preference",
			message: agentext.Message{
				Role:    "user",
				Content: "请用中文回答",
			},
			wantPreferences: 1,
			checkFunc: func(t *testing.T, prefs []*Preference) {
				if prefs[0].Category != CategoryLanguage {
					t.Errorf("Category = %s, want %s", prefs[0].Category, CategoryLanguage)
				}
				if prefs[0].Value != "zh" {
					t.Errorf("Value = %s, want 'zh'", prefs[0].Value)
				}
			},
		},
		{
			name: "extract format preference",
			message: agentext.Message{
				Role:    "user",
				Content: "Use spaces for indentation",
			},
			wantPreferences: 1,
			checkFunc: func(t *testing.T, prefs []*Preference) {
				if prefs[0].Category != CategoryFormat {
					t.Errorf("Category = %s, want %s", prefs[0].Category, CategoryFormat)
				}
				if prefs[0].Value != "spaces" {
					t.Errorf("Value = %s, want 'spaces'", prefs[0].Value)
				}
			},
		},
		{
			name: "no preferences",
			message: agentext.Message{
				Role:    "user",
				Content: "What's the weather?",
			},
			wantPreferences: 0,
			checkFunc:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs, err := extractor.ExtractFromMessage(
				context.Background(),
				"user-1",
				tt.message,
			)

			if err != nil {
				t.Fatalf("ExtractFromMessage failed: %v", err)
			}

			if len(prefs) != tt.wantPreferences {
				t.Errorf("extracted %d preferences, want %d", len(prefs), tt.wantPreferences)
			}

			if tt.checkFunc != nil && len(prefs) > 0 {
				tt.checkFunc(t, prefs)
			}
		})
	}
}

func TestDefaultPreferenceManagerConfig(t *testing.T) {
	config := DefaultPreferenceManagerConfig()

	if config.StrengthDecay <= 0 {
		t.Error("StrengthDecay should be > 0")
	}

	if config.MinStrength <= 0 {
		t.Error("MinStrength should be > 0")
	}

	if config.MaxPreferencesPerUser <= 0 {
		t.Error("MaxPreferencesPerUser should be > 0")
	}

	if !config.AutoExtract {
		t.Error("AutoExtract should be true")
	}
}
