"use client";

import { AIChatDialogue, AIChatInput } from "@douyinfe/semi-ui";
import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import { CopilotKit, useCopilotChatInternal } from "@copilotkit/react-core";
import { useCopilotKit } from "@copilotkit/react-core/v2";
import { EventType } from "@ag-ui/client";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { theme } from "../../theme";
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

const A2UIMessageRenderer = createA2UIMessageRenderer({ theme });
const activityRenderers = [A2UIMessageRenderer];
const API_BASE = "/api/backend";
const quickActionHints = [
  "给我推荐几道今日菜单",
  "我想要两人套餐，包含荤素搭配",
  "请避开辣味和海鲜",
];
const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

type ActivityMessageLike = {
  id: string;
  role: "activity";
  content: Record<string, unknown>;
  activityType: string;
};

const safeStringify = (value: unknown) => {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
};

const normalizeContentItems = (content: unknown) => {
  if (typeof content === "string") return [content];
  if (Array.isArray(content)) return content;
  if (content && typeof content === "object") {
    const container = content as { content?: unknown; text?: unknown };
    if (Array.isArray(container.content)) return container.content;
    if (typeof container.text === "string") return [container];
    if (typeof container.content === "string") return [container.content];
  }
  if (content === null || content === undefined) return [];
  return [content];
};

const collectMessageParts = (message: {
  role?: string;
  content?: unknown;
  toolCalls?: unknown[];
}) => {
  const textParts: string[] = [];
  const toolParts: string[] = [];
  const reasoningParts: string[] = [];
  const role = message.role;
  const items = normalizeContentItems(message.content);

  const pushValue = (target: string[], value: string) => {
    if (!value) return;
    target.push(value);
  };

  for (const item of items) {
    if (typeof item === "string") {
      if (role === "tool") {
        pushValue(toolParts, item);
      } else {
        pushValue(textParts, item);
      }
      continue;
    }
    if (typeof item === "number" || typeof item === "boolean") {
      const value = String(item);
      if (role === "tool") {
        pushValue(toolParts, value);
      } else {
        pushValue(textParts, value);
      }
      continue;
    }
    if (item && typeof item === "object") {
      const entry = item as {
        type?: string;
        text?: string;
        content?: string;
        reasoning?: string;
      };
      const type = typeof entry.type === "string" ? entry.type.toLowerCase() : "";
      const value =
        (typeof entry.text === "string" && entry.text) ||
        (typeof entry.content === "string" && entry.content) ||
        (typeof entry.reasoning === "string" && entry.reasoning) ||
        safeStringify(item);
      if (type.includes("reasoning") || type.includes("thinking")) {
        pushValue(reasoningParts, value);
      } else if (type.includes("tool")) {
        pushValue(toolParts, value);
      } else if (type.includes("text")) {
        pushValue(textParts, value);
      } else if (role === "tool") {
        pushValue(toolParts, value);
      } else {
        pushValue(textParts, value);
      }
    }
  }

  if (Array.isArray(message.toolCalls)) {
    for (const toolCall of message.toolCalls) {
      pushValue(toolParts, safeStringify(toolCall));
    }
  }

  return { textParts, toolParts, reasoningParts };
};

const getTextFromContent = (content: unknown) => {
  const { textParts } = collectMessageParts({ content, role: "user" });
  return textParts.join("\n").trim();
};

const extractInputText = (payload?: { inputContents?: unknown[] }) => {
  const items = Array.isArray(payload?.inputContents) ? payload.inputContents : [];
  const parts: string[] = [];
  for (const item of items) {
    if (typeof item === "string") {
      parts.push(item);
      continue;
    }
    if (item && typeof item === "object") {
      const entry = item as { text?: string; content?: unknown; type?: string };
      if (typeof entry.text === "string") {
        parts.push(entry.text);
        continue;
      }
      if (typeof entry.content === "string") {
        parts.push(entry.content);
        continue;
      }
      if (Array.isArray(entry.content)) {
        const text = collectMessageParts({ role: "user", content: entry.content })
          .textParts.join("\n")
          .trim();
        if (text) parts.push(text);
      }
    }
  }
  return parts.join("\n").trim();
};

function ChatContent() {
  const { copilotkit } = useCopilotKit();
  const {
    messages,
    sendMessage,
    setMessages,
    isLoading,
    stopGeneration,
    agent,
    threadId,
  } = useCopilotChatInternal();
  const [historyLoaded, setHistoryLoaded] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState("");
  const [historyPage, setHistoryPage] = useState(1);
  const [historyPageSize] = useState(20);
  const [historyHasMore, setHistoryHasMore] = useState(false);
  const [liveThinkingByRunId, setLiveThinkingByRunId] = useState<
    Record<string, Record<string, string>>
  >({});
  const [completedThinkingByRunId, setCompletedThinkingByRunId] = useState<
    Record<string, Record<string, string[]>>
  >({});
  const previousThreadId = useRef<string | undefined>(threadId);
  const isPrependingRef = useRef(false);
  const messagesRef = useRef(messages);
  const thinkingBufferByRunIdRef = useRef<Record<string, string>>({});
  const currentRunIdRef = useRef<string | null>(null);
  const currentThinkingMsgIdByRunIdRef = useRef<Record<string, string>>({});
  const runIdToUserMessageIdRef = useRef<Record<string, string>>({});
  const runIdOrderRef = useRef<string[]>([]);
  const router = useRouter();
  const ActivityRenderer = A2UIMessageRenderer.render;
  const getLatestUserMessageId = () => {
    const list = messagesRef.current;
    for (let i = list.length - 1; i >= 0; i -= 1) {
      const message = list[i];
      if (message?.role === "user" && typeof message.id === "string") {
        return message.id;
      }
    }
    return null;
  };

  useEffect(() => {
    if (!agent || !threadId) return;
    if (agent.threadId !== threadId) {
      agent.threadId = threadId;
      copilotkit.connectAgent({ agent }).catch(() => {});
    }
  }, [agent, threadId, copilotkit]);

  useEffect(() => {
    if (!agent) return;
    const applyThinkingEvent = (incoming: unknown) => {
      const event = incoming as { type?: string; delta?: string; runId?: string };
      if (!event?.type) return;
      if (event.type === EventType.RUN_STARTED) {
        const runId = typeof event.runId === "string" ? event.runId : null;
        currentRunIdRef.current = runId;
        if (runId) {
          thinkingBufferByRunIdRef.current[runId] = "";
        }
        if (runId) {
          const latestUserId = getLatestUserMessageId();
          if (latestUserId) {
            runIdToUserMessageIdRef.current[runId] = latestUserId;
          }
          runIdOrderRef.current = [...runIdOrderRef.current, runId];
        }
        return;
      }
      if (event.type === EventType.THINKING_START) {
        const runId = currentRunIdRef.current;
        if (!runId) return;
        const thinkingMsgId = createThreadId();
        currentThinkingMsgIdByRunIdRef.current[runId] = thinkingMsgId;
        thinkingBufferByRunIdRef.current[runId] = "";
        setLiveThinkingByRunId((prev) => ({
          ...prev,
          [runId]: {
            ...(prev[runId] ?? {}),
            [thinkingMsgId]: "",
          },
        }));
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_START) {
        const runId = currentRunIdRef.current;
        if (!runId) return;
        thinkingBufferByRunIdRef.current[runId] = "";
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_CONTENT) {
        const delta = typeof event.delta === "string" ? event.delta : "";
        if (!delta) return;
        const runId = currentRunIdRef.current;
        if (!runId) return;
        const thinkingMsgId = currentThinkingMsgIdByRunIdRef.current[runId];
        if (!thinkingMsgId) return;
        const current = thinkingBufferByRunIdRef.current[runId] ?? "";
        const next = `${current}${delta}`;
        thinkingBufferByRunIdRef.current[runId] = next;
        setLiveThinkingByRunId((prev) => {
          const runLive = prev[runId] ?? {};
          if (runLive[thinkingMsgId] === next) return prev;
          return {
            ...prev,
            [runId]: {
              ...runLive,
              [thinkingMsgId]: next,
            },
          };
        });
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_END) {
        const runId = currentRunIdRef.current;
        if (!runId) return;
        const thinkingMsgId = currentThinkingMsgIdByRunIdRef.current[runId];
        if (!thinkingMsgId) return;
        const value = thinkingBufferByRunIdRef.current[runId] ?? "";
        thinkingBufferByRunIdRef.current[runId] = "";
        if (!runId || !value) return;
        setCompletedThinkingByRunId((prev) => ({
          ...prev,
          [runId]: {
            ...(prev[runId] ?? {}),
            [thinkingMsgId]: [...(prev[runId]?.[thinkingMsgId] ?? []), value],
          },
        }));
        setLiveThinkingByRunId((prev) => ({
          ...prev,
          [runId]: {
            ...(prev[runId] ?? {}),
            [thinkingMsgId]: "",
          },
        }));
      }
      if (event.type === EventType.THINKING_END) {
        const runId = currentRunIdRef.current;
        if (!runId) return;
        delete currentThinkingMsgIdByRunIdRef.current[runId];
        thinkingBufferByRunIdRef.current[runId] = "";
      }
    };
    const { unsubscribe } = agent.subscribe({
      onEvent: ({ event }) => {
        applyThinkingEvent(event);
      },
    });
    return () => {
      unsubscribe();
    };
  }, [agent, setMessages]);

  useEffect(() => {
    if (previousThreadId.current === undefined) {
      previousThreadId.current = threadId;
      return;
    }
    if (previousThreadId.current !== threadId) {
      previousThreadId.current = threadId;
      setLiveThinkingByRunId({});
      setCompletedThinkingByRunId({});
      runIdToUserMessageIdRef.current = {};
      runIdOrderRef.current = [];
      currentThinkingMsgIdByRunIdRef.current = {};
      thinkingBufferByRunIdRef.current = {};
      setHistoryLoaded(false);
      setHistoryLoading(false);
      setHistoryError("");
      setHistoryPage(1);
      setHistoryHasMore(false);
      setMessages([]);
    }
  }, [threadId, setMessages]);

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  const parseMetadata = useCallback((metadata?: string) => {
    if (!metadata) return {};
    try {
      return JSON.parse(metadata);
    } catch {
      return {};
    }
  }, []);

  const parseActivityContent = useCallback((content: string) => {
    if (!content) return content;
    try {
      return JSON.parse(content);
    } catch {
      return content;
    }
  }, []);
  const parseMessagePayload = useCallback((content?: string) => {
    if (!content) return {};
    try {
      return JSON.parse(content) as {
        role?: string;
        content?: unknown;
        tool_calls?: unknown[];
        toolCalls?: unknown[];
        activityType?: string;
        activity_type?: string;
        toolCallId?: string;
      };
    } catch {
      return {};
    }
  }, []);
  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const mapHistoryMessage = useCallback(
    (item: {
      msg_id?: string;
      run_id?: string;
      type?: string;
      role?: string;
      content?: string;
      metadata?: string;
    }) => {
      const metadata = parseMetadata(item.metadata);
      const activityType =
        typeof metadata.activity_type === "string"
          ? metadata.activity_type
          : undefined;
      const runId = typeof item.run_id === "string" ? item.run_id : undefined;
      const roleFromMeta =
        typeof metadata.role === "string" ? metadata.role : undefined;
      const isActivity = item.type === "activity" || Boolean(activityType);
      const payload = parseMessagePayload(item.content);
      const activityTypeFromPayload =
        typeof payload.activityType === "string"
          ? payload.activityType
          : typeof payload.activity_type === "string"
          ? payload.activity_type
          : undefined;
      const roleFromPayload =
        typeof payload.role === "string" ? payload.role : undefined;
      const roleFromItem = typeof item.role === "string" ? item.role : undefined;
      const roleFromType = typeof item.type === "string" ? item.type : undefined;
      const roleCandidate =
        roleFromPayload ?? roleFromItem ?? roleFromMeta ?? roleFromType;
      const normalizedRole =
        roleCandidate === "user" ||
        roleCandidate === "assistant" ||
        roleCandidate === "system" ||
        roleCandidate === "tool" ||
        roleCandidate === "reasoning"
          ? roleCandidate
          : "assistant";
      const role = isActivity ? "activity" : normalizedRole;
      const content = isActivity
        ? payload.content ?? parseActivityContent(item.content ?? "")
        : (payload.content ?? "");
      const toolCalls =
        payload.tool_calls ??
        payload.toolCalls ??
        (typeof payload.tool_calls === "undefined" &&
        typeof payload.toolCalls === "undefined"
          ? undefined
          : []);
      return {
        id: item.msg_id ?? createThreadId(),
        role,
        content,
        ...(activityType || activityTypeFromPayload
          ? { activityType: activityType ?? activityTypeFromPayload }
          : {}),
        ...(runId ? { runId } : {}),
        ...(toolCalls ? { toolCalls } : {}),
        ...(typeof payload.toolCallId === "string"
          ? { toolCallId: payload.toolCallId }
          : {}),
        ...(typeof item.type === "string" ? { messageType: item.type } : {}),
      };
    },
    [parseActivityContent, parseMessagePayload, parseMetadata]
  );

  const fetchHistoryPage = useCallback(
    async (pageToLoad: number, replace: boolean) => {
      if (!threadId) return;
      const token = getValidToken();
      if (!token) return;
      setHistoryLoading(true);
      setHistoryError("");
      try {
        const params = new URLSearchParams();
        params.set("page", String(pageToLoad));
        params.set("page_size", String(historyPageSize));
        const res = await fetch(
          `${API_BASE}/v1/threads/${threadId}/messages?${params.toString()}`,
          {
            headers: buildAuthHeaders(token),
          }
        );
        updateTokenFromResponse(res);
        const data = await res.json();
        const message = typeof data?.message === "string" ? data.message : "";
        if (!res.ok) {
          if (res.status === 404 && message === "thread not found") {
            setMessages(replace ? [] : messagesRef.current);
            setHistoryLoaded(true);
            setHistoryPage(pageToLoad);
            setHistoryHasMore(false);
            return;
          }
          throw new Error(message || "获取历史消息失败");
        }
        if (data.code !== 0) {
          if (message === "thread not found") {
            setMessages(replace ? [] : messagesRef.current);
            setHistoryLoaded(true);
            setHistoryPage(pageToLoad);
            setHistoryHasMore(false);
            return;
          }
          throw new Error(message || "获取历史消息失败");
        }
        const items = Array.isArray(data.data?.data) ? data.data.data : [];
        const total = typeof data.data?.total === "number" ? data.data.total : 0;
        const normalized = items.map(mapHistoryMessage).reverse();
        setMessages(replace ? normalized : [...normalized, ...messagesRef.current]);
        setHistoryLoaded(true);
        setHistoryPage(pageToLoad);
        setHistoryHasMore(pageToLoad * historyPageSize < total);
      } catch (err) {
        if (err instanceof Error && err.message) {
          setHistoryError(err.message);
        } else {
          setHistoryError("获取历史消息失败");
        }
      } finally {
        setHistoryLoading(false);
      }
    },
    [getValidToken, historyPageSize, mapHistoryMessage, setMessages, threadId]
  );

  useEffect(() => {
    if (historyLoaded || !threadId) return;
    fetchHistoryPage(1, true);
  }, [fetchHistoryPage, historyLoaded, threadId]);

  const handleQuickSend = useCallback(
    async (content: string) => {
      const trimmed = content.trim();
      if (!trimmed) return;
      await sendMessage({
        id: createThreadId(),
        role: "user",
        content: trimmed,
      });
      window.dispatchEvent(new Event("threads-refresh"));
    },
    [sendMessage]
  );

  const handleInputSend = useCallback(
    (payload: { inputContents?: unknown[] }) => {
      const text = extractInputText(payload);
      if (!text) return;
      void handleQuickSend(text);
    },
    [handleQuickSend]
  );

  const groupedItems = useMemo(() => {
    type Segment = {
      id: string;
      kind: "text" | "reasoning" | "tool_call" | "tool_result" | "activity";
      text?: string;
      toolCall?: unknown;
      toolCallId?: string;
      activity?: ActivityMessageLike;
    };
    const items: Array<{
      kind: "user" | "assistant";
      id: string;
      content?: string;
      segments?: Segment[];
    }> = [];
    const groups = new Map<string, (typeof items)[number]>();
    const groupSegmentMaps = new Map<string, Map<string, Segment>>();
    let currentBatchKey: string | null = null;
    const runIdToUserMessageId = runIdToUserMessageIdRef.current;
    const runIdOrder = runIdOrderRef.current;
    const latestThinkingRunId = (() => {
      const ids = [
        ...Object.keys(liveThinkingByRunId),
        ...Object.keys(completedThinkingByRunId),
      ];
      return ids.length > 0 ? ids[ids.length - 1] : undefined;
    })();
    const getThinkingItems = (runId: string) => {
      const completedById = completedThinkingByRunId[runId] ?? {};
      const liveById = liveThinkingByRunId[runId] ?? {};
      const ids = new Set([...Object.keys(completedById), ...Object.keys(liveById)]);
      return Array.from(ids)
        .map((id) => {
          const completed = completedById[id] ?? [];
          const live = liveById[id];
          const texts = [...completed, ...(live ? [live] : [])];
          return { id, texts };
        })
        .filter((item) => item.texts.length > 0);
    };
    const getGroup = (key: string) => {
      let group = groups.get(key);
      if (!group) {
        group = {
          kind: "assistant",
          id: key,
          segments: [],
        };
        groups.set(key, group);
        items.push(group);
      }
      return group;
    };
    const getSegment = (
      group: (typeof items)[number],
      kind: Segment["kind"],
      id: string
    ) => {
      let map = groupSegmentMaps.get(group.id);
      if (!map) {
        map = new Map();
        groupSegmentMaps.set(group.id, map);
      }
      const key = `${kind}:${id}`;
      let segment = map.get(key);
      if (!segment) {
        segment = { kind, id };
        map.set(key, segment);
        group.segments?.push(segment);
      }
      return segment;
    };
    const getMessageText = (content: unknown, role?: string) => {
      const text = collectMessageParts({ content, role }).textParts.join("\n").trim();
      return text || (typeof content === "string" ? content : "");
    };

    const getRunIdForUser = (userId?: string) => {
      if (!userId) return undefined;
      for (let i = runIdOrder.length - 1; i >= 0; i -= 1) {
        const runId = runIdOrder[i];
        if (!runId) continue;
        if (runIdToUserMessageId[runId] === userId) {
          return runId;
        }
      }
      return undefined;
    };

    for (const message of messages) {
      if (message.role === "user") {
        const userText = getTextFromContent(message.content);
        items.push({
          kind: "user",
          id: message.id,
          content: userText || safeStringify(message.content),
        });
        currentBatchKey = `user-${message.id}`;
        continue;
      }

      const runId = (message as { runId?: string }).runId;
      const latestUserId = currentBatchKey?.startsWith("user-")
        ? currentBatchKey.slice("user-".length)
        : undefined;
      const resolvedRunId =
        runId ??
        getRunIdForUser(latestUserId) ??
        currentRunIdRef.current ??
        runIdOrder[runIdOrder.length - 1] ??
        latestThinkingRunId ??
        undefined;
      const key = resolvedRunId ?? currentBatchKey ?? message.id;
      const group = getGroup(key);
      if (message.role === "activity") {
        if (message.activityType === A2UIMessageRenderer.activityType) {
          const activity = message as ActivityMessageLike;
          const segment = getSegment(group, "activity", activity.id ?? message.id);
          segment.activity = activity;
        }
        continue;
      }
      const messageType = (message as { messageType?: string }).messageType;
      if (messageType === "thinking_message") {
        const text = getMessageText(message.content, "assistant");
        if (text) {
          const segment = getSegment(group, "reasoning", message.id);
          segment.text = segment.text ? `${segment.text}\n${text}` : text;
        }
        continue;
      }
      if (message.role === "tool") {
        const toolCallId = (message as { toolCallId?: string }).toolCallId;
        const text = getMessageText(message.content, "tool");
        const segment = getSegment(group, "tool_result", toolCallId ?? message.id);
        segment.toolCallId = toolCallId;
        if (text) {
          segment.text = segment.text ? `${segment.text}\n${text}` : text;
        }
        continue;
      }
      const toolCalls = (message as { toolCalls?: unknown[] }).toolCalls;
      if (Array.isArray(toolCalls)) {
        for (const toolCall of toolCalls) {
          const toolCallId =
            toolCall && typeof toolCall === "object" && "id" in toolCall
              ? String((toolCall as { id?: string }).id ?? "")
              : "";
          const segment = getSegment(
            group,
            "tool_call",
            toolCallId || `${message.id}-tool-${group.segments?.length ?? 0}`
          );
          segment.toolCall = toolCall;
          segment.toolCallId = toolCallId || segment.toolCallId;
        }
      }
      const text = getMessageText(message.content, "assistant");
      if (text) {
        const segment = getSegment(group, "text", message.id);
        segment.text = segment.text ? `${segment.text}\n${text}` : text;
      }
    }

    const thinkingRunIds = new Set([
      ...Object.keys(completedThinkingByRunId),
      ...Object.keys(liveThinkingByRunId),
    ]);

    for (const runId of thinkingRunIds) {
      const thinkingItems = getThinkingItems(runId);
      if (thinkingItems.length === 0) continue;
      if (groups.has(runId)) {
        const existing = groups.get(runId);
        if (existing && existing.kind === "assistant") {
          thinkingItems.forEach((thinking) => {
            const segment = getSegment(existing, "reasoning", thinking.id);
            const text = thinking.texts.join("\n").trim();
            if (text) {
              segment.text = segment.text ? `${segment.text}\n${text}` : text;
            }
          });
        }
        continue;
      }
      const targetUserId = runIdToUserMessageId[runId];
      if (targetUserId) {
        const insertIndex = items.findIndex(
          (item) => item.kind === "user" && item.id === targetUserId
        );
        if (insertIndex !== -1) {
          const group = {
            kind: "assistant",
            id: runId,
            segments: [],
          } as (typeof items)[number];
          thinkingItems.forEach((thinking) => {
            const segment = getSegment(group, "reasoning", thinking.id);
            const text = thinking.texts.join("\n").trim();
            if (text) {
              segment.text = segment.text ? `${segment.text}\n${text}` : text;
            }
          });
          items.splice(insertIndex + 1, 0, group);
          continue;
        }
      }
      const group = {
        kind: "assistant",
        id: runId,
        segments: [],
      } as (typeof items)[number];
      thinkingItems.forEach((thinking) => {
        const segment = getSegment(group, "reasoning", thinking.id);
        const text = thinking.texts.join("\n").trim();
        if (text) {
          segment.text = segment.text ? `${segment.text}\n${text}` : text;
        }
      });
      items.push(group);
    }

    return items;
  }, [completedThinkingByRunId, liveThinkingByRunId, messages]);

  const roleConfig = useMemo(
    () => ({
      user: { name: "你" },
      assistant: { name: "助手" },
      system: { name: "系统" },
    }),
    []
  );

  const renderDialogueContentItem = useMemo(
    () => ({
      activity: (item: {
        content?: Record<string, unknown>;
        activityType?: string;
        message?: ActivityMessageLike;
      }) => {
        const activityType =
          typeof item?.activityType === "string" ? item.activityType : "";
        if (!activityType) return null;
        const content = item?.content ?? {};
        const message =
          item?.message ?? ({
            id: createThreadId(),
            role: "activity",
            content,
            activityType,
          } as ActivityMessageLike);
        return (
          <div className="rounded-xl border bg-white p-4">
            <ActivityRenderer
              activityType={activityType}
              content={content}
              message={message}
              agent={agent}
            />
          </div>
        );
      },
    }),
    [ActivityRenderer, agent]
  );

  const dialogueMessages = useMemo(() => {
    return groupedItems.map((item) => {
      if (item.kind === "user") {
        return {
          id: item.id,
          role: "user",
          content: item.content ?? "",
        };
      }
      const contentItems: Array<Record<string, unknown>> = [];
      const segments = item.segments ?? [];
      segments.forEach((segment) => {
        if (segment.kind === "reasoning") {
          if (!segment.text) return;
          contentItems.push({
            type: "reasoning",
            status: isLoading ? "in_progress" : "completed",
            summary: [
              {
                type: "summary_text",
                text: segment.text,
              },
            ],
          });
          return;
        }
        if (segment.kind === "tool_call") {
          const argumentsValue =
            typeof segment.toolCall === "string"
              ? segment.toolCall
              : safeStringify(segment.toolCall);
          contentItems.push({
            type: "function_call",
            name: "tool",
            arguments: argumentsValue,
            status: "completed",
            call_id: segment.toolCallId ?? segment.id,
          });
          return;
        }
        if (segment.kind === "tool_result") {
          if (!segment.text) return;
          contentItems.push({
            type: "message",
            content: [
              {
                type: "output_text",
                text: segment.text,
              },
            ],
            status: "completed",
          });
          return;
        }
        if (segment.kind === "text") {
          if (!segment.text) return;
          contentItems.push({
            type: "message",
            content: [
              {
                type: "output_text",
                text: segment.text,
              },
            ],
            status: "completed",
          });
          return;
        }
        if (segment.kind === "activity") {
          const activity = segment.activity;
          if (!activity) return;
          contentItems.push({
            type: "activity",
            activityType: activity.activityType,
            content: activity.content,
            message: activity,
            id: activity.id ?? `${item.id}-activity`,
          });
        }
      });
      return {
        id: item.id,
        role: "assistant",
        content: contentItems.length > 0 ? contentItems : "",
        status: isLoading ? "in_progress" : "completed",
      };
    });
  }, [groupedItems, isLoading]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 overflow-auto p-6">
        {historyHasMore && (
          <div className="flex justify-center">
            <button
              type="button"
              disabled={historyLoading}
              onClick={() => {
                isPrependingRef.current = true;
                fetchHistoryPage(historyPage + 1, false);
              }}
              className="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {historyLoading ? "加载中..." : "加载更多"}
            </button>
          </div>
        )}
        {historyError && (
          <div className="text-sm text-red-500">{historyError}</div>
        )}
        {dialogueMessages.length === 0 ? (
          <div className="text-sm text-gray-400">
            {historyLoading ? "加载中..." : "开始对话吧"}
          </div>
        ) : (
          <AIChatDialogue
            chats={dialogueMessages}
            roleConfig={roleConfig}
            renderDialogueContentItem={renderDialogueContentItem}
            hints={quickActionHints}
            onHintClick={handleQuickSend}
          />
        )}
      </div>
      <div className="sticky bottom-6 mx-6 mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-[0_16px_40px_rgba(15,23,42,0.12)] backdrop-blur">
        <AIChatInput
          placeholder="请输入你的问题"
          generating={isLoading}
          immediatelyRender={false}
          onMessageSend={handleInputSend}
          onStopGenerate={stopGeneration}
        />
      </div>
    </div>
  );
}

export default function ChatPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [resolvedThreadId, setResolvedThreadId] = useState<string | null>(null);
  const [copilotHeaders, setCopilotHeaders] = useState<
    Record<string, string> | undefined
  >(undefined);
  const createdThreadIdRef = useRef<string | null>(null);
  const getValidToken = useCallback(() => readValidToken(router), [router]);
  const threadIdFromUrl = useMemo(
    () => searchParams.get("threadId"),
    [searchParams]
  );

  useEffect(() => {
    const token = getValidToken();
    if (token) {
      const storedUser = readStoredUser();
      const storedUserId =
        typeof storedUser?.user_id === "string" || typeof storedUser?.user_id === "number"
          ? String(storedUser.user_id)
          : "";
      const userId = storedUserId || getUserIdFromToken(token);
      setCopilotHeaders(buildAuthHeaders(token, userId));
    } else {
      setCopilotHeaders(undefined);
    }
  }, [getValidToken]);

  useEffect(() => {
    if (threadIdFromUrl) {
      setResolvedThreadId(threadIdFromUrl);
      createdThreadIdRef.current = null;
      return;
    }
    if (!createdThreadIdRef.current) {
      createdThreadIdRef.current = createThreadId();
    }
    const newThreadId = createdThreadIdRef.current;
    if (!newThreadId) return;
    setResolvedThreadId(newThreadId);
    router.replace(`/chat?threadId=${newThreadId}`);
  }, [router, threadIdFromUrl]);

  if (!resolvedThreadId) {
    return null;
  }

  return (
    <CopilotKit
      runtimeUrl="/api/copilotkit"
      renderActivityMessages={activityRenderers}
      threadId={resolvedThreadId}
      headers={copilotHeaders}
    >
      <ChatContent />
    </CopilotKit>
  );
}
