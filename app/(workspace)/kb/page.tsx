"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type MouseEvent,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import {
  buildAuthHeaders,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import {
  Card,
  List,
  Space,
  Spin,
  Tree,
  Typography,
  Upload,
} from "@douyinfe/semi-ui-19";
import { Modal } from "../a2ui/components/Modal";
import { ConfirmDialog } from "../a2ui/components/ConfirmDialog";
import {
  IconDelete,
  IconFolder,
  IconFile,
  IconPlus,
  IconRefresh,
  IconUpload,
} from "@douyinfe/semi-icons";

type KnowledgeBase = {
  name?: string;
  uri?: string;
};

type TreeEntry = {
  rel_path?: string;
  name?: string;
  isDir?: boolean;
  uri?: string;
  size?: number;
};

type TreeNode = {
  key: string;
  label: string;
  children?: TreeNode[];
  isLeaf?: boolean;
  isDir?: boolean;
  path: string;
};

type SelectedNode = {
  path: string;
  isDir: boolean;
};

const API_BASE = "/api/backend";

const buildNodeKey = (segments: string[], isDir: boolean) => {
  if (segments.length === 0) return "";
  const base = segments.join("/");
  return isDir ? `${base}/` : base;
};

const normalizeRelPath = (relPath: string) => {
  const trimmed = relPath.replace(/^\/+/, "");
  return trimmed;
};

const splitPath = (relPath: string) => {
  return relPath.split("/").filter(Boolean);
};

const parseEntryUriPath = (kbName: string, uri?: string) => {
  if (!kbName) return "";
  if (!uri) return "";
  const prefix = `viking://resources/${kbName}/`;
  if (!uri.startsWith(prefix)) return "";
  return uri.slice(prefix.length);
};

const buildTreeNodes = (entries: TreeEntry[], kbName: string) => {
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
    const rawPath = parseEntryUriPath(kbName, entry.uri);
    if (!rawPath) return;
    let relPath = normalizeRelPath(rawPath);
    const isDir = entry.isDir || relPath.endsWith("/");
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

const getParentDir = (pathValue: string) => {
  const trimmed = pathValue.replace(/\/+$/, "");
  const parts = trimmed.split("/").filter(Boolean);
  if (parts.length <= 1) return "";
  return `${parts.slice(0, -1).join("/")}/`;
};

const getBaseName = (pathValue: string) => {
  const trimmed = pathValue.replace(/\/+$/, "");
  const parts = trimmed.split("/").filter(Boolean);
  return parts[parts.length - 1] ?? "";
};

const buildUri = (kbName: string, relPath: string) => {
  const normalized = relPath.replace(/^\/+/, "");
  if (!normalized) return `viking://resources/${kbName}/`;
  return `viking://resources/${kbName}/${normalized}`;
};

const formatNodePath = (pathValue: string) => {
  const trimmed = pathValue.replace(/\/+$/, "");
  return trimmed || "/";
};

export default function KnowledgeBasePage() {
  const router = useRouter();
  const [kbs, setKbs] = useState<KnowledgeBase[]>([]);
  const [loading, setLoading] = useState(false);
  const [treeLoading, setTreeLoading] = useState(false);
  const [selectedKb, setSelectedKb] = useState("");
  const [treeEntries, setTreeEntries] = useState<TreeEntry[]>([]);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);
  const [createVisible, setCreateVisible] = useState(false);
  const [createName, setCreateName] = useState("");
  const [createFile, setCreateFile] = useState<File | null>(null);
  const [creating, setCreating] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [deleteKbVisible, setDeleteKbVisible] = useState(false);
  const [deleteEntryVisible, setDeleteEntryVisible] = useState(false);
  const [pendingEntry, setPendingEntry] = useState<TreeNode | null>(null);
  const [deletingKb, setDeletingKb] = useState(false);
  const [deletingEntry, setDeletingEntry] = useState(false);

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
    const token = getValidToken();
    if (!token) return;
    setLoading(true);
    setErrorMessage("");
    try {
      const res = await fetch(`${API_BASE}/v1/kbs`, {
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取知识库列表失败");
      }
      const list = data.data ?? [];
      setKbs(list);
      if (list.length && !list.find((item: KnowledgeBase) => item.name === selectedKb)) {
        setSelectedKb(list[0].name ?? "");
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
  }, [getValidToken, selectedKb]);

  const fetchTree = useCallback(
    async (kbName: string) => {
      const token = getValidToken();
      if (!token || !kbName) return;
      setTreeLoading(true);
      setErrorMessage("");
      try {
        const res = await fetch(`${API_BASE}/v1/kbs/${encodeURIComponent(kbName)}/tree`, {
          headers: buildAuthHeaders(token),
        });
        updateTokenFromResponse(res);
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "获取知识库文件失败");
        }
        setTreeEntries(data.data ?? []);
      } catch (err) {
        const message = err instanceof Error ? err.message : "获取知识库文件失败";
        showError(message);
      } finally {
        setTreeLoading(false);
      }
    },
    [getValidToken]
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

  const treeData = useMemo(
    () => buildTreeNodes(treeEntries, selectedKb),
    [treeEntries, selectedKb]
  );
  const kbStats = useMemo(() => {
    let folders = 0;
    let files = 0;

    treeEntries.forEach((entry) => {
      if (entry.isDir) {
        folders += 1;
      } else {
        files += 1;
      }
    });

    return {
      folders,
      files,
    };
  }, [treeEntries]);
  const uploadTargetLabel = useMemo(() => {
    if (!selectedNode) return "根目录";
    if (selectedNode.isDir) {
      return formatNodePath(selectedNode.path);
    }
    const parentDir = getParentDir(selectedNode.path);
    return formatNodePath(parentDir);
  }, [selectedNode]);

  const handleCreate = async () => {
    if (!createName.trim()) {
      showError("请输入知识库名称");
      return;
    }
    const token = getValidToken();
    if (!token) return;
    setCreating(true);
    setErrorMessage("");
    try {
      const formData = new FormData();
      formData.append("kb_name", createName.trim());
      if (createFile) {
        formData.append("file", createFile);
      }
      const res = await fetch(`${API_BASE}/v1/kbs/import`, {
        method: "POST",
        headers: buildAuthHeaders(token),
        body: formData,
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "创建知识库失败");
      }
      showSuccess("知识库创建成功");
      setCreateVisible(false);
      setCreateName("");
      setCreateFile(null);
      await fetchList();
    } catch (err) {
      const message = err instanceof Error ? err.message : "创建知识库失败";
      showError(message);
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteKb = () => {
    if (!selectedKb) return;
    setDeleteKbVisible(true);
  };

  const confirmDeleteKb = async () => {
    if (!selectedKb) return;
    const token = getValidToken();
    if (!token) return;
    setDeletingKb(true);
    setErrorMessage("");
    try {
      const res = await fetch(`${API_BASE}/v1/kbs/${encodeURIComponent(selectedKb)}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "删除知识库失败");
      }
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

  const handleDeleteEntry = (node: TreeNode) => {
    setPendingEntry(node);
    setDeleteEntryVisible(true);
  };

  const confirmDeleteEntry = async () => {
    if (!selectedKb || !pendingEntry) return;
    const token = getValidToken();
    if (!token) return;
    setDeletingEntry(true);
    setErrorMessage("");
    try {
      const params = new URLSearchParams();
      params.set("uri", buildUri(selectedKb, pendingEntry.path));
      if (pendingEntry.isDir) {
        params.set("recursive", "true");
      }
      const res = await fetch(`${API_BASE}/v1/kbs/entry?${params.toString()}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "删除失败");
      }
      showSuccess("删除成功");
      setDeleteEntryVisible(false);
      setPendingEntry(null);
      await fetchTree(selectedKb);
    } catch (err) {
      const message = err instanceof Error ? err.message : "删除失败";
      showError(message);
    } finally {
      setDeletingEntry(false);
    }
  };

  const handleMove = async (
    draggingNode: TreeNode,
    targetNode: TreeNode,
    dropToGap: boolean
  ) => {
    if (!selectedKb) return;
    const token = getValidToken();
    if (!token) return;
    const fromPath = draggingNode.path;
    const dragBase = getBaseName(fromPath);
    const targetDir = dropToGap
      ? getParentDir(targetNode.path)
      : targetNode.isDir
      ? targetNode.path
      : getParentDir(targetNode.path);
    const toPath = targetDir ? `${targetDir}${dragBase}${draggingNode.isDir ? "/" : ""}` : `${dragBase}${draggingNode.isDir ? "/" : ""}`;
    if (fromPath === toPath) return;
    try {
      setErrorMessage("");
      const res = await fetch(`${API_BASE}/v1/kbs/drag`, {
        method: "POST",
        headers: {
          ...buildAuthHeaders(token),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          from_uri: buildUri(selectedKb, fromPath),
          to_uri: buildUri(selectedKb, toPath),
        }),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "移动失败");
      }
      showSuccess("移动成功");
      await fetchTree(selectedKb);
    } catch (err) {
      const message = err instanceof Error ? err.message : "移动失败";
      showError(message);
    }
  };

  const handleUploadFile = async (file: File) => {
    if (!selectedKb) return;
    const token = getValidToken();
    if (!token) return;
    setUploading(true);
    setErrorMessage("");
    try {
      const targetDir = selectedNode
        ? selectedNode.isDir
          ? selectedNode.path
          : getParentDir(selectedNode.path)
        : "";
      const formData = new FormData();
      formData.append("kb_name", selectedKb);
      formData.append("file", file);
      formData.append("target", targetDir);
      const res = await fetch(`${API_BASE}/v1/kbs/file`, {
        method: "POST",
        headers: buildAuthHeaders(token),
        body: formData,
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "上传失败");
      }
      showSuccess("上传成功");
      await fetchTree(selectedKb);
    } catch (err) {
      const message = err instanceof Error ? err.message : "上传失败";
      showError(message);
    } finally {
      setUploading(false);
    }
  };

  const treeRenderLabel = (label?: ReactNode, treeNode?: any) => {
    const node = treeNode as TreeNode | undefined;
    const displayLabel = typeof label === "string" ? label : node?.label ?? "";
    if (!node) return <span>{displayLabel}</span>;
    const isSelected = selectedNode?.path === node.path;
    return (
      <div
        className={`group flex w-full items-center justify-between gap-2 rounded-[var(--radius-md)] px-1 py-1 ${
          isSelected
            ? "bg-[linear-gradient(90deg,rgba(37,99,255,0.12),rgba(37,99,255,0.04))] text-[var(--color-text-primary)]"
            : "text-[var(--color-text-secondary)]"
        }`}
      >
        <span className="flex min-w-0 items-center gap-2">
          {node.isDir ? <IconFolder /> : <IconFile />}
          <span className="truncate">{displayLabel}</span>
        </span>
        <UiButton
          provider="semi"
          variant="danger"
          size="sm"
          className="opacity-70 motion-safe-highlight group-hover:opacity-100"
          onClick={(event: MouseEvent) => {
            event.stopPropagation();
            handleDeleteEntry(node);
          }}
        >
          <IconDelete />
        </UiButton>
      </div>
    );
  };

  return (
    <div className="kb-page workspace-gradient-surface workspace-gradient-surface--kb h-full overflow-auto p-4 md:p-6">
      <div className="mx-auto flex max-w-6xl flex-col gap-4">
        <div className="workspace-panel-card motion-safe-fade-in rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5 backdrop-blur-sm">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-3">
              <div>
                <Typography.Title heading={4} className="mb-0">
                  知识库管理
                </Typography.Title>
                <div className="mt-1 text-sm text-[var(--color-text-muted)]">
                  统一管理知识库、文件树和上传操作
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <span className="rounded-full border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.08)] px-3 py-1 text-xs font-medium text-[var(--color-action-primary)]">
                  知识库 {kbs.length}
                </span>
                <span className="rounded-full border border-[rgba(0,191,165,0.14)] bg-[rgba(0,191,165,0.08)] px-3 py-1 text-xs font-medium text-[#0f766e]">
                  文件 {kbStats.files}
                </span>
                <span className="rounded-full border border-[rgba(148,163,184,0.18)] bg-[rgba(148,163,184,0.08)] px-3 py-1 text-xs font-medium text-[var(--color-text-secondary)]">
                  目录 {kbStats.folders}
                </span>
              </div>
            </div>
            <Space wrap>
              <UiButton
                provider="semi"
                size="lg"
                className="ui-button-spotlight"
                onClick={() => setCreateVisible(true)}
              >
                <IconPlus />
                新建知识库
              </UiButton>
              <UiButton
                provider="semi"
                variant="secondary"
                onClick={fetchList}
                loading={loading}
              >
                <IconRefresh />
                刷新列表
              </UiButton>
              <UiButton
                provider="semi"
                variant="danger"
                disabled={!selectedKb}
                onClick={handleDeleteKb}
              >
                <IconDelete />
                删除知识库
              </UiButton>
            </Space>
          </div>
        </div>

        {errorMessage && (
          <div className="motion-safe-slide-up rounded-[var(--radius-lg)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
            {errorMessage}
          </div>
        )}
        {successMessage && (
          <div className="motion-safe-slide-up rounded-[var(--radius-lg)] border border-[rgba(22,163,74,0.18)] bg-[rgba(22,163,74,0.08)] px-4 py-3 text-sm text-[var(--color-state-success)]">
            {successMessage}
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[280px_1fr]">
          <Card
            className="motion-safe-lift"
            title="知识库列表"
            bodyStyle={{ padding: 0 }}
          >
            {loading && kbs.length === 0 ? (
              <div className="space-y-3 p-4">
                <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.14)]" />
                <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.1)]" />
                <div className="h-9 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.08)]" />
              </div>
            ) : (
              <Spin spinning={loading}>
                {kbs.length ? (
                  <List
                    dataSource={kbs}
                    renderItem={(item: KnowledgeBase) => {
                      const name = item.name ?? "";
                      const active = name === selectedKb;
                      return (
                        <List.Item
                          onClick={() => setSelectedKb(name)}
                          className={`motion-safe-highlight cursor-pointer px-4 py-3 ${
                            active
                              ? "bg-[linear-gradient(90deg,rgba(37,99,255,0.12),rgba(37,99,255,0.04))] text-[var(--color-text-primary)]"
                              : "text-[var(--color-text-secondary)] hover:bg-[rgba(241,245,249,0.9)]"
                          }`}
                        >
                          <span className="text-sm font-medium">
                            {name || "未命名知识库"}
                          </span>
                        </List.Item>
                      );
                    }}
                  />
                ) : (
                  <div className="px-4 py-6 text-sm text-[var(--color-text-muted)]">
                    暂无知识库，先创建一个新的空间。
                  </div>
                )}
              </Spin>
            )}
          </Card>

          <Card
            className="motion-safe-lift"
            title={selectedKb ? `${selectedKb} 文件树` : "文件树"}
            headerExtraContent={
              <Space wrap>
                <Upload
                  action={`${API_BASE}/v1/kbs/file`}
                  customRequest={(options: any) => {
                    const { fileInstance, onSuccess, onError } = options ?? {};
                    if (!fileInstance || !(fileInstance instanceof File)) return;
                    handleUploadFile(fileInstance)
                      .then(() => onSuccess?.({}))
                      .catch(() => onError?.({ status: 500 }, undefined));
                  }}
                  showUploadList={false}
                >
                  <UiButton
                    provider="semi"
                    size="lg"
                    className="ui-button-spotlight"
                    disabled={!selectedKb}
                    loading={uploading}
                  >
                    <IconUpload />
                    上传文件
                  </UiButton>
                </Upload>
                <UiButton
                  provider="semi"
                  variant="secondary"
                  onClick={() => selectedKb && fetchTree(selectedKb)}
                  disabled={!selectedKb}
                >
                  <IconRefresh />
                  刷新
                </UiButton>
              </Space>
            }
          >
            {selectedKb && (
              <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(248,250,252,0.78)] px-3 py-2">
                <div className="min-w-0">
                  <div className="text-xs font-medium text-[var(--color-text-secondary)]">
                    当前上传目录
                  </div>
                  <div className="truncate text-sm text-[var(--color-text-primary)]">
                    {uploadTargetLabel}
                  </div>
                </div>
                {selectedNode && (
                  <UiButton
                    provider="semi"
                    variant="secondary"
                    size="sm"
                    onClick={() => setSelectedNode(null)}
                  >
                    取消选择
                  </UiButton>
                )}
              </div>
            )}
            {treeLoading && treeEntries.length === 0 ? (
              <div className="space-y-3 p-2">
                <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.12)]" />
                <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.1)]" />
                <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.08)]" />
                <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.06)]" />
              </div>
            ) : (
              <Spin spinning={treeLoading}>
                {selectedKb ? (
                  treeData.length ? (
                    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(248,250,252,0.72)] p-2">
                      <Tree
                        treeData={treeData}
                        className="kb-tree"
                        draggable
                        expandAll
                        renderLabel={treeRenderLabel}
                        onSelect={(
                          selectedKey: string,
                          selected: boolean,
                          selectedNode: any
                        ) => {
                          if (selected && selectedNode) {
                            setSelectedNode({
                              path: selectedNode.path ?? selectedKey,
                              isDir: !!selectedNode.isDir,
                            });
                          }
                        }}
                        onDrop={(info: any) => {
                          const dragNode = info?.dragNode;
                          const node = info?.node;
                          if (dragNode?.path && node?.path) {
                            handleMove(dragNode, node, !!info.dropToGap);
                          }
                        }}
                      />
                    </div>
                  ) : (
                    <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] bg-[rgba(248,250,252,0.78)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
                      当前知识库为空，可上传文件或拖入 zip 初始化内容。
                    </div>
                  )
                ) : (
                  <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] bg-[rgba(248,250,252,0.78)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
                    暂无知识库，请先从左侧选择或创建。
                  </div>
                )}
              </Spin>
            )}
          </Card>
        </div>
      </div>

      <Modal
        open={createVisible}
        title="新建知识库"
        onClose={() => setCreateVisible(false)}
        footer={
          <>
            <UiButton
              type="button"
              variant="secondary"
              onClick={() => setCreateVisible(false)}
            >
              取消
            </UiButton>
            <UiButton
              type="button"
              onClick={handleCreate}
              disabled={creating}
              className="min-w-24"
            >
              {creating ? "创建中..." : "创建"}
            </UiButton>
          </>
        }
      >
        <div className="space-y-4">
          <UiInput
            provider="semi"
            placeholder="输入知识库名称"
            value={createName}
            onChange={(event) => setCreateName(event.target.value)}
          />
          <Upload
            action={`${API_BASE}/v1/kbs/import`}
            draggable
            uploadTrigger="custom"
            onFileChange={(files: File[]) => setCreateFile(files[0] ?? null)}
            customRequest={(options: any) => options?.onSuccess?.({})}
            showUploadList
            accept=".zip"
            dragMainText="选择或拖拽知识库 zip"
            dragSubText="支持空知识库，zip 可选"
          />
          <div className="text-xs text-[var(--color-text-muted)]">
            {createFile ? `已选择 ${createFile.name}` : "未选择 zip 文件"}
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={deleteKbVisible}
        title="删除知识库"
        description={`确认删除知识库 ${selectedKb} 吗？此操作会删除全部文件。`}
        confirmText="删除"
        cancelText="取消"
        confirming={deletingKb}
        onConfirm={confirmDeleteKb}
        onCancel={() => setDeleteKbVisible(false)}
      />

      <ConfirmDialog
        open={deleteEntryVisible}
        title="删除文件"
        description={`确认删除 ${pendingEntry?.label ?? "该文件"} 吗？`}
        confirmText="删除"
        cancelText="取消"
        confirming={deletingEntry}
        onConfirm={confirmDeleteEntry}
        onCancel={() => {
          setDeleteEntryVisible(false);
          setPendingEntry(null);
        }}
      />
    </div>
  );
}
