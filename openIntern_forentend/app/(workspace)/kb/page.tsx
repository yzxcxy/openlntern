"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type MouseEvent,
} from "react";
import { useRouter } from "next/navigation";
import { readStoredUser, readValidToken, requestBackend } from "../auth";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiConfirmDialog as ConfirmDialog } from "../../components/ui/UiConfirmDialog";
import { UiModal as Modal } from "../../components/ui/UiModal";

type KnowledgeBase = {
  name?: string;
};

type TreeEntry = {
  path?: string;
  name?: string;
  is_dir?: boolean;
  size?: number;
};

type TreeNode = {
  key: string;
  label: string;
  children?: TreeNode[];
  isLeaf: boolean;
  isDir: boolean;
  path: string;
};

const buildNodeKey = (segments: string[], isDir: boolean) => {
  if (segments.length === 0) return "";
  const base = segments.join("/");
  return isDir ? `${base}/` : base;
};

const normalizeRelPath = (relPath: string) => {
  return relPath.replace(/^\/+/, "");
};

const readString = (...values: unknown[]) => {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) {
      return value;
    }
  }
  return "";
};

const readBoolean = (...values: unknown[]) => {
  for (const value of values) {
    if (typeof value === "boolean") {
      return value;
    }
    if (typeof value === "string") {
      const normalized = value.trim().toLowerCase();
      if (normalized === "true" || normalized === "1") {
        return true;
      }
      if (normalized === "false" || normalized === "0") {
        return false;
      }
    }
    if (typeof value === "number") {
      return value !== 0;
    }
  }
  return false;
};

const readNumber = (...values: unknown[]) => {
  for (const value of values) {
    if (typeof value === "number" && Number.isFinite(value)) {
      return value;
    }
    if (typeof value === "string" && value.trim()) {
      const parsed = Number(value);
      if (Number.isFinite(parsed)) {
        return parsed;
      }
    }
  }
  return undefined;
};

const splitPath = (relPath: string) => {
  return relPath.split("/").filter(Boolean);
};

const buildTreeNodes = (entries: TreeEntry[]): TreeNode[] => {
  const rootNodes: TreeNode[] = [];
  const nodeMap = new Map<string, TreeNode>();

  const ensureNode = (segments: string[], isDir: boolean) => {
    const key = buildNodeKey(segments, isDir);
    if (!key) return null;
    const existing = nodeMap.get(key);
    if (existing) return existing;
    const label = segments[segments.length - 1] ?? "";
    const node: TreeNode = {
      key,
      label,
      isLeaf: !isDir,
      isDir,
      path: key,
      children: isDir ? [] : undefined,
    };
    nodeMap.set(key, node);
    const parentSegments = segments.slice(0, -1);
    const parentKey = buildNodeKey(parentSegments, true);
    if (!parentSegments.length) {
      rootNodes.push(node);
    } else {
      const parent = nodeMap.get(parentKey);
      if (parent) {
        parent.children = parent.children ?? [];
        parent.children.push(node);
      } else {
        const fallbackParent = ensureNode(parentSegments, true);
        if (fallbackParent) {
          fallbackParent.children = fallbackParent.children ?? [];
          fallbackParent.children.push(node);
        } else {
          rootNodes.push(node);
        }
      }
    }
    return node;
  };

  entries.forEach((entry) => {
    const rawPath = entry.path ? normalizeRelPath(entry.path) : "";
    if (!rawPath) return;
    let relPath = normalizeRelPath(rawPath);
    const isDir = entry.is_dir || relPath.endsWith("/");
    if (isDir && !relPath.endsWith("/")) {
      relPath = `${relPath}/`;
    }
    const segments = splitPath(relPath);
    if (segments.length === 0) return;
    segments.forEach((_, index) => {
      const isLast = index === segments.length - 1;
      const nodeIsDir = isLast ? isDir : true;
      const nodeSegments = segments.slice(0, index + 1);
      ensureNode(nodeSegments, nodeIsDir);
    });
  });

  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.isDir !== b.isDir) {
        return a.isDir ? -1 : 1;
      }
      return a.label.localeCompare(b.label, "zh-Hans-CN");
    });
    nodes.forEach((node) => {
      if (node.children?.length) {
        sortNodes(node.children);
      }
    });
  };

  sortNodes(rootNodes);
  return rootNodes;
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

// Folder Icon
const IconFolder = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
  </svg>
);

// File Icon
const IconFile = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <polyline points="14 2 14 8 20 8" />
  </svg>
);

// Plus Icon
const IconPlus = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <line x1="12" y1="5" x2="12" y2="19" />
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
);

// Refresh Icon
const IconRefresh = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M21 12a9 9 0 1 1-2.64-6.36" />
    <path d="M21 3v6h-6" />
  </svg>
);

// Upload Icon
const IconUpload = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
    <polyline points="17 8 12 3 7 8" />
    <line x1="12" y1="3" x2="12" y2="15" />
  </svg>
);

// Delete Icon
const IconDelete = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <polyline points="3 6 5 6 21 6" />
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  </svg>
);

// Eye Icon for preview
const IconEye = ({ className }: { className?: string }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
    <circle cx="12" cy="12" r="3" />
  </svg>
);

// Chevron Icon for tree expand/collapse
const IconChevron = ({ className, expanded }: { className?: string; expanded?: boolean }) => (
  <svg
    className={joinClasses(className, "transition-transform duration-200", expanded && "rotate-90")}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

// Tree Item Component
const TreeItem = ({
  node,
  level,
  selectedNode,
  onSelect,
  onPreview,
  expandedNodes,
  toggleExpand,
}: {
  node: TreeNode;
  level: number;
  selectedNode: TreeNode | null;
  onSelect: (node: TreeNode) => void;
  onPreview: (node: TreeNode) => void;
  expandedNodes: Set<string>;
  toggleExpand: (key: string) => void;
}) => {
  const isSelected = selectedNode?.path === node.path;
  const isExpanded = expandedNodes.has(node.key);
  const hasChildren = node.children && node.children.length > 0;

  const handleClick = () => {
    onSelect(node);
    if (node.isDir && hasChildren) {
      toggleExpand(node.key);
    }
  };

  const handleDoubleClick = () => {
    if (!node.isDir) {
      onPreview(node);
    }
  };

  return (
    <div>
      <div
        className={joinClasses(
          "group flex cursor-pointer items-center gap-2 rounded-[var(--radius-md)] px-2 py-2 transition-colors",
          isSelected
            ? "border-[rgba(199,104,67,0.18)] bg-[linear-gradient(135deg,rgba(255,247,240,0.98),rgba(245,231,219,0.78))] text-[var(--color-text-primary)]"
            : "text-[var(--color-text-secondary)] hover:bg-[rgba(255,252,247,0.9)]"
        )}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
      >
        {node.isDir ? (
          <>
            {hasChildren && (
              <IconChevron className="h-4 w-4 shrink-0 text-[var(--color-text-muted)]" expanded={isExpanded} />
            )}
            {!hasChildren && <span className="w-4" />}
            <IconFolder className="h-4 w-4 shrink-0 text-[var(--color-text-muted)]" />
          </>
        ) : (
          <>
            <span className="w-4" />
            <IconFile className="h-4 w-4 shrink-0 text-[var(--color-text-muted)]" />
          </>
        )}
        <span className="min-w-0 flex-1 truncate text-sm">{node.label}</span>
        {!node.isDir && (
          <UiButton
            variant="ghost"
            size="sm"
            className="opacity-0 group-hover:opacity-100"
            onClick={(e: MouseEvent) => {
              e.stopPropagation();
              onPreview(node);
            }}
          >
            <IconEye className="h-4 w-4" />
          </UiButton>
        )}
      </div>
      {node.isDir && hasChildren && isExpanded && (
        <div>
          {node.children!.map((child) => (
            <TreeItem
              key={child.key}
              node={child}
              level={level + 1}
              selectedNode={selectedNode}
              onSelect={onSelect}
              onPreview={onPreview}
              expandedNodes={expandedNodes}
              toggleExpand={toggleExpand}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default function KnowledgeBasePage() {
  const router = useRouter();
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [loading, setLoading] = useState(false);
  const [treeLoading, setTreeLoading] = useState(false);
  const [selectedKb, setSelectedKb] = useState("");
  const [treeEntries, setTreeEntries] = useState<TreeEntry[]>([]);
  const [selectedNode, setSelectedNode] = useState<TreeNode | null>(null);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [createVisible, setCreateVisible] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createFile, setCreateFile] = useState<File | null>(null);
  const [creating, setCreating] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [deleteKbVisible, setDeleteKbVisible] = useState(false);
  const [deletingKb, setDeletingKb] = useState(false);

  // File preview state
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewNode, setPreviewNode] = useState<TreeNode | null>(null);
  const [previewContent, setPreviewContent] = useState<string>("");
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");

  const showError = (message: string) => {
    setErrorMessage(message);
    setSuccessMessage("");
  };

  const showSuccess = (message: string) => {
    setSuccessMessage(message);
    setErrorMessage("");
  };

  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const fetchList = useCallback(async () => {
    if (!getValidToken()) return;
    setLoading(true);
    setErrorMessage("");
    try {
      const data = await requestBackend<KnowledgeBase[]>("/v1/kbs", {
        fallbackMessage: "获取知识库列表失败",
        router,
      });
      const list = data.data ?? [];
      setKbs(list);
      if (list.length && !list.find((item: KnowledgeBase) => item.name === selectedKb)) {
        setSelectedKb(list[0]?.name ?? "");
      }
      if (!list.length) {
        setSelectedKb("");
        setTreeEntries([]);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "获取知识库列表失败";
      showError(message);
    } finally {
      setLoading(false);
    }
  }, [getValidToken, router, selectedKb]);

  const fetchTree = useCallback(
    async (kbName: string) => {
      if (!getValidToken() || !kbName) return;
      setTreeLoading(true);
      setErrorMessage("");
      try {
        const data = await requestBackend<TreeEntry[]>(
          `/v1/kbs/${encodeURIComponent(kbName)}/tree`,
          {
            fallbackMessage: "获取知识库文件失败",
            router,
          }
        );
        const entries = (data.data ?? []).map((entry: TreeEntry) => ({
          path: readString(entry.path),
          name: readString(entry.name),
          is_dir: readBoolean(entry.is_dir),
          size: readNumber(entry.size),
        }));
        setTreeEntries(entries);
        // Auto expand all nodes
        const allKeys = new Set<string>();
        const collectKeys = (nodes: TreeNode[]) => {
          nodes.forEach((n) => {
            if (n.isDir) {
              allKeys.add(n.key);
              if (n.children) collectKeys(n.children);
            }
          });
        };
        const treeNodes = buildTreeNodes(entries);
        collectKeys(treeNodes);
        setExpandedNodes(allKeys);
      } catch (err) {
        const message = err instanceof Error ? err.message : "获取知识库文件失败";
        showError(message);
      } finally {
        setTreeLoading(false);
      }
    },
    [getValidToken, router]
  );

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  useEffect(() => {
    if (selectedKb) {
      fetchTree(selectedKb);
    }
  }, [fetchTree, selectedKb]);

  useEffect(() => {
    setSelectedNode(null);
  }, [selectedKb]);

  const treeData = useMemo(() => buildTreeNodes(treeEntries), [treeEntries]);

  const resetCreateModal = useCallback(() => {
    setCreateVisible(false);
    setCreateName("");
    setCreateFile(null);
  }, []);

  const handleCreate = useCallback(async () => {
    if (!createName.trim()) {
      showError("请输入知识库名称");
      return;
    }
    if (!createFile) {
      showError("请选择zip文件");
      return;
    }
    if (!getValidToken()) return;
    setCreating(true);
    setErrorMessage("");
    try {
      const formData = new FormData();
      formData.append("kb_name", createName.trim());
      formData.append("file", createFile);
      await requestBackend("/v1/kbs/import", {
        method: "POST",
        body: formData,
        fallbackMessage: "创建知识库失败",
        router,
      });
      showSuccess("知识库创建请求已受理，后台处理中，请稍后刷新查看结果");
      resetCreateModal();
      await fetchList();
    } catch (err) {
      const message = err instanceof Error ? err.message : "创建知识库失败";
      showError(message);
    } finally {
      setCreating(false);
    }
  }, [createFile, createName, fetchList, getValidToken, resetCreateModal, router]);

  const handleDeleteKb = () => {
    if (!selectedKb) return;
    setDeleteKbVisible(true);
  };

  const confirmDeleteKb = async () => {
    if (!selectedKb) return;
    if (!getValidToken()) return;
    setDeletingKb(true);
    setErrorMessage("");
    try {
      await requestBackend(`/v1/kbs/${encodeURIComponent(selectedKb)}`, {
        method: "DELETE",
        fallbackMessage: "删除知识库失败",
        router,
      });
      showSuccess("知识库已删除");
      setDeleteKbVisible(false);
      await fetchList();
    } catch (err) {
      const message = err instanceof Error ? err.message : "删除知识库失败";
      showError(message);
    } finally {
      setDeletingKb(false);
    }
  };

  const toggleExpand = (key: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  // File type detection
  const getFileType = (filename: string): "text" | "image" | "binary" => {
    const ext = filename.split(".").pop()?.toLowerCase() || "";
    const textExts = ["txt", "md", "json", "yaml", "yml", "xml", "html", "css", "js", "ts", "jsx", "tsx", "py", "go", "java", "c", "cpp", "h", "sh", "bash", "zsh", "sql", "csv", "log", "conf", "ini", "toml", "env", "gitignore", "dockerfile", "makefile", "rst", "adoc", "tex", "vue", "svelte", "scss", "sass", "less"];
    const imageExts = ["png", "jpg", "jpeg", "gif", "webp", "svg", "ico", "bmp"];

    if (imageExts.includes(ext)) return "image";
    if (textExts.includes(ext) || !ext) return "text";
    return "binary";
  };

  // Fetch file content for preview
  const fetchFileContent = useCallback(async (node: TreeNode) => {
    if (!selectedKb || !node.path) return;

    const fileType = getFileType(node.label);
    if (fileType === "binary") {
      setPreviewError("该文件类型暂不支持预览");
      setPreviewContent("");
      return;
    }

    setPreviewLoading(true);
    setPreviewError("");
    setPreviewContent("");

    try {
      const params = new URLSearchParams();
      params.set("kb_name", selectedKb);
      params.set("path", node.path);

      const data = await requestBackend<{ content?: string }>(
        `/v1/kbs/entry/content?${params.toString()}`,
        {
          router,
          fallbackMessage: "获取文件内容失败",
        }
      );

      if (typeof data.data?.content === "string") {
        setPreviewContent(data.data.content);
      } else if (data.data) {
        setPreviewContent(JSON.stringify(data.data, null, 2));
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "获取文件内容失败";
      setPreviewError(message);
    } finally {
      setPreviewLoading(false);
    }
  }, [selectedKb, router]);

  const handlePreviewFile = useCallback((node: TreeNode) => {
    setPreviewNode(node);
    setPreviewVisible(true);
    void fetchFileContent(node);
  }, [fetchFileContent]);

  const closePreview = useCallback(() => {
    setPreviewVisible(false);
    setPreviewNode(null);
    setPreviewContent("");
    setPreviewError("");
  }, []);

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--kb h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="workspace-page-stack">
          {/* Filter Panel */}
          <section className="workspace-filter-panel">
            <div className="flex flex-wrap items-center gap-3">
              <UiButton
                onClick={() => setCreateVisible(true)}
                className="ui-button-soft-accent"
              >
                <IconPlus className="h-4 w-4" />
                导入知识库
              </UiButton>
              <UiButton
                variant="secondary"
                onClick={fetchList}
                disabled={loading}
              >
                <IconRefresh className={joinClasses("h-4 w-4", loading && "animate-spin")} />
                刷新列表
              </UiButton>
              <UiButton
                variant="danger"
                disabled={!selectedKb}
                onClick={handleDeleteKb}
              >
                <IconDelete className="h-4 w-4" />
                删除知识库
              </UiButton>
            </div>

            {(errorMessage || successMessage) && (
              <div className="mt-4 space-y-2">
                {errorMessage && (
                  <div className="rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                    {errorMessage}
                  </div>
                )}
                {successMessage && (
                  <div className="rounded-[18px] border border-[rgba(47,122,87,0.16)] bg-[rgba(47,122,87,0.08)] px-4 py-3 text-sm text-[var(--color-state-success)]">
                    {successMessage}
                  </div>
                )}
              </div>
            )}
          </section>

          {/* Main Content */}
          <section>
            <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[280px_1fr]">
              {/* KB List */}
              <div className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)]">
                <div className="border-b border-[var(--color-border-default)] px-4 py-3">
                  <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">知识库列表</h3>
                </div>
                <div className="max-h-[400px] overflow-auto">
                  {loading && kbs.length === 0 ? (
                    <div className="space-y-3 p-4">
                      <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.14)]" />
                      <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.1)]" />
                      <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.08)]" />
                    </div>
                  ) : kbs.length > 0 ? (
                    <div>
                      {kbs.map((item) => {
                        const name = item.name ?? "";
                        const active = name === selectedKb;
                        return (
                          <div
                            key={name}
                            onClick={() => setSelectedKb(name)}
                            className={joinClasses(
                              "cursor-pointer px-4 py-3 transition-colors",
                              active
                                ? "border-[rgba(199,104,67,0.18)] bg-[linear-gradient(135deg,rgba(255,247,240,0.98),rgba(245,231,219,0.78))] text-[var(--color-text-primary)]"
                                : "text-[var(--color-text-secondary)] hover:bg-[rgba(255,252,247,0.9)]"
                            )}
                          >
                            <span className="text-sm font-medium">
                              {name || "未命名知识库"}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="px-4 py-6 text-sm text-[var(--color-text-muted)]">
                      暂无知识库，点击上方按钮导入。
                    </div>
                  )}
                </div>
              </div>

              {/* File Tree */}
              <div className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)]">
                <div className="flex items-center justify-between border-b border-[var(--color-border-default)] px-4 py-3">
                  <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
                    {selectedKb ? `${selectedKb} 文件树` : "文件树"}
                  </h3>
                  <div className="flex items-center gap-2">
                    <UiButton
                      variant="secondary"
                      size="sm"
                      disabled={!selectedKb}
                      onClick={() => selectedKb && fetchTree(selectedKb)}
                    >
                      <IconRefresh className={joinClasses("h-4 w-4", treeLoading && "animate-spin")} />
                    </UiButton>
                  </div>
                </div>
                <div className="max-h-[400px] overflow-auto p-2">
                  {treeLoading && treeEntries.length === 0 ? (
                    <div className="space-y-3 p-2">
                      <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.12)]" />
                      <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.1)]" />
                      <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(209,157,86,0.08)]" />
                    </div>
                  ) : selectedKb ? (
                    treeData.length > 0 ? (
                      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.6)] p-2">
                        {treeData.map((node) => (
                          <TreeItem
                            key={node.key}
                            node={node}
                            level={0}
                            selectedNode={selectedNode}
                            onSelect={setSelectedNode}
                            onPreview={handlePreviewFile}
                            expandedNodes={expandedNodes}
                            toggleExpand={toggleExpand}
                          />
                        ))}
                      </div>
                    ) : (
                      <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] bg-[rgba(255,252,247,0.5)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
                        当前知识库为空。
                      </div>
                    )
                  ) : (
                    <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] bg-[rgba(255,252,247,0.5)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
                      暂无知识库，请先从左侧选择或导入。
                    </div>
                  )}
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      {/* Create Modal */}
      <Modal
        open={createVisible}
        title="导入知识库"
        onClose={resetCreateModal}
        footer={
          <div className="flex justify-end gap-3">
            <UiButton variant="secondary" onClick={resetCreateModal}>
              取消
            </UiButton>
            <UiButton onClick={handleCreate} disabled={creating}>
              {creating ? "导入中..." : "导入"}
            </UiButton>
          </div>
        }
      >
        <div className="space-y-4">
          <UiInput
            placeholder="输入知识库名称"
            value={createName}
            onChange={(event) => setCreateName(event.target.value)}
          />
          <div>
            <label className="cursor-pointer">
              <input
                type="file"
                accept=".zip"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  setCreateFile(file ?? null);
                }}
              />
              <div className="rounded-[var(--radius-lg)] border-2 border-dashed border-[var(--color-border-default)] bg-[rgba(255,252,247,0.5)] p-8 text-center transition-colors hover:border-[rgba(199,104,67,0.24)] hover:bg-[rgba(255,247,240,0.8)]">
                <IconUpload className="mx-auto h-8 w-8 text-[var(--color-text-muted)]" />
                <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
                  点击选择知识库 zip 文件
                </p>
                <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                  必须上传 zip 文件
                </p>
              </div>
            </label>
          </div>
          <div className="text-xs text-[var(--color-text-muted)]">
            {createFile ? `已选择 ${createFile.name}` : "未选择 zip 文件"}
          </div>
        </div>
      </Modal>

      {/* Delete KB Confirm */}
      <ConfirmDialog
        open={deleteKbVisible}
        title="删除知识库"
        description={`确认删除知识库 ${selectedKb} 吗？此操作会删除全部文件且不可恢复。`}
        confirmText="删除"
        cancelText="取消"
        confirming={deletingKb}
        onConfirm={confirmDeleteKb}
        onCancel={() => setDeleteKbVisible(false)}
      />

      {/* File Preview Modal */}
      <Modal
        open={previewVisible}
        title={previewNode?.label ?? "文件预览"}
        onClose={closePreview}
        footer={null}
      >
        <div className="min-h-[200px]">
          {previewLoading ? (
            <div className="flex items-center justify-center py-12">
              <IconRefresh className="h-6 w-6 animate-spin text-[var(--color-text-muted)]" />
              <span className="ml-2 text-sm text-[var(--color-text-muted)]">加载中...</span>
            </div>
          ) : previewError ? (
            <div className="rounded-[var(--radius-lg)] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-6 text-center text-sm text-[var(--color-state-error)]">
              {previewError}
            </div>
          ) : previewNode && getFileType(previewNode.label) === "image" ? (
            <div className="flex items-center justify-center rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.5)] p-4">
              {previewContent.startsWith("data:") || previewContent.startsWith("http") ? (
                <img
                  src={previewContent}
                  alt={previewNode.label}
                  className="max-h-[400px] max-w-full rounded object-contain"
                />
              ) : (
                <span className="text-sm text-[var(--color-text-muted)]">无法预览此图片</span>
              )}
            </div>
          ) : (
            <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.7)] p-4">
              <pre className="max-h-[400px] overflow-auto whitespace-pre-wrap break-all font-mono text-xs leading-relaxed text-[var(--color-text-secondary)]">
                {previewContent}
              </pre>
            </div>
          )}
          {previewNode && (
            <div className="mt-4 text-xs text-[var(--color-text-muted)]">
              路径: {previewNode.path}
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}