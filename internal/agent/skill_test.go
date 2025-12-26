package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mainbong/storage_doctor/internal/filesystem"
)

func TestNewSkillManager(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewSkillManager() returned nil")
	}

	if manager.skillsDir != skillsDir {
		t.Errorf("Expected skillsDir '%s', got '%s'", skillsDir, manager.skillsDir)
	}
}

func TestLoadSkills_DefaultSkills(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	skills := manager.GetSkills()
	if len(skills) == 0 {
		t.Error("Expected default skills to be loaded, got none")
	}

	// Verify default skills were created
	expectedSkills := []string{"storage_diagnosis", "file_operations", "log_analysis"}
	skillNames := make(map[string]bool)
	for _, skill := range skills {
		skillNames[skill.Name] = true
	}

	for _, expected := range expectedSkills {
		if !skillNames[expected] {
			t.Errorf("Expected skill '%s' to be loaded, but it wasn't", expected)
		}
	}
}

func TestLoadSkills_ExistingSkills(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	// Create existing skill
	skillDir := filepath.Join(skillsDir, "custom_skill")
	skillFile := filepath.Join(skillDir, "SKILL.md")
	skillContent := `---
name: custom_skill
description: Custom skill for testing
---
# Custom Skill
This is a custom skill.
`

	mockFS.AddDir(skillsDir, 0755)
	mockFS.AddDir(skillDir, 0755)
	mockFS.AddFile(skillFile, []byte(skillContent), 0644)

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	skills := manager.GetSkills()
	found := false
	for _, skill := range skills {
		if skill.Name == "custom_skill" {
			found = true
			if skill.Description != "Custom skill for testing" {
				t.Errorf("Expected description 'Custom skill for testing', got '%s'", skill.Description)
			}
			if !strings.Contains(skill.Content, "Custom Skill") {
				t.Error("Expected skill content to contain 'Custom Skill'")
			}
			break
		}
	}

	if !found {
		t.Error("Expected custom_skill to be loaded, but it wasn't")
	}
}

func TestLoadSkill_InvalidFormat(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	// Create skill with invalid format (no YAML frontmatter)
	skillDir := filepath.Join(skillsDir, "invalid_skill")
	skillFile := filepath.Join(skillDir, "SKILL.md")
	invalidContent := "No YAML frontmatter here"

	mockFS.AddDir(skillsDir, 0755)
	mockFS.AddDir(skillDir, 0755)
	mockFS.AddFile(skillFile, []byte(invalidContent), 0644)

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err == nil {
		t.Error("Expected error for invalid skill format, got nil")
	}

	if manager != nil {
		skills := manager.GetSkills()
		// Should not have invalid skill
		for _, skill := range skills {
			if skill.Name == "invalid_skill" {
				t.Error("Expected invalid skill to not be loaded")
			}
		}
	}
}

func TestGetSkillMetadata(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	metadata := manager.GetSkillMetadata()
	if metadata == "" {
		t.Error("Expected skill metadata, got empty string")
	}

	// Verify metadata contains skill information
	if !strings.Contains(metadata, "사용 가능한 스킬") {
		t.Error("Expected metadata to contain '사용 가능한 스킬'")
	}
}

func TestGetSkillMetadata_NoSkills(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	// Create empty skills directory
	mockFS.AddDir(skillsDir, 0755)

	manager := &SkillManager{
		skillsDir: skillsDir,
		skills:    make([]Skill, 0),
		fs:        mockFS,
	}

	metadata := manager.GetSkillMetadata()
	if metadata != "" {
		t.Errorf("Expected empty metadata for no skills, got '%s'", metadata)
	}
}

func TestActivateSkill(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	// Activate a default skill
	content, err := manager.ActivateSkill("storage_diagnosis")
	if err != nil {
		t.Fatalf("ActivateSkill() failed: %v", err)
	}

	if content == "" {
		t.Error("Expected skill content, got empty string")
	}

	if !strings.Contains(content, "Storage Diagnosis Skill") {
		t.Error("Expected skill content to contain 'Storage Diagnosis Skill'")
	}
}

func TestActivateSkill_NotFound(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager, err := NewSkillManagerWithFS(skillsDir, mockFS)
	if err != nil {
		t.Fatalf("NewSkillManager() failed: %v", err)
	}

	_, err = manager.ActivateSkill("nonexistent_skill")
	if err == nil {
		t.Error("Expected error for nonexistent skill, got nil")
	}

	if !strings.Contains(err.Error(), "skill not found") {
		t.Errorf("Expected error message to contain 'skill not found', got '%v'", err)
	}
}

func TestLoadSkill_ReadError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	skillDir := filepath.Join(skillsDir, "test_skill")
	skillFile := filepath.Join(skillDir, "SKILL.md")

	mockFS.AddDir(skillsDir, 0755)
	mockFS.AddDir(skillDir, 0755)
	mockFS.AddFile(skillFile, []byte("test"), 0644)
	mockFS.SetReadError(skillFile, os.ErrPermission)

	manager := &SkillManager{
		skillsDir: skillsDir,
		skills:    make([]Skill, 0),
		fs:        mockFS,
	}

	err := manager.LoadSkills()
	if err == nil {
		t.Error("Expected error for read failure, got nil")
	}
}

func TestCreateDefaultSkills(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	manager := &SkillManager{
		skillsDir: skillsDir,
		skills:    make([]Skill, 0),
		fs:        mockFS,
	}

	err := manager.createDefaultSkills()
	if err != nil {
		t.Fatalf("createDefaultSkills() failed: %v", err)
	}

	// Verify default skills were created
	expectedSkills := []string{"storage_diagnosis", "file_operations", "log_analysis"}
	for _, skillName := range expectedSkills {
		skillDir := filepath.Join(skillsDir, skillName)
		skillFile := filepath.Join(skillDir, "SKILL.md")

		content := mockFS.GetFile(skillFile)
		if len(content) == 0 {
			t.Errorf("Expected skill file '%s' to be created, but it's empty", skillFile)
		}
	}
}

func TestCreateDefaultSkills_WriteError(t *testing.T) {
	mockFS := filesystem.NewMockFileSystem()
	skillsDir := "/test/skills"

	skillDir := filepath.Join(skillsDir, "storage_diagnosis")
	skillFile := filepath.Join(skillDir, "SKILL.md")

	mockFS.SetWriteError(skillFile, os.ErrPermission)

	manager := &SkillManager{
		skillsDir: skillsDir,
		skills:    make([]Skill, 0),
		fs:        mockFS,
	}

	err := manager.createDefaultSkills()
	if err == nil {
		t.Error("Expected error for write failure, got nil")
	}
}





