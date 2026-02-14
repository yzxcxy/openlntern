package services

import (
	"context"
	"openIntern/internal/agui"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)


func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error{
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}

	s := agui.NewSenderWithThreadID(ctx, w, threadID)

	//msgID := "msg_" + time.Now().Format("150405")

	// Start Run (uses injected IDs)
	if err := s.Start(); err != nil {
		return err
	}

	// // 测试
	// a2uiResp := a2ui.A2UIResponse{
	// 	MsgID: msgID,
	// 	SurfaceID: "default",
	// 	UIJSON: collectFormUIJSON,
	// }

	// // 发送A2UI响应
	// if err := a2ui.SendA2UIResponse(s,a2uiResp); err != nil {
	// 	return err
	// }

	// End Run (uses injected IDs)
	if err := s.Finish(); err != nil {
		return err
	}

	return nil
}