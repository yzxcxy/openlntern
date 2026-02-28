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
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
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

type UserInfo = {
  user_id?: string | number;
  username?: string;
  email?: string;
  role?: string;
};

const API_BASE = "/api/backend";

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
  const [previewTarget, setPreviewTarget] = useState<A2UI | null>(null);
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
      const res = await fetch(`${API_BASE}/v1/a2uis?${params.toString()}`, {
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取 A2UI 列表失败");
      }
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
  }, [getUserId, getValidToken, page, pageSize, searchKeyword]);

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
        const res = await fetch(`${API_BASE}/v1/a2uis`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...buildAuthHeaders(token, userId),
          },
          body: JSON.stringify({
            ...payload,
          }),
        });
        updateTokenFromResponse(res);
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "新增 A2UI 失败");
        }
      } else if (activeId) {
        const res = await fetch(`${API_BASE}/v1/a2uis/${activeId}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            ...buildAuthHeaders(token, userId),
          },
          body: JSON.stringify(payload),
        });
        updateTokenFromResponse(res);
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "更新 A2UI 失败");
        }
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
      const res = await fetch(`${API_BASE}/v1/a2uis/${deleteTarget.a2ui_id}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "删除 A2UI 失败");
      }
      closeDelete();
      fetchList();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("删除失败");
      }
    } finally {
      setDeleting(false);
    }
  };

  const openPreview = (item: A2UI) => {
    setPreviewTarget(item);
  };

  const closePreview = () => {
    setPreviewTarget(null);
  };

  const previewContent = useMemo(() => {
    if (!previewTarget) {
      return {
        components: null as A2UIViewerProps["components"] | null,
        root: null as string | null,
        data: undefined as Record<string, unknown> | undefined,
        error: "",
      };
    }

    let components: A2UIViewerProps["components"] | null = null;
    let root: string | null = null;
    let data: Record<string, unknown> | undefined;
    let error = "";

    try {
      const uiText = previewTarget.ui_json?.trim();
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
          } else if (
            Array.isArray(uiObj.components) &&
            typeof uiObj.rootId === "string"
          ) {
            components = uiObj.components as A2UIViewerProps["components"];
            root = uiObj.rootId as string;
          } else if (
            Array.isArray(uiObj.components) &&
            typeof (uiObj.root as { id?: string } | undefined)?.id === "string"
          ) {
            components = uiObj.components as A2UIViewerProps["components"];
            root = (uiObj.root as { id?: string }).id ?? null;
          } else if (
            Array.isArray(uiObj.nodes) &&
            typeof uiObj.root === "string"
          ) {
            components = uiObj.nodes as A2UIViewerProps["components"];
            root = uiObj.root as string;
          } else {
            error = "UI JSON 格式不符合预览要求";
          }
        } else {
          error = "UI JSON 格式不正确";
        }
      }
    } catch {
      error = "UI JSON 解析失败";
    }

    try {
      const dataText = previewTarget.data_json?.trim();
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
      error = "缺少可渲染的组件或根节点";
    }

    return { components, root, data, error };
  }, [previewTarget]);

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
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-6">
      <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4 shadow-[var(--shadow-sm)]">
          <div className="flex flex-wrap items-center gap-3">
          <UiInput
            className="w-full max-w-xs"
            placeholder="搜索名称或描述"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <UiButton type="button" variant="secondary" onClick={handleSearch}>
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

          <div className="mt-4 flex items-center justify-between">
            <div className="text-sm text-[var(--color-text-muted)]">共 {total} 条</div>
            <UiButton type="button" onClick={openCreate}>
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

          <div className="mt-4 space-y-3">
            {loading ? (
              <div className="text-sm text-[var(--color-text-muted)]">加载中...</div>
            ) : items.length === 0 ? (
              <div className="text-sm text-[var(--color-text-muted)]">暂无数据</div>
            ) : (
              items.map((item) => {
              const createdAt = formatDateDisplay(item.created_at);
              const updatedAt = formatDateDisplay(item.updated_at);
              return (
                <div
                  key={item.a2ui_id}
                  className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4"
                >
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                        {item.name}
                      </div>
                      <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <UiButton
                        className="px-3"
                        type="button"
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
                        className="px-3"
                        type="button"
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
                        className="px-3"
                        type="button"
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
                  <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-[var(--color-text-muted)]">
                    <div title={createdAt.full}>创建：{createdAt.label}</div>
                    <div title={updatedAt.full}>更新：{updatedAt.label}</div>
                  </div>
                </div>
              );
              })
            )}
          </div>

          <div className="mt-4 flex flex-wrap items-center justify-end gap-3 text-sm text-[var(--color-text-secondary)]">
            <div className="flex items-center gap-2">
              <span>每页</span>
              <UiSelect
                className="w-24"
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.target.value));
                  setPage(1);
                }}
              >
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
              </UiSelect>
            </div>
            <UiButton
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              disabled={page <= 1}
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
                <path d="M15 6l-6 6 6 6" />
              </svg>
              上一页
            </UiButton>
            <span>
              {page} / {totalPages}
            </span>
            <UiButton
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
              disabled={page >= totalPages}
            >
              下一页
              <svg
                className="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M9 6l6 6-6 6" />
              </svg>
            </UiButton>
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
            {previewContent.error ? (
              <div className="text-sm text-[var(--color-state-error)]">{previewContent.error}</div>
            ) : previewContent.components && previewContent.root ? (
              <div className="min-h-[220px]">
                <A2UIViewer
                  root={previewContent.root}
                  components={previewContent.components}
                  data={previewContent.data}
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
