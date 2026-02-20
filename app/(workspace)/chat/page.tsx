"use client";

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
import { RunFold } from "./components/RunFold";
import { MarkdownMessage } from "./components/MarkdownMessage";

const A2UIMessageRenderer = createA2UIMessageRenderer({ theme });
const activityRenderers = [A2UIMessageRenderer];
const API_BASE = "/api/backend";
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
      } else if (role === "system" || role === "developer") {
        pushValue(reasoningParts, item);
      } else {
        pushValue(textParts, item);
      }
      continue;
    }
    if (typeof item === "number" || typeof item === "boolean") {
      const value = String(item);
      if (role === "tool") {
        pushValue(toolParts, value);
      } else if (role === "system" || role === "developer") {
        pushValue(reasoningParts, value);
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
      } else if (role === "system" || role === "developer") {
        pushValue(reasoningParts, value);
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
function ChatContent({ isNewThread }: { isNewThread: boolean }) {
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
  const [inputValue, setInputValue] = useState("");
  const [liveThinkingByRunId, setLiveThinkingByRunId] = useState<
    Record<string, string>
  >({});
  const [completedThinkingByRunId, setCompletedThinkingByRunId] = useState<
    Record<string, string[]>
  >({});
  const previousThreadId = useRef<string | undefined>(threadId);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const isPrependingRef = useRef(false);
  const messagesRef = useRef(messages);
  const thinkingBufferRef = useRef("");
  const currentRunIdRef = useRef<string | null>(null);
  const router = useRouter();
  const ActivityRenderer = A2UIMessageRenderer.render;

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
        thinkingBufferRef.current = "";
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_START) {
        thinkingBufferRef.current = "";
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_CONTENT) {
        const delta = typeof event.delta === "string" ? event.delta : "";
        if (!delta) return;
        thinkingBufferRef.current += delta;
        const runId = currentRunIdRef.current;
        if (!runId) return;
        setLiveThinkingByRunId((prev) =>
          prev[runId] === thinkingBufferRef.current
            ? prev
            : { ...prev, [runId]: thinkingBufferRef.current }
        );
        return;
      }
      if (event.type === EventType.THINKING_TEXT_MESSAGE_END) {
        const runId = currentRunIdRef.current;
        const value = thinkingBufferRef.current;
        thinkingBufferRef.current = "";
        if (!runId || !value) return;
        setCompletedThinkingByRunId((prev) => ({
          ...prev,
          [runId]: [...(prev[runId] ?? []), value],
        }));
        setLiveThinkingByRunId((prev) => ({ ...prev, [runId]: "" }));
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
        roleCandidate === "tool"
          ? roleCandidate
          : roleCandidate === "reasoning"
          ? "system"
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

  useEffect(() => {
    if (isPrependingRef.current) {
      isPrependingRef.current = false;
      return;
    }
    bottomRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages, isLoading]);

  const quickActions = [
    { label: "菜单", value: "给我推荐几道今日菜单" },
    { label: "两人餐", value: "我想要两人套餐，包含荤素搭配" },
    { label: "忌口", value: "请避开辣味和海鲜" },
  ];

  const handleSend = async () => {
    const content = inputValue.trim();
    if (!content) return;
    setInputValue("");
    await sendMessage({
      id: createThreadId(),
      role: "user",
      content,
    });
    window.dispatchEvent(new Event("threads-refresh"));
  };

  const groupedItems = useMemo(() => {
    const items: Array<{
      kind: "user" | "assistant";
      id: string;
      content?: string;
      textParts?: string[];
      toolParts?: string[];
      reasoningParts?: string[];
      activities?: ActivityMessageLike[];
    }> = [];
    const groups = new Map<string, (typeof items)[number]>();
    let currentBatchKey: string | null = null;
    let lastAssistantGroupId: string | null = null;

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
      const key = runId ?? currentBatchKey ?? message.id;

      let group = groups.get(key);
      if (!group) {
        group = {
          kind: "assistant",
          id: key,
          textParts: [],
          toolParts: [],
          reasoningParts: [],
          activities: [],
        };
        groups.set(key, group);
        items.push(group);
      }
      lastAssistantGroupId = group.id;

      if (message.role === "activity") {
        if (message.activityType === A2UIMessageRenderer.activityType) {
          group.activities?.push(message as ActivityMessageLike);
        }
        continue;
      }

      const parts = collectMessageParts(message);
      group.textParts?.push(...parts.textParts);
      group.toolParts?.push(...parts.toolParts);
      group.reasoningParts?.push(...parts.reasoningParts);
    }

    for (const item of items) {
      if (item.kind !== "assistant") continue;
      const historyThinking = completedThinkingByRunId[item.id] ?? [];
      const liveThinking = liveThinkingByRunId[item.id];
      const merged = [
        ...(item.reasoningParts ?? []),
        ...historyThinking,
        ...(liveThinking ? [liveThinking] : []),
      ];
      item.reasoningParts = merged.length > 0 ? merged : item.reasoningParts;
    }

    const thinkingRunIds = new Set([
      ...Object.keys(completedThinkingByRunId),
      ...Object.keys(liveThinkingByRunId),
    ]);

    for (const runId of thinkingRunIds) {
      if (groups.has(runId)) continue;
      const historyThinking = completedThinkingByRunId[runId] ?? [];
      const liveThinking = liveThinkingByRunId[runId];
      const mergedThinking = [
        ...historyThinking,
        ...(liveThinking ? [liveThinking] : []),
      ];
      if (mergedThinking.length === 0) continue;
      if (lastAssistantGroupId) {
        const group = groups.get(lastAssistantGroupId);
        if (group) {
          group.reasoningParts = [
            ...(group.reasoningParts ?? []),
            ...mergedThinking,
          ];
          continue;
        }
      }
      items.push({
        kind: "assistant",
        id: runId,
        textParts: [],
        toolParts: [],
        reasoningParts: mergedThinking,
        activities: [],
      });
    }

    return items;
  }, [completedThinkingByRunId, liveThinkingByRunId, messages]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 space-y-4 overflow-auto p-6 pb-36 scroll-pb-[140px]">
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
        {messages.length === 0 ? (
          <div className="text-sm text-gray-400">
            {historyLoading ? "加载中..." : "开始对话吧"}
          </div>
        ) : (
          groupedItems.map((item, index) => {
            if (item.kind === "user") {
              return (
                <div key={`user-${item.id}-${index}`} className="flex justify-end">
                  <div className="max-w-[80%] rounded-2xl bg-gray-900 px-4 py-3 text-sm text-white">
                    <MarkdownMessage
                      variant="user"
                      content={item.content ?? ""}
                    />
                  </div>
                </div>
              );
            }

            const text = item.textParts?.join("\n").trim() ?? "";
            const activities = item.activities ?? [];
            return (
              <div key={`assistant-${item.id}-${index}`} className="flex justify-start">
                <div className="max-w-[80%] space-y-3">
                  <RunFold
                    toolItems={item.toolParts}
                    reasoningItems={item.reasoningParts}
                  />
                  {(text || activities.length > 0) && (
                    <div className="space-y-3">
                      {text && (
                        <div className="rounded-2xl bg-white px-4 py-3 text-sm text-gray-900 shadow">
                          <MarkdownMessage content={text} />
                        </div>
                      )}
                      {activities.map((message) => (
                        <div
                          key={message.id}
                          className="rounded-xl border bg-white p-4"
                        >
                          <ActivityRenderer
                            activityType={message.activityType}
                            content={message.content}
                            message={message}
                            agent={agent}
                          />
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            );
          })
        )}
        <div ref={bottomRef} className="h-28" />
      </div>
      <div className="sticky bottom-6 mx-6 mb-6 rounded-3xl border border-gray-200 bg-white/90 p-4 shadow-[0_16px_40px_rgba(15,23,42,0.12)] backdrop-blur">
        <div className="mb-3 flex flex-wrap gap-2">
          {quickActions.map((item) => (
            <button
              key={item.label}
              type="button"
              onClick={() => setInputValue(item.value)}
              className="rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-xs text-gray-700 hover:bg-gray-100"
            >
              {item.label}
            </button>
          ))}
        </div>
        <form
          className="flex items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50/60 p-2"
          onSubmit={(event) => {
            event.preventDefault();
            void handleSend();
          }}
        >
          <textarea
            value={inputValue}
            onChange={(event) => setInputValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key !== "Enter" || event.shiftKey) return;
              event.preventDefault();
              void handleSend();
            }}
            placeholder="请输入你的问题"
            rows={2}
            className="flex-1 resize-none rounded-xl border border-transparent bg-white px-3 py-2 text-sm leading-5 text-gray-900 outline-none placeholder:text-gray-400 focus:border-gray-200 focus:ring-2 focus:ring-gray-200"
          />
          <div className="flex items-center gap-2">
            <button
              type="submit"
              className="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-900 text-white shadow hover:bg-gray-800"
            >
              <svg
                className="h-5 w-5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M22 2L11 13" />
                <path d="M22 2L15 22 11 13 2 9 22 2z" />
              </svg>
            </button>
            {isLoading && (
              <button
                type="button"
                onClick={stopGeneration}
                className="flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 bg-white text-gray-600 shadow-sm hover:bg-gray-50"
              >
                <svg
                  className="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <rect x="6.5" y="6.5" width="11" height="11" rx="2" />
                </svg>
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}

export default function ChatPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [resolvedThreadId, setResolvedThreadId] = useState<string | null>(null);
  const [isNewThread, setIsNewThread] = useState(false);
  const [copilotHeaders, setCopilotHeaders] = useState<
    Record<string, string> | undefined
  >(undefined);
  const createdThreadIdRef = useRef<string | null>(null);
  const getValidToken = useCallback(() => readValidToken(router), [router]);

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
    const existingThreadId = searchParams.get("threadId");
    const isNewParam = searchParams.get("new") === "1";
    if (existingThreadId) {
      setResolvedThreadId(existingThreadId);
      if (isNewParam) {
        createdThreadIdRef.current = existingThreadId;
        setIsNewThread(true);
        router.replace(`/chat?threadId=${existingThreadId}`);
        return;
      }
      if (createdThreadIdRef.current === existingThreadId) {
        setIsNewThread(true);
        return;
      }
      setIsNewThread(false);
      return;
    }
    const newThreadId = createThreadId();
    setResolvedThreadId(newThreadId);
    setIsNewThread(true);
    createdThreadIdRef.current = newThreadId;
    router.replace(`/chat?threadId=${newThreadId}&new=1`);
  }, [router, searchParams]);

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
      <ChatContent isNewThread={isNewThread} />
    </CopilotKit>
  );
}
