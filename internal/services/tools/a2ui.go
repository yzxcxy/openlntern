package tools

import (
	"context"
	"encoding/json"
	"errors"
	"openIntern/internal/agui"
	"openIntern/internal/a2ui"
	"openIntern/internal/models"
	"strings"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// A2UIServiceInterface 供 list/get 使用的 A2UI 查询接口，由 services 注入到 context
type A2UIServiceInterface interface {
	ListA2UIs(page, pageSize int, keyword string) ([]models.A2UI, int64, error)
	GetA2UIByID(id string) (*models.A2UI, error)
}

var (
	errA2UIIDRequired         = errors.New("a2ui_id is required")
	errA2UINotFoundOrNoAccess = errors.New("a2ui not found or no access")
	errA2UISenderNotAvailable = errors.New("a2ui sender not available in this context")
	errMsgIDRequired          = errors.New("msg_id is required")
	errA2UIServiceNotInCtx    = errors.New("a2ui service not available in context")
)

// Context key 类型，用于从 context 中获取当前用户 ID 和 A2UI Sender
type contextKey string

const (
	ContextKeyUserID       contextKey = "openintern_user_id"
	ContextKeyA2UISender   contextKey = "openintern_a2ui_sender"
	ContextKeyA2UIService  contextKey = "openintern_a2ui_service"
)

// ListA2UIsInput 列出可访问 A2UI 的入参（无参数，一次返回全部）
type ListA2UIsInput struct{}

// GetA2UIInput 获取单个 A2UI 详情的入参
type GetA2UIInput struct {
	A2UIID string `json:"a2ui_id" jsonschema_description:"A2UI 的业务 ID"`
}

// SendA2UIInput 渲染数据并发送 A2UI 事件的入参
type SendA2UIInput struct {
	MsgID     string `json:"msg_id" jsonschema_description:"消息 ID，用于关联 A2UI 活动"`
	SurfaceID string `json:"surface_id" jsonschema_description:"Surface ID，如 default，可选"`
	A2UIID    string `json:"a2ui_id" jsonschema_description:"A2UI 的业务 ID"`
	DataJSON  string `json:"data_json" jsonschema_description:"数据模型更新内容的 JSON 字符串，注意如果查询对应A2UI返回的data_json为空，说明其是一个纯展示的A2UI，该字段就不应该被传入，可选"`
}

const listA2UIsLimit = 10000 // 一次拉取上限，视为「全部」

func listA2UIsImpl(ctx context.Context, input ListA2UIsInput) (string, error) {
	svc, _ := ctx.Value(ContextKeyA2UIService).(A2UIServiceInterface)
	if svc == nil {
		return "", errA2UIServiceNotInCtx
	}
	a2uiList, _, err := svc.ListA2UIs(1, listA2UIsLimit, "")
	if err != nil {
		return "", err
	}

	type brief struct {
		A2UIID      string `json:"a2ui_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var result []brief
	for _, a := range a2uiList {
		result = append(result, brief{
			A2UIID:      a.A2UIID,
			Name:        a.Name,
			Description: a.Description,
		})
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getA2UIImpl(ctx context.Context, input GetA2UIInput) (string, error) {
	svc, _ := ctx.Value(ContextKeyA2UIService).(A2UIServiceInterface)
	if svc == nil {
		return "", errA2UIServiceNotInCtx
	}
	if strings.TrimSpace(input.A2UIID) == "" {
		return "", errA2UIIDRequired
	}
	a, err := svc.GetA2UIByID(input.A2UIID)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sendA2UIImpl(ctx context.Context, input SendA2UIInput) (string, error) {
	sender, _ := ctx.Value(ContextKeyA2UISender).(agui.A2UISender)
	if sender == nil {
		return "", errA2UISenderNotAvailable
	}
	svc, _ := ctx.Value(ContextKeyA2UIService).(A2UIServiceInterface)
	if svc == nil {
		return "", errA2UIServiceNotInCtx
	}
	if strings.TrimSpace(input.MsgID) == "" {
		return "", errMsgIDRequired
	}
	if strings.TrimSpace(input.A2UIID) == "" {
		return "", errA2UIIDRequired
	}

	a, err := svc.GetA2UIByID(input.A2UIID)
	if err != nil {
		return "", err
	}
	resp := a2ui.A2UIResponse{
		MsgID:     input.MsgID,
		SurfaceID: input.SurfaceID,
		UIJSON:    a.UIJSON,
		DataJSON:  input.DataJSON,
	}
	if err := a2ui.SendA2UIResponse(sender, resp); err != nil {
		return "", err
	}
	return `{"status":"ok","message":"A2UI 已发送"}`, nil
}

// GetA2UITools 返回 A2UI 相关的三个工具（依赖从 context 读取 user_id 与 a2ui_sender）
func GetA2UITools(ctx context.Context) ([]einoTool.BaseTool, error) {
	listTool, err := utils.InferTool[ListA2UIsInput, string](
		"list_a2uis",
		"列出当前用户可以访问的全部 A2UI，一次返回所有。返回 a2ui_id、name、description 等简要信息。",
		listA2UIsImpl,
	)
	if err != nil {
		return nil, err
	}
	getTool, err := utils.InferTool[GetA2UIInput, string](
		"get_a2ui",
		"根据 a2ui_id 获取单个 A2UI 的详细信息（含 ui_json、data_json 等）。",
		getA2UIImpl,
	)
	if err != nil {
		return nil, err
	}
	sendTool, err := utils.InferTool[SendA2UIInput, string](
		"send_a2ui",
		"根据提供的 a2ui_id 与数据 JSON 渲染 A2UI 并发送事件到当前会话。需要 msg_id、a2ui_id，可选 surface_id、data_json。",
		sendA2UIImpl,
	)
	if err != nil {
		return nil, err
	}
	return []einoTool.BaseTool{listTool, getTool, sendTool}, nil
}
