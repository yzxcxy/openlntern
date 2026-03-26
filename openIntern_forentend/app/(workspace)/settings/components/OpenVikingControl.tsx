"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../../components/ui/UiButton";
import { readValidToken, requestBackend } from "../../auth";

type StatusResponse = {
  running?: boolean;
  status?: string;
  pid?: number;
  start_time?: string;
  uptime?: string;
  last_error?: string;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export function OpenVikingControl() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [actioning, setActioning] = useState(false);
  const [error, setError] = useState("");
  const router = useRouter();
  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const fetchStatus = useCallback(async () => {
    if (!getValidToken()) return;
    try {
      const data = await requestBackend<StatusResponse>("/v1/openviking/status", {
        fallbackMessage: "获取状态失败",
        router,
      });
      setStatus(data.data ?? null);
    } catch {
      // 忽略错误，保持当前状态
    } finally {
      setLoading(false);
    }
  }, [getValidToken, router]);

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  const handleStart = async () => {
    if (!getValidToken()) return;
    setActioning(true);
    setError("");
    try {
      await requestBackend("/v1/openviking/start", {
        method: "POST",
        fallbackMessage: "启动失败",
        router,
      });
      await fetchStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "启动失败");
    } finally {
      setActioning(false);
    }
  };

  const handleStop = async () => {
    if (!getValidToken()) return;
    setActioning(true);
    setError("");
    try {
      await requestBackend("/v1/openviking/stop", {
        method: "POST",
        fallbackMessage: "停止失败",
        router,
      });
      await fetchStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "停止失败");
    } finally {
      setActioning(false);
    }
  };

  const handleRestart = async () => {
    if (!getValidToken()) return;
    setActioning(true);
    setError("");
    try {
      await requestBackend("/v1/openviking/restart", {
        method: "POST",
        fallbackMessage: "重启失败",
        router,
      });
      await fetchStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "重启失败");
    } finally {
      setActioning(false);
    }
  };

  const statusColor = status?.running
    ? "bg-[rgba(82,196,26,0.15)] text-[#52c41a]"
    : "bg-[rgba(255,77,79,0.15)] text-[#ff4d4f]";

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
          OpenViking 进程控制
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          管理 OpenViking 服务的启动、停止和重启
        </p>
      </div>

      {error && (
        <div className="rounded-[var(--radius-lg)] bg-[rgba(255,77,79,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
          {error}
        </div>
      )}

      {/* Status Card */}
      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={joinClasses(
                "h-3 w-3 rounded-full",
                status?.running ? "bg-[#52c41a]" : "bg-[#ff4d4f]"
              )}
            />
            <span className="font-medium text-[var(--color-text-primary)]">
              {loading ? "加载中..." : status?.running ? "运行中" : "已停止"}
            </span>
          </div>
          <span
            className={joinClasses(
              "rounded-full px-2.5 py-1 text-xs font-medium",
              statusColor
            )}
          >
            {status?.status ?? "未知"}
          </span>
        </div>

        {status?.running && (
          <div className="mt-4 grid gap-3 text-sm sm:grid-cols-3">
            <div>
              <span className="text-[var(--color-text-muted)]">PID</span>
              <div className="font-medium text-[var(--color-text-primary)]">
                {status.pid ?? "-"}
              </div>
            </div>
            <div>
              <span className="text-[var(--color-text-muted)]">运行时间</span>
              <div className="font-medium text-[var(--color-text-primary)]">
                {status.uptime ?? "-"}
              </div>
            </div>
            <div>
              <span className="text-[var(--color-text-muted)]">启动时间</span>
              <div className="font-medium text-[var(--color-text-primary)]">
                {status.start_time
                  ? new Date(status.start_time).toLocaleString("zh-CN")
                  : "-"}
              </div>
            </div>
          </div>
        )}

        {status?.last_error && (
          <div className="mt-4 rounded-[var(--radius-md)] bg-[rgba(255,77,79,0.08)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            错误: {status.last_error}
          </div>
        )}
      </div>

      {/* Control Buttons */}
      <div className="flex flex-wrap gap-3">
        <UiButton
          onClick={handleStart}
          disabled={actioning || status?.running}
          variant={status?.running ? "secondary" : "primary"}
        >
          <svg
            className="h-4 w-4"
            viewBox="0 0 24 24"
            fill="currentColor"
          >
            <path d="M8 5v14l11-7z" />
          </svg>
          启动
        </UiButton>
        <UiButton
          onClick={handleStop}
          disabled={actioning || !status?.running}
          variant="secondary"
        >
          <svg
            className="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <rect x="6" y="6" width="12" height="12" />
          </svg>
          停止
        </UiButton>
        <UiButton
          onClick={handleRestart}
          disabled={actioning}
          variant="secondary"
        >
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
          重启
        </UiButton>
      </div>

      <div className="rounded-[var(--radius-lg)] bg-[var(--color-bg-page)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
        <strong>注意:</strong> 修改 OpenViking 服务配置后，需要重启服务才能生效。
      </div>
    </div>
  );
}