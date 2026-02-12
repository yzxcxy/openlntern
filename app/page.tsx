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

    // 模拟从后端加载一条包含 A2UI 的历史 ActivityMessage
    const loadHistory = async () => {
      // 这里模拟一个异步请求，例如 fetch("/api/chat/history")
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

      const activityMessage = {
        id: "msg_224331",
        role: "activity" as const,
        activityType: "a2ui-surface",
        content: {
          operations: [
            {
              surfaceUpdate: {
                components: [
                  {
                    component: {
                      Column: {
                        alignment: "stretch",
                        children: { explicitList: ["dishList"] },
                        distribution: "start",
                      },
                    },
                    id: "root",
                  },
                  {
                    component: {
                      List: {
                        alignment: "stretch",
                        children: {
                          template: {
                            componentId: "dishItem",
                            dataBinding: "/dishes",
                          },
                        },
                        direction: "vertical",
                      },
                    },
                    id: "dishList",
                  },
                  {
                    component: {
                      Card: {
                        child: "dishContent",
                      },
                    },
                    id: "dishItem",
                  },
                  {
                    component: {
                      Row: {
                        alignment: "center",
                        children: {
                          explicitList: ["dishImage", "dishDetails"],
                        },
                        distribution: "start",
                      },
                    },
                    id: "dishContent",
                  },
                  {
                    component: {
                      Image: {
                        fit: "cover",
                        url: { path: "/imageUrl" },
                        usageHint: "mediumFeature",
                      },
                    },
                    id: "dishImage",
                  },
                  {
                    component: {
                      Column: {
                        alignment: "start",
                        children: {
                          explicitList: ["dishName", "dishDescription"],
                        },
                        distribution: "start",
                      },
                    },
                    id: "dishDetails",
                  },
                  {
                    component: {
                      Text: {
                        text: { path: "/name" },
                        usageHint: "h3",
                      },
                    },
                    id: "dishName",
                  },
                  {
                    component: {
                      Text: {
                        text: { path: "/description" },
                        usageHint: "body",
                      },
                    },
                    id: "dishDescription",
                  },
                ],
                surfaceId: "default",
              },
            },
            {
              dataModelUpdate: {
                contents: [
                  {
                    key: "dishes",
                    valueMap: [
                      {
                        key: "dish0",
                        valueMap: [
                          { key: "name", valueString: "宫保鸡丁" },
                          { key: "description", valueString: "经典川菜，鲜香微辣" },
                          {
                            key: "imageUrl",
                            valueString:
                              "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s",
                          },
                        ],
                      },
                      {
                        key: "dish1",
                        valueMap: [
                          { key: "name", valueString: "鱼香肉丝" },
                          { key: "description", valueString: "酸甜鲜辣，开胃下饭" },
                          {
                            key: "imageUrl",
                            valueString:
                              "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s",
                          },
                        ],
                      },
                      {
                        key: "dish2",
                        valueMap: [
                          { key: "name", valueString: "麻婆豆腐" },
                          { key: "description", valueString: "麻辣鲜香，豆腐嫩滑" },
                          {
                            key: "imageUrl",
                            valueString:
                              "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s",
                          },
                        ],
                      },
                      {
                        key: "dish3",
                        valueMap: [
                          { key: "name", valueString: "拍黄瓜" },
                          { key: "description", valueString: "清爽解腻，脆嫩爽口" },
                          {
                            key: "imageUrl",
                            valueString:
                              "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTObmT7jUhWYIUW5nW0agmwc0X4pyFis_IBsw&s",
                          },
                        ],
                      },
                    ],
                  },
                ],
                surfaceId: "default",
              },
            },
            {
              beginRendering: {
                root: "root",
                surfaceId: "default",
              },
            },
          ],
        },
      } as const;

      // 把文本历史 + ActivityMessage 一次性追加到当前消息列表中
      agent.setMessages([...agent.messages, ...textHistory, activityMessage]);

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
