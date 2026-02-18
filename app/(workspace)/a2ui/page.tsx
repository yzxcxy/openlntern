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
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

type A2UIType = "official" | "custom";

type A2UI = {
  a2ui_id: string;
  name: string;
  description?: string;
  type: A2UIType;
  ui_json: string;
  data_json?: string;
  user_id?: number;
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
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [category, setCategory] = useState<A2UIType>("official");
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

  const isAdmin = userInfo?.role === "admin";
  const canManage = useMemo(
    () => category === "custom" || isAdmin,
    [category, isAdmin]
  );

  const applyUser = useCallback(() => readStoredUser<UserInfo>(), []);
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
    setUserInfo(applyUser());
  }, [applyUser, getValidToken, router]);

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
      const url =
        category === "official"
          ? `${API_BASE}/v1/a2uis/official?${params.toString()}`
          : `${API_BASE}/v1/a2uis/custom?${params.toString()}`;
      const res = await fetch(url, {
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
  }, [category, getUserId, getValidToken, page, pageSize, searchKeyword]);

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
            type: category,
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
    <div className="h-full overflow-auto p-6">
      <div className="rounded-xl border bg-white p-4 shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <input
            className="w-full max-w-xs rounded-md border px-3 py-2 text-sm"
            placeholder="搜索名称或描述"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <div className="flex items-center gap-2">
            <button
              type="button"
              className={`flex items-center gap-2 rounded-md border px-4 py-2 text-sm ${
                category === "official"
                  ? "border-gray-400 bg-gray-50 text-gray-900"
                  : "border-gray-200 text-gray-600 hover:bg-gray-50"
              }`}
              onClick={() => {
                setCategory("official");
                setPage(1);
              }}
            >
              <svg
                className="h-4 w-4 text-gray-500"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M12 3l2.5 5 5.5.8-4 3.9.9 5.5-4.9-2.7-4.9 2.7.9-5.5-4-3.9 5.5-.8L12 3z" />
              </svg>
              官方 A2UI
            </button>
            <button
              type="button"
              className={`flex items-center gap-2 rounded-md border px-4 py-2 text-sm ${
                category === "custom"
                  ? "border-gray-400 bg-gray-50 text-gray-900"
                  : "border-gray-200 text-gray-600 hover:bg-gray-50"
              }`}
              onClick={() => {
                setCategory("custom");
                setPage(1);
              }}
            >
              <svg
                className="h-4 w-4 text-gray-500"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M8 3a3 3 0 0 1 6 0v1h1.2a2.8 2.8 0 1 1 0 5.6H14V12h2.2a2.8 2.8 0 1 1 0 5.6H14V20a3 3 0 0 1-6 0v-1H6.8a2.8 2.8 0 1 1 0-5.6H8V9.6H5.8a2.8 2.8 0 1 1 0-5.6H8V3z" />
              </svg>
              自定义 A2UI
            </button>
          </div>
          <button
            className="flex items-center gap-2 rounded-md border bg-gray-100 px-4 py-2 text-sm text-gray-700 hover:bg-gray-200"
            type="button"
            onClick={handleSearch}
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
          </button>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-gray-500">
            共 {total} 条
          </div>
          {canManage && (
            <button
              className="flex items-center gap-2 rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
              type="button"
              onClick={openCreate}
            >
              <svg
                className="h-4 w-4 text-gray-500"
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
            </button>
          )}
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

        {error && <div className="mt-4 text-sm text-red-600">{error}</div>}

        <div className="mt-4 space-y-3">
          {loading ? (
            <div className="text-sm text-gray-500">加载中...</div>
          ) : items.length === 0 ? (
            <div className="text-sm text-gray-500">暂无数据</div>
          ) : (
            items.map((item) => {
              const createdAt = formatDateDisplay(item.created_at);
              const updatedAt = formatDateDisplay(item.updated_at);
              return (
                <div
                  key={item.a2ui_id}
                  className="rounded-lg border bg-white p-4"
                >
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-semibold text-gray-900">
                        {item.name}
                      </div>
                      <div className="mt-1 text-xs text-gray-500">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        className="flex items-center gap-1 rounded-md border px-3 py-1 text-xs text-gray-700 hover:bg-gray-50"
                        type="button"
                        onClick={() => openPreview(item)}
                      >
                        <svg
                          className="h-3.5 w-3.5 text-gray-500"
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
                      </button>
                      {canManage && (
                        <>
                          <button
                            className="flex items-center gap-1 rounded-md border px-3 py-1 text-xs text-gray-700 hover:bg-gray-50"
                            type="button"
                            onClick={() => openEdit(item)}
                          >
                            <svg
                              className="h-3.5 w-3.5 text-gray-500"
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
                          </button>
                          <button
                            className="flex items-center gap-1 rounded-md border px-3 py-1 text-xs text-red-600 hover:bg-red-50"
                            type="button"
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
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-gray-500">
                    <div title={createdAt.full}>创建：{createdAt.label}</div>
                    <div title={updatedAt.full}>更新：{updatedAt.label}</div>
                  </div>
                </div>
              );
            })
          )}
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-end gap-3 text-sm text-gray-600">
          <div className="flex items-center gap-2">
            <span>每页</span>
            <select
              className="rounded-md border px-2 py-1 text-sm"
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value));
                setPage(1);
              }}
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
            </select>
          </div>
          <button
            className="flex items-center gap-2 rounded-md border px-3 py-1 disabled:opacity-50"
            type="button"
            onClick={() => setPage((prev) => Math.max(1, prev - 1))}
            disabled={page <= 1}
          >
            <svg
              className="h-4 w-4 text-gray-500"
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
          </button>
          <span>
            {page} / {totalPages}
          </span>
          <button
            className="flex items-center gap-2 rounded-md border px-3 py-1 disabled:opacity-50"
            type="button"
            onClick={() =>
              setPage((prev) => Math.min(totalPages, prev + 1))
            }
            disabled={page >= totalPages}
          >
            下一页
            <svg
              className="h-4 w-4 text-gray-500"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M9 6l6 6-6 6" />
            </svg>
          </button>
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
          <div className="rounded-lg border bg-white p-3">
            {previewContent.error ? (
              <div className="text-sm text-red-600">{previewContent.error}</div>
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
              <div className="text-sm text-gray-500">暂无可预览内容</div>
            )}
          </div>
          <div className="space-y-3">
            <label className="block text-xs text-gray-500">UI JSON</label>
            <textarea
              className="min-h-[160px] w-full rounded-md border px-3 py-2 text-xs text-gray-700"
              value={previewTarget?.ui_json ?? ""}
              readOnly
            />
            <label className="block text-xs text-gray-500">数据 JSON</label>
            <textarea
              className="min-h-[120px] w-full rounded-md border px-3 py-2 text-xs text-gray-700"
              value={previewTarget?.data_json ?? ""}
              readOnly
            />
          </div>
        </div>
      </Modal>
    </div>
  );
}
