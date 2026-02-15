"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  A2uiEditorModal,
  A2uiFormValues,
} from "./components/A2uiEditorModal";
import { ConfirmDialog } from "./components/ConfirmDialog";

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
  user_id?: string;
  username?: string;
  email?: string;
  role?: string;
};

const API_BASE = "http://localhost:8080";

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
  const router = useRouter();

  const isAdmin = userInfo?.role === "admin";
  const canManage = useMemo(
    () => category === "custom" || isAdmin,
    [category, isAdmin]
  );

  const applyUser = useCallback(() => {
    if (typeof window === "undefined") return null;
    const storedUser = localStorage.getItem("user");
    if (!storedUser) return null;
    try {
      return JSON.parse(storedUser);
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    setUserInfo(applyUser());
  }, [applyUser, router]);

  const fetchList = useCallback(async () => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
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
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
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
  }, [category, page, pageSize, router, searchKeyword]);

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
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
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
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            ...payload,
            type: category,
          }),
        });
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "新增 A2UI 失败");
        }
      } else if (activeId) {
        const res = await fetch(`${API_BASE}/v1/a2uis/${activeId}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
            "X-User-ID": userInfo?.user_id || "",
          },
          body: JSON.stringify(payload),
        });
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
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    setError("");
    setDeleting(true);
    try {
      const res = await fetch(`${API_BASE}/v1/a2uis/${deleteTarget.a2ui_id}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
          "X-User-ID": userInfo?.user_id || "",
        },
      });
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

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const formatDateDisplay = (value?: string) => {
    if (!value) {
      return { label: "-", full: "-" };
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return { label: value, full: value };
    }
    const now = Date.now();
    const diff = now - date.getTime();
    if (diff < 60_000) {
      return { label: "刚刚", full: date.toISOString() };
    }
    const minutes = Math.floor(diff / 60_000);
    if (minutes < 60) {
      return { label: `${minutes} 分钟前`, full: date.toISOString() };
    }
    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
      return { label: `${hours} 小时前`, full: date.toISOString() };
    }
    const days = Math.floor(hours / 24);
    if (days <= 7) {
      return { label: `${days} 天前`, full: date.toISOString() };
    }
    const pad = (num: number) => String(num).padStart(2, "0");
    const full = `${date.getFullYear()}-${pad(
      date.getMonth() + 1
    )}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
      date.getMinutes()
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
          <select
            className="rounded-md border px-3 py-2 text-sm"
            value={category}
            onChange={(e) => {
              setCategory(e.target.value as A2UIType);
              setPage(1);
            }}
          >
            <option value="official">官方 A2UI</option>
            <option value="custom">自定义 A2UI</option>
          </select>
          <button
            className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-500"
            type="button"
            onClick={handleSearch}
          >
            搜索
          </button>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-gray-500">
            共 {total} 条
          </div>
          {canManage && (
            <button
              className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
              type="button"
              onClick={openCreate}
            >
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
                    {canManage && (
                      <div className="flex items-center gap-2">
                        <button
                          className="rounded-md border px-3 py-1 text-xs text-gray-700 hover:bg-gray-50"
                          type="button"
                          onClick={() => openEdit(item)}
                        >
                          编辑
                        </button>
                        <button
                          className="rounded-md border px-3 py-1 text-xs text-red-600 hover:bg-red-50"
                          type="button"
                          onClick={() => openDelete(item)}
                        >
                          删除
                        </button>
                      </div>
                    )}
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
          <div className="mr-auto flex items-center gap-2">
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
            className="rounded-md border px-3 py-1 disabled:opacity-50"
            type="button"
            onClick={() => setPage((prev) => Math.max(1, prev - 1))}
            disabled={page <= 1}
          >
            上一页
          </button>
          <span>
            {page} / {totalPages}
          </span>
          <button
            className="rounded-md border px-3 py-1 disabled:opacity-50"
            type="button"
            onClick={() =>
              setPage((prev) => Math.min(totalPages, prev + 1))
            }
            disabled={page >= totalPages}
          >
            下一页
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
    </div>
  );
}
