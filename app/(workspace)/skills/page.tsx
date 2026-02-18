"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  buildAuthHeaders,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

type SkillType = "official" | "custom";

type Skill = {
  skill_id?: string;
  name?: string;
  description?: string;
  type?: SkillType;
  source?: string;
  icon?: string;
  path?: string;
  user_id?: string;
};

const API_BASE = "/api/backend";
export default function SkillsPage() {
  const [category, setCategory] = useState<SkillType>("official");
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [items, setItems] = useState<Skill[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(12);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const router = useRouter();

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
      const url =
        category === "official"
          ? `${API_BASE}/v1/skills/meta/official?${params.toString()}`
          : `${API_BASE}/v1/skills/meta/custom?${params.toString()}`;
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
  }, [category, getValidToken, page, pageSize, searchKeyword]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleSearch = () => {
    setPage(1);
    setSearchKeyword(keyword);
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
              官方 Skill
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
              自定义 Skill
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
          <div className="text-sm text-gray-500">共 {total} 条</div>
        </div>

        {error && <div className="mt-4 text-sm text-red-600">{error}</div>}

        <div className="mt-4">
          {loading ? (
            <div className="text-sm text-gray-500">加载中...</div>
          ) : cards.length === 0 ? (
            <div className="text-sm text-gray-500">暂无数据</div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {cards.map((item) => (
                <button
                  key={item.path ?? item.skillName}
                  type="button"
                  className="flex h-full flex-col rounded-xl border bg-white p-4 text-left shadow-sm transition hover:border-gray-300 hover:shadow"
                  onClick={() =>
                    router.push(
                      `/skills/detail?scope=${category}&name=${encodeURIComponent(
                        item.skillName
                      )}`
                    )
                  }
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
                  <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-gray-500">
                    <span className="rounded-full border px-2 py-0.5">
                      {category === "official" ? "官方" : "自定义"}
                    </span>
                    {item.source && (
                      <span className="rounded-full border px-2 py-0.5">
                        {item.source}
                      </span>
                    )}
                    <span className="rounded-full border px-2 py-0.5">
                      {item.skillName}
                    </span>
                  </div>
                </button>
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
