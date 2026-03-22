"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { readValidToken, requestBackend } from "../auth";

type Skill = {
  skill_id?: string;
  name?: string;
  description?: string;
  icon?: string;
  path?: string;
};

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
    if (!getValidToken()) return;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (searchKeyword.trim()) {
        params.set("keyword", searchKeyword.trim());
      }
      const data = await requestBackend<{ data: Skill[]; total: number }>(
        `/v1/skills/meta?${params.toString()}`,
        {
          fallbackMessage: "获取 Skill 列表失败",
          router,
        }
      );
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
  }, [getValidToken, page, pageSize, router, searchKeyword]);

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
    if (!getValidToken()) return;
    setUploading(true);
    setUploadError("");
    setUploadSuccess("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      await requestBackend("/v1/skills/import", {
        method: "POST",
        body: formData,
        fallbackMessage: "上传失败",
        router,
      });
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
            <UiButton
              type="button"
              onClick={handleUploadClick}
              disabled={uploading}
              className="ui-button-soft-accent"
            >
              {uploading ? "上传中..." : "上传 Skill"}
            </UiButton>
            <input
              ref={uploadInputRef}
              type="file"
              accept=".zip"
              className="hidden"
              onChange={handleUploadChange}
            />
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="text-sm text-[var(--color-text-muted)]">共 {total} 条</div>
        </div>

        {error && (
          <div className="mt-4 text-sm text-[var(--color-state-error)]">{error}</div>
        )}
        {uploadError && (
          <div className="mt-2 text-sm text-[var(--color-state-error)]">{uploadError}</div>
        )}
        {uploadSuccess && (
          <div className="mt-2 text-sm text-[var(--color-state-success)]">{uploadSuccess}</div>
        )}
        <div className="mt-4">
          {loading ? (
            <div className="text-sm text-[var(--color-text-muted)]">加载中...</div>
          ) : cards.length === 0 ? (
            <div className="text-sm text-[var(--color-text-muted)]">暂无数据</div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {cards.map((item) => (
                <div
                  key={item.path ?? item.skillName}
                  className="workspace-item-surface workspace-item-hover-lift flex h-full flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4 text-left shadow-[var(--shadow-sm)] transition hover:-translate-y-1 hover:border-[var(--color-border-strong)]"
                  role="button"
                  tabIndex={0}
                  onClick={() =>
                    router.push(
                      `/skills/detail?name=${encodeURIComponent(item.skillName)}`
                    )
                  }
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      router.push(
                        `/skills/detail?name=${encodeURIComponent(item.skillName)}`
                      );
                    }
                  }}
                >
                  <div className="flex items-start gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-bg-page)] text-xl">
                      {item.icon || "🧩"}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                        {item.name || item.skillName}
                      </div>
                      <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="mt-5 flex flex-wrap items-center justify-end gap-3 text-sm text-[var(--color-text-secondary)]">
          <div className="flex shrink-0 items-center gap-2">
            <span className="shrink-0 whitespace-nowrap">每页</span>
            <UiSelect
              className="w-24"
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value));
                setPage(1);
              }}
            >
              <option value={12}>12</option>
              <option value={24}>24</option>
              <option value={48}>48</option>
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
    </div>
  );
}
