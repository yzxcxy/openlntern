"use client";

import type { ChangeEvent, RefObject } from "react";
import { PendingUploadList } from "./PendingUploadList";
import { joinClasses, type MentionSelectionItem, type MentionTargetOption, type MentionTriggerSymbol, type UploadAssetItem } from "./chat-helpers";

// 输入框上方的 mention 提示、已选上下文和待发送附件集中在这个组件中。

export function ChatComposerAssist({
  mentionOpen,
  mentionCandidates,
  mentionTriggerSymbol,
  mentionActiveIndex,
  onMentionHover,
  onMentionSelect,
  selectedMentions,
  onRemoveMention,
  uploadInputRef,
  onUploadInputChange,
  pendingUploads,
  uploading,
  onRemovePendingUpload,
}: {
  mentionOpen: boolean;
  mentionCandidates: MentionTargetOption[];
  mentionTriggerSymbol: MentionTriggerSymbol | null;
  mentionActiveIndex: number;
  onMentionHover: (index: number) => void;
  onMentionSelect: (item: MentionTargetOption) => void;
  selectedMentions: MentionSelectionItem[];
  onRemoveMention: (item: MentionSelectionItem) => void;
  uploadInputRef: RefObject<HTMLInputElement | null>;
  onUploadInputChange: (event: ChangeEvent<HTMLInputElement>) => void;
  pendingUploads: UploadAssetItem[];
  uploading: boolean;
  onRemovePendingUpload: (assetId: string) => void;
}) {
  return (
    <>
      {mentionOpen && mentionCandidates.length > 0 && (
        <div className="absolute bottom-full left-0 right-0 z-20 mb-2 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-md)]">
          <div className="border-b border-[var(--color-border-default)] px-3 py-2 text-xs text-[var(--color-text-muted)]">
            {mentionTriggerSymbol === "@"
              ? "当前选择：知识库（@）"
              : "当前选择：Skill（#）"}
          </div>
          {mentionCandidates.map((item, index) => (
            <button
              key={`${item.type}:${item.id}`}
              type="button"
              className={joinClasses(
                "flex w-full items-center justify-between px-3 py-2 text-left text-sm transition",
                mentionActiveIndex === index
                  ? "bg-[var(--color-bg-page)]"
                  : "hover:bg-[var(--color-bg-page)]"
              )}
              onMouseEnter={() => onMentionHover(index)}
              onClick={() => onMentionSelect(item)}
            >
              <span className="truncate text-[var(--color-text-primary)]">
                {item.displayName}
              </span>
              <span className="ml-3 shrink-0 rounded-full border border-[var(--color-border-default)] px-2 py-0.5 text-[10px] text-[var(--color-text-muted)]">
                {item.type === "skill" ? "Skill" : "知识库"}
              </span>
            </button>
          ))}
        </div>
      )}
      {selectedMentions.length > 0 && (
        <div className="mb-3 flex flex-wrap gap-2">
          {selectedMentions.map((item) => (
            <span
              key={`${item.type}:${item.id}`}
              className="inline-flex items-center gap-1.5 rounded-full border border-[rgba(126,96,69,0.14)] bg-[rgba(255,252,247,0.84)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)]"
            >
              <span className="font-medium text-[var(--color-text-primary)]">
                {item.type === "skill" ? "Skill" : "知识库"}
              </span>
              <span className="max-w-[220px] truncate">{item.name || item.id}</span>
              <button
                type="button"
                className="rounded px-1 leading-none text-[var(--color-text-muted)] hover:bg-[var(--color-bg-overlay)]"
                onClick={() => onRemoveMention(item)}
                aria-label="删除已选项"
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}
      <input
        ref={uploadInputRef}
        type="file"
        className="hidden"
        multiple
        onChange={onUploadInputChange}
      />
      <PendingUploadList
        items={pendingUploads}
        uploading={uploading}
        onRemove={onRemovePendingUpload}
      />
    </>
  );
}
