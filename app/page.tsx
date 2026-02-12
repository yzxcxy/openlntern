"use client";

import { CopilotKitProvider } from "@copilotkit/react-core/v2";
import { CopilotChat, useAgent } from "@copilotkit/react-core/v2";
import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import { useEffect, useState } from "react";
import { theme } from "./theme";

// Disable static optimization for this page
export const dynamic = "force-dynamic";

const A2UIMessageRenderer = createA2UIMessageRenderer({ theme });
const activityRenderers = [A2UIMessageRenderer];

function ChatWithMockHistory() {
  const { agent } = useAgent();
  const [historyLoaded, setHistoryLoaded] = useState(false);

  useEffect(() => {
    if (historyLoaded) return;

    // 模拟从后端加载历史聊天记录
    const loadHistory = async () => {
      // 这里模拟一个异步请求，例如 fetch("/api/chat/history")
      await new Promise((resolve) => setTimeout(resolve, 500));

      const mockHistory = [
        {
          id: "h1",
          role: "user" as const,
          content: "之前我问过：A2UI 可以做什么？",
        },
        {
          id: "h2",
          role: "assistant" as const,
          content: "A2UI 可以把智能体结果渲染成交互式 UI 组件。",
        },
      ];

      for (const msg of mockHistory) {
        agent.addMessage({
          id: msg.id,
          role: msg.role,
          content: msg.content,
        });
      }

      setHistoryLoaded(true);
    };

    loadHistory();
  }, [agent, historyLoaded]);

  return (
    <main
      className="h-full overflow-auto w-screen"
      style={{ minHeight: "100dvh" }}
    >
      <CopilotChat
        className="h-full"
        labels={{
          chatInputPlaceholder: "请输入你的问题哦",
        }}
      />
    </main>
  );
}

export default function Home() {
  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      showDevConsole="auto"
      renderActivityMessages={activityRenderers}
    >
      <ChatWithMockHistory />
    </CopilotKitProvider>
  );
}
