"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  buildAuthHeaders,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

type Skill = {
  skill_id?: string;
  name?: string;
  description?: string;
  icon?: string;
  path?: string;
};

const API_BASE = "/api/backend";
export default function SkillsPage() {
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [items, setItems] = useState<Skill[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(12);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [uploadSuccess, setUploadSuccess] = useState("");
  const router = useRouter();
  const uploadInputRef = useRef<HTMLInputElement | null>(null);

  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const fetchList = useCallback(async () => {
    const token = getValidToken();
    if (!token) return;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (searchKeyword.trim()) {
        params.set("keyword", searchKeyword.trim());
      }
      const url = `${API_BASE}/v1/skills/meta?${params.toString()}`;
      const res = await fetch(url, {
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取 Skill 列表失败");
      }
      setItems(data.data?.data ?? []);
      setTotal(data.data?.total ?? 0);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("获取 Skill 列表失败");
      }
    } finally {
      setLoading(false);
    }
  }, [getValidToken, page, pageSize, searchKeyword]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleSearch = () => {
    setPage(1);
    setSearchKeyword(keyword);
  };

  const handleUploadClick = () => {
    uploadInputRef.current?.click();
  };

  const handleUploadChange = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!file.name.toLowerCase().endsWith(".zip")) {
      setUploadError("仅支持 .zip 文件");
      return;
    }
    const token = getValidToken();
    if (!token) return;
    setUploading(true);
    setUploadError("");
    setUploadSuccess("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(`${API_BASE}/v1/skills/import`, {
        method: "POST",
        headers: buildAuthHeaders(token),
        body: formData,
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "上传失败");
      }
      setUploadSuccess("上传成功");
      await fetchList();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setUploadError(err.message);
      } else {
        setUploadError("上传失败");
      }
    } finally {
      setUploading(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const getSkillName = useCallback((skill: Skill) => {
    if (skill.path) {
      const parts = skill.path.split("/");
      return parts[parts.length - 1] || "";
    }
    return skill.name ?? "";
  }, []);

  const cards = useMemo(() => {
    return items.map((item) => {
      const skillName = getSkillName(item);
      return {
        ...item,
        skillName,
      };
    });
  }, [getSkillName, items]);

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
          <button
            className="flex items-center gap-2 rounded-md border bg-gray-900 px-4 py-2 text-sm text-white disabled:opacity-60"
            type="button"
            onClick={handleUploadClick}
            disabled={uploading}
          >
            {uploading ? "上传中..." : "上传 Skill"}
          </button>
          <input
            ref={uploadInputRef}
            type="file"
            accept=".zip"
            className="hidden"
            onChange={handleUploadChange}
          />
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-gray-500">共 {total} 条</div>
        </div>

        {error && <div className="mt-4 text-sm text-red-600">{error}</div>}
        {uploadError && (
          <div className="mt-2 text-sm text-red-600">{uploadError}</div>
        )}
        {uploadSuccess && (
          <div className="mt-2 text-sm text-green-600">{uploadSuccess}</div>
        )}
        <div className="mt-4">
          {loading ? (
            <div className="text-sm text-gray-500">加载中...</div>
          ) : cards.length === 0 ? (
            <div className="text-sm text-gray-500">暂无数据</div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {cards.map((item) => (
                <div
                  key={item.path ?? item.skillName}
                  className="flex h-full flex-col rounded-xl border bg-white p-4 text-left shadow-sm transition hover:border-gray-300 hover:shadow"
                  role="button"
                  tabIndex={0}
                  onClick={() =>
                    router.push(
                      `/skills/detail?name=${encodeURIComponent(
                        item.skillName
                      )}`
                    )
                  }
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      router.push(
                        `/skills/detail?name=${encodeURIComponent(
                          item.skillName
                        )}`
                      );
                    }
                  }}
                >
                  <div className="flex items-start gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 text-xl">
                      {item.icon || "🧩"}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-semibold text-gray-900">
                        {item.name || item.skillName}
                      </div>
                      <div className="mt-1 text-xs text-gray-500">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="mt-5 flex flex-wrap items-center justify-end gap-3 text-sm text-gray-600">
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
              <option value={12}>12</option>
              <option value={24}>24</option>
              <option value={48}>48</option>
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
            onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
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
    </div>
  );
}
