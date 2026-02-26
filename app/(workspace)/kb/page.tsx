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
import {
  Button,
  Card,
  Input,
  List,
  Modal,
  Space,
  Spin,
  Tree,
  Typography,
  Upload,
} from "@douyinfe/semi-ui-19";
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

  const treeData = useMemo(
    () => buildTreeNodes(treeEntries, selectedKb),
    [treeEntries, selectedKb]
  );

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
    return (
      <div className="flex w-full items-center justify-between gap-2">
        <span className="flex items-center gap-2">
          {node.isDir ? <IconFolder /> : <IconFile />}
          <span className="truncate">{displayLabel}</span>
        </span>
        <Button
          theme="borderless"
          type="danger"
          size="small"
          icon={<IconDelete />}
          onClick={(event: MouseEvent) => {
            event.stopPropagation();
            handleDeleteEntry(node);
          }}
        />
      </div>
    );
  };

  return (
    <div className="h-full overflow-auto p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Typography.Title heading={4} className="mb-0">
          知识库管理
        </Typography.Title>
        <Space>
          <Button icon={<IconPlus />} theme="solid" onClick={() => setCreateVisible(true)}>
            新建知识库
          </Button>
          <Button icon={<IconRefresh />} onClick={fetchList} loading={loading}>
            刷新列表
          </Button>
          <Button
            icon={<IconDelete />}
            type="danger"
            disabled={!selectedKb}
            onClick={handleDeleteKb}
          >
            删除知识库
          </Button>
        </Space>
      </div>

      {errorMessage && (
        <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">
          {errorMessage}
        </div>
      )}
      {successMessage && (
        <div className="mt-3 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-600">
          {successMessage}
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[260px_1fr]">
        <Card title="知识库列表" bodyStyle={{ padding: 0 }}>
          <Spin spinning={loading}>
            <List
              dataSource={kbs}
              renderItem={(item: KnowledgeBase) => {
                const name = item.name ?? "";
                const active = name === selectedKb;
                return (
                  <List.Item
                    onClick={() => setSelectedKb(name)}
                    className={`cursor-pointer px-4 py-2 ${
                      active ? "bg-blue-50" : ""
                    }`}
                  >
                    <span className="text-sm">{name || "未命名知识库"}</span>
                  </List.Item>
                );
              }}
            />
          </Spin>
        </Card>

        <Card
          title={selectedKb ? `${selectedKb} 文件树` : "文件树"}
          headerExtraContent={
            <Space>
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
                <Button icon={<IconUpload />} disabled={!selectedKb} loading={uploading}>
                  上传文件
                </Button>
              </Upload>
              <Button
                icon={<IconRefresh />}
                onClick={() => selectedKb && fetchTree(selectedKb)}
                disabled={!selectedKb}
              >
                刷新
              </Button>
            </Space>
          }
        >
          <Spin spinning={treeLoading}>
            {selectedKb ? (
              <Tree
                treeData={treeData}
                draggable
                expandAll
                renderLabel={treeRenderLabel}
                onSelect={(selectedKey: string, selected: boolean, selectedNode: any) => {
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
            ) : (
              <div className="text-sm text-gray-500">暂无知识库</div>
            )}
          </Spin>
        </Card>
      </div>

      <Modal
        title="新建知识库"
        visible={createVisible}
        onCancel={() => setCreateVisible(false)}
        onOk={handleCreate}
        confirmLoading={creating}
      >
        <div className="space-y-4">
          <Input
            placeholder="输入知识库名称"
            value={createName}
            onChange={(value: string) => setCreateName(value)}
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
          <div className="text-xs text-gray-500">
            {createFile ? `已选择 ${createFile.name}` : "未选择 zip 文件"}
          </div>
        </div>
      </Modal>

      <Modal
        title="删除知识库"
        visible={deleteKbVisible}
        onCancel={() => setDeleteKbVisible(false)}
        onOk={confirmDeleteKb}
        confirmLoading={deletingKb}
      >
        <div className="text-sm text-gray-600">
          确认删除知识库 {selectedKb} 吗？此操作会删除全部文件。
        </div>
      </Modal>

      <Modal
        title="删除文件"
        visible={deleteEntryVisible}
        onCancel={() => {
          setDeleteEntryVisible(false);
          setPendingEntry(null);
        }}
        onOk={confirmDeleteEntry}
        confirmLoading={deletingEntry}
      >
        <div className="text-sm text-gray-600">
          确认删除 {pendingEntry?.label ?? "该文件"} 吗？
        </div>
      </Modal>
    </div>
  );
}
