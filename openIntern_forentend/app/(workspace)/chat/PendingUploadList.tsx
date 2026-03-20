"use client";

import { formatFileSize, type UploadAssetItem } from "./chat-helpers";

// 待发送附件展示独立成组件，聊天页只保留上传流程和状态。

export function PendingUploadList({
  items,
  uploading,
  onRemove,
}: {
  items: UploadAssetItem[];
  uploading: boolean;
  onRemove: (assetId: string) => void;
}) {
  if (items.length === 0 && !uploading) {
    return null;
  }

  return (
    <div className="mb-2 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-2">
      <div className="mb-2 flex items-center justify-between text-xs text-[var(--color-text-muted)]">
        <span>待发送附件</span>
        {uploading && <span>上传中...</span>}
      </div>
      <div className="flex flex-wrap gap-2">
        {items.map((asset) => (
          <span
            key={asset.id}
            className="inline-flex max-w-full items-center gap-1 rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-2 py-1 text-xs text-[var(--color-text-secondary)]"
          >
            <span className="font-medium text-[var(--color-text-primary)]">
              {asset.mediaKind}
            </span>
            <span className="max-w-[220px] truncate">{asset.fileName}</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">
              {formatFileSize(asset.size)}
            </span>
            <button
              type="button"
              className="rounded px-1 leading-none text-[var(--color-text-muted)] hover:bg-[var(--color-bg-overlay)]"
              onClick={() => onRemove(asset.id)}
              aria-label="删除附件"
            >
              ×
            </button>
          </span>
        ))}
      </div>
    </div>
  );
}
