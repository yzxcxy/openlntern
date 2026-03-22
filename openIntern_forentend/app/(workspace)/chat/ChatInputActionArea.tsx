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
      className={joinClasses(className, "chat-action-rail flex flex-wrap items-center justify-end gap-2")}
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => event.stopPropagation()}
    >
      {conversationMode === "chat" && (
        <div
          className="relative w-[320px] max-w-full rounded-full border border-[rgba(126,96,69,0.16)] bg-[rgba(255,252,247,0.82)] px-4 py-2.5 shadow-[inset_0_1px_0_rgba(255,255,255,0.56)] transition focus-within:border-[rgba(199,104,67,0.28)] focus-within:bg-[rgba(255,250,243,0.96)]"
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
        className="relative inline-flex h-10 w-10 items-center justify-center rounded-full border border-[rgba(126,96,69,0.16)] bg-[rgba(255,252,247,0.82)] text-[var(--color-text-secondary)] shadow-[inset_0_1px_0_rgba(255,255,255,0.56)] transition hover:border-[rgba(199,104,67,0.24)] hover:bg-[rgba(255,250,243,0.96)] hover:text-[var(--color-text-primary)] disabled:cursor-not-allowed disabled:opacity-50"
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
      <div className="flex items-center gap-2">{menuItem}</div>
    </div>
  );
}
