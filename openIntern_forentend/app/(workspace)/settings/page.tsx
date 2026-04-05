"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { readValidToken, requestBackend } from "../auth";
import { AgentSettings } from "./components/AgentSettings";
import { AdvancedSettings } from "./components/AdvancedSettings";
import { SystemSettings } from "./components/SystemSettings";

type ConfigResponse = {
  agent?: {
    max_iterations?: number;
  };
  tools?: {
    sandbox?: {
      url?: string;
    };
    memory?: {
      provider?: string;
    };
  };
  context_compression?: {
    enabled?: boolean;
    soft_limit_tokens?: number;
    hard_limit_tokens?: number;
    output_reserve_tokens?: number;
    max_recent_messages?: number;
    estimated_chars_per_token?: number;
  };
  summary_llm?: {
    model?: string;
    api_key?: string;
    base_url?: string;
    provider?: string;
  };
  minio?: {
    endpoint?: string;
    access_key?: string;
    secret_key?: string;
    bucket?: string;
    use_ssl?: boolean;
    public_base_url?: string;
  };
  apmplus?: {
    host?: string;
    app_key?: string;
    service_name?: string;
    release?: string;
  };
};

type TabKey = "agent" | "advanced" | "system";

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export default function SettingsPage() {
  const [config, setConfig] = useState<ConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [activeTab, setActiveTab] = useState<TabKey>("agent");
  const router = useRouter();
  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const fetchConfig = useCallback(async () => {
    if (!getValidToken()) return;
    setLoading(true);
    setError("");
    try {
      const data = await requestBackend<ConfigResponse>("/v1/config", {
        fallbackMessage: "获取配置失败",
        router,
      });
      setConfig(data.data ?? null);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("获取配置失败");
      }
    } finally {
      setLoading(false);
    }
  }, [getValidToken, router]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async (section: string, updates: Record<string, unknown>) => {
    if (!getValidToken()) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await requestBackend("/v1/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ [section]: updates }),
        fallbackMessage: "更新配置失败",
        router,
      });
      setSuccess("配置保存成功");
      await fetchConfig();
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("更新配置失败");
      }
    } finally {
      setSaving(false);
    }
  };

  const handleReload = async () => {
    if (!getValidToken()) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await requestBackend("/v1/config/reload", {
        method: "POST",
        fallbackMessage: "重新加载配置失败",
        router,
      });
      setSuccess("配置已重新加载");
      await fetchConfig();
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("重新加载配置失败");
      }
    } finally {
      setSaving(false);
    }
  };

  const tabs: { key: TabKey; label: string }[] = [
    { key: "agent", label: "Agent 设置" },
    { key: "advanced", label: "高级设置" },
    { key: "system", label: "系统配置" },
  ];

  const renderTabContent = () => {
    switch (activeTab) {
      case "agent":
        return (
          <AgentSettings
            config={config?.agent}
            onSave={(updates) => handleSave("agent", updates)}
            saving={saving}
          />
        );
      case "advanced":
        return (
          <AdvancedSettings
            config={config?.context_compression}
            onSave={(updates) => handleSave("context_compression", updates)}
            saving={saving}
          />
        );
      case "system":
        return (
          <SystemSettings
            summaryLLM={config?.summary_llm}
            minio={config?.minio}
            apmplus={config?.apmplus}
            onSave={handleSave}
            saving={saving}
          />
        );
      default:
        return null;
    }
  };

  return (
    <div className="h-full w-full overflow-y-auto px-4 py-8">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">
              系统设置
            </h1>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">
              配置 Agent 和系统参数
            </p>
          </div>
          <UiButton variant="secondary" onClick={handleReload} disabled={saving}>
            <svg
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
              <path d="M21 3v5h-5" />
              <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
              <path d="M8 16H3v5" />
            </svg>
            重新加载配置
          </UiButton>
        </div>

        {/* Error/Success Messages */}
        {error && (
          <div className="rounded-[var(--radius-lg)] bg-[rgba(255,77,79,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
            {error}
          </div>
        )}
        {success && (
          <div className="rounded-[var(--radius-lg)] bg-[rgba(82,196,26,0.08)] px-4 py-3 text-sm text-[var(--color-state-success)]">
            {success}
          </div>
        )}

        {/* Tabs */}
        <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] shadow-[var(--shadow-sm)]">
          <div className="border-b border-[var(--color-border-default)] px-4">
            <nav className="flex gap-1 overflow-x-auto py-2" role="tablist">
              {tabs.map((tab) => (
                <button
                  key={tab.key}
                  role="tab"
                  aria-selected={activeTab === tab.key}
                  onClick={() => setActiveTab(tab.key)}
                  className={joinClasses(
                    "whitespace-nowrap rounded-lg px-4 py-2 text-sm font-medium transition-colors",
                    activeTab === tab.key
                      ? "bg-[var(--color-action-primary)] text-white"
                      : "text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]"
                  )}
                >
                  {tab.label}
                </button>
              ))}
            </nav>
          </div>

          <div className="p-6">
            {loading ? (
              <div className="py-8 text-center text-sm text-[var(--color-text-muted)]">
                加载中...
              </div>
            ) : (
              renderTabContent()
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
