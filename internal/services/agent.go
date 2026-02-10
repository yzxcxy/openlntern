package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"agent_backend/internal/agui"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

const menuUIJSON = `[ 
   { 
     "id": "root", 
     "component": { 
       "Column": { 
         "children": { 
           "explicitList": [ 
             "dishList" 
           ] 
         }, 
         "alignment": "stretch", 
         "distribution": "start" 
       } 
     } 
   }, 
   { 
     "id": "dishList", 
     "component": { 
       "List": { 
         "children": { 
           "template": { 
             "componentId": "dishItem", 
             "dataBinding": "/dishes" 
           } 
         }, 
         "direction": "vertical", 
         "alignment": "stretch" 
       } 
     } 
   }, 
   { 
     "id": "dishItem", 
     "component": { 
       "Card": { 
         "child": "dishContent" 
       } 
     } 
   }, 
   { 
     "id": "dishContent", 
     "component": { 
       "Row": { 
         "children": { 
           "explicitList": [ 
             "dishImage", 
             "dishDetails" 
           ] 
         }, 
         "alignment": "center", 
         "distribution": "start" 
       } 
     } 
   }, 
   { 
     "id": "dishImage", 
     "component": { 
       "Image": { 
         "url": { 
           "path": "/imageUrl" 
         }, 
         "fit": "cover", 
         "usageHint": "mediumFeature" 
       } 
     } 
   }, 
   { 
     "id": "dishDetails", 
     "component": { 
       "Column": { 
         "children": { 
           "explicitList": [ 
             "dishName", 
             "dishDescription" 
           ] 
         }, 
         "alignment": "start", 
         "distribution": "start" 
       } 
     } 
   }, 
   { 
     "id": "dishName", 
     "component": { 
       "Text": { 
         "text": { 
           "path": "/name" 
         }, 
         "usageHint": "h3" 
       } 
     } 
   }, 
   { 
     "id": "dishDescription", 
     "component": { 
       "Text": { 
         "text": { 
           "path": "/description" 
         }, 
         "usageHint": "body" 
       } 
     } 
   } 
 ]`

const menuDataJSON = `{ 
   "dishes": [ 
     { 
       "name": "宫保鸡丁", 
       "description": "经典川菜，鲜香微辣", 
       "imageUrl": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s" 
     }, 
     { 
       "name": "鱼香肉丝", 
       "description": "酸甜鲜辣，开胃下饭", 
       "imageUrl": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s" 
     }, 
     { 
       "name": "麻婆豆腐", 
       "description": "麻辣鲜香，豆腐嫩滑", 
       "imageUrl": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s" 
     },
     {
       "name": "拍黄瓜",
       "description": "清爽解腻，脆嫩爽口",
       "imageUrl": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s"
     }
   ] 
 }`

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
          "literalString": "欢迎你！"
        },
        "usageHint": "h2"
      }
    }
  }
]
 `


func RunAgent(ctx context.Context, w io.Writer, input *types.RunAgentInput) error {
	threadID := input.ThreadID

	if threadID == "" {
		threadID = events.GenerateThreadID()
	}

	s := agui.NewSenderWithThreadID(ctx, w, threadID)

	msgID := "msg_" + time.Now().Format("150405")

	// 3. Stream Content
	userMsg := ""
	if len(input.Messages) > 0 {
		lastMsg := input.Messages[len(input.Messages)-1]
		if lastMsg.Role == types.RoleUser {
			switch v := lastMsg.Content.(type) {
			case string:
				userMsg = v
			default:
				userMsg = fmt.Sprintf("%v", v)
			}
		}
	}

	if strings.Contains(userMsg, "欢迎") {
		// Start Run (uses injected IDs)
		if err := s.Start(); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)

		// Send Text Message
		// if err := s.StartMessage(msgID, "assistant"); err != nil {
		// 	return err
		// }
		// if err := s.SendContent(msgID, "好的，这是我们的菜单："); err != nil {
		// 	return err
		// }

		// Send A2UI with surfaceUpdate, dataModelUpdate, and beginRendering
		var ui []interface{}
		if err := json.Unmarshal([]byte(WelcomeText), &ui); err != nil {
			return err
		}

		// var data map[string]interface{}
		// if err := json.Unmarshal([]byte(menuDataJSON), &data); err != nil {
		// 	return err
		// }

		// Send initial empty snapshot to create the message/activity
		if err := s.SendA2UI(msgID, map[string]interface{}{
			"operations": []interface{}{},
		}); err != nil {
			return err
		}


		// Send surfaceUpdate
		if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/operations/-",
				Value: map[string]interface{}{
					"surfaceUpdate": map[string]interface{}{
						"surfaceId":  "default",
						"components": ui,
					},
				},
			},
		}); err != nil {
			return err
		}

    // Send beginRendering
		if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/operations/-",
				Value: map[string]interface{}{
					"beginRendering": map[string]interface{}{
						"surfaceId": "default",
						"root":      "root",
					},
				},
			},
		}); err != nil {
			return err
		}

		// Send dataModelUpdate
		// if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
		// 	{
		// 		Op:   "add",
		// 		Path: "/operations/-",
		// 		Value: map[string]interface{}{
		// 			"dataModelUpdate": map[string]interface{}{
		// 				"data": data,
		// 			},
		// 		},
		// 	},
		// }); err != nil {
		// 	return err
		// }

		// if err := s.EndMessage(msgID); err != nil {
		// 	return err
		// }

		// End Run (uses injected IDs)
		if err := s.Finish(); err != nil {
			return err
		}

		return nil
	}

	// 1. Start Run (uses injected IDs)
	if err := s.Start(); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	// 2. Start Message
	if err := s.StartMessage(msgID, "assistant"); err != nil {
		return err
	}

	message := fmt.Sprintf("Echo: %s\n\nI am sending this via AG-UI protocol using the Eino-compatible wrapper with injected context.", userMsg)

	// Simulate token streaming
	for _, char := range message {
		if err := s.SendContent(msgID, string(char)); err != nil {
			return err
		}
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	// 4. End Message
	if err := s.EndMessage(msgID); err != nil {
		return err
	}

	// 5. End Run (uses injected IDs)
	if err := s.Finish(); err != nil {
		return err
	}

	return nil
}
