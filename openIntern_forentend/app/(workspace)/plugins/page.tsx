"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import { readValidToken, requestBackend } from "../auth";
import {
  applyCodeToolDefaults,
  buildPayload,
  createField,
  createPluginDraft,
  createToolDraft,
  ensureRuntimeTools,
  flattenFields,
  formatTime,
  getCodeTemplate,
  getDefaultCodeBodyFields,
  getEffectiveCodeBodyFields,
  getInputSections,
  getOutputFields,
  getRuntimeBadgeClassName,
  getSourceBadgeClassName,
  getToolKey,
  isRecord,
  mcpProtocolLabel,
  normalizeMCPProtocolValue,
  parseFields,
  requiresRuntimeTools,
  responseModeLabel,
  runtimeLabel,
  sanitizeOutputFields,
  validateDraft,
  type DetailFieldSection,
  type FieldType,
  type MCPProtocol,
  type PluginDraft,
  type PluginField,
  type PluginRecord,
  type PluginTool,
  type RequestType,
  type ResponseMode,
  type RuntimeType,
  type ToolDraft,
} from "./plugin-editor";
import {
  CodeToolEditor,
  DetailSectionTitle,
  FieldListEditor,
  FieldTable,
  FormFieldRow,
  PluginAvatar,
  ToolDraftSwitcher,
} from "./plugin-editor-components";

export default function PluginsPage() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [runtimeFilter, setRuntimeFilter] = useState("");
  const [items, setItems] = useState<PluginRecord[]>([]);
  const [selectedPlugin, setSelectedPlugin] = useState<PluginRecord | null>(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(9);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [isWizardOpen, setIsWizardOpen] = useState(false);
  const [wizardStep, setWizardStep] = useState(1);
  const [draft, setDraft] = useState<PluginDraft>(createPluginDraft());
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [selectedToolKey, setSelectedToolKey] = useState("");
  const [wizardToolIndex, setWizardToolIndex] = useState(0);
  const [uploadingIcon, setUploadingIcon] = useState(false);
  const [defaultPluginIconURL, setDefaultPluginIconURL] = useState("");
  const iconUploadInputRef = useRef<HTMLInputElement | null>(null);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const previewJSON = useMemo(() => JSON.stringify(buildPayload(draft), null, 2), [draft]);
  const activeDraftTool = useMemo(() => {
    const nextTools = ensureRuntimeTools(draft.tools, draft.runtimeType);
    if (nextTools.length === 0) return null;
    return nextTools[Math.min(wizardToolIndex, nextTools.length - 1)] ?? null;
  }, [draft.runtimeType, draft.tools, wizardToolIndex]);
  const activeTool = useMemo(() => {
    const tools = selectedPlugin?.tools ?? [];
    if (tools.length === 0) return null;
    return (
      tools.find((tool, index) => getToolKey(tool, index) === selectedToolKey) ?? tools[0] ?? null
    );
  }, [selectedPlugin, selectedToolKey]);
  const activeToolInputSections = useMemo(
    () =>
      activeTool ? getInputSections(selectedPlugin?.runtime_type, activeTool) : [],
    [activeTool, selectedPlugin?.runtime_type]
  );
  const activeToolOutputFields = useMemo(
    () => (activeTool ? getOutputFields(activeTool) : []),
    [activeTool]
  );

  const getToken = useCallback(() => readValidToken(router), [router]);

  useEffect(() => {
    const tools = selectedPlugin?.tools ?? [];
    if (tools.length === 0) {
      setSelectedToolKey("");
      return;
    }

    const nextKeys = tools.map((tool, index) => getToolKey(tool, index));
    setSelectedToolKey((current) =>
      current && nextKeys.includes(current) ? current : (nextKeys[0] ?? "")
    );
  }, [selectedPlugin]);

  useEffect(() => {
    if (!requiresRuntimeTools(draft.runtimeType)) {
      setWizardToolIndex(0);
      return;
    }
    setWizardToolIndex((current) => {
      const nextTools = ensureRuntimeTools(draft.tools, draft.runtimeType);
      if (nextTools.length === 0) return 0;
      return Math.min(current, nextTools.length - 1);
    });
  }, [draft.runtimeType, draft.tools]);

  const fetchList = useCallback(async () => {
    if (!getToken()) return;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (searchKeyword.trim()) params.set("keyword", searchKeyword.trim());
      if (sourceFilter) params.set("source", sourceFilter);
      if (runtimeFilter) params.set("runtime_type", runtimeFilter);
      const data = await requestBackend<{ data: PluginRecord[]; total: number }>(
        `/v1/plugins?${params.toString()}`,
        {
          fallbackMessage: "获取插件列表失败",
          router,
        }
      );
      setItems(Array.isArray(data.data?.data) ? data.data.data : []);
      setTotal(typeof data.data?.total === "number" ? data.data.total : 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取插件列表失败");
    } finally {
      setLoading(false);
    }
  }, [
    getToken,
    page,
    pageSize,
    router,
    runtimeFilter,
    searchKeyword,
    sourceFilter,
  ]);

  useEffect(() => {
    void fetchList();
  }, [fetchList]);

  const fetchPluginDefaults = useCallback(async () => {
    if (!getToken()) return;
    try {
      const data = await requestBackend<{ default_icon_url?: string }>("/v1/plugins/defaults", {
        fallbackMessage: "获取插件默认配置失败",
        router,
      });
      const nextURL =
        typeof data.data?.default_icon_url === "string" ? data.data.default_icon_url.trim() : "";
      setDefaultPluginIconURL(nextURL);
    } catch {
      setDefaultPluginIconURL("");
    }
  }, [getToken, router]);

  useEffect(() => {
    void fetchPluginDefaults();
  }, [fetchPluginDefaults]);

  const fetchPluginDetail = async (pluginId: string) => {
    if (!getToken()) return null;
    const data = await requestBackend<PluginRecord>(`/v1/plugins/${pluginId}`, {
      fallbackMessage: "获取插件详情失败",
      router,
    });
    return (data.data ?? null) as PluginRecord | null;
  };

  const fillDraftFromPlugin = (plugin: PluginRecord) => {
    const runtimeType = plugin.runtime_type ?? "";
    const draftTools: ToolDraft[] = (plugin.tools ?? []).map((tool): ToolDraft => ({
      toolId: tool.tool_id,
      toolName: tool.tool_name ?? "",
      description: tool.description ?? "",
      toolResponseMode:
        tool.tool_response_mode === "streaming" ? "streaming" : "non_streaming",
      apiRequestType: tool.api_request_type ?? "GET",
      requestURL: tool.request_url ?? "",
      authConfigRef: tool.auth_config_ref ?? "",
      timeoutMS:
        typeof tool.timeout_ms === "number" && tool.timeout_ms >= 1
          ? tool.timeout_ms
          : 30000,
      queryFields: parseFields(tool.query_fields),
      headerFields: parseFields(tool.header_fields),
      bodyFields: parseFields(tool.body_fields),
      outputFields: getOutputFields(tool),
      codeLanguage: tool.code_language === "python" ? "python" : "javascript",
      code:
        tool.code && tool.code.trim()
          ? tool.code
          : getCodeTemplate(tool.code_language === "python" ? "python" : "javascript"),
    }));
    setDraft({
      pluginId: plugin.plugin_id,
      name: plugin.name ?? "",
      description: plugin.description ?? "",
      icon: plugin.icon ?? "",
      enabled: plugin.status !== "disabled",
      runtimeType,
      mcpURL: plugin.mcp_url ?? "",
      mcpProtocol: normalizeMCPProtocolValue(plugin.mcp_protocol),
      tools: ensureRuntimeTools(draftTools, runtimeType),
    });
    setWizardToolIndex(0);
  };

  const openCreate = () => {
    if (sourceFilter === "builtin") {
      return;
    }
    setDraft(createPluginDraft());
    setWizardToolIndex(0);
    setWizardStep(1);
    setFormError("");
    setIsWizardOpen(true);
  };

  const openEdit = async (pluginId?: string) => {
    if (!pluginId) return;
    setLoadingDetail(true);
    setFormError("");
    try {
      const detail = await fetchPluginDetail(pluginId);
      if (!detail) return;
      fillDraftFromPlugin(detail);
      setWizardStep(1);
      setIsWizardOpen(true);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "获取插件详情失败");
    } finally {
      setLoadingDetail(false);
    }
  };

  const openDetail = async (pluginId?: string) => {
    if (!pluginId) return;
    setLoadingDetail(true);
    setError("");
    try {
      const detail = await fetchPluginDetail(pluginId);
      setSelectedPlugin(detail);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取插件详情失败");
    } finally {
      setLoadingDetail(false);
    }
  };

  const handleSearch = () => {
    setPage(1);
    setSearchKeyword(keyword);
  };

  const closeWizard = () => {
    setIsWizardOpen(false);
    setSaving(false);
  };

  const closeDetail = () => {
    setSelectedPlugin(null);
  };

  const openIconUpload = () => {
    setFormError("");
    iconUploadInputRef.current?.click();
  };

  const handleIconFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!getToken()) return;
    setUploadingIcon(true);
    setFormError("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const data = await requestBackend<{ url?: string }>("/v1/plugins/icon", {
        method: "POST",
        body: formData,
        fallbackMessage: "上传头像失败",
        router,
      });
      const url = typeof data.data?.url === "string" ? data.data.url : "";
      if (!url) {
        throw new Error("上传头像失败");
      }
      setDraft((current) => ({ ...current, icon: url }));
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "上传头像失败");
    } finally {
      setUploadingIcon(false);
    }
  };

  const debugCodeTool = useCallback(
    async (tool: ToolDraft, input: Record<string, unknown>) => {
      if (!getToken()) {
        throw new Error("登录已失效，请重新登录后再试");
      }

      const data = await requestBackend(`/v1/plugins/code/debug`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          code: tool.code,
          code_language: tool.codeLanguage.trim(),
          input,
          timeout_ms:
            Number.isFinite(tool.timeoutMS) && tool.timeoutMS >= 1
              ? tool.timeoutMS
              : 30000,
        }),
        fallbackMessage: "调试执行失败",
        router,
      });
      return data.data ?? null;
    },
    [getToken, router]
  );

  const updateDraftToolAt = useCallback(
    (index: number, updater: (tool: ToolDraft) => ToolDraft) => {
      setDraft((current) => {
        const currentTools = ensureRuntimeTools(current.tools, current.runtimeType);
        if (currentTools.length === 0) {
          return current;
        }
        const safeIndex = Math.min(Math.max(index, 0), currentTools.length - 1);
        const nextTools = currentTools.map((tool, toolIndex) =>
          toolIndex === safeIndex ? updater(tool) : tool
        );
        return { ...current, tools: nextTools };
      });
    },
    []
  );

  const addDraftTool = useCallback(() => {
    const nextIndex = ensureRuntimeTools(draft.tools, draft.runtimeType).length;
    setDraft((current) => {
      const currentTools = ensureRuntimeTools(current.tools, current.runtimeType);
      const nextTool =
        current.runtimeType === "code" ? applyCodeToolDefaults(createToolDraft()) : createToolDraft();
      return {
        ...current,
        tools: [...currentTools, nextTool],
      };
    });
    setWizardToolIndex(nextIndex);
  }, [draft.runtimeType, draft.tools]);

  const removeDraftToolAt = useCallback((index: number) => {
    setDraft((current) => {
      if (!requiresRuntimeTools(current.runtimeType) || current.tools.length <= 1) {
        return current;
      }
      return {
        ...current,
        tools: current.tools.filter((_, toolIndex) => toolIndex !== index),
      };
    });
    setWizardToolIndex((current) => {
      if (current < index) {
        return current;
      }
      if (current === index) {
        return Math.max(0, current - 1);
      }
      return current - 1;
    });
  }, []);

  const goNext = async () => {
    if (wizardStep === 1 && !draft.runtimeType) {
      setFormError("请选择插件运行方式");
      return;
    }
    if (wizardStep === 2 && !draft.name.trim()) {
      setFormError("请输入插件名称");
      return;
    }
    if (wizardStep === 3) {
      const message = validateDraft(draft);
      if (message) {
        setFormError(message);
        return;
      }
    }
    setFormError("");
    if (wizardStep < 4) {
      setWizardStep((current) => current + 1);
      return;
    }
    if (!getToken()) return;
    setSaving(true);
    try {
      const payload = buildPayload(draft);
      const isEditing = Boolean(draft.pluginId);
      const data = await requestBackend<PluginRecord>(
        isEditing
          ? `/v1/plugins/${draft.pluginId}`
          : "/v1/plugins",
        {
          method: isEditing ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
          fallbackMessage: "保存插件失败",
          router,
        }
      );
      closeWizard();
      const savedPlugin = (data.data ?? null) as PluginRecord | null;
      setSelectedPlugin((current) =>
        current?.plugin_id && current.plugin_id === savedPlugin?.plugin_id ? savedPlugin : current
      );
      await fetchList();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "保存插件失败");
    } finally {
      setSaving(false);
    }
  };

  const changeStatus = async (plugin: PluginRecord, enable: boolean) => {
    if (!plugin.plugin_id) return;
    if (!getToken()) return;
    setError("");
    try {
      const data = await requestBackend<PluginRecord>(
        `/v1/plugins/${plugin.plugin_id}/${enable ? "enable" : "disable"}`,
        {
          method: "POST",
          fallbackMessage: "更新插件状态失败",
          router,
        }
      );
      const nextPlugin = data.data as PluginRecord;
      setItems((current) =>
        current.map((item) => (item.plugin_id === nextPlugin.plugin_id ? nextPlugin : item))
      );
      setSelectedPlugin((current) =>
        current?.plugin_id === nextPlugin.plugin_id ? nextPlugin : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新插件状态失败");
    }
  };

  const syncPlugin = async (plugin: PluginRecord) => {
    if (!plugin.plugin_id) return;
    if (!getToken()) return;
    setError("");
    try {
      const data = await requestBackend<PluginRecord>(`/v1/plugins/${plugin.plugin_id}/sync`, {
        method: "POST",
        fallbackMessage: "同步失败",
        router,
      });
      const nextPlugin = data.data as PluginRecord;
      setItems((current) =>
        current.map((item) => (item.plugin_id === nextPlugin.plugin_id ? nextPlugin : item))
      );
      setSelectedPlugin((current) =>
        current?.plugin_id === nextPlugin.plugin_id ? nextPlugin : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "同步失败");
    }
  };

  const removePlugin = async (plugin: PluginRecord) => {
    if (!plugin.plugin_id) return;
    if (!window.confirm(`确认删除插件「${plugin.name || plugin.plugin_id}」吗？`)) {
      return;
    }
    if (!getToken()) return;
    setError("");
    try {
      await requestBackend(`/v1/plugins/${plugin.plugin_id}`, {
        method: "DELETE",
        fallbackMessage: "删除插件失败",
        router,
      });
      setItems((current) => current.filter((item) => item.plugin_id !== plugin.plugin_id));
      setTotal((current) => Math.max(0, current - 1));
      setSelectedPlugin((current) =>
        current?.plugin_id === plugin.plugin_id ? null : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除插件失败");
    }
  };

  const isBuiltinFilterActive = sourceFilter === "builtin";

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        {!selectedPlugin && (
          <>
            <div className="workspace-toolbar-surface rounded-[var(--radius-lg)] border p-3">
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(260px,2.2fr)_160px_160px_auto_auto]">
                <UiInput
                  className="min-w-0"
                  placeholder="搜索插件名称或描述"
                  value={keyword}
                  onChange={(event) => setKeyword(event.target.value)}
                />
                <UiSelect
                  className="min-w-0"
                  value={sourceFilter}
                  onChange={(event) => {
                    setSourceFilter(event.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">全部来源</option>
                  <option value="custom">自定义</option>
                  <option value="builtin">内建</option>
                </UiSelect>
                <UiSelect
                  className="min-w-0"
                  value={runtimeFilter}
                  onChange={(event) => {
                    setRuntimeFilter(event.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">全部类型</option>
                  <option value="api">API</option>
                  <option value="builtin">内建</option>
                  <option value="mcp">MCP</option>
                  <option value="code">Code</option>
                </UiSelect>
                <UiButton
                  type="button"
                  variant="secondary"
                  className="w-full xl:w-auto"
                  onClick={handleSearch}
                >
                  搜索
                </UiButton>
                {!isBuiltinFilterActive && (
                  <UiButton type="button" className="w-full xl:w-auto" onClick={openCreate}>
                    新增插件
                  </UiButton>
                )}
              </div>
            </div>

            <div className="mt-4 flex items-center justify-between gap-3">
              <div className="text-sm text-[var(--color-text-muted)]">共 {total} 条</div>
              {loadingDetail && (
                <div className="text-xs text-[var(--color-text-muted)]">详情加载中...</div>
              )}
            </div>
          </>
        )}

        {error && (
          <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            {error}
          </div>
        )}
        {formError && !isWizardOpen && (
          <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            {formError}
          </div>
        )}

        {!selectedPlugin ? (
          <>
            <div className="mt-4">
              {loading ? (
                <div className="text-sm text-[var(--color-text-muted)]">加载中...</div>
              ) : items.length === 0 ? (
                <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] p-6 text-center">
                  <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                    {isBuiltinFilterActive ? "暂无内建插件" : "还没有自定义插件"}
                  </div>
                  <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                    {isBuiltinFilterActive
                      ? "当前筛选条件下没有可用的内建插件。"
                      : "从 API、MCP 或 Code 向导开始创建。"}
                  </div>
                  {!isBuiltinFilterActive && (
                    <div className="mt-4 flex flex-wrap justify-center gap-2">
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "api";
                          next.tools = ensureRuntimeTools(next.tools, "api");
                          setDraft(next);
                          setWizardToolIndex(0);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 API 插件
                      </UiButton>
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "mcp";
                          setDraft(next);
                          setWizardToolIndex(0);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 MCP 插件
                      </UiButton>
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "code";
                          next.tools = ensureRuntimeTools(next.tools, "code");
                          setDraft(next);
                          setWizardToolIndex(0);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 Code 插件
                      </UiButton>
                    </div>
                  )}
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  {items.map((item) => {
                    const toolCount = item.tool_count ?? item.tools?.length ?? 0;
                    return (
                      <button
                        key={item.plugin_id || item.name}
                        type="button"
                        onClick={() => void openDetail(item.plugin_id)}
                        className="workspace-item-surface workspace-item-hover-lift flex flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4 text-left shadow-[var(--shadow-sm)]"
                      >
                        <div className="flex items-start gap-3">
                          <PluginAvatar
                            src={item.icon}
                            name={item.name}
                            fallbackSrc={defaultPluginIconURL}
                            className="h-11 w-11 shrink-0"
                          />
                          <div className="min-w-0 flex-1">
                            <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
                              {item.name || "未命名插件"}
                            </div>
                          </div>
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2 text-xs">
                          <span
                            className={`rounded-full border px-2 py-1 ${getSourceBadgeClassName(item.source)}`}
                          >
                            {item.source === "builtin" ? "内建" : "自定义"}
                          </span>
                          <span
                            className={`rounded-full border px-2 py-1 ${getRuntimeBadgeClassName(
                              item.runtime_type
                            )}`}
                          >
                            {item.runtime_type
                              ? runtimeLabel[item.runtime_type as Exclude<RuntimeType, "">]
                              : "-"}
                          </span>
                          <span
                            className={`rounded-full px-2 py-1 ${
                              item.status === "enabled"
                                ? "bg-[rgba(22,163,74,0.12)] text-[var(--color-state-success)]"
                                : "bg-[rgba(148,163,184,0.14)] text-[var(--color-text-muted)]"
                            }`}
                          >
                            {item.status || "disabled"}
                          </span>
                          <span className="rounded-full bg-[rgba(37,99,255,0.08)] px-2 py-1 text-[var(--color-action-primary)]">
                            {toolCount} 个工具
                          </span>
                        </div>
                        <div
                          className="mt-3 line-clamp-2 text-xs text-[var(--color-text-muted)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                          dangerouslySetInnerHTML={{
                            __html: item.description?.trim() || "暂无描述",
                          }}
                        />
                      </button>
                    );
                  })}
                </div>
              )}
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
                    onChange={(event) => {
                      setPageSize(Number(event.target.value));
                      setPage(1);
                    }}
                    className="h-8 w-[60px] !py-0 !pl-2 !pr-6"
                  >
                    <option value={9}>9</option>
                    <option value={18}>18</option>
                    <option value={36}>36</option>
                    <option value={72}>72</option>
                  </UiSelect>
                  <span className="whitespace-nowrap text-sm text-[var(--color-text-muted)]">条/页</span>
                </div>
              </div>

              <div className="flex items-center gap-1">
                <UiButton
                  variant="secondary"
                  size="sm"
                  className="h-7 px-2"
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                  disabled={page <= 1}
                >
                  ←
                </UiButton>

                {(() => {
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
                  onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                  disabled={page >= totalPages}
                >
                  →
                </UiButton>
              </div>
            </div>
          </>
        ) : (
          <div className="mt-4 rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-5">
            {(() => {
              const detailToolCount =
                selectedPlugin.tool_count ?? selectedPlugin.tools?.length ?? 0;
              return (
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex min-w-0 items-start gap-4">
                    <button
                      type="button"
                      onClick={closeDetail}
                      className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-white text-[var(--color-text-secondary)]"
                      aria-label="返回插件列表"
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
                        <path d="M15 18l-6-6 6-6" />
                      </svg>
                    </button>
                    <PluginAvatar
                      src={selectedPlugin.icon}
                      name={selectedPlugin.name}
                      fallbackSrc={defaultPluginIconURL}
                      className="h-14 w-14 shrink-0"
                    />
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                        <div className="truncate text-lg font-semibold text-[var(--color-text-primary)]">
                          {selectedPlugin.name || "未命名插件"}
                        </div>
                        <div className="text-xs text-[var(--color-text-muted)]">
                          更新时间：{formatTime(selectedPlugin.updated_at)}
                        </div>
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2 text-xs">
                        <span
                          className={`rounded-full border px-2 py-1 ${getSourceBadgeClassName(
                            selectedPlugin.source
                          )}`}
                        >
                          {selectedPlugin.source === "builtin" ? "内建" : "自定义"}
                        </span>
                        <span
                          className={`rounded-full border px-2 py-1 ${getRuntimeBadgeClassName(
                            selectedPlugin.runtime_type
                          )}`}
                        >
                          {selectedPlugin.runtime_type
                            ? runtimeLabel[selectedPlugin.runtime_type as Exclude<RuntimeType, "">]
                            : "-"}
                        </span>
                        <span
                          className={`rounded-full border px-2 py-1 ${
                            selectedPlugin.status === "enabled"
                              ? "border-[rgba(22,163,74,0.2)] bg-[rgba(22,163,74,0.12)] text-[var(--color-state-success)]"
                              : "border-[rgba(148,163,184,0.2)] bg-[rgba(148,163,184,0.14)] text-[var(--color-text-muted)]"
                          }`}
                        >
                          {selectedPlugin.status || "disabled"}
                        </span>
                        <span className="rounded-full border border-[rgba(37,99,255,0.16)] bg-[rgba(37,99,255,0.08)] px-2 py-1 text-[var(--color-action-primary)]">
                          {detailToolCount} 个工具
                        </span>
                      </div>
                      <div
                        className="mt-2 text-sm text-[var(--color-text-secondary)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                        dangerouslySetInnerHTML={{
                          __html: selectedPlugin.description?.trim() || "暂无描述",
                        }}
                      />
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => void openEdit(selectedPlugin.plugin_id)}
                    >
                      编辑
                    </UiButton>
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() =>
                        void changeStatus(selectedPlugin, selectedPlugin.status !== "enabled")
                      }
                    >
                      {selectedPlugin.status === "enabled" ? "停用" : "启用"}
                    </UiButton>
                    {selectedPlugin.runtime_type === "mcp" && (
                      <UiButton
                        type="button"
                        variant="secondary"
                        size="sm"
                        onClick={() => void syncPlugin(selectedPlugin)}
                      >
                        手动同步
                      </UiButton>
                    )}
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => void removePlugin(selectedPlugin)}
                    >
                      删除
                    </UiButton>
                  </div>
                </div>
              );
            })()}

            {selectedPlugin.tools && selectedPlugin.tools.length > 0 && activeTool ? (
              <div className="mt-5 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-white">
                <div className="border-b border-[var(--color-border-default)] px-5 py-4">
                  <div className="flex min-w-full gap-2 overflow-x-auto pb-1">
                    {selectedPlugin.tools.map((tool, index) => {
                      const key = getToolKey(tool, index);
                      const active = key === selectedToolKey;
                      return (
                        <button
                          key={key}
                          type="button"
                          onClick={() => setSelectedToolKey(key)}
                          className={`shrink-0 whitespace-nowrap rounded-[var(--radius-md)] border px-4 py-2 text-sm transition ${
                            active
                              ? "border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.08)] font-semibold text-[var(--color-action-primary)]"
                              : "border-[var(--color-border-default)] text-[var(--color-text-secondary)]"
                          }`}
                        >
                          {tool.tool_name || `工具 ${index + 1}`}
                        </button>
                      );
                    })}
                  </div>
                </div>

                <div className="space-y-6 px-5 py-5">
                  <section className="space-y-4">
                    <DetailSectionTitle title="基础信息" />
                    <div className="grid gap-3 md:grid-cols-2">
                      <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                        <div className="text-xs text-[var(--color-text-muted)]">调用方式</div>
                        <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                          {activeTool.tool_response_mode
                            ? responseModeLabel[
                                activeTool.tool_response_mode as Exclude<ResponseMode, "">
                              ] || activeTool.tool_response_mode
                            : "-"}
                        </div>
                      </div>
                      {selectedPlugin.runtime_type === "api" && (
                        <>
                          <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                            <div className="text-xs text-[var(--color-text-muted)]">请求方式</div>
                            <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                              {activeTool.api_request_type || "-"}
                            </div>
                          </div>
                          <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                            <div className="text-xs text-[var(--color-text-muted)]">请求地址</div>
                            <div className="mt-1 break-all text-sm font-semibold text-[var(--color-text-primary)]">
                              {activeTool.request_url || "-"}
                            </div>
                          </div>
                        </>
                      )}
                      {selectedPlugin.runtime_type === "code" && (
                        <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                          <div className="text-xs text-[var(--color-text-muted)]">代码语言</div>
                          <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                            {activeTool.code_language || "-"}
                          </div>
                        </div>
                      )}
                      {selectedPlugin.runtime_type === "mcp" && (
                        <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                          <div className="text-xs text-[var(--color-text-muted)]">MCP 地址</div>
                          <div className="mt-1 break-all text-sm font-semibold text-[var(--color-text-primary)]">
                            {selectedPlugin.mcp_url || "-"}
                          </div>
                        </div>
                      )}
                      <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                        <div className="text-xs text-[var(--color-text-muted)]">工具描述</div>
                        <div
                          className="mt-1 text-sm text-[var(--color-text-primary)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                          dangerouslySetInnerHTML={{
                            __html: activeTool.description?.trim() || "暂无描述",
                          }}
                        />
                      </div>
                    </div>
                  </section>

                  <section className="space-y-4">
                    <DetailSectionTitle title="入参数" />
                    {activeToolInputSections.length > 0 ? (
                      <div className="space-y-4">
                        {activeToolInputSections.map((section) => (
                          <div key={section.key} className="space-y-2">
                            <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                              {section.label}
                            </div>
                            <FieldTable fields={section.fields} />
                          </div>
                        ))}
                      </div>
                    ) : (
                      <FieldTable fields={[]} emptyText="暂无入参数配置" />
                    )}
                  </section>

                  <section className="space-y-4">
                    <DetailSectionTitle title="出参数" />
                    <FieldTable
                      fields={activeToolOutputFields}
                      emptyText="暂无结构化出参数配置"
                      showConstraints={false}
                    />
                  </section>
                </div>
              </div>
            ) : (
              <div className="mt-5 text-sm text-[var(--color-text-muted)]">暂无工具</div>
            )}
          </div>
        )}
      </div>

      {isWizardOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(15,23,42,0.42)] p-2 md:p-6">
          <div className="flex h-[94vh] w-full max-w-[1440px] flex-col overflow-hidden rounded-[28px] border border-[var(--color-border-default)] bg-white shadow-[var(--shadow-lg)]">
            <div className="flex items-center justify-between gap-3 border-b border-[var(--color-border-default)] px-8 py-5">
              <div>
                <div className="text-xl font-semibold text-[var(--color-text-primary)]">
                  {draft.pluginId ? "编辑插件" : "新增插件"}
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  步骤 {wizardStep} / 4
                </div>
              </div>
              <button
                type="button"
                className="text-sm text-[var(--color-text-muted)]"
                onClick={closeWizard}
              >
                关闭
              </button>
            </div>

            <div className="flex-1 overflow-auto px-8 py-6">
              <div className="grid gap-3 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-3 md:grid-cols-4">
                {[
                  draft.pluginId ? "插件类型" : "选择类型",
                  "基础信息",
                  "运行配置",
                  "预览确认",
                ].map((label, index) => {
                  const step = index + 1;
                  const active = step === wizardStep;
                  return (
                    <div
                      key={label}
                      className={`flex items-center justify-center gap-3 rounded-[var(--radius-md)] border px-3 py-3 text-sm ${
                        active
                          ? "border-[var(--color-action-primary)] bg-[var(--color-action-primary)] font-semibold text-white shadow-[var(--shadow-sm)]"
                          : "border-transparent bg-white text-[var(--color-text-muted)]"
                      }`}
                    >
                      <span
                        className={`flex h-7 w-7 items-center justify-center rounded-full border ${
                          active
                            ? "border-white/50 bg-white/10"
                            : "border-[var(--color-border-default)]"
                        }`}
                      >
                        {step}
                      </span>
                      <span>{label}</span>
                    </div>
                  );
                })}
              </div>

              <div className="mt-6">
                {wizardStep === 1 && (
                  <div>
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      {draft.pluginId ? "第一步：查看插件类型" : "第一步：选择来源与运行方式"}
                    </div>
                    <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                      {draft.pluginId ? (
                        "插件类型在创建时已确定，编辑时不可修改。"
                      ) : (
                        <>来源固定为 `custom`，`builtin` 保持只读。</>
                      )}
                    </div>
                    {draft.pluginId ? (
                      <div className="mt-4 rounded-[var(--radius-lg)] border border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.06)] p-4">
                        <div className="text-base font-semibold text-[var(--color-text-primary)]">
                          {draft.runtimeType ? runtimeLabel[draft.runtimeType] : "未设置"}
                        </div>
                        <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                          {draft.runtimeType === "api" &&
                            "向导式配置请求方式、响应模式和 query/header/body 参数。"}
                          {draft.runtimeType === "mcp" &&
                            "维护连接地址与同步状态，当前主要管理定义与手动同步。"}
                          {draft.runtimeType === "code" &&
                            "配置输入参数、语言和脚本内容。"}
                        </div>
                      </div>
                    ) : (
                      <div className="mt-4 grid gap-3 md:grid-cols-3">
                        {(["api", "mcp", "code"] as Array<Exclude<RuntimeType, "">>).map(
                          (runtime) => (
                            <button
                              key={runtime}
                              type="button"
                              onClick={() => {
                                setDraft((current) => ({
                                  ...current,
                                  runtimeType: runtime,
                                  tools: ensureRuntimeTools(current.tools, runtime),
                                }));
                                setWizardToolIndex(0);
                              }}
                              className={`rounded-[var(--radius-lg)] border p-4 text-left ${
                                draft.runtimeType === runtime
                                  ? "border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.06)]"
                                  : "border-[var(--color-border-default)]"
                              }`}
                            >
                              <div className="text-base font-semibold text-[var(--color-text-primary)]">
                                {runtimeLabel[runtime]}
                              </div>
                              <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                                {runtime === "api" &&
                                  "向导式配置请求方式、响应模式和 query/header/body 参数。"}
                                {runtime === "mcp" &&
                                  "维护连接地址与同步状态，当前主要管理定义与手动同步。"}
                                {runtime === "code" &&
                                  "配置输入参数、语言和脚本内容，并固定使用非流式。"}
                              </div>
                            </button>
                          )
                        )}
                      </div>
                    )}
                  </div>
                )}

                {wizardStep === 2 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第二步：填写基础信息
                    </div>
                    <FormFieldRow label="插件名称" required>
                      <UiInput
                        placeholder="请输入插件名称"
                        value={draft.name}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, name: event.target.value }))
                        }
                      />
                    </FormFieldRow>
                    <FormFieldRow label="插件描述">
                      <UiInput
                        placeholder="请输入插件描述"
                        value={draft.description}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, description: event.target.value }))
                        }
                      />
                    </FormFieldRow>
                    <FormFieldRow label="头像">
                      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-3">
                        <div className="flex flex-col gap-3 lg:flex-row">
                          <div className="min-w-0 flex-1">
                            <UiInput
                              type="text"
                              inputMode="url"
                              autoComplete="off"
                              spellCheck={false}
                              placeholder="请输入头像 URL（可选）"
                              value={draft.icon}
                              onChange={(event) =>
                                setDraft((current) => ({ ...current, icon: event.target.value }))
                              }
                            />
                          </div>
                          <UiButton
                            variant="secondary"
                            onClick={openIconUpload}
                            loading={uploadingIcon}
                          >
                            上传头像
                          </UiButton>
                        </div>
                        <div className="mt-3 flex items-center gap-3 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] bg-white px-3 py-2 text-xs text-[var(--color-text-muted)]">
                          <PluginAvatar
                            src={draft.icon}
                            name={draft.name}
                            fallbackSrc={defaultPluginIconURL}
                            className="h-12 w-12 shrink-0"
                          />
                          <span className="truncate">
                            {draft.icon.trim()
                              ? "已回填头像地址，可继续手动调整"
                              : defaultPluginIconURL
                                ? "未填写时将使用后端默认头像，也可上传后回填 URL"
                                : "支持直接输入 URL，或上传图片后自动回填 URL"}
                          </span>
                        </div>
                        {/* eslint-disable-next-line no-restricted-syntax */}
                        <input
                          ref={iconUploadInputRef}
                          type="file"
                          accept="image/*"
                          className="hidden"
                          onChange={(event) => {
                            void handleIconFileChange(event);
                          }}
                        />
                      </div>
                    </FormFieldRow>
                    <label className="flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--color-border-default)] px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                      <input
                        type="checkbox"
                        checked={draft.enabled}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, enabled: event.target.checked }))
                        }
                      />
                      创建后立即启用
                    </label>
                  </div>
                )}

                {wizardStep === 3 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第三步：填写运行配置
                    </div>

                    {requiresRuntimeTools(draft.runtimeType) && (
                      <ToolDraftSwitcher
                        tools={ensureRuntimeTools(draft.tools, draft.runtimeType)}
                        activeIndex={wizardToolIndex}
                        onSelect={setWizardToolIndex}
                        onAdd={addDraftTool}
                        onRemove={removeDraftToolAt}
                      />
                    )}

                    {draft.runtimeType === "api" && activeDraftTool && (
                      <>
                        <div className="space-y-4">
                          <FormFieldRow label="工具名称" required>
                            <UiInput
                              placeholder="请输入工具名称"
                              value={activeDraftTool.toolName}
                              onChange={(event) =>
                                updateDraftToolAt(wizardToolIndex, (tool) => ({
                                  ...tool,
                                  toolName: event.target.value,
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="工具描述">
                            <UiInput
                              placeholder="请输入工具描述"
                              value={activeDraftTool.description}
                              onChange={(event) =>
                                updateDraftToolAt(wizardToolIndex, (tool) => ({
                                  ...tool,
                                  description: event.target.value,
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="请求方式" required>
                            <UiSelect
                              value={activeDraftTool.apiRequestType}
                              onChange={(event) =>
                                updateDraftToolAt(wizardToolIndex, (tool) => ({
                                  ...tool,
                                  apiRequestType: event.target.value as RequestType,
                                  bodyFields: event.target.value === "GET" ? [] : tool.bodyFields,
                                }))
                              }
                            >
                              <option value="GET">GET</option>
                              <option value="POST">POST</option>
                            </UiSelect>
                          </FormFieldRow>
                          <FormFieldRow label="响应模式" required>
                            <UiSelect
                              value={activeDraftTool.toolResponseMode}
                              onChange={(event) =>
                                updateDraftToolAt(wizardToolIndex, (tool) => ({
                                  ...tool,
                                  toolResponseMode: event.target.value as ResponseMode,
                                }))
                              }
                            >
                              <option value="non_streaming">{responseModeLabel.non_streaming}</option>
                              <option value="streaming">{responseModeLabel.streaming}</option>
                            </UiSelect>
                          </FormFieldRow>
                          <FormFieldRow label="调用地址" required>
                            <UiInput
                              placeholder="请输入 RequestURL"
                              value={activeDraftTool.requestURL}
                              onChange={(event) =>
                                updateDraftToolAt(wizardToolIndex, (tool) => ({
                                  ...tool,
                                  requestURL: event.target.value,
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="超时">
                            <div className="flex items-center gap-2">
                              <UiInput
                                type="number"
                                min={1}
                                step={1}
                                value={String(activeDraftTool.timeoutMS)}
                                onChange={(event) => {
                                  const nextValue = Number(event.target.value);
                                  updateDraftToolAt(wizardToolIndex, (tool) => ({
                                    ...tool,
                                    timeoutMS:
                                      Number.isFinite(nextValue) && nextValue >= 1 ? nextValue : 1,
                                  }));
                                }}
                              />
                              <span className="shrink-0 text-sm text-[var(--color-text-secondary)]">
                                毫秒
                              </span>
                            </div>
                          </FormFieldRow>
                        </div>
                        <div className="space-y-4">
                          <FieldListEditor
                            label="Query 字段"
                            fields={activeDraftTool.queryFields}
                            onChange={(nextFields) =>
                              updateDraftToolAt(wizardToolIndex, (tool) => ({
                                ...tool,
                                queryFields: nextFields,
                              }))
                            }
                          />
                          <FieldListEditor
                            label="Header 字段"
                            fields={activeDraftTool.headerFields}
                            onChange={(nextFields) =>
                              updateDraftToolAt(wizardToolIndex, (tool) => ({
                                ...tool,
                                headerFields: nextFields,
                              }))
                            }
                          />
                        </div>
                        {activeDraftTool.apiRequestType === "POST" ? (
                          <FieldListEditor
                            label="Body 字段"
                            fields={activeDraftTool.bodyFields}
                            onChange={(nextFields) =>
                              updateDraftToolAt(wizardToolIndex, (tool) => ({
                                ...tool,
                                bodyFields: nextFields,
                              }))
                            }
                          />
                        ) : null}
                      </>
                    )}

                    {draft.runtimeType === "mcp" && (
                      <div className="space-y-4">
                        <FormFieldRow label="MCP URL" required>
                          <UiInput
                            placeholder="请输入 MCP URL"
                            value={draft.mcpURL}
                            onChange={(event) =>
                              setDraft((current) => ({ ...current, mcpURL: event.target.value }))
                            }
                          />
                        </FormFieldRow>
                        <FormFieldRow label="MCP 协议">
                          <UiSelect
                            value={draft.mcpProtocol}
                            onChange={(event) =>
                              setDraft((current) => ({
                                ...current,
                                mcpProtocol: normalizeMCPProtocolValue(event.target.value),
                              }))
                            }
                          >
                            <option value="sse">{mcpProtocolLabel.sse}</option>
                            <option value="streamableHttp">
                              {mcpProtocolLabel.streamableHttp}
                            </option>
                          </UiSelect>
                        </FormFieldRow>
                      </div>
                    )}

                    {draft.runtimeType === "code" && activeDraftTool && (
                      <CodeToolEditor
                        tool={activeDraftTool}
                        onDebugRun={debugCodeTool}
                        onChange={(nextTool) =>
                          updateDraftToolAt(wizardToolIndex, () => nextTool)
                        }
                      />
                    )}
                  </div>
                )}

                {wizardStep === 4 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第四步：预览并确认
                    </div>
                    <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-3 py-3 text-xs text-[var(--color-text-muted)]">
                      将写入 plugin 定义，并生成对应的 tool 元数据；敏感信息仅保存引用，不经前端透传。
                    </div>
                    <pre className="overflow-auto rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgb(15,23,42)] p-4 text-xs text-[rgb(226,232,240)]">
                      {previewJSON}
                    </pre>
                  </div>
                )}

                {formError && (
                  <div className="mt-4 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
                    {formError}
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between gap-3 border-t border-[var(--color-border-default)] px-8 py-5">
              <UiButton
                type="button"
                variant="secondary"
                onClick={() => {
                  setFormError("");
                  setWizardStep((current) => Math.max(1, current - 1));
                }}
                disabled={wizardStep === 1 || saving}
              >
                上一步
              </UiButton>
              <UiButton type="button" onClick={() => void goNext()} disabled={saving}>
                {wizardStep === 4 ? (saving ? "保存中..." : "确认提交") : "下一步"}
              </UiButton>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
