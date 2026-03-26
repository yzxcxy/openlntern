"use client";

import { useState } from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";

type AgentConfig = {
  max_iterations?: number;
};

type Props = {
  config?: AgentConfig;
  onSave: (updates: Record<string, unknown>) => void;
  saving: boolean;
};

export function AgentSettings({ config, onSave, saving }: Props) {
  const [maxIterations, setMaxIterations] = useState(
    config?.max_iterations?.toString() ?? "10"
  );

  const handleSave = () => {
    const iterations = parseInt(maxIterations, 10);
    if (isNaN(iterations) || iterations < 1) {
      return;
    }
    onSave({ max_iterations: iterations });
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
          Agent 配置
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          配置 Agent 的运行参数
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            最大迭代次数
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            Agent 在单次对话中执行工具调用的最大次数。设置过小可能导致任务无法完成，过大可能增加响应时间。
          </p>
          <div className="mt-2 max-w-xs">
            <UiInput
              type="number"
              value={maxIterations}
              onChange={(e) => setMaxIterations(e.target.value)}
              min={1}
              max={100}
            />
          </div>
        </div>
      </div>

      <div className="flex justify-end">
        <UiButton onClick={handleSave} disabled={saving}>
          {saving ? "保存中..." : "保存"}
        </UiButton>
      </div>
    </div>
  );
}