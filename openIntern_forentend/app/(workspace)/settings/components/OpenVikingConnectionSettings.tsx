"use client";

import { useState } from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";

type OpenVikingConfig = {
  base_url?: string;
  api_key?: string;
  skills_root?: string;
  tools_root?: string;
  timeout_seconds?: number;
  memory_search_timeout_seconds?: number;
  memory_sync_delay_seconds?: number;
  memory_sync_poll_seconds?: number;
  memory_sync_timeout_seconds?: number;
  memory_sync_retry_seconds?: number;
};

type Props = {
  config?: OpenVikingConfig;
  onSave: (updates: Record<string, unknown>) => void;
  saving: boolean;
};

export function OpenVikingConnectionSettings({ config, onSave, saving }: Props) {
  const [baseUrl, setBaseUrl] = useState(config?.base_url ?? "");
  const [apiKey, setApiKey] = useState("");
  const [skillsRoot, setSkillsRoot] = useState(config?.skills_root ?? "");
  const [toolsRoot, setToolsRoot] = useState(config?.tools_root ?? "");
  const [timeoutSeconds, setTimeoutSeconds] = useState(
    config?.timeout_seconds?.toString() ?? "600"
  );

  const handleSave = () => {
    const updates: Record<string, unknown> = {
      base_url: baseUrl,
      skills_root: skillsRoot,
      tools_root: toolsRoot,
      timeout_seconds: parseInt(timeoutSeconds, 10) || 600,
    };

    // 只有当用户输入了新的 API Key 时才更新
    if (apiKey.trim()) {
      updates.api_key = apiKey.trim();
    }

    onSave(updates);
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
          OpenViking 连接配置
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          配置后端与 OpenViking 服务的连接参数
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            服务地址
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            OpenViking 服务的 HTTP 地址
          </p>
          <div className="mt-2 max-w-md">
            <UiInput
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="http://127.0.0.1:1933"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            API Key
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            当前: {config?.api_key || "未设置"}。输入新值以更新，留空保持不变。
          </p>
          <div className="mt-2 max-w-md">
            <UiInput
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="输入新的 API Key"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            Skills 根路径
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            OpenViking 中存储 Agent Skills 的根 URI
          </p>
          <div className="mt-2 max-w-md">
            <UiInput
              value={skillsRoot}
              onChange={(e) => setSkillsRoot(e.target.value)}
              placeholder="viking://agent/skills"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            Tools 根路径
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            OpenViking 中存储工具的根 URI
          </p>
          <div className="mt-2 max-w-md">
            <UiInput
              value={toolsRoot}
              onChange={(e) => setToolsRoot(e.target.value)}
              placeholder="viking://resources/tools"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            请求超时时间（秒）
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            与 OpenViking 通信的 HTTP 请求超时时间
          </p>
          <div className="mt-2 max-w-xs">
            <UiInput
              type="number"
              value={timeoutSeconds}
              onChange={(e) => setTimeoutSeconds(e.target.value)}
              min={1}
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