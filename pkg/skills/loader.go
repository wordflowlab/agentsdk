package skills

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wordflowlab/agentsdk/pkg/sandbox"
)

// SkillLoader 技能加载器
type SkillLoader struct {
	baseDir string            // 技能目录路径
	fs      sandbox.SandboxFS // 文件系统接口
}

// NewLoader 创建加载器
func NewLoader(baseDir string, fs sandbox.SandboxFS) *SkillLoader {
	if baseDir == "" {
		baseDir = "skills"
	}
	return &SkillLoader{
		baseDir: baseDir,
		fs:      fs,
	}
}

// Load 加载技能定义
func (sl *SkillLoader) Load(ctx context.Context, skillPath string) (*SkillDefinition, error) {
	// skillPath 可能是 "consistency-checker" 或 "quality-assurance/consistency-checker"
	// 需要找到对应的 SKILL.md 文件
	fullPath := filepath.Join(sl.baseDir, skillPath, "SKILL.md")

	content, err := sl.fs.Read(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	return sl.parse(skillPath, content)
}

// parse 解析技能文件
func (sl *SkillLoader) parse(name, content string) (*SkillDefinition, error) {
	// 分割 YAML frontmatter 和 Markdown 内容
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid skill format: missing YAML frontmatter")
	}

	skill := &SkillDefinition{
		Name: name,
	}

	// 解析 YAML frontmatter
	yamlContent := strings.TrimSpace(parts[1])
	if err := sl.parseYAML(yamlContent, skill); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// 提取 Markdown 内容作为知识库
	skill.KnowledgeBase = strings.TrimSpace(strings.Join(parts[2:], "---"))

	return skill, nil
}

// parseYAML 解析 YAML 配置
func (sl *SkillLoader) parseYAML(yamlContent string, skill *SkillDefinition) error {
	var config struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description"`
		AllowedTools []string `yaml:"allowed-tools"`
		Triggers     []struct {
			Type      string   `yaml:"type"`
			Keywords  []string `yaml:"keywords"`
			Condition string   `yaml:"condition"`
		} `yaml:"triggers"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		return err
	}

	if config.Name != "" {
		skill.Name = config.Name
	}
	skill.Description = config.Description
	skill.AllowedTools = config.AllowedTools

	// 转换 triggers
	skill.Triggers = make([]TriggerConfig, 0, len(config.Triggers))
	for _, t := range config.Triggers {
		skill.Triggers = append(skill.Triggers, TriggerConfig{
			Type:      t.Type,
			Keywords:  t.Keywords,
			Condition: t.Condition,
		})
	}

	return nil
}

// LoadMultiple 批量加载技能
func (sl *SkillLoader) LoadMultiple(ctx context.Context, skillPaths []string) (map[string]*SkillDefinition, error) {
	skills := make(map[string]*SkillDefinition)

	for _, path := range skillPaths {
		skill, err := sl.Load(ctx, path)
		if err != nil {
			// 记录错误但继续加载其他技能
			continue
		}
		skills[path] = skill
	}

	return skills, nil
}

// Discover 发现所有可用技能
func (sl *SkillLoader) Discover(ctx context.Context) ([]string, error) {
	// 递归搜索所有 SKILL.md 文件
	pattern := "**/SKILL.md"
	files, err := sl.fs.Glob(ctx, pattern, &sandbox.GlobOptions{
		CWD:      sl.baseDir,
		Absolute: false,
	})
	if err != nil {
		return nil, fmt.Errorf("glob skills: %w", err)
	}

	skills := make([]string, 0, len(files))
	for _, file := range files {
		// 提取技能路径（去掉 /SKILL.md）
		skillPath := strings.TrimSuffix(file, "/SKILL.md")
		skills = append(skills, skillPath)
	}

	return skills, nil
}
