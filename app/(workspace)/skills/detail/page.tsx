"use client";

import matter from "gray-matter";
import ReactMarkdown from "react-markdown";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ComponentPropsWithoutRef,
} from "react";
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

type SkillFileItem = {
  id?: string;
  type?: string;
  size?: number;
  date?: string;
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
  const [activeTab, setActiveTab] = useState<"doc" | "files">("doc");
  const [fileItems, setFileItems] = useState<SkillFileItem[] | null>(null);
  const [fileListLoaded, setFileListLoaded] = useState(false);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileError, setFileError] = useState("");
  const [selectedFile, setSelectedFile] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [fileContentLoading, setFileContentLoading] = useState(false);
  const [fileContentError, setFileContentError] = useState("");
  const [fileModalOpen, setFileModalOpen] = useState(false);
  const [fileModalPath, setFileModalPath] = useState("");
  const [docModalOpen, setDocModalOpen] = useState(false);
  const [docModalPath, setDocModalPath] = useState("");
  const [docModalContent, setDocModalContent] = useState("");
  const [docModalLoading, setDocModalLoading] = useState(false);
  const [docModalError, setDocModalError] = useState("");

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
  const parsedDocContent = useMemo(() => {
    if (!docModalContent) {
      return { body: "" };
    }
    try {
      const parsed = matter(docModalContent);
      return { body: parsed.content?.trim() ?? "" };
    } catch {
      return { body: docModalContent };
    }
  }, [docModalContent]);
  const docIsMarkdown = useMemo(() => {
    const value = docModalPath.toLowerCase();
    return value.endsWith(".md") || value.endsWith(".markdown");
  }, [docModalPath]);
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
  const isDocLink = (href?: string) => {
    if (!href) return false;
    const value = href.toLowerCase();
    if (
      value.startsWith("http://") ||
      value.startsWith("https://") ||
      value.startsWith("mailto:") ||
      value.startsWith("#")
    ) {
      return false;
    }
    return true;
  };
  const readContent = async (res: Response) => {
    const contentType = res.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      const data = await res.json();
      if (res.ok && data?.code === 0) {
        return String(data?.data?.content ?? "");
      }
      throw new Error(data?.message || "获取文档失败");
    }
    const text = (await res.text()).trim();
    if (!res.ok) {
      throw new Error(text || "获取文档失败");
    }
    return text;
  };
  const skillPath = useMemo(() => {
    if (skill?.path) {
      return `/${skill.path.replace(/^\/+/, "")}`;
    }
    if (!normalizedName) return "";
    if (scope === "official") {
      return `/official/${normalizedName}`;
    }
    if (skill?.user_id) {
      return `/${skill.user_id}/${normalizedName}`;
    }
    return "";
  }, [normalizedName, scope, skill?.path, skill?.user_id]);
  const listScope = scope === "official" ? "official" : "user";
  const fetchFileList = useCallback(async () => {
    const token = localStorage.getItem("token");
    if (!token) {
      setFileError("请先登录");
      return;
    }
    if (!skillPath) {
      setFileError("缺少技能路径");
      return;
    }
    setFileLoading(true);
    setFileError("");
    try {
      const params = new URLSearchParams();
      params.set("scope", listScope);
      params.set("path", skillPath);
      const res = await fetch(`${API_BASE}/v1/skills?${params.toString()}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      const contentType = res.headers.get("content-type") || "";
      if (!contentType.includes("application/json")) {
        const text = (await res.text()).trim();
        throw new Error(text || "响应格式错误");
      }
      const data = await res.json();
      if (!res.ok || data?.code !== 0) {
        throw new Error(data?.message || "获取技能文件失败");
      }
      setFileItems(Array.isArray(data?.data) ? data.data : []);
      setFileListLoaded(true);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setFileError(err.message);
      } else {
        setFileError("获取技能文件失败");
      }
      setFileListLoaded(true);
    } finally {
      setFileLoading(false);
    }
  }, [listScope, skillPath]);
  const fetchFileContent = async (path: string) => {
    const token = localStorage.getItem("token");
    if (!token) {
      setFileContentError("请先登录");
      return;
    }
    if (!normalizedName) {
      setFileContentError("缺少技能名称");
      return;
    }
    setFileContentLoading(true);
    setFileContentError("");
    try {
      const nameParam = encodeURIComponent(normalizedName);
      const pathParam = encodeURIComponent(path);
      const apiUrl = `${API_BASE}/v1/skills/content/${scope}/${nameParam}?path=${pathParam}`;
      const res = await fetch(apiUrl, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      const text = await readContent(res);
      setFileContent(text);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setFileContentError(err.message);
      } else {
        setFileContentError("获取文件内容失败");
      }
    } finally {
      setFileContentLoading(false);
    }
  };
  const toRelativeFilePath = useCallback(
    (fileId: string) => {
      if (!fileId) return "";
      const normalizedSkillPath = skillPath.replace(/\/+$/, "");
      if (!normalizedSkillPath) return fileId.replace(/^\/+/, "");
      if (fileId === normalizedSkillPath) return "";
      if (fileId.startsWith(`${normalizedSkillPath}/`)) {
        return fileId.slice(normalizedSkillPath.length + 1);
      }
      return fileId.replace(/^\/+/, "");
    },
    [skillPath]
  );
  const getFileExtension = (value: string) => {
    const name = value.split("/").filter(Boolean).pop() ?? "";
    const dotIndex = name.lastIndexOf(".");
    if (dotIndex <= 0 || dotIndex === name.length - 1) return "";
    return name.slice(dotIndex + 1).toLowerCase();
  };
  const renderFileIcon = (item: SkillFileItem, fileId: string) => {
    const type = (item.type ?? "").toLowerCase();
    if (type === "dir" || type === "directory") {
      return (
        <svg
          className="h-4 w-4 text-amber-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M3 7h6l2 2h10v9a2 2 0 0 1-2 2H3z" />
          <path d="M3 7V5a2 2 0 0 1 2-2h4l2 2" />
        </svg>
      );
    }
    const ext = getFileExtension(fileId);
    if (["md", "markdown"].includes(ext)) {
      return (
        <svg
          className="h-4 w-4 text-indigo-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
          <path d="M14 3v5h5" />
          <path d="M9 13h6M9 17h6" />
        </svg>
      );
    }
    if (["json", "yaml", "yml", "toml", "ini"].includes(ext)) {
      return (
        <svg
          className="h-4 w-4 text-emerald-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M4 4h6l2 2h8v14H4z" />
          <path d="M8 12h8M8 16h8M8 8h2" />
        </svg>
      );
    }
    if (["ts", "tsx", "js", "jsx", "py", "go", "rs", "java", "rb"].includes(ext)) {
      return (
        <svg
          className="h-4 w-4 text-blue-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M7 8l-4 4 4 4" />
          <path d="M17 8l4 4-4 4" />
          <path d="M10 20l4-16" />
        </svg>
      );
    }
    if (["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp"].includes(ext)) {
      return (
        <svg
          className="h-4 w-4 text-pink-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect x="3" y="5" width="18" height="14" rx="2" />
          <circle cx="8" cy="10" r="1.5" />
          <path d="M21 16l-5-5-6 6-3-3-4 4" />
        </svg>
      );
    }
    if (["zip", "rar", "7z", "tar", "gz"].includes(ext)) {
      return (
        <svg
          className="h-4 w-4 text-orange-500"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
          <path d="M14 3v5h5" />
          <path d="M11 10h2M11 13h2M11 16h2" />
        </svg>
      );
    }
    return (
      <svg
        className="h-4 w-4 text-gray-500"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
        <path d="M14 3v5h5" />
      </svg>
    );
  };
  const openFileModal = async (fileId: string) => {
    const relativePath = toRelativeFilePath(fileId);
    setSelectedFile(fileId);
    setFileContent("");
    setFileContentError("");
    setFileModalPath(fileId);
    setFileModalOpen(true);
    await fetchFileContent(relativePath);
  };
  const closeFileModal = () => {
    setFileModalOpen(false);
    setFileModalPath("");
    setFileContent("");
    setFileContentError("");
    setFileContentLoading(false);
  };
  const fetchDocContent = async (href: string) => {
    const token = localStorage.getItem("token");
    if (!token) {
      setDocModalError("请先登录");
      setDocModalLoading(false);
      return;
    }
    if (!normalizedName) {
      setDocModalError("缺少技能名称");
      setDocModalLoading(false);
      return;
    }
    const nameParam = encodeURIComponent(normalizedName);
    const pathParam = encodeURIComponent(href);
    const apiUrl = `${API_BASE}/v1/skills/content/${scope}/${nameParam}?path=${pathParam}`;
    try {
      const res = await fetch(apiUrl, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      const text = await readContent(res);
      setDocModalContent(text);
      setDocModalLoading(false);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setDocModalError(err.message);
      } else {
        setDocModalError("获取文档失败");
      }
      setDocModalLoading(false);
    }
  };
  const openDocModal = async (href: string) => {
    setDocModalOpen(true);
    setDocModalPath(href);
    setDocModalContent("");
    setDocModalError("");
    setDocModalLoading(true);
    await fetchDocContent(href);
  };
  const closeDocModal = () => {
    setDocModalOpen(false);
    setDocModalPath("");
    setDocModalContent("");
    setDocModalError("");
    setDocModalLoading(false);
  };
  const markdownComponentsWithLinks = {
    ...markdownComponents,
    a: ({
      className,
      href,
      onClick,
      ...props
    }: ComponentPropsWithoutRef<"a">) => {
      const docLink = isDocLink(href);
      const mergedClassName = ["text-sm text-blue-600 underline", className]
        .filter(Boolean)
        .join(" ");
      return (
        <a
          className={mergedClassName}
          href={href}
          target={docLink ? undefined : "_blank"}
          rel={docLink ? undefined : "noreferrer"}
          onClick={(event) => {
            onClick?.(event);
            if (!docLink || !href) return;
            event.preventDefault();
            openDocModal(href);
          }}
          {...props}
        />
      );
    },
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
  useEffect(() => {
    setFileItems(null);
    setFileListLoaded(false);
    setFileError("");
    setSelectedFile("");
    setFileContent("");
    setFileContentError("");
    setFileModalOpen(false);
    setFileModalPath("");
  }, [skillPath]);
  useEffect(() => {
    if (!skillPath) return;
    if (fileLoading || fileError || fileListLoaded) return;
    fetchFileList();
  }, [fetchFileList, fileError, fileListLoaded, fileLoading, skillPath]);
  useEffect(() => {
    if (activeTab !== "files") return;
    if (fileLoading || fileError || fileListLoaded) return;
    fetchFileList();
  }, [activeTab, fileError, fileListLoaded, fileLoading, fetchFileList]);

  const headerName = skill?.name || displayName;
  const headerDesc = skill?.description || "暂无描述";
  const headerIcon = skill?.icon || "🧩";
  const fileCount = fileItems?.length ?? 0;

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
            <div className="mt-6">
              <div className="flex flex-wrap items-center gap-3 border-b">
                <button
                  type="button"
                  className={`flex items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium ${
                    activeTab === "doc"
                      ? "border-gray-900 text-gray-900"
                      : "border-transparent text-gray-500 hover:text-gray-700"
                  }`}
                  onClick={() => setActiveTab("doc")}
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
                    <path d="M7 3h7l5 5v13a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" />
                    <path d="M14 3v5h5" />
                    <path d="M9 12h6M9 16h6" />
                  </svg>
                  说明文档
                </button>
                <button
                  type="button"
                  className={`flex items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium ${
                    activeTab === "files"
                      ? "border-gray-900 text-gray-900"
                      : "border-transparent text-gray-500 hover:text-gray-700"
                  }`}
                  onClick={() => setActiveTab("files")}
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
                    <path d="M3 7h18" />
                    <path d="M5 7v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7" />
                    <path d="M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                  技能文件
                  <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
                    {fileCount}
                  </span>
                </button>
              </div>
              {activeTab === "doc" ? (
                <div className="mt-4 rounded-lg border bg-gray-50 p-5">
                  {parsedContent.body ? (
                    <ReactMarkdown
                      components={markdownComponentsWithLinks}
                      remarkPlugins={[remarkGfm]}
                    >
                      {parsedContent.body}
                    </ReactMarkdown>
                  ) : (
                    <div className="text-sm text-gray-500">暂无文档内容</div>
                  )}
                </div>
              ) : (
                <div className="mt-4">
                  <div className="rounded-lg border bg-gray-50 p-3">
                    <div className="flex items-center justify-between px-2 py-1 text-xs font-medium text-gray-500">
                      <span>文件列表</span>
                      <button
                        type="button"
                        className="text-gray-400 hover:text-gray-600"
                        onClick={fetchFileList}
                      >
                        刷新
                      </button>
                    </div>
                    {fileLoading ? (
                      <div className="px-2 py-3 text-sm text-gray-500">
                        加载中...
                      </div>
                    ) : fileError ? (
                      <div className="px-2 py-3 text-sm text-red-600">
                        {fileError}
                      </div>
                    ) : (fileItems?.length ?? 0) === 0 ? (
                      <div className="px-2 py-3 text-sm text-gray-500">
                        暂无文件
                      </div>
                    ) : (
                      <div className="mt-2 space-y-1">
                        {(fileItems ?? []).map((item, index) => {
                          const fileId = item.id ?? "";
                          const relativePath = toRelativeFilePath(fileId);
                          const displayName =
                            relativePath ||
                            fileId.split("/").filter(Boolean).pop() ||
                            "未知文件";
                          const isActive = fileId === selectedFile;
                          return (
                            <button
                              key={`${fileId}-${index}`}
                              type="button"
                              className={`flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm ${
                                isActive
                                  ? "bg-white text-gray-900 shadow-sm"
                                  : "text-gray-600 hover:bg-white hover:text-gray-900"
                              }`}
                              onClick={() => openFileModal(fileId)}
                            >
                              {renderFileIcon(item, fileId)}
                              <span className="whitespace-normal break-all">
                                {displayName}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </>
        )}
      </div>
      {docModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={closeDocModal}
        >
          <div
            className="flex w-full max-w-3xl flex-col overflow-hidden rounded-xl bg-white shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b px-4 py-3">
              <div className="min-w-0 flex-1 text-sm font-semibold text-gray-900">
                <div className="truncate">{docModalPath || "文档内容"}</div>
              </div>
              <button
                className="ml-3 text-sm text-gray-500 hover:text-gray-700"
                type="button"
                onClick={closeDocModal}
              >
                关闭
              </button>
            </div>
            <div className="max-h-[70vh] overflow-auto p-4">
              {docModalLoading ? (
                <div className="text-sm text-gray-500">加载中...</div>
              ) : docModalError ? (
                <div className="text-sm text-red-600">{docModalError}</div>
              ) : docIsMarkdown && parsedDocContent.body ? (
                <ReactMarkdown
                  components={markdownComponentsWithLinks}
                  remarkPlugins={[remarkGfm]}
                >
                  {parsedDocContent.body}
                </ReactMarkdown>
              ) : docModalContent ? (
                <pre className="whitespace-pre-wrap text-sm text-gray-700">
                  {docModalContent}
                </pre>
              ) : (
                <div className="text-sm text-gray-500">暂无文档内容</div>
              )}
            </div>
          </div>
        </div>
      )}
      {fileModalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={closeFileModal}
        >
          <div
            className="flex w-full max-w-3xl flex-col overflow-hidden rounded-xl bg-white shadow-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b px-4 py-3">
              <div className="min-w-0 flex-1 text-sm font-semibold text-gray-900">
                <div className="truncate">
                  {fileModalPath || "文件内容"}
                </div>
              </div>
              <button
                className="ml-3 text-sm text-gray-500 hover:text-gray-700"
                type="button"
                onClick={closeFileModal}
              >
                关闭
              </button>
            </div>
            <div className="max-h-[70vh] overflow-auto p-4">
              {fileContentLoading ? (
                <div className="text-sm text-gray-500">加载中...</div>
              ) : fileContentError ? (
                <div className="text-sm text-red-600">{fileContentError}</div>
              ) : fileContent ? (
                <pre className="whitespace-pre-wrap text-sm text-gray-700">
                  {fileContent}
                </pre>
              ) : (
                <div className="text-sm text-gray-500">暂无文件内容</div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
