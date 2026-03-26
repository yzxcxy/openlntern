"use client";

import { useState } from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";

type ContextCompressionConfig = {
  enabled?: boolean;
  soft_limit_tokens?: number;
  hard_limit_tokens?: number;
  output_reserve_tokens?: number;
  max_recent_messages?: number;
  estimated_chars_per_token?: number;
};

type Props = {
  config?: ContextCompressionConfig;
  onSave: (updates: Record<string, unknown>) => void;
  saving: boolean;
};

export function AdvancedSettings({ config, onSave, saving }: Props) {
  const [enabled, setEnabled] = useState(config?.enabled ?? true);
  const [softLimit, setSoftLimit] = useState(
    config?.soft_limit_tokens?.toString() ?? "100000"
  );
  const [hardLimit, setHardLimit] = useState(
    config?.hard_limit_tokens?.toString() ?? "120000"
  );
  const [outputReserve, setOutputReserve] = useState(
    config?.output_reserve_tokens?.toString() ?? "4000"
  );
  const [maxRecent, setMaxRecent] = useState(
    config?.max_recent_messages?.toString() ?? "3"
  );
  const [charsPerToken, setCharsPerToken] = useState(
    config?.estimated_chars_per_token?.toString() ?? "1"
  );

  const handleSave = () => {
    onSave({
      enabled,
      soft_limit_tokens: parseInt(softLimit, 10) || 100000,
      hard_limit_tokens: parseInt(hardLimit, 10) || 120000,
      output_reserve_tokens: parseInt(outputReserve, 10) || 4000,
      max_recent_messages: parseInt(maxRecent, 10) || 3,
      estimated_chars_per_token: parseInt(charsPerToken, 10) || 1,
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
          高级设置
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          配置上下文压缩和其他高级参数
        </p>
      </div>

      {/* Context Compression */}
      <div className="space-y-4">
        <h4 className="font-medium text-[var(--color-text-primary)]">
          上下文压缩
        </h4>
        <p className="text-sm text-[var(--color-text-muted)]">
          当对话上下文接近模型限制时，自动压缩历史消息以节省 token。
        </p>

        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="cc-enabled"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="h-4 w-4 rounded border-[var(--color-border-default)] text-[var(--color-action-primary)] focus:ring-[var(--color-action-primary)]"
          />
          <label
            htmlFor="cc-enabled"
            className="text-sm font-medium text-[var(--color-text-secondary)]"
          >
            启用上下文压缩
          </label>
        </div>

        {enabled && (
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
                软限制 (tokens)
              </label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                超过此值时开始考虑压缩
              </p>
              <div className="mt-2 max-w-xs">
                <UiInput
                  type="number"
                  value={softLimit}
                  onChange={(e) => setSoftLimit(e.target.value)}
                  min={1000}
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
                硬限制 (tokens)
              </label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                超过此值时强制压缩
              </p>
              <div className="mt-2 max-w-xs">
                <UiInput
                  type="number"
                  value={hardLimit}
                  onChange={(e) => setHardLimit(e.target.value)}
                  min={1000}
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
                输出保留 (tokens)
              </label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                压缩时为模型输出预留的 token 数
              </p>
              <div className="mt-2 max-w-xs">
                <UiInput
                  type="number"
                  value={outputReserve}
                  onChange={(e) => setOutputReserve(e.target.value)}
                  min={100}
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
                最大最近消息数
              </label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                压缩时保留的最近完整消息数
              </p>
              <div className="mt-2 max-w-xs">
                <UiInput
                  type="number"
                  value={maxRecent}
                  onChange={(e) => setMaxRecent(e.target.value)}
                  min={1}
                  max={20}
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
                每Token字符数估计
              </label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                用于估算文本 token 数的字符比例
              </p>
              <div className="mt-2 max-w-xs">
                <UiInput
                  type="number"
                  value={charsPerToken}
                  onChange={(e) => setCharsPerToken(e.target.value)}
                  min={1}
                  max={10}
                />
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="flex justify-end">
        <UiButton onClick={handleSave} disabled={saving}>
          {saving ? "保存中..." : "保存"}
        </UiButton>
      </div>
    </div>
  );
}