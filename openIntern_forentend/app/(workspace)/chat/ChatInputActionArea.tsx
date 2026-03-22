"use client";

import type { ReactNode } from "react";
import { UiSelect } from "../../components/ui/UiSelect";
import { joinClasses, type ModelCatalogOption } from "./chat-helpers";

// 输入框 action area 单独拆出，页面主文件只保留状态和回调编排。

export function ChatInputActionArea({
  className,
  menuItem,
  conversationMode,
  selectedModelOption,
  availableModels,
  selectedModelId,
  onModelChange,
  onOpenUploadPicker,
  uploadDisabled,
  pendingUploadCount,
}: {
  className: string;
  menuItem: ReactNode[];
  conversationMode: "chat" | "agent";
  selectedModelOption: ModelCatalogOption | null;
  availableModels: ModelCatalogOption[];
  selectedModelId: string;
  onModelChange: (nextModelId: string) => void;
  onOpenUploadPicker: () => void;
  uploadDisabled: boolean;
  pendingUploadCount: number;
}) {
  return (
    <div
      className={joinClasses(className, "flex items-center gap-2")}
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => event.stopPropagation()}
    >
      {conversationMode === "chat" && (
        <div
          className="ui-select-control--glass relative w-[280px] max-w-[70vw] rounded-full border border-transparent px-4 py-2 focus-within:border-[var(--color-action-primary)]"
          title={
            selectedModelOption
              ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
              : "请先配置模型"
          }
        >
          <span className="pointer-events-none block overflow-hidden pr-11 text-ellipsis whitespace-nowrap text-sm font-medium text-[var(--color-text-primary)]">
            {selectedModelOption
              ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
              : "请先配置模型"}
          </span>
          <UiSelect
            value={selectedModelId}
            onChange={(event) => onModelChange(event.target.value)}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={(event) => event.stopPropagation()}
            title={
              selectedModelOption
                ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
                : "请先配置模型"
            }
            className="absolute inset-0 h-full w-full cursor-pointer rounded-full opacity-0"
          >
            {availableModels.length === 0 ? (
              <option value="">请先配置模型</option>
            ) : (
              availableModels.map((item) => (
                <option key={item.model_id} value={item.model_id}>
                  {item.provider_name} / {item.model_name}
                </option>
              ))
            )}
          </UiSelect>
        </div>
      )}
      <button
        type="button"
        onClick={onOpenUploadPicker}
        disabled={uploadDisabled}
        className="relative inline-flex h-9 w-9 items-center justify-center rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] text-[var(--color-text-secondary)] transition hover:bg-[var(--color-bg-page)] disabled:cursor-not-allowed disabled:opacity-50"
        title="上传图片/文件/音频/视频"
        aria-label="上传图片/文件/音频/视频"
      >
        {/* 聊天输入区的上传入口改为图标按钮，避免文案挤占模型选择区域。 */}
        <svg
          viewBox="0 0 24 24"
          className="h-[18px] w-[18px]"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M12 16V5" />
          <path d="m7.5 9.5 4.5-4.5 4.5 4.5" />
          <path d="M5 19h14" />
        </svg>
        {pendingUploadCount > 0 && (
          <span className="absolute -right-1 -top-1 min-w-[18px] rounded-full bg-[var(--color-bg-page)] px-1 py-0.5 text-center text-[10px] leading-none text-[var(--color-text-muted)] shadow-sm">
            {pendingUploadCount}
          </span>
        )}
      </button>
      {menuItem}
    </div>
  );
}
