"use client";

import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import { CopilotKit, useCopilotChatInternal } from "@copilotkit/react-core";
import { useCopilotKit } from "@copilotkit/react-core/v2";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
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
const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
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
  const previousThreadId = useRef<string | undefined>(threadId);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const isPrependingRef = useRef(false);
  const messagesRef = useRef(messages);
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
    if (previousThreadId.current === undefined) {
      previousThreadId.current = threadId;
      return;
    }
    if (previousThreadId.current !== threadId) {
      previousThreadId.current = threadId;
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
      return JSON.parse(content) as { role?: string; content?: unknown };
    } catch {
      return {};
    }
  }, []);
  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const mapHistoryMessage = useCallback(
    (item: {
      msg_id?: string;
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
      const roleFromMeta =
        typeof metadata.role === "string" ? metadata.role : undefined;
      const isActivity = item.type === "activity" || Boolean(activityType);
      const payload = isActivity ? {} : parseMessagePayload(item.content);
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
          : "assistant";
      const role = isActivity ? "activity" : normalizedRole;
      const content = isActivity
        ? parseActivityContent(item.content ?? "")
        : (payload.content ?? "");
      return {
        id: item.msg_id ?? createThreadId(),
        role,
        content,
        ...(activityType ? { activityType } : {}),
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
    if (isNewThread || historyLoaded || !threadId) return;
    fetchHistoryPage(1, true);
  }, [fetchHistoryPage, historyLoaded, isNewThread, threadId]);

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
    await sendMessage({
      id: createThreadId(),
      role: "user",
      content,
    });
    window.dispatchEvent(new Event("threads-refresh"));
    setInputValue("");
  };

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
          messages.map((message) => {
            if (
              message.role === "activity" &&
              message.activityType === A2UIMessageRenderer.activityType
            ) {
              return (
                <div key={message.id} className="rounded-xl border bg-white p-4">
                  <ActivityRenderer
                    activityType={message.activityType}
                    content={message.content}
                    message={message}
                    agent={agent}
                  />
                </div>
              );
            }

            const isUser = message.role === "user";
            const content =
              typeof message.content === "string"
                ? message.content
                : JSON.stringify(message.content, null, 2);
            return (
              <div
                key={message.id}
                className={`flex ${isUser ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`max-w-[80%] whitespace-pre-wrap rounded-2xl px-4 py-3 text-sm ${
                    isUser ? "bg-gray-900 text-white" : "bg-white text-gray-900 shadow"
                  }`}
                >
                  {content}
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
