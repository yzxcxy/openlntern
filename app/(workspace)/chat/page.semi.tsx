"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
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
    <div>
      <div
        className="semi-ai-chat-dialogue-content-tool-call"
        onClick={toggleOpen}
        role="button"
        tabIndex={0}
        onKeyDown={handleKeyDown}
        style={{ cursor: "pointer" }}
      >
        <IconWrench />
        <span>工具执行结果</span>
        {isOpen ? <IconChevronUp /> : <IconChevronDown />}
      </div>
      <Collapsible isOpen={isOpen}>
        <div className="semi-ai-chat-dialogue-content-bubble">
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

// 历史消息：按时间排序后聚合到 SemiMessage[]
const mapHistoryMessages = (items: BackendMessageItem[]) => {
  const sorted = [...items].sort((a, b) => {
    const aTime = toTimestamp(a.updated_at) ?? toTimestamp(a.created_at) ?? 0;
    const bTime = toTimestamp(b.updated_at) ?? toTimestamp(b.created_at) ?? 0;
    return aTime - bTime;
  });
  const result: SemiMessage[] = [];
  const runIndexMap = new Map<string, number>();

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

    // activity 消息独立映射，交由 A2UI 渲染
    if (role === "activity" || aguiMessage?.activityType) {
      result.push({
        id: item.msg_id,
        role: "activity",
        content: aguiMessage?.content as unknown as SemiMessage["content"],
        activityType: aguiMessage?.activityType,
        status: item.status,
        createdAt,
        updatedAt,
      } as SemiMessage);
      return;
    }

    // 其他消息按 run_id 归并到同一个 assistant SemiMessage
    if (!item.run_id) {
      return;
    }

    let runIndex = runIndexMap.get(item.run_id);
    if (runIndex === undefined) {
      const runMessage: SemiMessage = {
        id: item.run_id,
        role: "assistant",
        content: [],
        status: item.status,
        createdAt,
        updatedAt,
      };
      result.push(runMessage);
      runIndex = result.length - 1;
      runIndexMap.set(item.run_id, runIndex);
    }

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
      currentRunIdRef.current = "";
      textMessageMapRef.current.clear();
      toolCallMapRef.current.clear();
      reasoningMessageMapRef.current.clear();
    }
  }, [agent, resolvedThreadId, threadId]);

  // 拉取历史消息并映射为 SemiMessage[]
  const loadHistory = useCallback(async () => {
    if (!token || !threadId) return;
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

      // RUN_FINISHED 标记 run 结束
        if (eventType === "RUN_FINISHED") {
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          if (!runId) return;
        updateRunMessage(runId, (message) => ({
          ...message,
          status: "completed",
        }));
        currentRunIdRef.current = "";
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

      // activity 直接作为独立消息渲染
        if (eventType === "ACTIVITY_MESSAGE") {
          const activity = rawEvent.message ?? rawEvent;
          if (!activity?.activityType) return;
        setSemiMessages((prev) => [
          ...prev,
          {
            id: activity.id ?? createMessageId(),
            role: "activity",
            content: activity.content,
            activityType: activity.activityType,
            status: "completed",
            createdAt: Date.now(),
          },
        ]);
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

  const dialogueRenderConfig = useMemo(
    () => ({
      renderDialogueContent: ({
        message,
        defaultContent,
      }: {
        message?: SemiMessage;
        defaultContent?: ReactNode;
      }) => {
        if (!message) return defaultContent;
        if (message.role === "activity") {
          return renderActivityMessage(message as unknown as ActivityMessage);
        }
        return defaultContent;
      },
    }),
    [renderActivityMessage]
  );

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
    }),
    []
  );

  // 发送：先回显 user 消息，再触发 run
  const handleMessageSend = useCallback(
    (payload: MessageContent) => {
      if (agent.isRunning) return;
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
      agent.runAgent().catch((error) => {
        if (error instanceof Error && error.message) {
          setInputError(error.message);
        } else {
          setInputError("发送失败");
        }
      });
    },
    [agent]
  );

  const handleStopGenerate = useCallback(() => {
    agent.abortRun();
  }, [agent]);

  return (
    <div className="flex h-full w-full flex-col bg-white">
      <div className="flex-1 overflow-hidden">
        <AIChatDialogue
          align="leftRight"
          mode="bubble"
          chats={chats}
          dialogueRenderConfig={dialogueRenderConfig}
          renderDialogueContentItem={renderDialogueContentItem}
          roleConfig={roleConfig}
          className="h-full"
        />
      </div>
      <div className="border-t bg-white px-4 py-3">
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
          immediatelyRender = {false}
        />
        {inputError && (
          <div className="mt-2 text-xs text-red-500">{inputError}</div>
        )}
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
