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
        <div className="workspace-page-stack">
          <section className="workspace-filter-panel">
            <div className="workspace-section-title">
              <div>
                <h1>Skill 管理</h1>
                <p>搜索、上传和查看可用 Skill。</p>
              </div>
              <div className="workspace-stat-row">
                <div className="workspace-stat-chip">
                  <strong>{total}</strong>
                  <span>数量</span>
                </div>
                <div className="workspace-stat-chip">
                  <strong>{pageSize}</strong>
                  <span>每页</span>
                </div>
                <div className="workspace-stat-chip">
                  <strong>{totalPages}</strong>
                  <span>页数</span>
                </div>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap items-center gap-3">
              <UiInput
                className="w-full max-w-sm"
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

            {(error || uploadError || uploadSuccess) && (
              <div className="mt-4 space-y-2">
                {error && (
                  <div className="rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                    {error}
                  </div>
                )}
                {uploadError && (
                  <div className="rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                    {uploadError}
                  </div>
                )}
                {uploadSuccess && (
                  <div className="rounded-[18px] border border-[rgba(47,122,87,0.16)] bg-[rgba(47,122,87,0.08)] px-4 py-3 text-sm text-[var(--color-state-success)]">
                    {uploadSuccess}
                  </div>
                )}
              </div>
            )}
          </section>

          <section>
            <div className="workspace-section-title">
              <div>
                <h2>Skill 卡片</h2>
                <p>查看 Skill 名称、描述和详情入口。</p>
              </div>
            </div>

            <div className="mt-4">
              {loading ? (
                <div className="workspace-empty-state">
                  <strong>正在加载 Skill 列表</strong>
                  <span>请稍候。</span>
                </div>
              ) : cards.length === 0 ? (
                <div className="workspace-empty-state">
                  <strong>当前没有可展示的 Skill</strong>
                  <span>你可以先上传一个 zip 包，或者调整关键字重新搜索。</span>
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  {cards.map((item) => (
                    <div
                      key={item.path ?? item.skillName}
                      className="workspace-item-surface workspace-item-hover-lift flex h-full flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-5 text-left shadow-[var(--shadow-sm)] transition hover:-translate-y-1 hover:border-[var(--color-border-strong)]"
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
                      <div className="flex items-start gap-4">
                        <div className="flex h-12 w-12 items-center justify-center rounded-[18px] border border-[var(--color-border-default)] bg-[var(--color-surface-soft)] text-xs font-semibold uppercase tracking-[0.14em] text-[var(--color-action-primary)]">
                          {(item.name || item.skillName).slice(0, 2)}
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="text-base font-semibold tracking-[-0.02em] text-[var(--color-text-primary)]">
                            {item.name || item.skillName}
                          </div>
                          <div className="mt-1 text-xs uppercase tracking-[0.14em] text-[var(--color-text-muted)]">
                            {item.skillName}
                          </div>
                        </div>
                      </div>
                      <div className="mt-4 line-clamp-3 text-sm leading-7 text-[var(--color-text-secondary)]">
                        {item.description || "暂无描述"}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </section>
        </div>

        <div className="mt-5 flex items-center justify-between gap-3 border-t border-[rgba(126,96,69,0.14)] pt-4">
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
                <option value={12}>12</option>
                <option value={24}>24</option>
                <option value={48}>48</option>
                <option value={96}>96</option>
              </UiSelect>
              <span className="whitespace-nowrap text-sm text-[var(--color-text-muted)]">条/页</span>
            </div>
          </div>

          <div className="flex items-center gap-1">
            <UiButton
              variant="secondary"
              size="sm"
              className="h-7 px-2"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
            >
              ←
            </UiButton>

            {(() => {
              const totalPages = Math.ceil(total / pageSize);
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
              onClick={() => setPage((p) => Math.min(Math.ceil(total / pageSize), p + 1))}
              disabled={page >= Math.ceil(total / pageSize)}
            >
              →
            </UiButton>
          </div>
        </div>
      </div>
    </div>
  );
}
