"use client";

import { useCallback, useState } from "react";
import { Collapsible, MarkdownRender } from "@douyinfe/semi-ui-19";
import {
  IconChevronDown,
  IconChevronUp,
  IconWrench,
} from "@douyinfe/semi-icons";

// 工具结果折叠块单独放到组件文件，避免聊天页继续堆叠展示细节。

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
        <IconWrench />
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
