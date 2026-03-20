"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Content as AIChatInputContent } from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import {
  extractInputPlainText,
  matchMentionTrigger,
  stripMentionSuffix,
  type MentionSelectionItem,
  type MentionTargetOption,
  type MentionTargetType,
  type MentionTriggerSymbol,
} from "./chat-helpers";

// mention 相关状态机统一收口到 hook，页面只保留发送和业务编排。

export function useMentionComposer({
  mentionOptions,
  setComposerText,
}: {
  mentionOptions: MentionTargetOption[];
  setComposerText: (text: string) => void;
}) {
  const composerTextRef = useRef("");
  const [selectedMentions, setSelectedMentions] = useState<MentionSelectionItem[]>([]);
  const [mentionKeyword, setMentionKeyword] = useState("");
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionActiveIndex, setMentionActiveIndex] = useState(0);
  const [mentionTriggerSymbol, setMentionTriggerSymbol] =
    useState<MentionTriggerSymbol | null>(null);

  const mentionCandidates = useMemo(() => {
    if (!mentionOpen || !mentionTriggerSymbol) {
      return [];
    }
    const keyword = mentionKeyword.trim().toLowerCase();
    const targetType: MentionTargetType =
      mentionTriggerSymbol === "@" ? "kb" : "skill";
    return mentionOptions
      .filter((item) => item.type === targetType)
      .filter((item) => {
        if (!keyword) {
          return true;
        }
        return item.keyword.includes(keyword);
      })
      .filter((item) => {
        const key = `${item.type}:${item.id}`;
        return !selectedMentions.some(
          (selection) => `${selection.type}:${selection.id}` === key
        );
      })
      .slice(0, 8);
  }, [
    mentionKeyword,
    mentionOpen,
    mentionOptions,
    mentionTriggerSymbol,
    selectedMentions,
  ]);

  useEffect(() => {
    if (!mentionOpen || mentionCandidates.length === 0) {
      setMentionActiveIndex(0);
      return;
    }
    setMentionActiveIndex((current) =>
      Math.min(current, mentionCandidates.length - 1)
    );
  }, [mentionCandidates, mentionOpen]);

  const closeMentionMenu = useCallback(() => {
    setMentionOpen(false);
    setMentionKeyword("");
    setMentionTriggerSymbol(null);
    setMentionActiveIndex(0);
  }, []);

  const clearComposerDraft = useCallback(() => {
    composerTextRef.current = "";
  }, []);

  const handleInputContentChange = useCallback(
    (contents: AIChatInputContent[]) => {
      const text = extractInputPlainText(contents ?? []);
      composerTextRef.current = text;
      const trigger = matchMentionTrigger(text);
      if (!trigger) {
        closeMentionMenu();
        return;
      }
      setMentionTriggerSymbol(trigger.symbol);
      setMentionKeyword(trigger.keyword);
      setMentionOpen(true);
      setMentionActiveIndex(0);
    },
    [closeMentionMenu]
  );

  const handleMentionSelect = useCallback(
    (target: MentionTargetOption) => {
      setSelectedMentions((current) => {
        const key = `${target.type}:${target.id}`;
        if (current.some((item) => `${item.type}:${item.id}` === key)) {
          return current;
        }
        return [
          ...current,
          {
            type: target.type,
            id: target.id,
            name: target.name,
          },
        ];
      });
      const nextText = stripMentionSuffix(
        composerTextRef.current,
        mentionTriggerSymbol
      );
      composerTextRef.current = nextText;
      setComposerText(nextText);
      closeMentionMenu();
    },
    [closeMentionMenu, mentionTriggerSymbol, setComposerText]
  );

  // mention 下拉打开时拦截方向键与回车，支持纯键盘选择目标。
  const handleMentionKeyDownCapture = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (!mentionOpen || mentionCandidates.length === 0) {
        return;
      }
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setMentionActiveIndex((current) =>
          (current + 1) % mentionCandidates.length
        );
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setMentionActiveIndex((current) =>
          (current - 1 + mentionCandidates.length) % mentionCandidates.length
        );
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        const target =
          mentionCandidates[
            Math.max(0, Math.min(mentionActiveIndex, mentionCandidates.length - 1))
          ];
        if (target) {
          handleMentionSelect(target);
        }
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        closeMentionMenu();
      }
    },
    [
      closeMentionMenu,
      handleMentionSelect,
      mentionActiveIndex,
      mentionCandidates,
      mentionOpen,
    ]
  );

  const removeMentionSelection = useCallback((target: MentionSelectionItem) => {
    setSelectedMentions((current) =>
      current.filter(
        (item) => item.type !== target.type || item.id !== target.id
      )
    );
  }, []);

  const resetMentionComposer = useCallback(() => {
    setSelectedMentions([]);
    clearComposerDraft();
    closeMentionMenu();
  }, [clearComposerDraft, closeMentionMenu]);

  return {
    selectedMentions,
    mentionOpen,
    mentionActiveIndex,
    mentionTriggerSymbol,
    mentionCandidates,
    handleInputContentChange,
    handleMentionSelect,
    handleMentionKeyDownCapture,
    removeMentionSelection,
    closeMentionMenu,
    clearComposerDraft,
    resetMentionComposer,
    setMentionActiveIndex,
  };
}
