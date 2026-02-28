"use client";

import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import {
  CopilotKitProvider,
  UseAgentUpdate,
  useAgent,
  useRenderActivityMessage,
} from "@copilotkit/react-core/v2";
import {
  AIChatDialogue,
  AIChatInput,
  Collapsible,
  MarkdownRender,
} from "@douyinfe/semi-ui-19";
import {
  IconChevronDown,
  IconChevronUp,
  IconWrench,
} from "@douyinfe/semi-icons";
import type { Message as SemiMessage } from "@douyinfe/semi-ui-19/lib/es/aiChatDialogue/interface";
import type {
  ActivityMessage,
  Message as AguiMessage,
} from "@ag-ui/client";
import type { MessageContent } from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import { theme } from "../../theme";
import { UiSelect } from "../../components/ui/UiSelect";
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
  type StoredUser,
} from "../auth";

const A2UI_MESSAGE_RENDERER = createA2UIMessageRenderer({ theme });
// 需要稳定引用，避免 renderActivityMessages 触发不稳定数组报错
const ACTIVITY_RENDERERS = [A2UI_MESSAGE_RENDERER];
const TOOL_RESULT_TYPE = "tool_result_text";
const ACTIVITY_CONTENT_TYPE = "activity_message";
const ACTIVITY_EVENT_SNAPSHOT = "ACTIVITY_SNAPSHOT";
const ACTIVITY_EVENT_DELTA = "ACTIVITY_DELTA";
const A2UI_SURFACE_ACTIVITY_TYPE = "a2ui-surface";

type ToolResultCollapseProps = {
  text: string;
};

function ToolResultCollapse({ text }: ToolResultCollapseProps) {
  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        toggleOpen();
      }
    },
    [toggleOpen]
  );

  return (
    <div className="motion-safe-slide-up">
      <div
        className="semi-ai-chat-dialogue-content-tool-call motion-safe-highlight"
        onClick={toggleOpen}
        role="button"
        tabIndex={0}
        onKeyDown={handleKeyDown}
      >
        <IconWrench />
        <span>工具执行结果</span>
        {isOpen ? <IconChevronUp /> : <IconChevronDown />}
      </div>
      <Collapsible isOpen={isOpen}>
        <div className="semi-ai-chat-dialogue-content-bubble px-3 py-3">
          <MarkdownRender format="md" raw={text} />
        </div>
      </Collapsible>
    </div>
  );
}

const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const createMessageId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

// 基于文字生成头像，避免无图时对话头信息空白
const buildTextAvatarDataUrl = (
  text: string,
  options?: { background?: string; color?: string }
) => {
  const safeText = Array.from(text || "")
    .filter(Boolean)
    .slice(0, 2)
    .join("");
  const label = safeText || "AI";
  const background = options?.background ?? "#94A3B8";
  const color = options?.color ?? "#FFFFFF";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64">
    <rect width="64" height="64" rx="16" fill="${background}" />
    <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle"
      font-family="Arial, sans-serif" font-size="24" fill="${color}">${label}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
};

type BackendMessageItem = {
  msg_id: string;
  thread_id: string;
  run_id: string;
  type: string;
  content: string;
  status?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
};

type BackendMessagePage = {
  data: BackendMessageItem[];
  total: number;
  page?: number;
  size?: number;
};

type BackendResult<T> = {
  code: number;
  message: string;
  data?: T;
};

type ModelCatalogOption = {
  provider_id: string;
  provider_name: string;
  provider_avatar?: string;
  api_type: string;
  model_id: string;
  model_key: string;
  model_name: string;
  model_avatar?: string;
  is_system_default?: boolean;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

// 后端消息中的 content 为 JSON 字符串，解析失败时返回 null
const safeParseJson = <T,>(value: string): T | null => {
  try {
    return JSON.parse(value) as T;
  } catch {
    return null;
  }
};

// 从不同形态的 content 中提取文本
const extractText = (content: unknown) => {
  if (typeof content === "string") {
    return content;
  }
  if (Array.isArray(content)) {
    return content
      .map((item) =>
        typeof (item as { text?: unknown }).text === "string"
          ? String((item as { text?: string }).text)
          : ""
      )
      .filter(Boolean)
      .join("");
  }
  return "";
};

// tool_result 的 content 可能是 JSON 字符串，优先解析出 text 聚合
const extractToolResultText = (content: unknown) => {
  if (typeof content === "string") {
    const parsed = safeParseJson<{ content?: Array<{ text?: string }> }>(content);
    if (parsed?.content?.length) {
      return parsed.content
        .map((item) => (typeof item.text === "string" ? item.text : ""))
        .filter(Boolean)
        .join("");
    }
    return content;
  }
  if (content && typeof content === "object") {
    const items = (content as { content?: Array<{ text?: string }> }).content;
    if (Array.isArray(items)) {
      return items
        .map((item) => (typeof item.text === "string" ? item.text : ""))
        .filter(Boolean)
        .join("");
    }
  }
  return "";
};

// ISO 时间字符串转为时间戳，解析失败返回 undefined
const toTimestamp = (value?: string) => {
  if (!value) return undefined;
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return undefined;
  return parsed;
};

const toRecord = (value: unknown): Record<string, any> => {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, any>;
  }
  return {};
};

const cloneValue = <T,>(value: T): T => {
  if (typeof structuredClone === "function") {
    try {
      return structuredClone(value);
    } catch {
      // fallback to JSON clone
    }
  }
  try {
    return JSON.parse(JSON.stringify(value)) as T;
  } catch {
    return value;
  }
};

const applyA2UISurfacePatch = (previousContent: unknown, patch: unknown) => {
  const base = toRecord(cloneValue(previousContent));
  const operations = Array.isArray(base.operations) ? [...base.operations] : [];
  const patchItems = Array.isArray(patch) ? patch : [];
  patchItems.forEach((item) => {
    if (!item || typeof item !== "object") return;
    const op = String((item as { op?: unknown }).op ?? "");
    const path = String((item as { path?: unknown }).path ?? "");
    const value = (item as { value?: unknown }).value;

    if (op === "add" && path === "/operations/-") {
      operations.push(value);
      return;
    }

    const match = path.match(/^\/operations\/(\d+)$/);
    if (!match) return;
    const index = Number(match[1]);
    if (!Number.isInteger(index) || index < 0) return;

    if (op === "add" || op === "replace") {
      operations[index] = value;
      return;
    }
    if (op === "remove") {
      operations.splice(index, 1);
    }
  });
  return {
    ...base,
    operations,
  };
};

const mapActivityContent = (params: {
  activityType?: string;
  eventType?: string;
  content?: unknown;
  patch?: unknown;
  replace?: boolean;
  previousContent?: unknown;
}) => {
  const {
    activityType,
    eventType,
    content,
    patch,
    replace,
    previousContent,
  } = params;
  const normalizedEventType =
    eventType === ACTIVITY_EVENT_DELTA
      ? ACTIVITY_EVENT_DELTA
      : ACTIVITY_EVENT_SNAPSHOT;

  if (normalizedEventType === ACTIVITY_EVENT_DELTA) {
    if (activityType === A2UI_SURFACE_ACTIVITY_TYPE) {
      return applyA2UISurfacePatch(previousContent, patch);
    }
    return {
      ...toRecord(previousContent),
      patch: Array.isArray(patch) ? patch : [],
    };
  }

  const nextContent = toRecord(content);
  if (replace || !previousContent) {
    return nextContent;
  }
  return {
    ...toRecord(previousContent),
    ...nextContent,
  };
};

// 历史消息：按时间排序后聚合到 SemiMessage[]
const mapHistoryMessages = (items: BackendMessageItem[]) => {
  const sorted = [...items].sort((a, b) => {
    const aTime = toTimestamp(a.updated_at) ?? toTimestamp(a.created_at) ?? 0;
    const bTime = toTimestamp(b.updated_at) ?? toTimestamp(b.created_at) ?? 0;
    return aTime - bTime;
  });
  const result: SemiMessage[] = [];
  const runIndexMap = new Map<string, number>();

  const ensureAssistantMessage = (
    runId: string,
    status?: string,
    createdAt?: number,
    updatedAt?: number
  ) => {
    let runIndex = runIndexMap.get(runId);
    if (runIndex === undefined) {
      result.push({
        id: runId,
        role: "assistant",
        content: [],
        status,
        createdAt,
        updatedAt,
      });
      runIndex = result.length - 1;
      runIndexMap.set(runId, runIndex);
    }
    return runIndex;
  };

  sorted.forEach((item) => {
    const aguiMessage = safeParseJson<any>(item.content);
    const role = aguiMessage?.role;
    const createdAt =
      toTimestamp(item.created_at) ?? toTimestamp(item.updated_at);
    const updatedAt =
      toTimestamp(item.updated_at) ?? toTimestamp(item.created_at);

    // user 消息直接映射为独立的 SemiMessage
    if (role === "user") {
      result.push({
        id: item.msg_id,
        role: "user",
        content: extractText(aguiMessage?.content),
        status: item.status,
        createdAt,
        updatedAt,
      });
      return;
    }

    // activity 消息归并到 assistant.content 中，交由 A2UI 渲染
    if (role === "activity" || aguiMessage?.activityType) {
      const activityType =
        typeof aguiMessage?.activityType === "string"
          ? aguiMessage.activityType
          : "";
      if (!activityType) return;
      const activityEventType =
        aguiMessage?.type === ACTIVITY_EVENT_DELTA
          ? ACTIVITY_EVENT_DELTA
          : ACTIVITY_EVENT_SNAPSHOT;
      const runId = item.run_id || `history-activity-${item.msg_id}`;
      const runIndex = ensureAssistantMessage(runId, item.status, createdAt, updatedAt);
      const target = result[runIndex] as SemiMessage;
      const content = Array.isArray(target.content) ? [...target.content] : [];
      const existingIndex = content.findIndex(
        (contentItem) =>
          (contentItem as { type?: string }).type === ACTIVITY_CONTENT_TYPE &&
          (contentItem as { activityMessageId?: string }).activityMessageId ===
            item.msg_id
      );
      const previousItem =
        existingIndex === -1 ? null : (content[existingIndex] as Record<string, any>);
      const mappedContent = mapActivityContent({
        activityType,
        eventType: activityEventType,
        content: aguiMessage?.content,
        patch: aguiMessage?.patch,
        replace: aguiMessage?.replace,
        previousContent: previousItem?.content,
      });
      const activityItem = {
        ...(previousItem ?? {}),
        id: item.msg_id,
        type: ACTIVITY_CONTENT_TYPE,
        activityMessageId: item.msg_id,
        activityType,
        activityEventType,
        content: mappedContent,
        status: item.status ?? "completed",
        timestamp: createdAt ?? updatedAt ?? Date.now(),
      };
      if (existingIndex === -1) {
        content.push(activityItem);
      } else {
        content[existingIndex] = activityItem;
      }
      result[runIndex] = {
        ...target,
        content,
        updatedAt: updatedAt ?? target.updatedAt,
      } as SemiMessage;
      return;
    }

    // 其他消息按 run_id 归并到同一个 assistant SemiMessage
    if (!item.run_id) {
      return;
    }

    const runIndex = ensureAssistantMessage(
      item.run_id,
      item.status,
      createdAt,
      updatedAt
    );
    const target = result[runIndex] as SemiMessage;
    const content = Array.isArray(target.content) ? target.content : [];
    const nextContent = [...content];

    // text -> output_text
    if (item.type === "text") {
      const text = extractText(aguiMessage?.content);
      if (text) {
        nextContent.push({
          type: "message",
          content: [{ type: "output_text", text }],
          status: item.status ?? "completed",
        });
      }
    }

    // tool_call -> function_call
    if (item.type === "tool_call") {
      const toolCall = aguiMessage?.toolCalls?.[0];
      if (toolCall) {
        nextContent.push({
          type: "function_call",
          id: toolCall.id,
          call_id: toolCall.id,
          name: toolCall.function?.name,
          status: item.status ?? "completed",
          arguments: toolCall.function?.arguments ?? "",
        });
      }
    }

    if (item.type === "tool_result") {
      const text = extractToolResultText(aguiMessage?.content);
      if (text) {
        nextContent.push({
          type: TOOL_RESULT_TYPE,
          text,
          status: item.status ?? "completed",
        });
      }
    }

    // reasoning -> summary_text
    if (item.type === "reasoning" || item.type === "reasoning_message") {
      const text = extractText(aguiMessage?.content);
      nextContent.push({
        type: "reasoning",
        status: item.status ?? "completed",
        summary: text ? [{ type: "summary_text", text }] : [],
        encryptedValue: aguiMessage?.encryptedValue,
      });
    }

    result[runIndex] = {
      ...target,
      content: nextContent,
      updatedAt: updatedAt ?? target.updatedAt,
    } as SemiMessage;
  });

  return result;
};

type ChatContentProps = {
  token: string;
  userId: string;
  userName: string;
  userAvatar: string;
};

function ChatContent({ token, userId, userName, userAvatar }: ChatContentProps) {
  const searchParams = useSearchParams();
  const fallbackThreadIdRef = useRef<string>("");
  const currentRunIdRef = useRef<string>("");
  const textMessageMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const toolCallMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const reasoningMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );
  const activityMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );
  const { agent } = useAgent({
    updates: [
      UseAgentUpdate.OnRunStatusChanged,
      UseAgentUpdate.OnStateChanged,
    ],
  });
  const { renderActivityMessage } = useRenderActivityMessage();
  const [inputError, setInputError] = useState("");
  const [threadId, setThreadId] = useState("");
  const [semiMessages, setSemiMessages] = useState<SemiMessage[]>([]);
  const [conversationMode, setConversationMode] = useState<"chat" | "agent">("chat");
  const [availableModels, setAvailableModels] = useState<ModelCatalogOption[]>([]);
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");

  useEffect(() => {
    let active = true;
    const loadModelCatalog = async () => {
      if (!token) return;
      try {
        const response = await fetch("/api/backend/v1/models/catalog", {
          headers: buildAuthHeaders(token, userId),
        });
        updateTokenFromResponse(response);
        const data = (await response
          .json()
          .catch(() => null)) as BackendResult<ModelCatalogOption[]> | null;
        if (!response.ok || !data || data.code !== 0) {
          if (!active) return;
          setAvailableModels([]);
          setSelectedModelId("");
          setSelectedProviderId("");
          return;
        }
        const nextItems = Array.isArray(data.data) ? data.data : [];
        if (!active) return;
        setAvailableModels(nextItems);
        const defaultItem =
          nextItems.find((item) => item.is_system_default) ?? nextItems[0] ?? null;
        setSelectedModelId((prev) => {
          const matched = nextItems.find((item) => item.model_id === prev);
          return matched?.model_id ?? defaultItem?.model_id ?? "";
        });
        setSelectedProviderId(defaultItem?.provider_id ?? "");
      } catch {
        if (!active) return;
        setAvailableModels([]);
        setSelectedModelId("");
        setSelectedProviderId("");
      }
    };
    loadModelCatalog();
    return () => {
      active = false;
    };
  }, [token, userId]);

  useEffect(() => {
    if (!availableModels.length) {
      setSelectedModelId("");
      setSelectedProviderId("");
      return;
    }
    const selected =
      availableModels.find((item) => item.model_id === selectedModelId) ??
      availableModels.find((item) => item.is_system_default) ??
      availableModels[0];
    if (!selected) {
      return;
    }
    if (selected.model_id !== selectedModelId) {
      setSelectedModelId(selected.model_id);
    }
    if (selected.provider_id !== selectedProviderId) {
      setSelectedProviderId(selected.provider_id);
    }
  }, [availableModels, selectedModelId, selectedProviderId]);

  const selectedModelOption = useMemo(
    () => availableModels.find((item) => item.model_id === selectedModelId) ?? null,
    [availableModels, selectedModelId]
  );
  const [historyLoading, setHistoryLoading] = useState(false);

  // 根据 query 参数或生成新 thread_id
  const resolvedThreadId = useMemo(() => {
    const paramThreadId = searchParams.get("threadId");
    if (paramThreadId) {
      return paramThreadId;
    }
    if (!fallbackThreadIdRef.current) {
      fallbackThreadIdRef.current = createThreadId();
    }
    return fallbackThreadIdRef.current;
  }, [searchParams]);

  // 切换 thread 时重置消息与事件映射表
  useEffect(() => {
    if (!agent) return;
    if (resolvedThreadId !== threadId) {
      setThreadId(resolvedThreadId);
      agent.threadId = resolvedThreadId;
      agent.setMessages([]);
      setInputError("");
      setSemiMessages([]);
      setHistoryLoading(false);
      currentRunIdRef.current = "";
      textMessageMapRef.current.clear();
      toolCallMapRef.current.clear();
      reasoningMessageMapRef.current.clear();
      activityMessageMapRef.current.clear();
    }
  }, [agent, resolvedThreadId, threadId]);

  // 拉取历史消息并映射为 SemiMessage[]
  const loadHistory = useCallback(async () => {
    if (!token || !threadId) return;
    setHistoryLoading(true);
    setInputError("");
    try {
      const response = await fetch(`/api/backend/v1/threads/${threadId}/messages`, {
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(response);
      const data = (await response
        .json()
        .catch(() => null)) as BackendResult<BackendMessagePage> | null;
      if (!response.ok) {
        if (data?.code === 1004) {
          // 新建会话暂无历史，保持空消息列表
          setSemiMessages([]);
          return;
        }
        setInputError(data?.message || "历史消息加载失败");
        return;
      }
      if (!data || data.code !== 0) {
        if (data?.code === 1004) {
          // 新建会话暂无历史，保持空消息列表
          setSemiMessages([]);
          return;
        }
        setInputError(data?.message || "历史消息加载失败");
        return;
      }
      // 后端返回 data.data 才是消息列表
      const mapped = mapHistoryMessages(data.data?.data ?? []);
      setSemiMessages(mapped);
    } catch (error) {
      if (error instanceof Error && error.message) {
        setInputError(error.message);
      } else {
        setInputError("历史消息加载失败");
      }
    } finally {
      setHistoryLoading(false);
    }
  }, [threadId, token, userId]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // 按 run_id 定位或创建 assistant SemiMessage
  const updateRunMessage = useCallback(
    (runId: string, updater: (message: SemiMessage) => SemiMessage) => {
      setSemiMessages((prev) => {
        const next = [...prev];
        let index = next.findIndex(
          (message) => message.id === runId && message.role === "assistant"
        );
        const baseMessage =
          index === -1
            ? {
                id: runId,
                role: "assistant",
                content: [],
                status: "in_progress",
                createdAt: Date.now(),
              }
            : (next[index] as SemiMessage);
        const normalizedMessage = {
          ...baseMessage,
          content: Array.isArray(baseMessage.content)
            ? [...baseMessage.content]
            : [],
        } as SemiMessage;
        const updated = updater(normalizedMessage);
        if (index === -1) {
          next.push(updated);
        } else {
          next[index] = updated;
        }
        return next;
      });
    },
    []
  );

  const resolveRunId = (event: any) =>
    event?.runId ?? event?.run_id ?? event?.data?.runId ?? event?.data?.run_id;

  const resolveMessageId = (event: any) =>
    event?.messageId ??
    event?.message_id ??
    event?.data?.messageId ??
    event?.data?.message_id;

  const resolveActivityMessageId = (event: any) =>
    event?.messageId ??
    event?.message_id ??
    event?.id ??
    event?.data?.messageId ??
    event?.data?.message_id;

  // 订阅 AG-UI 事件流，增量拼装 ContentItem[]
  useEffect(() => {
    if (!agent) return;
    const subscription = agent.subscribe({
      onEvent: ({ event }) => {
        const rawEvent = event as any;
        const eventType = String(rawEvent?.type ?? "");
        if (!eventType) return;
        // 调试日志：确认事件类型与关键信息是否完整
        console.debug("[chat][event]", {
          type: eventType,
          runId: resolveRunId(rawEvent),
          messageId: resolveMessageId(rawEvent),
          threadId: rawEvent?.threadId ?? rawEvent?.thread_id,
        });

        // RUN_STARTED 用来确立 runId
        if (eventType === "RUN_STARTED") {
          const runId = resolveRunId(rawEvent);
          // 关键日志：runId 为空会导致后续流式消息无法归档
          console.debug("[chat][RUN_STARTED]", { runId });
          if (runId) {
            currentRunIdRef.current = runId;
            updateRunMessage(runId, (message) => message);
          }
          return;
        }

      // RUN_FINISHED 标记 run 结束，并清空 AG-UI 消息结构
        if (eventType === "RUN_FINISHED") {
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          if (runId) {
            updateRunMessage(runId, (message) => ({
              ...message,
              status: "completed",
            }));
          }
          agent.setMessages([]);
          currentRunIdRef.current = "";
          textMessageMapRef.current.clear();
          toolCallMapRef.current.clear();
          reasoningMessageMapRef.current.clear();
          activityMessageMapRef.current.clear();
          return;
        }

      // TEXT_MESSAGE_* 流式拼接 output_text
        if (eventType === "TEXT_MESSAGE_START") {
          const runId = currentRunIdRef.current;
          const messageId = resolveMessageId(rawEvent);
          if (!runId || !messageId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "message",
            content: [{ type: "output_text", text: "" }],
            status: "in_progress",
          });
            textMessageMapRef.current.set(messageId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TEXT_MESSAGE_CONTENT") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = textMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target || !Array.isArray(target.content)) {
            return message;
          }
          const nextTarget = { ...target };
          const items = [...target.content];
          const first = items[0];
          if (first && typeof first.text === "string") {
            items[0] = { ...first, text: `${first.text}${event.delta ?? ""}` };
          }
          nextTarget.content = items;
          content[mapping.index] = nextTarget;
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TEXT_MESSAGE_END") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = textMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
          textMessageMapRef.current.delete(messageId);
        return;
        }

      // REASONING_MESSAGE_* 流式拼接 summary_text
        if (eventType === "REASONING_MESSAGE_START") {
          const runId = currentRunIdRef.current;
          const messageId = resolveMessageId(rawEvent);
          if (!runId || !messageId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "reasoning",
            summary: [{ type: "summary_text", text: "" }],
            status: "in_progress",
          });
            reasoningMessageMapRef.current.set(messageId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "REASONING_MESSAGE_CONTENT") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target || !Array.isArray(target.summary)) {
            return message;
          }
          const nextTarget = { ...target };
          const items = [...target.summary];
          const first = items[0];
          if (first && typeof first.text === "string") {
            items[0] = { ...first, text: `${first.text}${event.delta ?? ""}` };
          }
          nextTarget.summary = items;
          content[mapping.index] = nextTarget;
          return { ...message, content };
        });
        return;
        }

        if (eventType === "REASONING_MESSAGE_END") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
          reasoningMessageMapRef.current.delete(messageId);
        return;
        }

      // REASONING_ENCRYPTED_VALUE 绑定到 reasoning 消息
        if (eventType === "REASONING_ENCRYPTED_VALUE") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            encryptedValue: event.encryptedValue,
          };
          return { ...message, content };
        });
        return;
        }

      // TOOL_CALL_* 处理 function_call 的参数流
        if (eventType === "TOOL_CALL_START") {
          const runId = currentRunIdRef.current;
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!runId || !toolCallId) return;
          // 实时 SSE 事件仅从 toolCallName 获取工具名
          const name = rawEvent.toolCallName;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "function_call",
            id: toolCallId,
            call_id: toolCallId,
            name,
            status: "in_progress",
            arguments: "",
          });
          toolCallMapRef.current.set(toolCallId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_ARGS") {
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!toolCallId) return;
          const mapping = toolCallMapRef.current.get(toolCallId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            arguments: `${target.arguments ?? ""}${rawEvent.delta ?? ""}`,
          };
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_END") {
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!toolCallId) return;
          const mapping = toolCallMapRef.current.get(toolCallId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index];
          if (!target) return message;
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_RESULT") {
          const runId = currentRunIdRef.current;
          if (!runId) return;
          const outputText = extractToolResultText(
            rawEvent.result ?? rawEvent.content
          );
        if (!outputText) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          content.push({
            type: TOOL_RESULT_TYPE,
            text: outputText,
            status: "completed",
          });
          return { ...message, content };
        });
        return;
        }

      // activity 归并到 assistant.content 中，由 content renderer 交给 A2UI
        if (
          eventType === ACTIVITY_EVENT_SNAPSHOT ||
          eventType === ACTIVITY_EVENT_DELTA
        ) {
          const activityEvent = rawEvent;
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          const activityType =
            typeof activityEvent?.activityType === "string"
              ? activityEvent.activityType
              : "";
          const activityMessageId = resolveActivityMessageId(activityEvent);
          if (!runId || !activityType || !activityMessageId) return;
          const activityMessageKey = String(activityMessageId);
          const activityEventType =
            eventType === ACTIVITY_EVENT_DELTA
              ? ACTIVITY_EVENT_DELTA
              : ACTIVITY_EVENT_SNAPSHOT;
          updateRunMessage(runId, (message) => {
            const content = Array.isArray(message.content) ? [...message.content] : [];
            const mapped = activityMessageMapRef.current.get(activityMessageKey);
            let index =
              mapped && mapped.runId === runId ? mapped.index : -1;
            if (
              index < 0 ||
              index >= content.length ||
              (content[index] as { activityMessageId?: string })?.activityMessageId !==
                activityMessageKey
            ) {
              index = content.findIndex(
                (contentItem) =>
                  (contentItem as { type?: string }).type === ACTIVITY_CONTENT_TYPE &&
                  (contentItem as { activityMessageId?: string }).activityMessageId ===
                    activityMessageKey
              );
            }
            const previousItem =
              index === -1 ? null : (content[index] as Record<string, any>);
            const mappedContent = mapActivityContent({
              activityType,
              eventType: activityEventType,
              content: activityEvent?.content,
              patch: activityEvent?.patch,
              replace: activityEvent?.replace,
              previousContent: previousItem?.content,
            });
            const activityContentItem = {
              ...(previousItem ?? {}),
              id: activityMessageKey,
              type: ACTIVITY_CONTENT_TYPE,
              activityMessageId: activityMessageKey,
              activityType,
              activityEventType,
              content: mappedContent,
              status:
                activityEvent?.status ??
                (activityEventType === ACTIVITY_EVENT_DELTA
                  ? "in_progress"
                  : "completed"),
              timestamp:
                activityEvent?.timestamp ??
                rawEvent?.timestamp ??
                Date.now(),
            };
            if (index === -1) {
              index = content.length;
              content.push(activityContentItem);
            } else {
              content[index] = activityContentItem;
            }
            activityMessageMapRef.current.set(activityMessageKey, {
              runId,
              index,
            });
            return {
              ...message,
              content,
            };
          });
          return;
        }
      },
    });

    return () => {
      subscription?.unsubscribe?.();
    };
  }, [agent, updateRunMessage]);

  const chats = useMemo<SemiMessage[]>(
    () =>
      semiMessages.map((message, index) => ({
        ...message,
        id: message.id ?? `${message.role}-${index}`,
      })),
    [semiMessages]
  );
  const showHistorySkeleton = historyLoading && chats.length === 0;
  const showEmptyState =
    !historyLoading && !agent.isRunning && chats.length === 0 && !inputError;

  const roleConfig = useMemo(
    () => ({
      user: {
        name: userName || "用户",
        avatar:
          userAvatar ||
          buildTextAvatarDataUrl(userName || "用户", {
            background: "#2DD4BF",
          }),
        color: "teal",
      },
      assistant: {
        name: "AI助手",
        avatar: buildTextAvatarDataUrl("AI", { background: "#6366F1" }),
        color: "indigo",
      },
    }),
    [userAvatar, userName]
  );

  const renderDialogueContentItem = useMemo(
    () => ({
      [TOOL_RESULT_TYPE]: (item: { text?: string }) => {
        if (!item?.text) return null;
        return <ToolResultCollapse text={item.text} />;
      },
      [ACTIVITY_CONTENT_TYPE]: (item: {
        id?: string;
        activityMessageId?: string;
        activityType?: string;
        content?: unknown;
      }) => {
        if (!item?.activityType) return null;
        const activityMessage: ActivityMessage = {
          id: String(item.activityMessageId ?? item.id ?? createMessageId()),
          role: "activity",
          activityType: item.activityType,
          content: toRecord(item.content),
        };
        const activityNode = renderActivityMessage(activityMessage);
        if (!activityNode) return null;
        return (
          <div className="motion-safe-slide-up my-2 overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-md)]">
            <div className="border-b border-[var(--color-border-default)] bg-[linear-gradient(90deg,rgba(37,99,255,0.08),rgba(14,165,233,0.08))] px-3 py-2">
              <span className="text-xs font-medium text-[var(--color-text-secondary)]">
                可视化内容
              </span>
            </div>
            <div className="p-3">{activityNode}</div>
          </div>
        );
      },
    }),
    [renderActivityMessage]
  );

  const renderConfigureArea = useCallback(
    () => (
      <div
        className="flex items-center gap-2"
        onMouseDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
      >
        <UiSelect
          value={conversationMode}
          onChange={(event) =>
            setConversationMode(event.target.value === "agent" ? "agent" : "chat")
          }
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
          className="rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-2.5 py-1 text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-action-primary)]"
        >
          <option value="chat">Chat</option>
          <option value="agent">Agent</option>
        </UiSelect>
      </div>
    ),
    [conversationMode]
  );

  const renderActionArea = useCallback(
    (props: { menuItem: ReactNode[]; className: string }) => (
      <div
        className={joinClasses(props.className, "flex items-center gap-2")}
        onMouseDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
      >
        {conversationMode === "chat" && (
          <UiSelect
            value={selectedModelId}
            onChange={(event) => {
              const nextModelId = event.target.value;
              const nextItem = availableModels.find((item) => item.model_id === nextModelId);
              setSelectedModelId(nextModelId);
              setSelectedProviderId(nextItem?.provider_id ?? "");
            }}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={(event) => event.stopPropagation()}
            className="max-w-[220px] rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-3 py-1.5 text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-action-primary)]"
          >
            {availableModels.length === 0 ? (
              <option value="">请先配置模型</option>
            ) : (
              availableModels.map((item) => (
                <option key={item.model_id} value={item.model_id}>
                  {item.provider_name} / {item.model_name}
                </option>
              ))
            )}
          </UiSelect>
        )}
        {props.menuItem}
      </div>
    ),
    [availableModels, conversationMode, selectedModelId]
  );

  // 发送：先回显 user 消息，再触发 run
  const handleMessageSend = useCallback(
    (payload: MessageContent) => {
      if (agent.isRunning) return;
      if (conversationMode === "agent") {
        setInputError("Agent 模式暂未开放");
        return;
      }
      if (!selectedModelOption) {
        setInputError("请先配置模型服务并选择模型");
        return;
      }
      const inputContents = payload?.inputContents ?? [];
      if (!inputContents.length) {
        setInputError("请输入内容");
        return;
      }
      const textChunks = inputContents
        .map((item) =>
          typeof (item as { text?: unknown }).text === "string"
            ? String((item as { text?: string }).text)
            : ""
        )
        .filter(Boolean);
      if (!textChunks.length) {
        setInputError("暂不支持该输入格式");
        return;
      }
      setInputError("");
      const message: AguiMessage = {
        id: createMessageId(),
        role: "user",
        content: textChunks.join(""),
      };
      setSemiMessages((prev) => [
        ...prev,
        {
          id: message.id,
          role: "user",
          content: message.content,
          status: "completed",
          createdAt: Date.now(),
        },
      ]);
      agent.setMessages([message]);
      agent
        .runAgent({
          forwardedProps: {
            agentConfig: {
              conversation: {
                mode: conversationMode,
              },
              model: {
                providerId: selectedProviderId || selectedModelOption.provider_id,
                modelId: selectedModelOption.model_id,
              },
              features: {},
            },
          },
        })
        .catch((error) => {
          if (error instanceof Error && error.message) {
            setInputError(error.message);
          } else {
            setInputError("发送失败");
          }
        });
    },
    [agent, conversationMode, selectedModelOption, selectedProviderId]
  );

  const handleStopGenerate = useCallback(() => {
    agent.abortRun();
  }, [agent]);

  return (
    <div className="chat-page workspace-gradient-surface workspace-gradient-surface--chat flex h-full w-full flex-col p-3 md:p-4">
      <div className="motion-safe-fade-in flex h-full min-h-0 flex-col gap-3">
        <div className="motion-safe-slide-up flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.86)] px-4 py-3 shadow-[var(--shadow-sm)] backdrop-blur-sm">
          <div>
            <div className="text-sm font-semibold text-[var(--color-text-primary)]">
              智能对话
            </div>
            <div className="text-xs text-[var(--color-text-muted)]">
              实时展示回复、工具调用与可视化活动消息
            </div>
          </div>
          <div
            className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${
              agent.isRunning
                ? "border-[rgba(37,99,255,0.2)] bg-[rgba(37,99,255,0.08)] text-[var(--color-action-primary)]"
                : "border-[rgba(22,163,74,0.18)] bg-[rgba(22,163,74,0.08)] text-[var(--color-state-success)]"
            }`}
          >
            {agent.isRunning ? "生成中" : "已就绪"}
          </div>
        </div>

        <div className="motion-safe-lift flex min-h-0 flex-1 flex-col overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.92)] shadow-[var(--shadow-sm)] backdrop-blur-sm">
          <div className="flex-1 overflow-hidden px-1 py-1">
            {showHistorySkeleton ? (
              <div className="flex h-full flex-col gap-4 px-4 py-5">
                <div className="flex justify-start">
                  <div className="w-full max-w-[68%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(226,232,240,0.8)] bg-[rgba(248,250,252,0.82)] p-4">
                    <div className="h-3 w-20 animate-pulse rounded-full bg-[rgba(148,163,184,0.16)]" />
                    <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(148,163,184,0.12)]" />
                    <div className="h-3 w-4/5 animate-pulse rounded-full bg-[rgba(148,163,184,0.1)]" />
                  </div>
                </div>
                <div className="flex justify-end">
                  <div className="w-full max-w-[52%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(37,99,255,0.12)] bg-[rgba(37,99,255,0.06)] p-4">
                    <div className="h-3 w-16 animate-pulse rounded-full bg-[rgba(37,99,255,0.12)]" />
                    <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(37,99,255,0.08)]" />
                  </div>
                </div>
                <div className="flex justify-start">
                  <div className="w-full max-w-[74%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(226,232,240,0.8)] bg-[rgba(248,250,252,0.82)] p-4">
                    <div className="h-3 w-24 animate-pulse rounded-full bg-[rgba(148,163,184,0.14)]" />
                    <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(148,163,184,0.1)]" />
                    <div className="h-3 w-3/4 animate-pulse rounded-full bg-[rgba(148,163,184,0.08)]" />
                  </div>
                </div>
              </div>
            ) : showEmptyState ? (
              <div className="motion-safe-fade-in flex h-full items-center justify-center px-6 py-8">
                <div className="max-w-md rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,250,252,0.94))] p-6 text-center shadow-[var(--shadow-sm)]">
                  <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.08)] text-[var(--color-action-primary)]">
                    <svg
                      viewBox="0 0 24 24"
                      className="h-6 w-6"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M8 10h8" />
                      <path d="M8 14h5" />
                      <path d="M6 4h12a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H9l-5 3V6a2 2 0 0 1 2-2z" />
                    </svg>
                  </div>
                  <div className="mt-4 text-sm font-semibold text-[var(--color-text-primary)]">
                    开始新的对话
                  </div>
                  <div className="mt-2 text-sm text-[var(--color-text-muted)]">
                    输入你的问题，系统会实时展示回复、工具调用和可视化内容。
                  </div>
                </div>
              </div>
            ) : (
              <AIChatDialogue
                align="leftRight"
                mode="bubble"
                chats={chats}
                renderDialogueContentItem={renderDialogueContentItem}
                roleConfig={roleConfig}
                className="h-full"
              />
            )}
          </div>
          <div className="border-t border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,250,252,0.96))] px-4 py-3">
            {agent.isRunning && (
              <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.06)] px-3 py-1 text-xs font-medium text-[var(--color-action-primary)]">
                <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-current" />
                AI 正在生成回复
              </div>
            )}
            <AIChatInput
              keepSkillAfterSend={false}
              onMessageSend={handleMessageSend}
              onStopGenerate={handleStopGenerate}
              generating={agent.isRunning}
              canSend={!agent.isRunning}
              showUploadButton={false}
              showUploadFile={false}
              showReference={false}
              round
              immediatelyRender={false}
              renderConfigureArea={renderConfigureArea}
              renderActionArea={renderActionArea}
            />
            {inputError && (
              <div
                role="alert"
                aria-live="polite"
                className="motion-safe-slide-up mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]"
              >
                {inputError}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function ChatPage() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [userInfo, setUserInfo] = useState<StoredUser | null>(null);
  const refreshUserInfo = useCallback(() => {
    setUserInfo(readStoredUser());
  }, []);

  useEffect(() => {
    const currentToken = readValidToken(router);
    if (!currentToken) {
      router.push("/login");
      return;
    }
    setToken(currentToken);
    refreshUserInfo();
  }, [refreshUserInfo, router]);

  // 同步 localStorage 用户信息变化
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handleStorage = (event: StorageEvent) => {
      if (event.key === "user") {
        refreshUserInfo();
      }
    };
    const handleFocus = () => {
      refreshUserInfo();
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("focus", handleFocus);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("focus", handleFocus);
    };
  }, [refreshUserInfo]);

  const userName = useMemo(() => {
    const name =
      typeof userInfo?.username === "string" ? userInfo.username.trim() : "";
    return name || "用户";
  }, [userInfo]);

  const userAvatar = useMemo(() => {
    const avatar =
      typeof userInfo?.avatar === "string" ? userInfo.avatar.trim() : "";
    return avatar;
  }, [userInfo]);

  const userId = useMemo(() => {
    if (!token) return "";
    const storedId = userInfo?.user_id;
    if (storedId !== undefined && storedId !== null) {
      return String(storedId);
    }
    return getUserIdFromToken(token);
  }, [token, userInfo]);

  const copilotHeaders = useMemo(
    () => buildAuthHeaders(token, userId),
    [token, userId]
  );

  if (!token) {
    return null;
  }

  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      renderActivityMessages={ACTIVITY_RENDERERS}
      headers={copilotHeaders}
    >
      <ChatContent
        token={token}
        userId={userId}
        userName={userName}
        userAvatar={userAvatar}
      />
    </CopilotKitProvider>
  );
}
