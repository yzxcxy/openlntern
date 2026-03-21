"use client";

import { useCallback, useState } from "react";
import { Collapsible, MarkdownRender } from "@douyinfe/semi-ui-19";
import {
  IconChevronDown,
  IconChevronUp,
} from "@douyinfe/semi-icons";

// 工具结果折叠块单独放到组件文件，避免聊天页继续堆叠展示细节。

// 工具相关图标统一为线性控制杆风格，和思考态的节点图标保持同一套视觉语言。
function ToolGlyphIcon() {
  return (
    <span
      aria-hidden="true"
      className="chat-tool-glyph inline-flex h-5 w-5 items-center justify-center rounded-full"
    >
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
  );
}

export function ToolResultCollapse({ text }: { text: string }) {
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
        <ToolGlyphIcon />
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
