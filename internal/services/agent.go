package services

import (
	"context"
	"io"
	"log"
	"openIntern/internal/a2ui"
	"openIntern/internal/agui"
	"time"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)


const WelcomeText = `[
  {
    "id": "root",
    "component": {
      "Card": {
        "child": "content"
      }
    }
  },
  {
    "id": "content",
    "component": {
      "Column": {
        "children": {
          "explicitList": [
            "welcomeText"
          ]
        },
        "distribution": "center",
        "alignment": "center"
      }
    }
  },
  {
    "id": "welcomeText",
    "component": {
      "Text": {
        "text": {
          "literalString": "我是小智，欢迎你！"
        },
        "usageHint": "h2"
      }
    }
  }
]
 `


func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error{
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}

	s := agui.NewSenderWithThreadID(ctx, w, threadID)

	msgID := "msg_" + time.Now().Format("150405")

	// Start Run (uses injected IDs)
	if err := s.Start(); err != nil {
		log.Printf("Start Run failed: %v", err)
		return err
	}

	// 测试
	a2uiResp := a2ui.A2UIResponse{
		MsgID: msgID,
		SurfaceID: "default",
		UIJSON: WelcomeText	,
	}

	// 发送A2UI响应
	if err := a2ui.SendA2UIResponse(s,a2uiResp); err != nil {
		return err
	}

	// End Run (uses injected IDs)
	if err := s.Finish(); err != nil {
		log.Printf("Finish Run failed: %v", err)
		return err
	}

	log.Printf("Run Agent %s %s success", input.ThreadID, input.RunID)
	return nil
}