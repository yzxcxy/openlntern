"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import { UiModal as Modal } from "../../components/ui/UiModal";
import {
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  requestBackend,
} from "../auth";

type UserInfo = {
  user_id?: string | number;
};

type BackendPage<T> = {
  data: T[];
  total: number;
  page?: number;
  size?: number;
};

type ModelProviderItem = {
  provider_id: string;
  name: string;
  api_type: string;
  base_url?: string;
  api_key_masked?: string;
  avatar?: string;
  extra_config_json?: string;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
};

type ModelItem = {
  model_id: string;
  provider_id: string;
  provider_name?: string;
  provider_avatar?: string;
  api_type?: string;
  model_key: string;
  name: string;
  avatar?: string;
  capabilities_json?: string;
  enabled: boolean;
  sort: number;
  is_system_default?: boolean;
  created_at?: string;
  updated_at?: string;
};

type DefaultModelResponse = {
  config_key?: string;
  model_id?: string;
};

type ProviderFormState = {
  name: string;
  api_type: string;
  base_url: string;
  api_key: string;
  avatar: string;
  extra_config_json: string;
  enabled: boolean;
};

type ModelFormState = {
  provider_id: string;
  model_key: string;
  name: string;
  avatar: string;
  capabilities_json: string;
  context_window: string;
  enabled: boolean;
  sort: string;
};

const EMPTY_PROVIDER_FORM: ProviderFormState = {
  name: "",
  api_type: "openai",
  base_url: "",
  api_key: "",
  avatar: "",
  extra_config_json: "",
  enabled: true,
};

const EMPTY_MODEL_FORM: ModelFormState = {
  provider_id: "",
  model_key: "",
  name: "",
  avatar: "",
  capabilities_json: "",
  context_window: "",
  enabled: true,
  sort: "0",
};

const MODEL_GROUP_ORDER = ["对话", "向量", "视觉", "重排", "音频", "其他", "未分类"] as const;

const collectCapabilityTokens = (value: unknown): string[] => {
  if (typeof value === "string") {
    return value
      .split(/[,\s/|]+/)
      .map((item) => item.trim())
      .filter(Boolean);
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => collectCapabilityTokens(item));
  }
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    if (Array.isArray(record.capabilities)) {
      return collectCapabilityTokens(record.capabilities);
    }
    return Object.entries(record)
      .filter(([, item]) => item === true)
      .map(([key]) => key);
  }
  return [];
};

const parseCapabilityTokens = (raw?: string) => {
  if (!raw?.trim()) {
    return [];
  }
  try {
    return Array.from(
      new Set(
        collectCapabilityTokens(JSON.parse(raw)).map((item) => item.toLowerCase())
      )
    );
  } catch {
    return Array.from(
      new Set(
        raw
          .split(/[,\s/|]+/)
          .map((item) => item.trim().toLowerCase())
          .filter(Boolean)
      )
    );
  }
};

const parsePositiveInteger = (value: unknown) => {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return Math.floor(value);
  }
  if (typeof value === "string") {
    const parsed = Number.parseInt(value.trim(), 10);
    if (!Number.isNaN(parsed) && parsed > 0) {
      return parsed;
    }
  }
  return 0;
};

const formatNumberLabel = (value: number) =>
  new Intl.NumberFormat("zh-CN").format(value);

// 将旧的纯标签写法提升为 JSON，便于和上下文窗口配置共存。
const buildCapabilitiesDraft = (raw?: string): Record<string, unknown> | null => {
  const trimmed = raw?.trim() ?? "";
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return { ...(parsed as Record<string, unknown>) };
    }
    const tokens = Array.from(new Set(collectCapabilityTokens(parsed)));
    return tokens.length > 0 ? { capabilities: tokens } : {};
  } catch {
    const tokens = Array.from(new Set(collectCapabilityTokens(trimmed)));
    if (tokens.length > 0) {
      return { capabilities: tokens };
    }
    return null;
  }
};

const extractContextWindow = (raw?: string) => {
  const trimmed = raw?.trim();
  if (!trimmed) {
    return "";
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return "";
    }
    const record = parsed as Record<string, unknown>;
    for (const key of ["context_window", "contextWindow", "max_input_tokens", "maxInputTokens"]) {
      const tokens = parsePositiveInteger(record[key]);
      if (tokens > 0) {
        return String(tokens);
      }
    }
    const contextValue = record.context;
    if (contextValue && typeof contextValue === "object" && !Array.isArray(contextValue)) {
      const contextRecord = contextValue as Record<string, unknown>;
      for (const key of ["window", "context_window", "max_input_tokens"]) {
        const tokens = parsePositiveInteger(contextRecord[key]);
        if (tokens > 0) {
          return String(tokens);
        }
      }
    }
  } catch {
    return "";
  }
  return "";
};

const mergeCapabilitiesWithContextWindow = (raw: string, contextWindow: string) => {
  const trimmedContextWindow = contextWindow.trim();
  const draft = buildCapabilitiesDraft(raw);
  if (draft === null) {
    throw new Error("capabilities_json 不是有效 JSON 或能力标签列表，无法合并上下文大小");
  }

  delete draft.context_window;
  delete draft.contextWindow;
  delete draft.max_input_tokens;
  delete draft.maxInputTokens;

  const contextValue = draft.context;
  if (contextValue && typeof contextValue === "object" && !Array.isArray(contextValue)) {
    const nextContext = { ...(contextValue as Record<string, unknown>) };
    delete nextContext.window;
    delete nextContext.context_window;
    delete nextContext.max_input_tokens;
    if (Object.keys(nextContext).length > 0) {
      draft.context = nextContext;
    } else {
      delete draft.context;
    }
  }

  if (trimmedContextWindow) {
    const tokens = parsePositiveInteger(trimmedContextWindow);
    if (tokens <= 0) {
      throw new Error("上下文大小必须是正整数");
    }
    draft.context_window = tokens;
  }

  return Object.keys(draft).length > 0 ? JSON.stringify(draft, null, 2) : "";
};

const getModelGroupLabel = (model: ModelItem) => {
  const tokens = parseCapabilityTokens(model.capabilities_json);
  if (!tokens.length) {
    return "未分类";
  }
  if (tokens.some((item) => item.includes("embedding") || item.includes("vector"))) {
    return "向量";
  }
  if (
    tokens.some(
      (item) =>
        item.includes("vision") ||
        item.includes("image") ||
        item.includes("multimodal") ||
        item.includes("vl")
    )
  ) {
    return "视觉";
  }
  if (tokens.some((item) => item.includes("rerank") || item.includes("rank"))) {
    return "重排";
  }
  if (
    tokens.some(
      (item) =>
        item.includes("audio") ||
        item.includes("speech") ||
        item.includes("tts") ||
        item.includes("asr")
    )
  ) {
    return "音频";
  }
  if (
    tokens.some(
      (item) =>
        item.includes("chat") ||
        item.includes("text") ||
        item.includes("completion") ||
        item.includes("reason")
    )
  ) {
    return "对话";
  }
  return "其他";
};

const formatDateLabel = (value?: string) => {
  if (!value) {
    return "未记录";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

export default function ModelsPage() {
  const router = useRouter();
  const [providers, setProviders] = useState<ModelProviderItem[]>([]);
  const [models, setModels] = useState<ModelItem[]>([]);
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [defaultModelId, setDefaultModelId] = useState("");
  const [savedDefaultModelId, setSavedDefaultModelId] = useState("");
  const [providerQuery, setProviderQuery] = useState("");
  const [providerForm, setProviderForm] = useState<ProviderFormState>(EMPTY_PROVIDER_FORM);
  const [providerEditId, setProviderEditId] = useState("");
  const [isProviderFormOpen, setIsProviderFormOpen] = useState(false);
  const [modelForm, setModelForm] = useState<ModelFormState>(EMPTY_MODEL_FORM);
  const [modelEditId, setModelEditId] = useState("");
  const [isModelFormOpen, setIsModelFormOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [savingProvider, setSavingProvider] = useState(false);
  const [savingModel, setSavingModel] = useState(false);
  const [savingDefault, setSavingDefault] = useState(false);
  const [error, setError] = useState("");

  const getValidToken = useCallback(() => readValidToken(router), [router]);

  const getUserId = useCallback((token: string) => {
    const user = readStoredUser<UserInfo>();
    const value = user?.user_id;
    if (typeof value === "string" || typeof value === "number") {
      return String(value);
    }
    return getUserIdFromToken(token);
  }, []);

  useEffect(() => {
    const token = getValidToken();
    if (!token) {
      router.push("/login");
    }
  }, [getValidToken, router]);

  const fetchAll = useCallback(async () => {
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setLoading(true);
    setError("");
    try {
      const [providerData, modelData, defaultData] = await Promise.all([
        requestBackend<BackendPage<ModelProviderItem>>("/v1/model-providers?page=1&page_size=100", {
          fallbackMessage: "获取模型提供商失败",
          router,
          userId,
        }),
        requestBackend<BackendPage<ModelItem>>("/v1/models?page=1&page_size=200", {
          fallbackMessage: "获取模型列表失败",
          router,
          userId,
        }),
        requestBackend<DefaultModelResponse>("/v1/models/default", {
          fallbackMessage: "获取默认模型失败",
          router,
          userId,
        }),
      ]);

      const nextProviders = providerData.data?.data ?? [];
      const nextModels = modelData.data?.data ?? [];
      const nextDefaultModelId =
        typeof defaultData.data?.model_id === "string" ? defaultData.data.model_id : "";

      setProviders(nextProviders);
      setModels(nextModels);
      setDefaultModelId(nextDefaultModelId);
      setSavedDefaultModelId(nextDefaultModelId);
      setModelForm((prev) => ({
        ...prev,
        provider_id:
          prev.provider_id || nextProviders[0]?.provider_id || "",
      }));
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("加载模型服务失败");
      }
    } finally {
      setLoading(false);
    }
  }, [getUserId, getValidToken, router]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  useEffect(() => {
    if (!providers.length) {
      setSelectedProviderId("");
      return;
    }
    const firstProviderId = providers[0]?.provider_id ?? "";
    setSelectedProviderId((prev) =>
      providers.some((item) => item.provider_id === prev) ? prev : firstProviderId
    );
  }, [providers]);

  const providerOptions = useMemo(
    () =>
      providers.map((item) => ({
        value: item.provider_id,
        label: item.name,
      })),
    [providers]
  );

  const providerModelCount = useMemo(() => {
    const counts = new Map<string, number>();
    models.forEach((item) => {
      counts.set(item.provider_id, (counts.get(item.provider_id) ?? 0) + 1);
    });
    return counts;
  }, [models]);

  const filteredProviders = useMemo(() => {
    const query = providerQuery.trim().toLowerCase();
    if (!query) {
      return providers;
    }
    return providers.filter((item) => {
      const haystack = `${item.name} ${item.api_type} ${item.base_url || ""}`.toLowerCase();
      return haystack.includes(query);
    });
  }, [providerQuery, providers]);

  const selectedProvider = useMemo(
    () => providers.find((item) => item.provider_id === selectedProviderId) ?? null,
    [providers, selectedProviderId]
  );

  const sortedModels = useMemo(
    () =>
      [...models].sort((left, right) => {
        const sortGap = (left.sort ?? 0) - (right.sort ?? 0);
        if (sortGap !== 0) {
          return sortGap;
        }
        return left.name.localeCompare(right.name, "zh-CN");
      }),
    [models]
  );

  const selectedProviderModels = useMemo(
    () => sortedModels.filter((item) => item.provider_id === selectedProviderId),
    [selectedProviderId, sortedModels]
  );

  const groupedProviderModels = useMemo(() => {
    const groups = new Map<string, ModelItem[]>();
    selectedProviderModels.forEach((item) => {
      const label = getModelGroupLabel(item);
      const current = groups.get(label) ?? [];
      current.push(item);
      groups.set(label, current);
    });
    return MODEL_GROUP_ORDER.filter((label) => groups.has(label)).map((label) => ({
      label,
      items: groups.get(label) ?? [],
    }));
  }, [selectedProviderModels]);

  const currentDefaultModel = useMemo(
    () => models.find((item) => item.model_id === savedDefaultModelId) ?? null,
    [savedDefaultModelId, models]
  );

  // 默认模型下拉不保留空选项，列表变化后自动收敛到一个有效模型。
  useEffect(() => {
    if (!sortedModels.length) {
      if (defaultModelId) {
        setDefaultModelId("");
      }
      return;
    }
    if (sortedModels.some((item) => item.model_id === defaultModelId)) {
      return;
    }
    setDefaultModelId(savedDefaultModelId || sortedModels[0]?.model_id || "");
  }, [defaultModelId, savedDefaultModelId, sortedModels]);

  const resetProviderForm = () => {
    setProviderForm(EMPTY_PROVIDER_FORM);
    setProviderEditId("");
  };

  const resetModelForm = () => {
    setModelForm({
      ...EMPTY_MODEL_FORM,
      provider_id: selectedProviderId || providers[0]?.provider_id || "",
    });
    setModelEditId("");
  };

  const handleCreateProvider = () => {
    resetProviderForm();
    setIsProviderFormOpen(true);
  };

  const handleEditProvider = (item: ModelProviderItem) => {
    setSelectedProviderId(item.provider_id);
    setProviderEditId(item.provider_id);
    setProviderForm({
      name: item.name || "",
      api_type: item.api_type || "openai",
      base_url: item.base_url || "",
      api_key: "",
      avatar: item.avatar || "",
      extra_config_json: item.extra_config_json || "",
      enabled: !!item.enabled,
    });
    setIsProviderFormOpen(true);
  };

  const handleCreateModel = () => {
    setModelEditId("");
    setModelForm({
      ...EMPTY_MODEL_FORM,
      provider_id: selectedProviderId || providers[0]?.provider_id || "",
    });
    setIsModelFormOpen(true);
  };

  const handleEditModel = (item: ModelItem) => {
    setSelectedProviderId(item.provider_id);
    setModelEditId(item.model_id);
    setModelForm({
      provider_id: item.provider_id || "",
      model_key: item.model_key || "",
      name: item.name || "",
      avatar: item.avatar || "",
      capabilities_json: item.capabilities_json || "",
      context_window: extractContextWindow(item.capabilities_json),
      enabled: !!item.enabled,
      sort: String(item.sort ?? 0),
    });
    setIsModelFormOpen(true);
  };

  const handleSaveProvider = async () => {
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    if (!providerForm.name.trim()) {
      setError("请填写提供商名称");
      return;
    }
    if (!providerForm.api_type.trim()) {
      setError("请选择 API 类型");
      return;
    }
    setSavingProvider(true);
    setError("");
    try {
      const payload = {
        name: providerForm.name.trim(),
        api_type: providerForm.api_type.trim(),
        base_url: providerForm.base_url.trim(),
        api_key: providerForm.api_key.trim(),
        avatar: providerForm.avatar.trim(),
        extra_config_json: providerForm.extra_config_json.trim(),
        enabled: providerForm.enabled,
      };
      await requestBackend(
        providerEditId
          ? `/v1/model-providers/${providerEditId}`
          : "/v1/model-providers",
        {
          method: providerEditId ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
          fallbackMessage: "保存提供商失败",
          router,
          userId,
        }
      );
      resetProviderForm();
      setIsProviderFormOpen(false);
      await fetchAll();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("保存提供商失败");
      }
    } finally {
      setSavingProvider(false);
    }
  };

  const handleSaveModel = async () => {
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    if (!modelForm.provider_id) {
      setError("请先选择提供商");
      return;
    }
    if (!modelForm.model_key.trim() || !modelForm.name.trim()) {
      setError("请填写模型标识和展示名称");
      return;
    }
    setSavingModel(true);
    setError("");
    try {
      const sortValue = Number.parseInt(modelForm.sort, 10);
      const hasEmbeddedContextWindow = extractContextWindow(modelForm.capabilities_json) !== "";
      const capabilitiesJSON =
        modelForm.context_window.trim() || hasEmbeddedContextWindow
          ? mergeCapabilitiesWithContextWindow(
              modelForm.capabilities_json,
              modelForm.context_window
            )
          : modelForm.capabilities_json.trim();
      const payload = {
        provider_id: modelForm.provider_id,
        model_key: modelForm.model_key.trim(),
        name: modelForm.name.trim(),
        avatar: modelForm.avatar.trim(),
        capabilities_json: capabilitiesJSON,
        enabled: modelForm.enabled,
        sort: Number.isNaN(sortValue) ? 0 : sortValue,
      };
      await requestBackend(
        modelEditId ? `/v1/models/${modelEditId}` : "/v1/models",
        {
          method: modelEditId ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
          fallbackMessage: "保存模型失败",
          router,
          userId,
        }
      );
      resetModelForm();
      setIsModelFormOpen(false);
      await fetchAll();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("保存模型失败");
      }
    } finally {
      setSavingModel(false);
    }
  };

  const handleDeleteProvider = async (providerId: string) => {
    if (!window.confirm("确认删除这个模型提供商？")) {
      return;
    }
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setError("");
    try {
      await requestBackend(`/v1/model-providers/${providerId}`, {
        method: "DELETE",
        fallbackMessage: "删除提供商失败",
        router,
        userId,
      });
      if (providerEditId === providerId) {
        resetProviderForm();
        setIsProviderFormOpen(false);
      }
      await fetchAll();
    } catch {
      setError("删除提供商失败");
    }
  };

  const handleDeleteModel = async (modelId: string) => {
    if (!window.confirm("确认删除这个模型？")) {
      return;
    }
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setError("");
    try {
      await requestBackend(`/v1/models/${modelId}`, {
        method: "DELETE",
        fallbackMessage: "删除模型失败",
        router,
        userId,
      });
      if (modelEditId === modelId) {
        resetModelForm();
        setIsModelFormOpen(false);
      }
      await fetchAll();
    } catch {
      setError("删除模型失败");
    }
  };

  const handleSaveDefaultModel = async () => {
    if (!defaultModelId) {
      setError("请选择默认模型");
      return;
    }
    const token = getValidToken();
    if (!token) return;
    const userId = getUserId(token);
    setSavingDefault(true);
    setError("");
    try {
      await requestBackend("/v1/models/default", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ model_id: defaultModelId }),
        fallbackMessage: "保存默认模型失败",
        router,
        userId,
      });
      await fetchAll();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setError(err.message);
      } else {
        setError("保存默认模型失败");
      }
    } finally {
      setSavingDefault(false);
    }
  };

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="workspace-page-stack">
          <section className="workspace-filter-panel">
            <div className="workspace-section-title">
              <div>
                <h1>模型服务</h1>
                <p>
                  {currentDefaultModel
                    ? `当前默认模型：${currentDefaultModel.provider_name || "未知提供商"} / ${currentDefaultModel.name}`
                    : "管理模型提供商、模型目录以及系统默认模型。"}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <UiButton type="button" variant="secondary" onClick={fetchAll} loading={loading}>
                  刷新
                </UiButton>
              </div>
            </div>
            <div className="mt-4 flex flex-wrap items-center gap-3">
              <div className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--color-text-secondary)]">
                系统默认模型
              </div>
              <UiSelect
                className="min-w-[260px] flex-1"
                value={defaultModelId}
                disabled={models.length === 0}
                onChange={(e) => setDefaultModelId(e.target.value)}
              >
                {sortedModels.map((item) => (
                  <option key={item.model_id} value={item.model_id}>
                    {item.provider_name || "未知提供商"} / {item.name}
                  </option>
                ))}
              </UiSelect>
              <UiButton
                type="button"
                onClick={handleSaveDefaultModel}
                loading={savingDefault}
                disabled={models.length === 0}
              >
                保存默认模型
              </UiButton>
            </div>

            {error && (
              <div className="mt-4 rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                {error}
              </div>
            )}
          </section>

          <div className="grid gap-6 xl:grid-cols-[300px_minmax(0,1fr)]">
          <aside className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                模型提供商
              </div>
              <UiButton type="button" size="sm" onClick={handleCreateProvider}>
                新增提供商
              </UiButton>
            </div>

            <div className="mt-4">
              <UiInput
                placeholder="搜索提供商"
                value={providerQuery}
                onChange={(e) => setProviderQuery(e.target.value)}
              />
            </div>

            <div className="mt-4 space-y-2">
              {providers.length === 0 ? (
                <div className="workspace-empty-state !px-4 !py-6">
                  <strong>暂无模型服务</strong>
                  <span>先新增一个提供商，再录入它的模型目录。</span>
                </div>
              ) : filteredProviders.length === 0 ? (
                <div className="workspace-empty-state !px-4 !py-6">
                  <strong>没有匹配的服务</strong>
                  <span>调整关键字后重新搜索提供商。</span>
                </div>
              ) : (
                filteredProviders.map((item) => {
                  const isActive = item.provider_id === selectedProviderId;
                  const modelCount = providerModelCount.get(item.provider_id) ?? 0;
                  return (
                    <UiButton
                      key={item.provider_id}
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedProviderId(item.provider_id)}
                      className={`!h-auto min-h-[76px] w-full !items-start !justify-start rounded-[var(--radius-md)] !px-3 !py-3 text-left ${
                        isActive
                          ? "border-[var(--color-action-primary)] bg-[var(--color-surface-panel)]"
                          : "border-[var(--color-border-default)] bg-[var(--color-surface-panel)] hover:border-[var(--color-action-primary)]"
                      }`}
                    >
                      <div className="flex w-full items-start gap-3">
                        {item.avatar ? (
                          <img
                            src={item.avatar}
                            alt={item.name}
                            className="h-8 w-8 shrink-0 rounded-full border border-[var(--color-border-default)] object-cover"
                          />
                        ) : (
                          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-[var(--color-border-default)] text-sm font-semibold text-[var(--color-text-muted)]">
                            {item.name.slice(0, 1)}
                          </div>
                        )}
                        <div className="min-w-0 flex-1 pt-0.5">
                          <div className="truncate text-sm font-semibold leading-tight text-[var(--color-text-primary)]">
                            {item.name}
                          </div>
                          <div className="mt-1 truncate text-xs leading-tight text-[var(--color-text-muted)]">
                            {modelCount} 个模型 · {item.enabled ? "已启用" : "已停用"}
                          </div>
                        </div>
                      </div>
                    </UiButton>
                  );
                })
              )}
            </div>
          </aside>

          <main className="space-y-6">
            {selectedProvider ? (
                <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
                  <div className="flex flex-wrap items-start justify-between gap-3 border-b border-[var(--color-border-default)] pb-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-3">
                        {selectedProvider.avatar ? (
                          <img
                            src={selectedProvider.avatar}
                            alt={selectedProvider.name}
                            className="h-10 w-10 rounded-full border border-[var(--color-border-default)] object-cover"
                          />
                        ) : (
                          <div className="flex h-10 w-10 items-center justify-center rounded-full border border-[var(--color-border-default)] text-sm font-semibold text-[var(--color-text-muted)]">
                            {selectedProvider.name.slice(0, 1)}
                          </div>
                        )}
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <div className="truncate text-base font-semibold text-[var(--color-text-primary)]">
                              {selectedProvider.name}
                            </div>
                            <span className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-text-muted)]">
                              {selectedProvider.api_type}
                            </span>
                            <span
                              className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${
                                selectedProvider.enabled
                                  ? "border border-[var(--color-action-primary)] bg-[var(--color-surface-panel)] text-[var(--color-action-primary)]"
                                  : "border border-[var(--color-border-default)] bg-[var(--color-surface-panel)] text-[var(--color-text-muted)]"
                              }`}
                            >
                              {selectedProvider.enabled ? "已启用" : "已停用"}
                            </span>
                          </div>
                          <div className="mt-1 truncate text-xs text-[var(--color-text-muted)]">
                            {selectedProvider.base_url || "使用默认地址"} · 更新于{" "}
                            {formatDateLabel(selectedProvider.updated_at)}
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <UiButton
                        type="button"
                        variant="secondary"
                        size="sm"
                        onClick={() => handleEditProvider(selectedProvider)}
                      >
                        编辑提供商
                      </UiButton>
                      <UiButton
                        type="button"
                        variant="danger"
                        size="sm"
                        onClick={() => void handleDeleteProvider(selectedProvider.provider_id)}
                      >
                        删除提供商
                      </UiButton>
                    </div>
                  </div>

                  <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                        模型列表
                      </div>
                      <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                        按能力分组展示当前提供商下的模型。
                      </div>
                    </div>
                    <UiButton type="button" size="sm" onClick={handleCreateModel}>
                      新增模型
                    </UiButton>
                  </div>

                  <div className="mt-4 space-y-4">
                    {selectedProviderModels.length === 0 ? (
                      <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] px-3 py-8 text-center text-sm text-[var(--color-text-muted)]">
                        当前服务下还没有模型，建议先新增一个常用对话模型。
                      </div>
                    ) : (
                      groupedProviderModels.map((group) => (
                        <div
                          key={group.label}
                          className="rounded-[var(--radius-md)] border border-[var(--color-border-default)]"
                        >
                          <div className="flex items-center justify-between gap-3 border-b border-[var(--color-border-default)] px-4 py-3">
                            <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                              {group.label}
                            </div>
                            <div className="text-xs text-[var(--color-text-muted)]">
                              {group.items.length} 个模型
                            </div>
                          </div>
                          <div className="divide-y divide-[var(--color-border-default)]">
                            {group.items.map((item) => {
                              const capabilityTokens = parseCapabilityTokens(item.capabilities_json);
                              const contextWindow = extractContextWindow(item.capabilities_json);
                              return (
                                <div
                                  key={item.model_id}
                                  className="flex flex-wrap items-start justify-between gap-3 px-4 py-3"
                                >
                                  <div className="min-w-0 flex-1">
                                    <div className="flex items-center gap-2">
                                      {item.avatar ? (
                                        <img
                                          src={item.avatar}
                                          alt={item.name}
                                          className="h-9 w-9 rounded-full border border-[var(--color-border-default)] object-cover"
                                        />
                                      ) : (
                                        <div className="flex h-9 w-9 items-center justify-center rounded-full border border-[var(--color-border-default)] text-xs font-semibold text-[var(--color-text-muted)]">
                                          {item.name.slice(0, 1)}
                                        </div>
                                      )}
                                      <div className="min-w-0">
                                        <div className="flex flex-wrap items-center gap-2">
                                          <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
                                            {item.name}
                                          </div>
                                          {item.is_system_default && (
                                            <span className="rounded-full border border-[var(--color-action-primary)] bg-[var(--color-surface-panel)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-action-primary)]">
                                              default
                                            </span>
                                          )}
                                          <span
                                            className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${
                                              item.enabled
                                                ? "border border-[var(--color-action-primary)] bg-[var(--color-surface-panel)] text-[var(--color-action-primary)]"
                                                : "border border-[var(--color-border-default)] bg-[var(--color-surface-panel)] text-[var(--color-text-muted)]"
                                            }`}
                                          >
                                            {item.enabled ? "已启用" : "已停用"}
                                          </span>
                                        </div>
                                        <div className="truncate text-xs text-[var(--color-text-muted)]">
                                          {item.model_key}
                                        </div>
                                      </div>
                                    </div>
                                    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-[var(--color-text-muted)]">
                                      <span>排序 {item.sort}</span>
                                      <span>API 类型 {item.api_type || selectedProvider.api_type}</span>
                                      {contextWindow && (
                                        <span className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5">
                                          上下文 {formatNumberLabel(Number.parseInt(contextWindow, 10))}
                                        </span>
                                      )}
                                      {capabilityTokens.length > 0 ? (
                                        capabilityTokens.slice(0, 4).map((token) => (
                                          <span
                                            key={`${item.model_id}-${token}`}
                                            className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5"
                                          >
                                            {token}
                                          </span>
                                        ))
                                      ) : (
                                        <span className="rounded-full border border-[var(--color-border-default)] px-2 py-0.5">
                                          无能力标签
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <UiButton
                                      type="button"
                                      variant="secondary"
                                      size="sm"
                                      onClick={() => handleEditModel(item)}
                                    >
                                      编辑
                                    </UiButton>
                                    <UiButton
                                      type="button"
                                      variant="danger"
                                      size="sm"
                                      onClick={() => void handleDeleteModel(item.model_id)}
                                    >
                                      删除
                                    </UiButton>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </section>
            ) : (
              <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-8 text-center">
                <div className="workspace-empty-state">
                  <strong>还没有可展示的模型提供商</strong>
                  <span>先新增一个提供商，随后再录入该提供商下的模型。</span>
                </div>
                <div className="mt-4">
                  <UiButton type="button" onClick={handleCreateProvider}>
                    新增提供商
                  </UiButton>
                </div>
              </section>
            )}
          </main>
          </div>
        </div>

        <Modal
          open={isProviderFormOpen}
          title={providerEditId ? "编辑提供商" : "新增提供商"}
          onClose={() => {
            resetProviderForm();
            setIsProviderFormOpen(false);
          }}
          footer={
            <>
              <UiButton
                type="button"
                variant="ghost"
                onClick={() => {
                  resetProviderForm();
                  setIsProviderFormOpen(false);
                }}
              >
                取消
              </UiButton>
              <UiButton type="button" onClick={handleSaveProvider} loading={savingProvider}>
                {providerEditId ? "更新提供商" : "新增提供商"}
              </UiButton>
            </>
          }
        >
          <div className="grid gap-3 sm:grid-cols-2">
            <UiInput
              placeholder="提供商名称"
              value={providerForm.name}
              onChange={(e) => setProviderForm((prev) => ({ ...prev, name: e.target.value }))}
            />
            <UiSelect
              value={providerForm.api_type}
              onChange={(e) =>
                setProviderForm((prev) => ({ ...prev, api_type: e.target.value }))
              }
            >
              <option value="openai">openai</option>
              <option value="ark">ark</option>
              <option value="deepseek">deepseek</option>
            </UiSelect>
            <UiInput
              className="sm:col-span-2"
              placeholder="Base URL（可选）"
              value={providerForm.base_url}
              onChange={(e) =>
                setProviderForm((prev) => ({ ...prev, base_url: e.target.value }))
              }
            />
            <UiInput
              placeholder={providerEditId ? "留空表示不更新 API Key" : "API Key"}
              value={providerForm.api_key}
              onChange={(e) => setProviderForm((prev) => ({ ...prev, api_key: e.target.value }))}
            />
            <UiInput
              placeholder="提供商头像 URL（可选）"
              value={providerForm.avatar}
              onChange={(e) => setProviderForm((prev) => ({ ...prev, avatar: e.target.value }))}
            />
            <UiTextarea
              className="sm:col-span-2 min-h-24"
              placeholder="extra_config_json（可选）"
              value={providerForm.extra_config_json}
              onChange={(e) =>
                setProviderForm((prev) => ({
                  ...prev,
                  extra_config_json: e.target.value,
                }))
              }
            />
          </div>
          <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-panel)] px-3 py-2">
            <div className="text-sm text-[var(--color-text-secondary)]">提供商状态</div>
            <UiButton
              type="button"
              variant={providerForm.enabled ? "secondary" : "ghost"}
              size="sm"
              onClick={() =>
                setProviderForm((prev) => ({
                  ...prev,
                  enabled: !prev.enabled,
                }))
              }
            >
              {providerForm.enabled ? "已启用" : "已停用"}
            </UiButton>
          </div>
        </Modal>

        <Modal
          open={isModelFormOpen}
          title={modelEditId ? "编辑模型" : "新增模型"}
          onClose={() => {
            resetModelForm();
            setIsModelFormOpen(false);
          }}
          footer={
            <>
              <UiButton
                type="button"
                variant="ghost"
                onClick={() => {
                  resetModelForm();
                  setIsModelFormOpen(false);
                }}
              >
                取消
              </UiButton>
              <UiButton type="button" onClick={handleSaveModel} loading={savingModel}>
                {modelEditId ? "更新模型" : "新增模型"}
              </UiButton>
            </>
          }
        >
          <div className="grid gap-3 sm:grid-cols-2">
            <UiSelect
              value={modelForm.provider_id}
              onChange={(e) =>
                setModelForm((prev) => ({
                  ...prev,
                  provider_id: e.target.value,
                }))
              }
            >
              <option value="">选择提供商</option>
              {providerOptions.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </UiSelect>
            <UiInput
              placeholder="模型标识，例如 gpt-4o-mini"
              value={modelForm.model_key}
              onChange={(e) =>
                setModelForm((prev) => ({
                  ...prev,
                  model_key: e.target.value,
                }))
              }
            />
            <UiInput
              placeholder="模型展示名称"
              value={modelForm.name}
              onChange={(e) => setModelForm((prev) => ({ ...prev, name: e.target.value }))}
            />
            <UiInput
              placeholder="模型头像 URL（可选）"
              value={modelForm.avatar}
              onChange={(e) => setModelForm((prev) => ({ ...prev, avatar: e.target.value }))}
            />
            <UiInput
              placeholder="排序"
              value={modelForm.sort}
              onChange={(e) => setModelForm((prev) => ({ ...prev, sort: e.target.value }))}
            />
            <UiInput
              type="number"
              min="1"
              step="1"
              placeholder="上下文大小，例如 128000"
              value={modelForm.context_window}
              onChange={(e) =>
                setModelForm((prev) => ({
                  ...prev,
                  context_window: e.target.value,
                }))
              }
            />
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-panel)] px-3 py-2 text-sm text-[var(--color-text-secondary)]">
              <span>模型状态</span>
              <UiButton
                type="button"
                variant={modelForm.enabled ? "secondary" : "ghost"}
                size="sm"
                onClick={() =>
                  setModelForm((prev) => ({
                    ...prev,
                    enabled: !prev.enabled,
                  }))
                }
              >
                {modelForm.enabled ? "已启用" : "已停用"}
              </UiButton>
            </div>
            <div className="sm:col-span-2 text-xs leading-5 text-[var(--color-text-muted)]">
              上下文大小会写入 <code>capabilities_json.context_window</code>。如果这里留空，会移除已有的上下文窗口配置。
            </div>
            <UiTextarea
              className="sm:col-span-2 min-h-24"
              placeholder="capabilities_json（可选）"
              value={modelForm.capabilities_json}
              onChange={(e) =>
                setModelForm((prev) => ({
                  ...prev,
                  capabilities_json: e.target.value,
                }))
              }
            />
          </div>
        </Modal>
      </div>
    </div>
  );
}
