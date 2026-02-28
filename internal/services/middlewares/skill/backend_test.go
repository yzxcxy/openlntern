package skillmiddleware_test

import (
	"context"
	"strings"
	"testing"

	"openIntern/internal/config"
	"openIntern/internal/database"
	"openIntern/internal/models"
	"openIntern/internal/services"
	skillmiddleware "openIntern/internal/services/middlewares/skill"
)

type realOpenVikingClient struct{}

func (realOpenVikingClient) SkillsRoot() string {
	return services.OpenViking.SkillsRoot()
}

func (realOpenVikingClient) List(ctx context.Context, uri string, recursive bool) ([]map[string]any, error) {
	return services.OpenVikingList(ctx, uri, recursive)
}

func (realOpenVikingClient) ReadAbstract(ctx context.Context, uri string) (string, error) {
	return services.OpenVikingReadAbstract(ctx, uri)
}

func (realOpenVikingClient) ReadContent(ctx context.Context, uri string) (string, error) {
	return services.OpenVikingReadContent(ctx, uri)
}

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
	cfg := config.LoadConfig("config.yaml")
	if err := database.Init(cfg.MySQL.DSN); err != nil {
		t.Fatalf("init database failed: %v", err)
	}
	services.InitOpenViking(cfg.Tools.OpenViking)
	if !services.OpenViking.Configured() {
		t.Fatalf("openviking not configured")
	}
}

func pickFirstSkillName(t *testing.T, ctx context.Context) string {
	t.Helper()
	root := services.OpenViking.SkillsRoot()
	entries, err := services.OpenVikingList(ctx, root, false)
	if err != nil {
		t.Fatalf("list openviking skills failed: %v", err)
	}
	for _, entry := range entries {
		if !services.OpenVikingEntryIsDir(entry) {
			continue
		}
		entryPath := services.OpenVikingEntryString(entry, "path", "uri")
		entryName := services.OpenVikingEntryString(entry, "name")
		skillPath := services.OpenVikingRelativePath(root, entryPath)
		if skillPath == "" {
			skillPath = entryName
		}
		if skillPath == "" {
			continue
		}
		if strings.Contains(skillPath, "/") {
			skillPath = strings.Split(skillPath, "/")[0]
		}
		if skillPath != "" {
			return skillPath
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

func TestOpenVikingBackend_List(t *testing.T) {
	initOpenViking(t)
	client := realOpenVikingClient{}
	store := realFrontmatterStore{}

	backend, err := skillmiddleware.NewOpenVikingBackend(client, store)
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

func TestOpenVikingBackend_Get(t *testing.T) {
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

	client := realOpenVikingClient{}
	store := realFrontmatterStore{}

	backend, err := skillmiddleware.NewOpenVikingBackend(client, store)
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
	root := services.OpenViking.SkillsRoot()
	if skill.BaseDirectory != buildSkillURI(root, skillName, "") {
		t.Fatalf("unexpected base directory: %s", skill.BaseDirectory)
	}
}
