"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { ActivityMessage } from "@ag-ui/client";
import { IconLoading } from "@douyinfe/semi-icons";
import {
  ACTIVITY_CONTENT_TYPE,
  PROCESS_PANEL_TYPE,
  TOOL_RESULT_TYPE,
  createMessageId,
  toRecord,
} from "./chat-helpers";
import { ToolResultCollapse } from "./ToolResultCollapse";

// 聊天消息的自定义渲染配置收口到 hook，避免页面文件继续堆叠渲染分支。

type ReasoningTextItem = {
  text?: string;
  type?: string;
};

type ReasoningRendererProps = {
  status?: string;
  summary?: ReasoningTextItem[];
  content?: ReasoningTextItem[];
};

type FunctionCallItem = {
  name?: string;
  arguments?: string;
  status?: string;
};

type ProcessPanelItem = {
  collapseOnOutput?: boolean;
  items?: Array<Record<string, any>>;
  status?: string;
};

function ProcessPanelStatusIcon({ status }: { status?: string }) {
  const isCompleted = status === "completed";

  return (
    <span
      aria-hidden="true"
      className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-[rgba(126,96,69,0.08)] text-[var(--color-text-secondary)]"
    >
      {isCompleted ? (
        <svg
          viewBox="0 0 20 20"
          fill="none"
          className="h-3.5 w-3.5"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M5.5 10.2L8.4 13.1L14.5 7"
            stroke="currentColor"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      ) : (
        <IconLoading className="text-sm animate-spin" />
      )}
    </span>
  );
}

// 用图形徽标替代“思”字，避免状态图标和文案都依赖文字表达。
function ReasoningStatusIcon({ status }: { status?: string }) {
  const isCompleted = status === "completed";

  return (
    <span
      aria-hidden="true"
      className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-[rgba(37,99,255,0.12)] text-[var(--color-action-primary)]"
    >
      {isCompleted ? (
        <svg
          viewBox="0 0 20 20"
          fill="none"
          className="h-3.5 w-3.5"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M6.2 6.2H13.8M7.3 7.3L9.1 11.1M12.7 7.3L10.9 11.1"
            stroke="currentColor"
            strokeWidth="1.6"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <circle cx="6" cy="6" r="1.7" fill="currentColor" />
          <circle cx="14" cy="6" r="1.7" fill="currentColor" />
          <circle cx="10" cy="13" r="1.9" fill="currentColor" />
        </svg>
      ) : (
        <IconLoading className="animate-spin text-sm" />
      )}
    </span>
  );
}

function ReasoningContentItem({
  status,
  summary,
  content,
}: ReasoningRendererProps) {
  const isCompleted = status === "completed";
  // 折叠状态只保留一份，避免运行态切到完成态时被重置为默认关闭。
  const [isOpen, setIsOpen] = useState(!isCompleted);

  const text = useMemo(() => {
    const source = Array.isArray(summary) && summary.length > 0 ? summary : content;
    return Array.isArray(source)
      ? source
          .map((item) => item?.text ?? "")
          .filter((item) => item.length > 0)
          .join("\n")
      : "";
  }, [content, summary]);

  const handleToggle = useCallback(() => {
    setIsOpen((current) => !current);
  }, []);

  useEffect(() => {
    if (isCompleted) {
      setIsOpen(false);
    }
  }, [isCompleted]);

  return (
    <div className="overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)]">
      <button
        type="button"
        onClick={handleToggle}
        className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left"
      >
        <span className="flex items-center gap-2">
          <ReasoningStatusIcon status={status} />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">
            {status === "completed" ? "思考完成" : "正在思考中..."}
          </span>
        </span>
        <span className="text-xs text-[var(--color-text-tertiary)]">
          {isOpen ? "收起" : "展开"}
        </span>
      </button>
      {isOpen ? (
        <div className="border-t border-[var(--color-border-default)] px-3 py-3 text-sm leading-6 whitespace-pre-wrap text-[var(--color-text-secondary)]">
          {text}
        </div>
      ) : null}
    </div>
  );
}

function ToolCallCollapse({ name, arguments: args, status }: FunctionCallItem) {
  const isCompleted = status === "completed";
  const hasArgs = typeof args === "string" && args.trim().length > 0;
  // 运行态默认展开，完成后沿用用户最后一次的显式选择。
  const [isOpen, setIsOpen] = useState(!isCompleted);
  const hasOpenContent = hasArgs && isOpen;

  const handleToggle = useCallback(() => {
    if (!hasArgs) return;
    setIsOpen((current) => !current);
  }, [hasArgs]);

  return (
    <div className="motion-safe-slide-up w-full">
      <div
        className="semi-ai-chat-dialogue-content-tool-call motion-safe-highlight w-full"
        onClick={handleToggle}
        role={hasArgs ? "button" : undefined}
        tabIndex={hasArgs ? 0 : -1}
      >
        <span className="chat-tool-glyph inline-flex h-5 w-5 items-center justify-center rounded-full">
          <svg
            viewBox="0 0 20 20"
            fill="none"
            className="h-3.5 w-3.5"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M6 3.5V16.5M10 3.5V16.5M14 3.5V16.5"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
            />
            <circle cx="6" cy="7" r="1.75" fill="currentColor" />
            <circle cx="10" cy="12" r="1.75" fill="currentColor" />
            <circle cx="14" cy="5.5" r="1.75" fill="currentColor" />
          </svg>
        </span>
        <span className="truncate">{name || "工具调用"}</span>
        <span className="ml-auto text-xs text-[var(--color-text-tertiary)]">
          {isCompleted ? "已完成" : "执行中"}
        </span>
      </div>
      {hasOpenContent ? (
        <div className="semi-ai-chat-dialogue-content-bubble mt-2 w-full px-3 py-3 text-sm whitespace-pre-wrap text-[var(--color-text-secondary)]">
          {args}
        </div>
      ) : null}
    </div>
  );
}

function ProcessPanelCollapse({ items, status, collapseOnOutput }: ProcessPanelItem) {
  const processItems = Array.isArray(items) ? items : [];
  const isCompleted = status === "completed";
  // 面板默认跟随运行态展开，只在助手正文真正到达后再自动收起。
  const [isOpen, setIsOpen] = useState(!isCompleted);
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const shouldStickToBottomRef = useRef(true);

  const handleToggle = useCallback(() => {
    setIsOpen((current) => {
      const next = !current;
      if (next) {
        shouldStickToBottomRef.current = true;
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (collapseOnOutput) {
      setIsOpen(false);
    }
  }, [collapseOnOutput]);

  const processLabel = useMemo(() => {
    const toolCount = processItems.filter((item) => item?.type === "function_call").length;
    const reasoningCount = processItems.filter((item) => item?.type === "reasoning").length;
    const labels = [];
    if (reasoningCount > 0) {
      labels.push(`${reasoningCount} 个思考`);
    }
    if (toolCount > 0) {
      labels.push(`${toolCount} 个工具`);
    }
    return labels.length > 0 ? labels.join("，") : `${processItems.length} 个过程项`;
  }, [processItems]);

  // 运行过程容器保持固定宽高，避免内部思考块或工具块开合时整体尺寸抖动。
  const bodyClassName = "h-72";
  const headerClassName =
    "flex w-full items-center justify-between gap-3 px-4 py-3 text-left";
  const handleBodyScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) {
      return;
    }
    const distanceFromBottom =
      container.scrollHeight - container.scrollTop - container.clientHeight;
    shouldStickToBottomRef.current = distanceFromBottom <= 48;
  }, []);

  const getProcessItemKey = useCallback((item: Record<string, any>, index: number) => {
    const itemType = String(item?.type ?? "process");
    const itemId =
      typeof item?.id === "string" && item.id.length > 0
        ? item.id
        : typeof item?.call_id === "string" && item.call_id.length > 0
          ? item.call_id
          : "";
    return itemId ? `${itemType}-${itemId}` : `${itemType}-${index}`;
  }, []);

  useEffect(() => {
    if (!isOpen) {
      return;
    }
    const container = scrollContainerRef.current;
    if (!container || !shouldStickToBottomRef.current) {
      return;
    }
    // 仅在用户停留在底部附近时自动跟随，避免手动回看过程时被强制拉回。
    container.scrollTop = container.scrollHeight;
  }, [isOpen, processItems, status]);

  return (
    <div
      className="w-full overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.92)] shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]"
    >
      <button
        type="button"
        onClick={handleToggle}
        className={headerClassName}
      >
        <span className="flex min-w-0 items-center gap-2">
          <ProcessPanelStatusIcon status={status} />
          <span className="text-sm font-normal text-[var(--color-text-tertiary)]">
            {isCompleted ? "执行过程" : "正在执行"}
          </span>
          <span className="truncate text-xs font-normal text-[var(--color-text-quaternary,#94A3B8)]">
            {processLabel}
          </span>
        </span>
        <span className="shrink-0 text-xs text-[var(--color-text-quaternary,#94A3B8)]">
          {isOpen ? "收起" : "展开"}
        </span>
      </button>
      {isOpen ? (
        <div
          ref={scrollContainerRef}
          onScroll={handleBodyScroll}
          className={`${bodyClassName} overflow-y-auto border-t border-[var(--color-border-default)] px-3 py-3`}
        >
          <div className="space-y-3">
          {processItems.map((item, index) => {
            const itemType = String(item?.type ?? "");
            if (itemType === "reasoning") {
              return (
                <ReasoningContentItem
                  key={getProcessItemKey(item, index)}
                  status={item?.status}
                  summary={Array.isArray(item?.summary) ? item.summary : []}
                  content={Array.isArray(item?.content) ? item.content : []}
                />
              );
            }
            if (itemType === "function_call") {
              return (
                <ToolCallCollapse
                  key={getProcessItemKey(item, index)}
                  name={item?.name}
                  arguments={item?.arguments}
                  status={item?.status}
                />
              );
            }
            if (itemType === TOOL_RESULT_TYPE) {
              return (
                <ToolResultCollapse
                  key={getProcessItemKey(item, index)}
                  text={String(item?.text ?? "")}
                />
              );
            }
            return null;
          })}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function useChatDialogueRenderers(
  renderActivityMessage: (message: ActivityMessage) => ReactNode
) {
  return useMemo(
    () => ({
      [PROCESS_PANEL_TYPE]: (item: ProcessPanelItem) => (
        <ProcessPanelCollapse
          items={item?.items}
          status={item?.status}
          collapseOnOutput={item?.collapseOnOutput}
        />
      ),
      reasoning: (item: {
        status?: string;
        summary?: ReasoningTextItem[];
        content?: ReasoningTextItem[];
      }) => (
        <ReasoningContentItem
          status={item?.status}
          summary={item?.summary}
          content={item?.content}
        />
      ),
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
            <div className="border-b border-[var(--color-border-default)] bg-[linear-gradient(90deg,rgba(199,104,67,0.08),rgba(209,157,86,0.06))] px-3 py-2">
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
