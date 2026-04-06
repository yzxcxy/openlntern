package dao

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"

	"openIntern/internal/database"
	"openIntern/internal/models"
)

type SkillFileEntry struct {
	Path       string
	Type       string
	Size       int64
	ModifiedAt string
}

type SkillStoreDAO struct{}

var SkillStore = new(SkillStoreDAO)

func (d *SkillStoreDAO) Configured() bool {
	return skillStoreReady()
}

func (d *SkillStoreDAO) RootURI(ctx context.Context) (string, error) {
	if !skillStoreReady() {
		return "", errors.New("skill storage root not configured")
	}
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	configuredRoot := strings.TrimSpace(database.Context.SkillsRoot())
	if configuredRoot != "" {
		// OpenViking scopes viking://agent/skills by tenant headers, so one configured root can remain user-isolated.
		return strings.TrimRight(configuredRoot, "/"), nil
	}
	return strings.TrimRight(UserSkillRootURI(userID), "/"), nil
}

func (d *SkillStoreDAO) BuildURI(ctx context.Context, skillPath string) (string, error) {
	root, err := d.RootURI(ctx)
	if err != nil {
		return "", err
	}
	skillPath = strings.Trim(skillPath, "/")
	if skillPath == "" {
		return root, nil
	}
	parts := strings.Split(skillPath, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	if len(cleaned) == 0 {
		return root, nil
	}
	return root + "/" + strings.Join(cleaned, "/"), nil
}

func (d *SkillStoreDAO) ListSkillNames(ctx context.Context) ([]string, error) {
	root, err := d.RootURI(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := listEntries(ctx, root, false)
	if err != nil {
		if isContextStoreNotFound(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}
		skillPath := relativePath(root, entry.Path)
		if skillPath == "" {
			skillPath = entry.Name
		}
		if skillPath == "" {
			continue
		}
		if strings.Contains(skillPath, "/") {
			skillPath = strings.Split(skillPath, "/")[0]
		}
		if skillPath == "" {
			continue
		}
		if _, ok := seen[skillPath]; ok {
			continue
		}
		seen[skillPath] = struct{}{}
		names = append(names, skillPath)
	}
	sort.Strings(names)
	return names, nil
}

func (d *SkillStoreDAO) ListSkillCatalog(ctx context.Context, keyword string, page int, pageSize int) ([]models.Skill, int64, error) {
	names, err := d.ListSkillNames(ctx)
	if err != nil {
		return nil, 0, err
	}
	skills := make([]models.Skill, 0, len(names))
	for _, name := range names {
		skills = append(skills, models.Skill{Name: name})
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		filtered := make([]models.Skill, 0, len(skills))
		for _, skill := range skills {
			if strings.Contains(strings.ToLower(skill.Name), strings.ToLower(keyword)) {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}
	total := int64(len(skills))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(skills) {
		return []models.Skill{}, total, nil
	}
	end := start + pageSize
	if end > len(skills) {
		end = len(skills)
	}
	return skills[start:end], total, nil
}

func (d *SkillStoreDAO) ListFiles(ctx context.Context, skillPath string, recursive bool) ([]ResourceEntry, error) {
	targetURI, err := d.BuildURI(ctx, skillPath)
	if err != nil {
		return nil, err
	}
	entries, err := listEntries(ctx, targetURI, recursive)
	if err != nil {
		if isContextStoreNotFound(err) {
			return []ResourceEntry{}, nil
		}
		return nil, err
	}
	return entries, nil
}

func (d *SkillStoreDAO) ListFilesInDirectory(ctx context.Context, skillName string, relPath string) ([]SkillFileEntry, error) {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return nil, errors.New("skill name is required")
	}
	rootURI, err := d.BuildURI(ctx, skillName)
	if err != nil {
		return nil, err
	}
	listURI := rootURI
	relPath = strings.Trim(strings.TrimSpace(relPath), "/")
	if relPath != "" {
		listURI, err = d.BuildURI(ctx, path.Join(skillName, relPath))
		if err != nil {
			return nil, err
		}
	}
	entries, err := listEntries(ctx, listURI, false)
	if err != nil {
		if isContextStoreNotFound(err) {
			return []SkillFileEntry{}, nil
		}
		return nil, err
	}
	result := make([]SkillFileEntry, 0, len(entries))
	for _, entry := range entries {
		rel := relativePath(rootURI, entry.Path)
		if rel == "" {
			rel = entry.Name
		}
		itemType := "file"
		if entry.IsDir {
			itemType = "dir"
		}
		mtime := ""
		if !entry.ModifiedAt.IsZero() {
			mtime = entry.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		result = append(result, SkillFileEntry{
			Path:       rel,
			Type:       itemType,
			Size:       entry.Size,
			ModifiedAt: mtime,
		})
	}
	return result, nil
}

func (d *SkillStoreDAO) ReadFile(ctx context.Context, skillPath string) (string, error) {
	targetURI, err := d.BuildURI(ctx, skillPath)
	if err != nil {
		return "", err
	}
	return readContent(ctx, targetURI, "/api/v1/content/read")
}

func (d *SkillStoreDAO) ReadSummary(ctx context.Context, skillName string) (string, error) {
	targetURI, err := d.BuildURI(ctx, skillName)
	if err != nil {
		return "", err
	}
	return readContent(ctx, targetURI, "/api/v1/content/abstract")
}

func (d *SkillStoreDAO) Import(ctx context.Context, rootDir string, skillName string) (*ImportResult, error) {
	_ = skillName
	// Skill archives must use the dedicated /api/v1/skills flow; resource imports only accept viking://resources/* targets.
	return importSkill(ctx, rootDir)
}

func (d *SkillStoreDAO) Delete(ctx context.Context, name string) error {
	targetURI, err := d.BuildURI(ctx, name)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(targetURI, "/") {
		targetURI += "/"
	}
	return deletePath(ctx, targetURI, true)
}
