package agui

import "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

type A2UISender interface {
	SendA2UI(messageID string, content any) error
	UpdateA2UI(messageID string, patch []events.JSONPatchOperation) error
}
