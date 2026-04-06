package dao

import "testing"

// TestBuildAddResourcePayloadFromTempFile verifies resource imports no longer send local paths.
func TestBuildAddResourcePayloadFromTempFile(t *testing.T) {
	payload := buildAddResourcePayload("tmp_123", "viking://resources/demo/", false, 0)
	if payload["temp_file_id"] != "tmp_123" {
		t.Fatalf("unexpected temp file id: %#v", payload["temp_file_id"])
	}
	if payload["target"] != "viking://resources/demo/" {
		t.Fatalf("unexpected target: %#v", payload["target"])
	}
	if payload["wait"] != false {
		t.Fatalf("unexpected wait flag: %#v", payload["wait"])
	}
	if _, exists := payload["path"]; exists {
		t.Fatalf("path should not be sent when temp_file_id is used")
	}
}

// TestBuildAddResourcePayloadIncludesTimeout verifies optional timeout is preserved.
func TestBuildAddResourcePayloadIncludesTimeout(t *testing.T) {
	payload := buildAddResourcePayload("tmp_123", "viking://resources/demo/", true, 30)
	if payload["timeout"] != 30.0 {
		t.Fatalf("unexpected timeout: %#v", payload["timeout"])
	}
}

// TestBuildAddSkillPayloadFromTempFile verifies skill imports no longer send backend-local directories.
func TestBuildAddSkillPayloadFromTempFile(t *testing.T) {
	payload := buildAddSkillPayload("tmp_skill_1", false, 0)
	if payload["temp_file_id"] != "tmp_skill_1" {
		t.Fatalf("unexpected temp file id: %#v", payload["temp_file_id"])
	}
	if payload["wait"] != false {
		t.Fatalf("unexpected wait flag: %#v", payload["wait"])
	}
	if _, exists := payload["data"]; exists {
		t.Fatalf("data should not be sent when temp_file_id is used")
	}
}

// TestBuildAddSkillPayloadIncludesTimeout verifies skill imports retain timeout when requested.
func TestBuildAddSkillPayloadIncludesTimeout(t *testing.T) {
	payload := buildAddSkillPayload("tmp_skill_1", true, 15)
	if payload["timeout"] != 15.0 {
		t.Fatalf("unexpected timeout: %#v", payload["timeout"])
	}
}
