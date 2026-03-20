"use client";

import { useMemo, type ReactNode } from "react";
import type { ActivityMessage } from "@ag-ui/client";
import {
  ACTIVITY_CONTENT_TYPE,
  TOOL_RESULT_TYPE,
  createMessageId,
  toRecord,
} from "./chat-helpers";
import { ToolResultCollapse } from "./ToolResultCollapse";

// 聊天消息的自定义渲染配置收口到 hook，避免页面文件继续堆叠渲染分支。

export function useChatDialogueRenderers(
  renderActivityMessage: (message: ActivityMessage) => ReactNode
) {
  return useMemo(
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
}
