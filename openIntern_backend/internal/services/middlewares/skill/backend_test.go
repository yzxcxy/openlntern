package skillmiddleware_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"openIntern/internal/config"
	"openIntern/internal/dao"
	"openIntern/internal/database"
	"openIntern/internal/models"
	"openIntern/internal/services"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
)

type realFrontmatterStore struct{}

func (realFrontmatterStore) ListByNames(names []string) ([]skillmiddleware.SkillFrontmatterRecord, error) {
	items, err := services.SkillFrontmatter.ListByNames(names)
	if err != nil {
		return nil, err
	}
	records := make([]skillmiddleware.SkillFrontmatterRecord, 0, len(items))
	for _, item := range items {
		records = append(records, skillmiddleware.SkillFrontmatterRecord{
			SkillName: item.SkillName,
			Raw:       item.Raw,
		})
	}
	return records, nil
}

func (realFrontmatterStore) GetByName(name string) (*skillmiddleware.SkillFrontmatterRecord, error) {
	item, err := services.SkillFrontmatter.GetByName(name)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &skillmiddleware.SkillFrontmatterRecord{
		SkillName: item.SkillName,
		Raw:       item.Raw,
	}, nil
}

func initOpenViking(t *testing.T) {
	t.Helper()
	cfg := config.LoadConfig(testConfigPath(t))
	if err := database.Init(cfg.MySQL.DSN); err != nil {
		t.Fatalf("init database failed: %v", err)
	}
	database.InitContextStore(cfg.Tools.OpenViking)
	if !dao.SkillStore.Configured() {
		t.Fatalf("openviking not configured")
	}
}

func testConfigPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file path failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "config.yaml"))
}

func pickFirstSkillName(t *testing.T, ctx context.Context) string {
	t.Helper()
	names, err := dao.SkillStore.ListSkillNames(ctx)
	if err != nil {
		t.Fatalf("list openviking skills failed: %v", err)
	}
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			return name
		}
	}
	t.Fatalf("no skill found in openviking")
	return ""
}

func buildSkillURI(root string, skill string, suffix string) string {
	base := strings.TrimRight(root, "/") + "/" + strings.TrimLeft(skill, "/")
	if suffix == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(suffix, "/")
}

func TestRemoteBackend_List(t *testing.T) {
	initOpenViking(t)
	store := realFrontmatterStore{}

	backend, err := skillmiddleware.NewRemoteBackend(dao.SkillStore, store)
	if err != nil {
		t.Fatalf("new backend failed: %v", err)
	}

	frontmatters, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("list skills failed: %v", err)
	}
	if len(frontmatters) == 0 {
		t.Fatalf("empty frontmatter result")
	}
	t.Logf("res: %v", frontmatters)
}

func TestRemoteBackend_Get(t *testing.T) {
	initOpenViking(t)
	ctx := context.Background()
	skillName := pickFirstSkillName(t, ctx)
	raw := "name: " + skillName + "\n" + "description: Integration description\n"
	if err := services.SkillFrontmatter.CreateOrReplaceByName(&models.SkillFrontmatter{
		SkillName: skillName,
		Raw:       raw,
	}); err != nil {
		t.Fatalf("create frontmatter failed: %v", err)
	}
	t.Cleanup(func() {
		_ = services.SkillFrontmatter.DeleteByName(skillName)
	})

	store := realFrontmatterStore{}

	backend, err := skillmiddleware.NewRemoteBackend(dao.SkillStore, store)
	if err != nil {
		t.Fatalf("new backend failed: %v", err)
	}

	skill, err := backend.Get(ctx, skillName)
	if err != nil {
		t.Fatalf("get skill failed: %v", err)
	}
	if skill.FrontMatter.Name != skillName {
		t.Fatalf("unexpected skill name: %s", skill.FrontMatter.Name)
	}
	if skill.FrontMatter.Description != "Integration description" {
		t.Fatalf("unexpected description: %s", skill.FrontMatter.Description)
	}
	if skill.Content == "" {
		t.Fatalf("unexpected content: %s", skill.Content)
	}
	root := dao.SkillStore.RootURI()
	if skill.BaseDirectory != buildSkillURI(root, skillName, "") {
		t.Fatalf("unexpected base directory: %s", skill.BaseDirectory)
	}
	
	t.Log(skill)
}
