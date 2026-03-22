"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiConfirmDialog } from "../../components/ui/UiConfirmDialog";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { getUserIdFromToken, readValidToken } from "../auth";
import {
  type AgentListItem,
  disableAgent,
  enableAgent,
  listAgents,
  removeAgent,
} from "./agent-api";

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export default function AgentsPage() {
  const router = useRouter();
  const token = readValidToken(router);
  const userId = getUserIdFromToken(token);

  const [agents, setAgents] = useState<AgentListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<AgentListItem | null>(null);
  const [deleting, setDeleting] = useState(false);

  const requestContext = useMemo(
    () => ({
      router,
      userId,
    }),
    [router, userId]
  );

  const loadAgents = useCallback(async () => {
    if (!token) {
      return;
    }
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", "1");
      params.set("page_size", "200");
      if (keyword.trim()) {
        params.set("keyword", keyword.trim());
      }
      if (statusFilter) {
        params.set("status", statusFilter);
      }
      if (typeFilter) {
        params.set("agent_type", typeFilter);
      }
      const response = await listAgents(params, requestContext);
      setAgents(Array.isArray(response.data?.data) ? response.data.data : []);
      setTotal(typeof response.data?.total === "number" ? response.data.total : 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取 Agent 列表失败");
    } finally {
      setLoading(false);
    }
  }, [keyword, requestContext, statusFilter, token, typeFilter]);

  useEffect(() => {
    void loadAgents();
  }, [loadAgents]);

  const handleToggleEnabled = useCallback(
    async (item: AgentListItem, nextEnabled: boolean) => {
      try {
        if (nextEnabled) {
          await enableAgent(item.agent_id, requestContext);
        } else {
          await disableAgent(item.agent_id, requestContext);
        }
        await loadAgents();
      } catch (err) {
        setError(err instanceof Error ? err.message : "更新 Agent 状态失败");
      }
    },
    [loadAgents, requestContext]
  );

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) {
      return;
    }
    setDeleting(true);
    try {
      await removeAgent(deleteTarget.agent_id, requestContext);
      setDeleteTarget(null);
      await loadAgents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除 Agent 失败");
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget, loadAgents, requestContext]);

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="workspace-toolbar-surface rounded-[var(--radius-lg)] border p-4">
          <div className="grid items-center gap-3 grid-cols-[minmax(220px,1.4fr)_minmax(180px,1fr)_minmax(180px,1fr)_48px_auto]">
            <UiInput
              className="h-10"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="搜索名字或描述"
            />
            <UiSelect
              className="h-10"
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value)}
            >
              <option value="">全部状态</option>
              <option value="draft">Draft</option>
              <option value="enabled">Enabled</option>
              <option value="disabled">Disabled</option>
            </UiSelect>
            <UiSelect className="h-10" value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)}>
              <option value="">全部类型</option>
              <option value="single">Single</option>
              <option value="supervisor">Supervisor</option>
            </UiSelect>
            <UiButton
              variant="secondary"
              className="h-10 w-12 px-0 justify-self-center"
              onClick={() => void loadAgents()}
              disabled={loading}
              aria-label="刷新列表"
              title="刷新列表"
            >
              <svg
                className={joinClasses("h-4.5 w-4.5", loading && "animate-spin")}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M21 12a9 9 0 1 1-2.64-6.36" />
                <path d="M21 3v6h-6" />
              </svg>
            </UiButton>
            <UiButton className="h-10 px-4 justify-self-start" onClick={() => router.push("/agents/editor")}>
              创建 Agent
            </UiButton>
          </div>

          <div className="mt-3 text-xs text-[var(--color-text-muted)]">共 {total} 个 Agent</div>
          {error ? (
            <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(255,255,255,0.7)] px-3 py-2 text-sm text-[var(--color-danger)]">
              {error}
            </div>
          ) : null}
        </div>

        <div className="mt-5 grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
          {agents.map((item) => (
            <section
              key={item.agent_id}
              className="workspace-item-surface workspace-item-hover-lift rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-base font-semibold text-[var(--color-text-primary)]">{item.name}</span>
                    <span className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5 text-[11px] text-[var(--color-text-muted)]">
                      {item.agent_type === "supervisor" ? "Supervisor" : "Single"}
                    </span>
                    <span
                      className={joinClasses(
                        "rounded-full border px-2 py-0.5 text-[11px]",
                        item.status === "enabled"
                          ? "border-[rgba(22,163,74,0.22)] text-[rgb(22,163,74)]"
                          : item.status === "disabled"
                            ? "border-[rgba(234,88,12,0.22)] text-[rgb(234,88,12)]"
                            : "border-[var(--color-border-default)] text-[var(--color-text-muted)]"
                      )}
                    >
                      {item.status}
                    </span>
                  </div>
                  <div className="mt-1 line-clamp-2 text-sm text-[var(--color-text-secondary)]">
                    {item.description || "暂无描述"}
                  </div>
                </div>
                {item.avatar_url ? (
                  <img
                    src={item.avatar_url}
                    alt={item.name}
                    className="h-12 w-12 rounded-[14px] border border-[var(--color-border-default)] object-cover"
                  />
                ) : (
                  <div className="flex h-12 w-12 items-center justify-center rounded-[14px] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] text-xs font-semibold text-[var(--color-text-muted)]">
                    AG
                  </div>
                )}
              </div>

              <div className="mt-4 space-y-2 text-xs text-[var(--color-text-secondary)]">
                <div>默认模型：{item.default_model_name || item.default_model_id || "系统默认"}</div>
                <div className="flex flex-wrap gap-2">
                  <span>工具 {item.tool_count}</span>
                  <span>Skill {item.skill_count}</span>
                  <span>知识库 {item.knowledge_base_count}</span>
                  <span>SubAgent {item.sub_agent_count}</span>
                </div>
                <div>长期记忆：{item.agent_memory_enabled ? "开启" : "关闭"}</div>
              </div>

              <div className="mt-4 flex flex-wrap gap-2">
                <UiButton
                  variant="secondary"
                  size="sm"
                  onClick={() => router.push(`/agents/editor?agent_id=${encodeURIComponent(item.agent_id)}`)}
                >
                  编辑
                </UiButton>
                {item.status === "enabled" ? (
                  <UiButton variant="ghost" size="sm" onClick={() => void handleToggleEnabled(item, false)}>
                    停用
                  </UiButton>
                ) : (
                  <UiButton variant="primary" size="sm" onClick={() => void handleToggleEnabled(item, true)}>
                    启用
                  </UiButton>
                )}
                <UiButton variant="danger" size="sm" onClick={() => setDeleteTarget(item)}>
                  删除
                </UiButton>
              </div>
            </section>
          ))}
        </div>

        {!loading && agents.length === 0 ? (
          <div className="mt-6 rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] px-6 py-10 text-center text-sm text-[var(--color-text-muted)]">
            当前没有匹配的 Agent，先创建一个 single 或 supervisor 试试。
          </div>
        ) : null}
      </div>

      <UiConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除 Agent"
        description={
          deleteTarget
            ? `删除后无法恢复：${deleteTarget.name}。若它仍被启用中的 supervisor 引用，后端会拒绝删除。`
            : ""
        }
        confirmText="确认删除"
        confirming={deleting}
        onCancel={() => {
          if (deleting) return;
          setDeleteTarget(null);
        }}
        onConfirm={() => void handleDelete()}
      />
    </div>
  );
}
