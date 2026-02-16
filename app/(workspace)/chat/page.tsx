"use client";

import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import { CopilotKit, useCopilotChatInternal } from "@copilotkit/react-core";
import { useCopilotKit } from "@copilotkit/react-core/v2";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { theme } from "../../theme";

const A2UIMessageRenderer = createA2UIMessageRenderer({ theme });
const activityRenderers = [A2UIMessageRenderer];
const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
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
  const [inputValue, setInputValue] = useState("");
  const previousThreadId = useRef<string | undefined>(threadId);
  const bottomRef = useRef<HTMLDivElement | null>(null);
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
      setMessages([]);
    }
  }, [threadId, setMessages]);

  useEffect(() => {
    if (historyLoaded || !threadId) return;

    const loadHistory = async () => {
      setHistoryLoaded(true);
      await new Promise((resolve) => setTimeout(resolve, 500));

      const textHistory = [
        {
          id: "h1",
          role: "user" as const,
          content: "菜单",
        },
        {
          id: "h2",
          role: "assistant" as const,
          content: "这是之前帮你生成的菜品列表界面。",
        },
      ];

      setMessages([...textHistory]);
    };

    loadHistory();
  }, [historyLoaded, setMessages, threadId]);

  useEffect(() => {
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
    setInputValue("");
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 space-y-4 overflow-auto p-6 pb-36 scroll-pb-[140px]">
        {messages.length === 0 ? (
          <div className="text-sm text-gray-400">开始对话吧</div>
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

  useEffect(() => {
    const existingThreadId = searchParams.get("threadId");
    if (existingThreadId) {
      setResolvedThreadId(existingThreadId);
      return;
    }
    const newThreadId = createThreadId();
    setResolvedThreadId(newThreadId);
    router.replace(`/chat?threadId=${newThreadId}`);
  }, [router, searchParams]);

  if (!resolvedThreadId) {
    return null;
  }

  return (
    <CopilotKit
      runtimeUrl="/api/copilotkit"
      renderActivityMessages={activityRenderers}
      threadId={resolvedThreadId}
    >
      <ChatContent />
    </CopilotKit>
  );
}
