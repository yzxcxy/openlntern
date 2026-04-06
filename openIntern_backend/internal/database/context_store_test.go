package database

import "testing"

// TestExtractPayloadSummaryPrefersTargetURI ensures target_uri is surfaced in request logs.
func TestExtractPayloadSummaryPrefersTargetURI(t *testing.T) {
	target, pathValue := extractPayloadSummary(map[string]any{
		"query":      "go1.26",
		"target_uri": "viking://resources/golang/",
		"path":       "/tmp/sample.md",
	})
	if target != "viking://resources/golang/" {
		t.Fatalf("unexpected target: %q", target)
	}
	if pathValue != "/tmp/sample.md" {
		t.Fatalf("unexpected path: %q", pathValue)
	}
}

// TestExtractPayloadSummaryFallsBackToTarget ensures legacy target key still works.
func TestExtractPayloadSummaryFallsBackToTarget(t *testing.T) {
	target, pathValue := extractPayloadSummary(map[string]any{
		"target": "viking://resources/legacy/",
	})
	if target != "viking://resources/legacy/" {
		t.Fatalf("unexpected target: %q", target)
	}
	if pathValue != "" {
		t.Fatalf("unexpected path: %q", pathValue)
	}
}

// TestExtractPayloadSummarySupportsParent ensures directory-style imports surface the parent URI in request logs.
func TestExtractPayloadSummarySupportsParent(t *testing.T) {
	target, pathValue := extractPayloadSummary(map[string]any{
		"parent": "viking://resources/demo/",
	})
	if target != "viking://resources/demo/" {
		t.Fatalf("unexpected target: %q", target)
	}
	if pathValue != "" {
		t.Fatalf("unexpected path: %q", pathValue)
	}
}

// TestExtractPayloadSummarySupportsTempFileID ensures temp_file_id imports surface a useful log summary.
func TestExtractPayloadSummarySupportsTempFileID(t *testing.T) {
	target, pathValue := extractPayloadSummary(map[string]any{
		"parent":       "viking://resources/demo/",
		"temp_file_id": "upload_123.zip",
	})
	if target != "viking://resources/demo/" {
		t.Fatalf("unexpected target: %q", target)
	}
	if pathValue != "temp_file_id:upload_123.zip" {
		t.Fatalf("unexpected path: %q", pathValue)
	}
}
