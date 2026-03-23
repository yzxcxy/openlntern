"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { A2UIViewer, type A2UIViewerProps } from "@copilotkit/a2ui-renderer";
import {
  A2uiEditorModal,
  A2uiFormValues,
} from "./components/A2uiEditorModal";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { Modal } from "./components/Modal";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import {
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  requestBackend,
} from "../auth";

type A2UI = {
  a2ui_id: string;
  name: string;
  description?: string;
  ui_json: string;
  data_json?: string;
  created_at?: string;
  updated_at?: string;
};

type A2UICard = A2UI & {
  previewComponents: A2UIViewerProps["components"] | null;
  previewRoot: string | null;
  previewData: Record<string, unknown> | undefined;
  previewError: string;
};

type UserInfo = {
  user_id?: string | number;
  username?: string;
  email?: string;
  role?: string;
};

export default function A2uiPage() {
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [items, setItems] = useState<A2UI[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [editorOpen, setEditorOpen] = useState(false);
  const [editorMode, setEditorMode] = useState<"create" | "edit">("create");
  const [activeId, setActiveId] = useState<string | null>(null);
  const [formValues, setFormValues] = useState<A2uiFormValues>({
    name: "",
    description: "",
    ui_json: "",
    data_json: "",
  });
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<A2UI | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [previewTarget, setPreviewTarget] = useState<A2UICard | null>(null);
  const router = useRouter();

  const getUserId = useCallback((token: string) => {
    const user = readStoredUser<UserInfo>();
    const userId = user?.user_id;
    if (typeof userId === "string" || typeof userId === "number") {
      return String(userId);
    }
    return getUserIdFromToken(token);
  }, []);

  const getValidToken = useCallback(() => readValidToken(router), [router]);

  useEffect(() => {
    const token = getValidToken();
    if (!token) {
      router.push("/login");
      return;
    }
  }, [getValidToken, router]);

  const fetchList = useCallback(async () => {
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (searchKeyword.trim()) {
        params.set("keyword", searchKeyword.trim());
      }
      const data = await requestBackend<{ data: A2UI[]; total: number }>(
        `/v1/a2uis?${params.toString()}`,
        {
          fallbackMessage: "获取 A2UI 列表失败",
          router,
          userId,
        }
      );
      setItems(data.data?.data ?? []);
      setTotal(data.data?.total ?? 0);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("获取 A2UI 列表失败");
      }
    } finally {
      setLoading(false);
    }
  }, [getUserId, getValidToken, page, pageSize, router, searchKeyword]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleSearch = () => {
    setPage(1);
    setSearchKeyword(keyword);
  };

  const openCreate = () => {
    setEditorMode("create");
    setActiveId(null);
    setFormValues({
      name: "",
      description: "",
      ui_json: "",
      data_json: "",
    });
    setEditorOpen(true);
  };

  const openEdit = (item: A2UI) => {
    setEditorMode("edit");
    setActiveId(item.a2ui_id);
    setFormValues({
      name: item.name ?? "",
      description: item.description ?? "",
      ui_json: item.ui_json ?? "",
      data_json: item.data_json ?? "",
    });
    setEditorOpen(true);
  };

  const closeEditor = () => {
    setEditorOpen(false);
    setActiveId(null);
  };

  const handleSave = async () => {
    if (!formValues.name.trim()) {
      setError("请填写名称");
      return;
    }
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setSaving(true);
    setError("");
    try {
      const payload: Record<string, string> = {
        name: formValues.name.trim(),
        description: formValues.description.trim(),
        ui_json: formValues.ui_json,
        data_json: formValues.data_json,
      };
      if (editorMode === "create") {
        await requestBackend("/v1/a2uis", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            ...payload,
          }),
          fallbackMessage: "新增 A2UI 失败",
          router,
          userId,
        });
      } else if (activeId) {
        await requestBackend(`/v1/a2uis/${activeId}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
          fallbackMessage: "更新 A2UI 失败",
          router,
          userId,
        });
      }
      closeEditor();
      fetchList();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("保存失败");
      }
    } finally {
      setSaving(false);
    }
  };

  const openDelete = (item: A2UI) => {
    setDeleteTarget(item);
  };

  const closeDelete = () => {
    setDeleteTarget(null);
    setDeleting(false);
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setError("");
    setDeleting(true);
    try {
      await requestBackend(`/v1/a2uis/${deleteTarget.a2ui_id}`, {
        method: "DELETE",
        fallbackMessage: "删除 A2UI 失败",
        router,
        userId,
      });
      closeDelete();
      fetchList();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("获取 A2UI 列表失败");
      }
    } finally {
      setDeleting(false);
    }
  };

  const openPreview = (item: A2UICard) => {
    setPreviewTarget(item);
  };

  const closePreview = () => {
    setPreviewTarget(null);
  };

  const cards = useMemo(() => {
    return items.map((item) => {
      let components: A2UIViewerProps["components"] | null = null;
      let root: string | null = null;
      let data: Record<string, unknown> | undefined;
      let error = "";

      try {
        const uiText = item.ui_json?.trim();
        if (!uiText) {
          error = "UI JSON 为空";
        } else {
          const ui = JSON.parse(uiText);
          if (Array.isArray(ui)) {
            components = ui;
            root = ui[0]?.id ?? null;
          } else if (ui && typeof ui === "object") {
            const uiObj = ui as Record<string, unknown>;
            if (Array.isArray(uiObj.components) && typeof uiObj.root === "string") {
              components = uiObj.components as A2UIViewerProps["components"];
              root = uiObj.root as string;
            } else if (Array.isArray(uiObj.components) && typeof uiObj.rootId === "string") {
              components = uiObj.components as A2UIViewerProps["components"];
              root = uiObj.rootId as string;
            } else if (Array.isArray(uiObj.components) && typeof (uiObj.root as { id?: string } | undefined)?.id === "string") {
              components = uiObj.components as A2UIViewerProps["components"];
              root = (uiObj.root as { id?: string }).id ?? null;
            } else if (Array.isArray(uiObj.nodes) && typeof uiObj.root === "string") {
              components = uiObj.nodes as A2UIViewerProps["components"];
              root = uiObj.root as string;
            } else {
              error = "UI JSON 格式不符合要求";
            }
          } else {
            error = "UI JSON 格式不正确";
          }
        }
      } catch {
        error = "UI JSON 解析失败";
      }

      try {
        const dataText = item.data_json?.trim();
        if (dataText) {
          const parsed = JSON.parse(dataText);
          if (parsed && typeof parsed === "object") {
            data = parsed as Record<string, unknown>;
          }
        }
      } catch {
        error = error ? `${error}；数据 JSON 解析失败` : "数据 JSON 解析失败";
      }

      if (!error && (!components || !root)) {
        error = "缺少可渲染的组件";
      }

      return {
        ...item,
        previewComponents: components,
        previewRoot: root,
        previewData: data,
        previewError: error,
      };
    });
  }, [items]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const formatDateDisplay = (value?: string) => {
    if (!value) {
      return { label: "-", full: "-" };
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return { label: value, full: value };
    }
    const pad = (num: number) => String(num).padStart(2, "0");
    const full = `${date.getFullYear()}:${pad(date.getMonth() + 1)}:${pad(
      date.getDate()
    )} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(
      date.getSeconds()
    )}`;
    return { label: full, full };
  };

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="workspace-toolbar-surface rounded-[var(--radius-lg)] border p-3">
          <div className="flex flex-wrap items-center gap-3">
            <UiInput
              className="w-full max-w-xs"
              placeholder="搜索名称或描述"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
            <UiButton
              type="button"
              variant="secondary"
              onClick={handleSearch}
              className="px-5"
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
                <circle cx="11" cy="11" r="7" />
                <path d="M20 20l-3.5-3.5" />
              </svg>
              搜索
            </UiButton>
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-[var(--color-text-muted)]">共 {total} 条</div>
          <UiButton
            type="button"
            onClick={openCreate}
            className="ui-button-soft-accent"
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
              <path d="M12 5v14" />
              <path d="M5 12h14" />
            </svg>
            新增 A2UI
          </UiButton>
        </div>

        <A2uiEditorModal
          open={editorOpen}
          mode={editorMode}
          values={formValues}
          onChange={setFormValues}
          onClose={closeEditor}
          onSave={handleSave}
          saving={saving}
        />

        {error && (
          <div className="mt-4 text-sm text-[var(--color-state-error)]">{error}</div>
        )}

        <section>
            <div className="mt-4">
              {loading ? (
                <div className="workspace-empty-state">
                  <span>加载中...</span>
                </div>
              ) : cards.length === 0 ? (
                <div className="workspace-empty-state">
                  <span>暂无 A2UI</span>
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  {cards.map((item) => {
                    const createdAt = formatDateDisplay(item.created_at);
                    return (
                      <div
                        key={item.a2ui_id}
                        className="workspace-item-surface workspace-item-hover-lift flex h-full flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-default)] text-left shadow-[var(--shadow-sm)] transition hover:-translate-y-1 hover:border-[var(--color-border-strong)]"
                      >
                        {/* 卡片主体：渲染后的内容 */}
                        <div className="min-h-[180px] flex-1 overflow-hidden rounded-t-[var(--radius-lg)] bg-[var(--color-surface-soft)] p-4">
                          {item.previewError ? (
                            <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">
                              {item.previewError}
                            </div>
                          ) : item.previewComponents && item.previewRoot ? (
                            <A2UIViewer
                              root={item.previewRoot}
                              components={item.previewComponents}
                              data={item.previewData}
                              className="h-full w-full"
                            />
                          ) : (
                            <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">
                              暂无内容
                            </div>
                          )}
                        </div>

                        {/* 卡片底部：信息与操作 */}
                        <div className="border-t border-[var(--color-border-default)] p-4">
                          <div className="flex items-center justify-between gap-2">
                            <div className="truncate text-base font-semibold tracking-[-0.02em] text-[var(--color-text-primary)]">
                              {item.name}
                            </div>
                            <div className="text-xs text-[var(--color-text-muted)]" title={createdAt.full}>
                              {createdAt.label}
                            </div>
                          </div>

                          <div className="mt-3 flex gap-2">
                            <UiButton
                              variant="secondary"
                              size="sm"
                              onClick={() => openPreview(item)}
                            >
                              <svg
                                className="h-3.5 w-3.5"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="1.8"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                              >
                                <circle cx="12" cy="12" r="3" />
                                <path d="M2 12s4-6 10-6 10 6 10 6-4 6-10 6-10-6-10-6z" />
                              </svg>
                              预览
                            </UiButton>
                            <UiButton
                              variant="secondary"
                              size="sm"
                              onClick={() => openEdit(item)}
                            >
                              <svg
                                className="h-3.5 w-3.5"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="1.8"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                              >
                                <path d="M12 20h9" />
                                <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
                              </svg>
                              编辑
                            </UiButton>
                            <UiButton
                              variant="danger"
                              size="sm"
                              onClick={() => openDelete(item)}
                            >
                              <svg
                                className="h-3.5 w-3.5"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="1.8"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                              >
                                <path d="M3 6h18" />
                                <path d="M8 6V4h8v2" />
                                <path d="M6 6l1 14h10l1-14" />
                              </svg>
                              删除
                            </UiButton>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </section>

        <div className="mt-4 flex items-center justify-between gap-3 border-t border-[rgba(126,96,69,0.14)] pt-4">
          <div className="flex shrink-0 items-center gap-2">
            <span className="whitespace-nowrap text-sm text-[var(--color-text-muted)]">
              共 {total} 条
            </span>
            <span className="text-sm text-[var(--color-text-muted)]">/</span>
            <div className="flex items-center gap-1">
              <UiSelect
                value={String(pageSize)}
                onChange={(e) => {
                  setPageSize(Number(e.target.value));
                  setPage(1);
                }}
                className="h-8 w-[60px] !py-0 !pl-2 !pr-6"
              >
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
              </UiSelect>
              <span className="whitespace-nowrap text-sm text-[var(--color-text-muted)]">条/页</span>
            </div>
          </div>

          <div className="flex items-center gap-1">
            <UiButton
              variant="secondary"
              size="sm"
              className="h-7 px-2"
              onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              disabled={page <= 1}
            >
              ←
            </UiButton>

            {(() => {
              const pages: (number | string)[] = [];
              if (totalPages <= 7) {
                for (let i = 1; i <= totalPages; i++) pages.push(i);
              } else {
                if (page <= 4) {
                  pages.push(1, 2, 3, 4, 5, "...", totalPages);
                } else if (page >= totalPages - 3) {
                  pages.push(1, "...", totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages);
                } else {
                  pages.push(1, "...", page - 1, page, page + 1, "...", totalPages);
                }
              }
              return pages.map((p, idx) =>
                p === "..." ? (
                  <span key={`dot-${idx}`} className="px-1 text-sm text-[var(--color-text-muted)]">
                    ...
                  </span>
                ) : (
                  <UiButton
                    key={p}
                    variant={p === page ? "primary" : "secondary"}
                    size="sm"
                    className="h-7 min-w-[28px] px-2"
                    onClick={() => setPage(p as number)}
                  >
                    {p}
                  </UiButton>
                )
              );
            })()}

            <UiButton
              variant="secondary"
              size="sm"
              className="h-7 px-2"
              onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
              disabled={page >= totalPages}
            >
              →
            </UiButton>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除 A2UI"
        description={
          deleteTarget ? `确认删除「${deleteTarget.name}」吗？` : "确认删除吗？"
        }
        confirmText="删除"
        confirming={deleting}
        onConfirm={handleDelete}
        onCancel={closeDelete}
      />

      <Modal
        open={Boolean(previewTarget)}
        title={previewTarget?.name ? `预览：${previewTarget.name}` : "预览"}
        onClose={closePreview}
      >
        <div className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
          <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-3">
            {previewTarget?.previewError ? (
              <div className="text-sm text-[var(--color-state-error)]">{previewTarget.previewError}</div>
            ) : previewTarget?.previewComponents && previewTarget.previewRoot ? (
              <div className="min-h-[220px]">
                <A2UIViewer
                  root={previewTarget.previewRoot}
                  components={previewTarget.previewComponents}
                  data={previewTarget.previewData}
                  className="w-full"
                />
              </div>
            ) : (
              <div className="text-sm text-[var(--color-text-muted)]">暂无可预览内容</div>
            )}
          </div>
          <div className="space-y-3">
            <label className="block text-xs text-[var(--color-text-muted)]">UI JSON</label>
            <UiTextarea
              className="min-h-[160px] text-xs"
              value={previewTarget?.ui_json ?? ""}
              readOnly
            />
            <label className="block text-xs text-[var(--color-text-muted)]">数据 JSON</label>
            <UiTextarea
              className="min-h-[120px] text-xs"
              value={previewTarget?.data_json ?? ""}
              readOnly
            />
          </div>
        </div>
      </Modal>
    </div>
  );
}
