package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mainbong/storage_doctor/internal/filesystem"
	"gopkg.in/yaml.v3"
)

// Skill represents an agent skill
type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string // Full SKILL.md content
	Path        string // Path to skill directory
}

// SkillManager manages agent skills
type SkillManager struct {
	skillsDir string
	skills    []Skill
	fs        filesystem.FileSystem
}

// NewSkillManager creates a new skill manager
func NewSkillManager(skillsDir string) (*SkillManager, error) {
	return NewSkillManagerWithFS(skillsDir, filesystem.NewOSFileSystem())
}

// NewSkillManagerWithFS creates a new skill manager with a custom FileSystem (for testing)
func NewSkillManagerWithFS(skillsDir string, fs filesystem.FileSystem) (*SkillManager, error) {
	sm := &SkillManager{
		skillsDir: skillsDir,
		skills:    make([]Skill, 0),
		fs:        fs,
	}

	// Load skills on initialization
	if err := sm.LoadSkills(); err != nil {
		return nil, fmt.Errorf("failed to load skills: %w", err)
	}

	return sm, nil
}

// LoadSkills loads all skills from the skills directory
func (sm *SkillManager) LoadSkills() error {
	if _, err := sm.fs.Stat(sm.skillsDir); os.IsNotExist(err) {
		// Create default skills directory
		if err := sm.fs.MkdirAll(sm.skillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}
		// Create default skills
		if err := sm.createDefaultSkills(); err != nil {
			return fmt.Errorf("failed to create default skills: %w", err)
		}
	}

	sm.skills = make([]Skill, 0)

	// Walk through skills directory
	err := sm.fs.Walk(sm.skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for SKILL.md files
		if info.Name() == "SKILL.md" {
			skill, err := sm.loadSkill(path)
			if err != nil {
				return fmt.Errorf("failed to load skill at %s: %w", path, err)
			}
			sm.skills = append(sm.skills, skill)
		}

		return nil
	})

	return err
}

// loadSkill loads a single skill from SKILL.md file
func (sm *SkillManager) loadSkill(path string) (Skill, error) {
	data, err := sm.fs.ReadFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to read skill file: %w", err)
	}

	content := string(data)

	// Parse YAML frontmatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Skill{}, fmt.Errorf("invalid skill format: missing YAML frontmatter")
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(parts[1]), &skill); err != nil {
		return Skill{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Store full content and path
	skill.Content = content
	skill.Path = filepath.Dir(path)

	return skill, nil
}

// GetSkills returns all loaded skills
func (sm *SkillManager) GetSkills() []Skill {
	return sm.skills
}

// GetSkillMetadata returns skill metadata (name and description) for system prompt
func (sm *SkillManager) GetSkillMetadata() string {
	if len(sm.skills) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("사용 가능한 스킬:\n")
	for i, skill := range sm.skills {
		builder.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, skill.Name, skill.Description))
	}
	builder.WriteString("\n작업과 관련된 스킬이 있다면 해당 스킬을 활성화하여 사용하세요.\n")

	return builder.String()
}

// ActivateSkill activates a skill by name and returns its full content
func (sm *SkillManager) ActivateSkill(name string) (string, error) {
	for _, skill := range sm.skills {
		if skill.Name == name {
			return skill.Content, nil
		}
	}
	return "", fmt.Errorf("skill not found: %s", name)
}

// createDefaultSkills creates default skills for storage doctor
func (sm *SkillManager) createDefaultSkills() error {
	// Storage Diagnosis Skill
	storageSkill := `---
name: storage_diagnosis
description: Kubernetes 및 클라우드 스토리지 문제 진단 및 해결
---

# Storage Diagnosis Skill

이 스킬은 Kubernetes 및 클라우드 환경에서 발생하는 스토리지 문제를 진단하고 해결하는 전문 지식을 제공합니다.

## 주요 기능

### 1. PVC 문제 진단
- PVC 상태 확인
- StorageClass 설정 검증
- 볼륨 바인딩 문제 해결

### 2. 스토리지 드라이버 문제
- CSI 드라이버 상태 확인
- 드라이버 로그 분석
- 드라이버 재시작 및 복구

### 3. 디스크 공간 문제
- 디스크 사용량 확인
- 오래된 리소스 정리
- 스토리지 확장

## 사용 방법

문제가 발생하면 다음 순서로 진단하세요:
1. 관련 리소스 상태 확인 (kubectl get)
2. 이벤트 및 로그 확인
3. 설정 파일 검증
4. 웹 검색을 통한 유사 사례 확인
5. 해결 방안 적용

## 주의사항

- 프로덕션 환경에서는 항상 백업 후 작업
- 변경 사항은 단계적으로 적용
- 롤백 계획을 항상 준비
`

	// File Operations Skill
	fileSkill := `---
name: file_operations
description: 설정 파일 읽기, 수정, 백업 및 복구
---

# File Operations Skill

이 스킬은 Kubernetes 매니페스트, 설정 파일 등을 안전하게 수정하는 방법을 제공합니다.

## 주요 기능

### 1. 파일 읽기
- YAML, JSON, TOML 파일 파싱
- 설정 검증
- 구조 분석

### 2. 파일 수정
- 자동 백업 생성
- 안전한 수정 절차
- 변경 사항 검증

### 3. 롤백
- 백업에서 복구
- 변경 이력 관리

## 모범 사례

- 항상 수정 전 백업
- 변경 사항을 단계적으로 적용
- 수정 후 검증 수행
`

	// Log Analysis Skill
	logSkill := `---
name: log_analysis
description: 로그 파일 모니터링, 패턴 검색 및 분석
---

# Log Analysis Skill

이 스킬은 로그 파일을 효과적으로 분석하고 문제를 찾는 방법을 제공합니다.

## 주요 기능

### 1. 실시간 모니터링
- tail -f 스타일 모니터링
- 키워드 필터링
- 에러 패턴 감지

### 2. 로그 검색
- 정규식 패턴 검색
- 시간 범위 필터링
- 로그 레벨 필터링

### 3. 로그 분석
- 통계 생성
- 에러 요약
- 패턴 분석

## 사용 방법

1. 로그 파일 경로 확인
2. 적절한 액션 선택 (tail/search/filter/summarize)
3. 필요한 경우 패턴 지정
4. 결과 분석 및 문제 파악
`

	// Write skills to files
	skills := map[string]string{
		"storage_diagnosis": storageSkill,
		"file_operations":   fileSkill,
		"log_analysis":      logSkill,
	}

	for name, content := range skills {
		skillDir := filepath.Join(sm.skillsDir, name)
		if err := sm.fs.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory: %w", err)
		}

		skillFile := filepath.Join(skillDir, "SKILL.md")
		if err := sm.fs.WriteFile(skillFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write skill file: %w", err)
		}
	}

	return nil
}
