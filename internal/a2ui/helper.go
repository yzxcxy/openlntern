package a2ui

import (
	"encoding/json"
	"fmt"
	"openIntern/internal/agui"
	"sort"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

// A2UIResponse 定义了发送 A2UI 响应所需的内容
// 设计为便于 AI 生成和调用的结构
type A2UIResponse struct {
	MsgID     string // 消息 ID
	SurfaceID string // 必填: Surface ID (例如 "default")
	UIJSON    string // 必填: UI 描述的 JSON 字符串
	DataJSON  string // 可选: 数据模型更新内容的 JSON 字符串 (包含 contents 等字段)
}

// SendA2UIResponse 统一封装 A2UI 消息的发送流程
func SendA2UIResponse(s *agui.Sender, resp A2UIResponse) error {
	// 默认 SurfaceID 处理
	if resp.SurfaceID == "" {
		resp.SurfaceID = "default"
	}

	// 1. 解析 UI JSON
	var ui []interface{}
	if err := json.Unmarshal([]byte(resp.UIJSON), &ui); err != nil {
		return fmt.Errorf("unmarshal UI JSON failed: %w", err)
	}

	// 2. 解析 Data JSON (如果有) 并转换为 A2UI 数据模型
	var dataModelUpdate map[string]interface{}
	if resp.DataJSON != "" {
		var rawData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.DataJSON), &rawData); err != nil {
			return fmt.Errorf("unmarshal Data JSON failed: %w", err)
		}

		contents := make([]interface{}, 0, len(rawData))

		// 为了保证确定性（主要是为了测试和调试），我们对 map 的 key 进行排序
		keys := make([]string, 0, len(rawData))
		for k := range rawData {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			contents = append(contents, convertToA2UIEntry(k, rawData[k]))
		}

		dataModelUpdate = map[string]interface{}{
			"surfaceId": resp.SurfaceID,
			"contents":  contents,
		}
	}

	// 3. 发送初始空快照
	if err := s.SendA2UI(resp.MsgID, map[string]interface{}{
		"operations": []interface{}{},
	}); err != nil {
		return fmt.Errorf("send initial snapshot failed: %w", err)
	}

	// 4. 发送 Surface Update
	if err := s.UpdateA2UI(resp.MsgID, []events.JSONPatchOperation{
		{
			Op:   "add",
			Path: "/operations/-",
			Value: map[string]interface{}{
				"surfaceUpdate": map[string]interface{}{
					"surfaceId":  resp.SurfaceID,
					"components": ui,
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("send surface update failed: %w", err)
	}

	// 5. 发送 Data Model Update (如果解析出了数据)
	if dataModelUpdate != nil {
		if err := s.UpdateA2UI(resp.MsgID, []events.JSONPatchOperation{
			{
				Op:   "add",
				Path: "/operations/-",
				Value: map[string]interface{}{
					"dataModelUpdate": dataModelUpdate,
				},
			},
		}); err != nil {
			return fmt.Errorf("send data update failed: %w", err)
		}
	}

	// 6. 发送 Begin Rendering
	if err := s.UpdateA2UI(resp.MsgID, []events.JSONPatchOperation{
		{
			Op:   "add",
			Path: "/operations/-",
			Value: map[string]interface{}{
				"beginRendering": map[string]interface{}{
					"surfaceId": resp.SurfaceID,
					"root":      "root", // 默认根节点 ID 为 root
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("send begin rendering failed: %w", err)
	}

	return nil
}

// convertToA2UIEntry 递归将任意值转换为 A2UI 的数据模型条目
func convertToA2UIEntry(key string, value interface{}) map[string]interface{} {
	entry := map[string]interface{}{
		"key": key,
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Map -> ValueMap
		children := make([]interface{}, 0, len(v))

		// 排序 keys 保证确定性
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, subK := range keys {
			children = append(children, convertToA2UIEntry(subK, v[subK]))
		}
		entry["valueMap"] = children

	case []interface{}:
		// Array/Slice -> ValueMap (使用索引作为 key)
		children := make([]interface{}, 0, len(v))
		for i, item := range v {
			// 使用字符串化的索引作为 key，例如 "0", "1"
			// 注意：如果业务需要特定的 key 前缀 (如 "dish0")，需要在原始数据中预处理为 map，
			// 或者在此处使用通用索引，并在 UI 模板中使用通用索引访问。
			// 鉴于通用性，这里只使用纯数字索引字符串。
			children = append(children, convertToA2UIEntry(fmt.Sprintf("%d", i), item))
		}
		entry["valueMap"] = children

	default:
		// 基本类型处理
		switch val := v.(type) {
		case bool:
			entry["valueBool"] = val
		case string:
			entry["valueString"] = val
		case float64:
			// JSON 数字默认解析为 float64
			// 如果是整数，去掉小数点
			if val == float64(int64(val)) {
				entry["valueString"] = fmt.Sprintf("%d", int64(val))
			} else {
				entry["valueString"] = fmt.Sprintf("%v", val)
			}
		case int, int8, int16, int32, int64:
			entry["valueString"] = fmt.Sprintf("%d", val)
		case uint, uint8, uint16, uint32, uint64:
			entry["valueString"] = fmt.Sprintf("%d", val)
		default:
			// 其他类型兜底转字符串
			entry["valueString"] = fmt.Sprintf("%v", v)
		}
	}

	return entry
}
