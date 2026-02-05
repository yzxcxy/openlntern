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
       "imageUrl": "https://example.com/images/gongbao.jpg" 
     }, 
     { 
       "name": "鱼香肉丝", 
       "description": "酸甜鲜辣，开胃下饭", 
       "imageUrl": "https://example.com/images/yuxiang.jpg" 
     }, 
     { 
       "name": "麻婆豆腐", 
       "description": "麻辣鲜香，豆腐嫩滑", 
       "imageUrl": "https://example.com/images/mapo.jpg" 
     } 
   ] 
 }`

const orderUIJSON = `[ 
   { 
     "id": "root", 
     "component": { 
       "Column": { 
         "children": { 
           "explicitList": [ 
             "dishField", 
             "quantityField", 
             "submitBtn" 
           ] 
         }, 
         "distribution": "start", 
         "alignment": "stretch" 
       } 
     } 
   }, 
   { 
     "id": "dishField", 
     "component": { 
       "TextField": { 
         "label": { 
           "literalString": "菜品" 
         }, 
         "text": { 
           "path": "/form/dish" 
         }, 
         "textFieldType": "shortText" 
       } 
     } 
   }, 
   { 
     "id": "quantityField", 
     "component": { 
       "TextField": { 
         "label": { 
           "literalString": "数量" 
         }, 
         "text": { 
           "path": "/form/quantity" 
         }, 
         "textFieldType": "number" 
       } 
     } 
   }, 
   { 
     "id": "submitBtnText", 
     "component": { 
       "Text": { 
         "text": { 
           "literalString": "提交" 
         } 
       } 
     } 
   }, 
   { 
     "id": "submitBtn", 
     "component": { 
       "Button": { 
         "child": "submitBtnText", 
         "primary": true, 
         "action": { 
           "name": "submitOrder", 
           "context": [ 
             { 
               "key": "dish", 
               "value": { 
                 "path": "/form/dish" 
               } 
             }, 
             { 
               "key": "quantity", 
               "value": { 
                 "path": "/form/quantity" 
               } 
             } 
           ] 
         } 
       } 
     } 
   } 
 ]`

const orderDataJSON = `{
  "form": {
    "dish": "",
    "quantity": "1"
  }
}`

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

	if strings.Contains(userMsg, "查看菜单") {
		// Start Run (uses injected IDs)
		if err := s.Start(); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)

		// Send A2UI with surfaceUpdate, dataModelUpdate, and beginRendering
		var ui []interface{}
		if err := json.Unmarshal([]byte(menuUIJSON), &ui); err != nil {
			return err
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(menuDataJSON), &data); err != nil {
			return err
		}

		// Send initial empty snapshot to create the message/activity
		if err := s.SendA2UI(msgID, map[string]interface{}{}); err != nil {
			return err
		}

		// Send surfaceUpdate
		if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/surfaceUpdate",
				Value: map[string]interface{}{
					"components": ui,
				},
			},
		}); err != nil {
			return err
		}

		// Send dataModelUpdate
		if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/dataModelUpdate",
				Value: map[string]interface{}{
					"data": data,
				},
			},
		}); err != nil {
			return err
		}

		// Send beginRendering
		if err := s.UpdateA2UI(msgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/beginRendering",
				Value: map[string]interface{}{
					"root": "root",
				},
			},
		}); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)

		// End Run (uses injected IDs)
		if err := s.Finish(); err != nil {
			return err
		}

		return nil
	} else if strings.Contains(userMsg, "我要点单") {
		// Start Run (uses injected IDs)
		if err := s.Start(); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)

		// Send A2UI with surfaceUpdate, dataModelUpdate, and beginRendering
		var ui []interface{}
		if err := json.Unmarshal([]byte(orderUIJSON), &ui); err != nil {
			return err
		}

		a2uiPayload := map[string]interface{}{
			"surfaceUpdate": map[string]interface{}{
				"components": ui,
			},
			"beginRendering": map[string]interface{}{
				"root": "root",
			},
		}

		if err := s.SendA2UI(msgID, a2uiPayload); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)

		//  End Run (uses injected IDs)
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
