"use client";

import matter from "gray-matter";
import ReactMarkdown from "react-markdown";
import { useEffect, useMemo, useState, type ComponentPropsWithoutRef } from "react";
import remarkGfm from "remark-gfm";
import { useRouter, useSearchParams } from "next/navigation";

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

const markdownComponents = {
  h1: (props: ComponentPropsWithoutRef<"h1">) => (
    <h1 className="mb-4 text-2xl font-semibold text-gray-900" {...props} />
  ),
  h2: (props: ComponentPropsWithoutRef<"h2">) => (
    <h2 className="mb-3 mt-6 text-xl font-semibold text-gray-900" {...props} />
  ),
  h3: (props: ComponentPropsWithoutRef<"h3">) => (
    <h3 className="mb-2 mt-5 text-lg font-semibold text-gray-900" {...props} />
  ),
  p: (props: ComponentPropsWithoutRef<"p">) => (
    <p className="my-3 text-sm text-gray-700" {...props} />
  ),
  ul: (props: ComponentPropsWithoutRef<"ul">) => (
    <ul
      className="my-3 list-disc space-y-1 pl-6 text-sm text-gray-700"
      {...props}
    />
  ),
  ol: (props: ComponentPropsWithoutRef<"ol">) => (
    <ol
      className="my-3 list-decimal space-y-1 pl-6 text-sm text-gray-700"
      {...props}
    />
  ),
  li: (props: ComponentPropsWithoutRef<"li">) => (
    <li className="text-sm text-gray-700" {...props} />
  ),
  a: (props: ComponentPropsWithoutRef<"a">) => (
    <a
      className="text-sm text-blue-600 underline"
      target="_blank"
      rel="noreferrer"
      {...props}
    />
  ),
  code: (props: ComponentPropsWithoutRef<"code">) => (
    <code
      className="rounded bg-gray-200 px-1 py-0.5 text-xs text-gray-800"
      {...props}
    />
  ),
  pre: (props: ComponentPropsWithoutRef<"pre">) => (
    <pre
      className="my-3 overflow-auto rounded-lg bg-gray-900 p-4 text-xs text-gray-100"
      {...props}
    />
  ),
  table: ({
    className,
    ...props
  }: ComponentPropsWithoutRef<"table">) => (
    <div className="my-4 w-full overflow-x-auto">
      <table
        className={`w-full border-collapse text-sm text-gray-700 ${className ?? ""}`}
        {...props}
      />
    </div>
  ),
  thead: (props: ComponentPropsWithoutRef<"thead">) => (
    <thead className="bg-gray-100" {...props} />
  ),
  tbody: (props: ComponentPropsWithoutRef<"tbody">) => (
    <tbody className="divide-y divide-gray-200" {...props} />
  ),
  tr: (props: ComponentPropsWithoutRef<"tr">) => (
    <tr className="hover:bg-gray-50" {...props} />
  ),
  th: (props: ComponentPropsWithoutRef<"th">) => (
    <th
      className="border border-gray-200 px-3 py-2 text-left text-xs font-semibold text-gray-700"
      {...props}
    />
  ),
  td: (props: ComponentPropsWithoutRef<"td">) => (
    <td className="border border-gray-200 px-3 py-2 text-sm" {...props} />
  ),
  blockquote: (props: ComponentPropsWithoutRef<"blockquote">) => (
    <blockquote
      className="my-3 border-l-4 border-gray-300 pl-4 text-sm text-gray-600"
      {...props}
    />
  ),
  hr: (props: ComponentPropsWithoutRef<"hr">) => (
    <hr className="my-6 border-gray-200" {...props} />
  ),
};

export default function SkillDetailPage() {
  const params = useSearchParams();
  const router = useRouter();
  const [skill, setSkill] = useState<Skill | null>(null);
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const scope = (params.get("scope") as SkillType) || "official";
  const rawName = params.get("name") ?? "";
  const normalizedName = useMemo(() => {
    const trimmed = rawName.trim();
    if (!trimmed) return "";
    const parts = trimmed.split("/").filter(Boolean);
    return parts[parts.length - 1] ?? "";
  }, [rawName]);
  const displayName = useMemo(
    () => normalizedName || rawName || "未知技能",
    [normalizedName, rawName]
  );
  const parsedContent = useMemo(() => {
    if (!content) {
      return { body: "", frontmatter: {} as Record<string, unknown> };
    }
    try {
      const parsed = matter(content);
      return {
        body: parsed.content?.trim() ?? "",
        frontmatter: (parsed.data ?? {}) as Record<string, unknown>,
      };
    } catch {
      return { body: content, frontmatter: {} as Record<string, unknown> };
    }
  }, [content]);
  const frontmatterEntries = useMemo(
    () => Object.entries(parsedContent.frontmatter ?? {}),
    [parsedContent.frontmatter]
  );
  const formatFrontmatterValue = (value: unknown) => {
    if (value === null || value === undefined) return "—";
    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean"
    ) {
      return String(value);
    }
    return JSON.stringify(value);
  };

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    if (!normalizedName) {
      setError("缺少技能名称");
      setLoading(false);
      return;
    }
    const readJson = async (res: Response) => {
      const contentType = res.headers.get("content-type") || "";
      if (!contentType.includes("application/json")) {
        const text = (await res.text()).trim();
        throw new Error(text || "响应格式错误");
      }
      return res.json();
    };
    const fetchDetail = async () => {
      setLoading(true);
      setError("");
      try {
        const nameParam = encodeURIComponent(normalizedName);
        const metaRes = await fetch(
          `${API_BASE}/v1/skills/meta/${scope}/${nameParam}`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          }
        );
        const metaData = await readJson(metaRes);
        if (!metaRes.ok || metaData.code !== 0) {
          throw new Error(metaData.message || "获取技能信息失败");
        }
        setSkill(metaData.data ?? null);
        const contentRes = await fetch(
          `${API_BASE}/v1/skills/content/${scope}/${nameParam}`,
          {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          }
        );
        const contentData = await readJson(contentRes);
        if (!contentRes.ok || contentData.code !== 0) {
          throw new Error(contentData.message || "获取技能文档失败");
        }
        setContent(contentData.data?.content ?? "");
      } catch (err) {
        if (err instanceof Error && err.message) {
          setError(err.message);
        } else {
          setError("加载失败");
        }
      } finally {
        setLoading(false);
      }
    };
    fetchDetail();
  }, [normalizedName, rawName, router, scope]);

  const headerName = skill?.name || displayName;
  const headerDesc = skill?.description || "暂无描述";
  const headerIcon = skill?.icon || "🧩";

  return (
    <div className="h-full overflow-auto p-6">
      <div className="rounded-xl border bg-white p-6 shadow-sm">
        <button
          className="mb-4 text-sm text-gray-500 hover:text-gray-700"
          type="button"
          onClick={() => router.push("/skills")}
        >
          返回 Skill 市场
        </button>
        {loading ? (
          <div className="text-sm text-gray-500">加载中...</div>
        ) : error ? (
          <div className="text-sm text-red-600">{error}</div>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gray-100 text-2xl">
                {headerIcon}
              </div>
              <div className="flex-1">
                <div className="text-lg font-semibold text-gray-900">
                  {headerName}
                </div>
                <div className="mt-1 text-sm text-gray-500">{headerDesc}</div>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-gray-500">
                <span className="rounded-full border px-2 py-0.5">
                  {scope === "official" ? "官方" : "自定义"}
                </span>
                {skill?.source && (
                  <span className="rounded-full border px-2 py-0.5">
                    {skill.source}
                  </span>
                )}
              </div>
            </div>
            {frontmatterEntries.length > 0 && (
              <div className="mt-4 rounded-lg border bg-gray-50 p-4">
                <div className="text-sm font-semibold text-gray-900">元信息</div>
                <dl className="mt-3 grid gap-x-6 gap-y-3 text-sm text-gray-600 sm:grid-cols-2">
                  {frontmatterEntries.map(([key, value]) => (
                    <div key={key} className="flex flex-col">
                      <dt className="text-xs font-medium uppercase text-gray-500">
                        {key}
                      </dt>
                      <dd className="mt-1 text-gray-700">
                        {formatFrontmatterValue(value)}
                      </dd>
                    </div>
                  ))}
                </dl>
              </div>
            )}
            <div className="mt-6 rounded-lg border bg-gray-50 p-5">
              {parsedContent.body ? (
                <ReactMarkdown
                  components={markdownComponents}
                  remarkPlugins={[remarkGfm]}
                >
                  {parsedContent.body}
                </ReactMarkdown>
              ) : (
                <div className="text-sm text-gray-500">暂无文档内容</div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
