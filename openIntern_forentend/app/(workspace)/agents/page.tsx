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
        <div className="workspace-page-stack">
          <section className="workspace-filter-panel">
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">Agent 管理</h1>
                <p className="text-sm text-[var(--color-text-muted)]">共 {total} 个 Agent</p>
              </div>
              <UiButton className="h-10 px-4" onClick={() => router.push("/agents/editor")}>
                创建 Agent
              </UiButton>
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-3">
              <UiInput
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="搜索名字或描述"
                className="h-10 w-48"
              />
              <UiSelect
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value)}
                className="h-10 w-28"
              >
                <option value="">全部状态</option>
                <option value="draft">Draft</option>
                <option value="enabled">Enabled</option>
                <option value="disabled">Disabled</option>
              </UiSelect>
              <UiSelect value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)} className="h-10 w-28">
                <option value="">全部类型</option>
                <option value="single">Single</option>
                <option value="supervisor">Supervisor</option>
              </UiSelect>
              <UiButton
                variant="secondary"
                className="h-10 w-10 px-0"
                onClick={() => void loadAgents()}
                disabled={loading}
                aria-label="刷新列表"
                title="刷新列表"
              >
                <svg
                  className={joinClasses("h-4 w-4", loading && "animate-spin")}
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
            </div>

            {error ? (
              <div className="mt-4 rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                {error}
              </div>
            ) : null}
          </section>

          <section>
            <div className="mt-4 grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
              {agents.map((item) => (
                <section
                  key={item.agent_id}
                  className="workspace-item-surface workspace-item-hover-lift rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-5"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-base font-semibold tracking-[-0.02em] text-[var(--color-text-primary)]">
                          {item.name}
                        </span>
                        <span className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5 text-[11px] text-[var(--color-text-muted)]">
                          {item.agent_type === "supervisor" ? "Supervisor" : "Single"}
                        </span>
                        <span
                          className={joinClasses(
                            "rounded-full border px-2 py-0.5 text-[11px]",
                            item.status === "enabled"
                              ? "border-[rgba(47,122,87,0.22)] text-[rgb(47,122,87)]"
                              : item.status === "disabled"
                                ? "border-[rgba(183,121,31,0.22)] text-[rgb(183,121,31)]"
                                : "border-[var(--color-border-default)] text-[var(--color-text-muted)]"
                          )}
                        >
                          {item.status}
                        </span>
                      </div>
                      <div className="mt-2 line-clamp-3 text-sm leading-7 text-[var(--color-text-secondary)]">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                    {item.avatar_url ? (
                      <img
                        src={item.avatar_url}
                        alt={item.name}
                        className="h-12 w-12 rounded-[16px] border border-[var(--color-border-default)] object-cover"
                      />
                    ) : (
                      <div className="flex h-12 w-12 items-center justify-center rounded-[16px] border border-[var(--color-border-default)] bg-[var(--color-surface-soft)] text-xs font-semibold text-[var(--color-text-muted)]">
                        AG
                      </div>
                    )}
                  </div>

                  <div className="mt-4 flex flex-wrap gap-2">
                    {(item.tool_count > 0 || item.skill_count > 0 || item.knowledge_base_count > 0) && (
                      <span className="text-xs text-[var(--color-text-muted)]">
                        {[
                          item.tool_count > 0 && `${item.tool_count} 工具`,
                          item.skill_count > 0 && `${item.skill_count} Skill`,
                          item.knowledge_base_count > 0 && `${item.knowledge_base_count} 知识库`,
                        ].filter(Boolean).join(" · ")}
                      </span>
                    )}
                  </div>

                  <div className="mt-4 flex gap-2">
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
              <div className="mt-6 workspace-empty-state">
                <strong>当前没有匹配的 Agent</strong>
                <span>先创建一个 single 或 supervisor，再回到这里统一管理生命周期。</span>
              </div>
            ) : null}
          </section>
        </div>
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
