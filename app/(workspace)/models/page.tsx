"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

type UserInfo = {
  user_id?: string | number;
};

type BackendResult<T> = {
  code: number;
  message: string;
  data?: T;
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
  enabled: boolean;
  sort: string;
};

const API_BASE = "/api/backend";

const EMPTY_PROVIDER_FORM: ProviderFormState = {
  name: "",
  api_type: "ark",
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
  enabled: true,
  sort: "0",
};

const parseErrorMessage = async (response: Response, fallback: string) => {
  const data = (await response.json().catch(() => null)) as BackendResult<unknown> | null;
  return data?.message || fallback;
};

export default function ModelsPage() {
  const router = useRouter();
  const [providers, setProviders] = useState<ModelProviderItem[]>([]);
  const [models, setModels] = useState<ModelItem[]>([]);
  const [defaultModelId, setDefaultModelId] = useState("");
  const [providerForm, setProviderForm] = useState<ProviderFormState>(EMPTY_PROVIDER_FORM);
  const [providerEditId, setProviderEditId] = useState("");
  const [modelForm, setModelForm] = useState<ModelFormState>(EMPTY_MODEL_FORM);
  const [modelEditId, setModelEditId] = useState("");
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
      const [providerRes, modelRes, defaultRes] = await Promise.all([
        fetch(`${API_BASE}/v1/model-providers?page=1&page_size=100`, {
          headers: buildAuthHeaders(token, userId),
        }),
        fetch(`${API_BASE}/v1/models?page=1&page_size=200`, {
          headers: buildAuthHeaders(token, userId),
        }),
        fetch(`${API_BASE}/v1/models/default`, {
          headers: buildAuthHeaders(token, userId),
        }),
      ]);

      updateTokenFromResponse(providerRes);
      updateTokenFromResponse(modelRes);
      updateTokenFromResponse(defaultRes);

      const providerData = (await providerRes.json()) as BackendResult<BackendPage<ModelProviderItem>>;
      const modelData = (await modelRes.json()) as BackendResult<BackendPage<ModelItem>>;
      const defaultData = (await defaultRes.json()) as BackendResult<DefaultModelResponse>;

      if (!providerRes.ok || providerData.code !== 0) {
        throw new Error(providerData.message || "获取模型提供商失败");
      }
      if (!modelRes.ok || modelData.code !== 0) {
        throw new Error(modelData.message || "获取模型列表失败");
      }
      if (!defaultRes.ok || defaultData.code !== 0) {
        throw new Error(defaultData.message || "获取默认模型失败");
      }

      const nextProviders = providerData.data?.data ?? [];
      const nextModels = modelData.data?.data ?? [];
      const nextDefaultModelId =
        typeof defaultData.data?.model_id === "string" ? defaultData.data.model_id : "";

      setProviders(nextProviders);
      setModels(nextModels);
      setDefaultModelId(nextDefaultModelId);
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
  }, [getUserId, getValidToken]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const providerOptions = useMemo(
    () =>
      providers.map((item) => ({
        value: item.provider_id,
        label: item.name,
      })),
    [providers]
  );

  const resetProviderForm = () => {
    setProviderForm(EMPTY_PROVIDER_FORM);
    setProviderEditId("");
  };

  const resetModelForm = () => {
    setModelForm({
      ...EMPTY_MODEL_FORM,
      provider_id: providers[0]?.provider_id || "",
    });
    setModelEditId("");
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
      const response = await fetch(
        providerEditId
          ? `${API_BASE}/v1/model-providers/${providerEditId}`
          : `${API_BASE}/v1/model-providers`,
        {
          method: providerEditId ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
            ...buildAuthHeaders(token, userId),
          },
          body: JSON.stringify(payload),
        }
      );
      updateTokenFromResponse(response);
      if (!response.ok) {
        throw new Error(await parseErrorMessage(response, "保存提供商失败"));
      }
      const data = (await response.json()) as BackendResult<unknown>;
      if (data.code !== 0) {
        throw new Error(data.message || "保存提供商失败");
      }
      resetProviderForm();
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
      const payload = {
        provider_id: modelForm.provider_id,
        model_key: modelForm.model_key.trim(),
        name: modelForm.name.trim(),
        avatar: modelForm.avatar.trim(),
        capabilities_json: modelForm.capabilities_json.trim(),
        enabled: modelForm.enabled,
        sort: Number.isNaN(sortValue) ? 0 : sortValue,
      };
      const response = await fetch(
        modelEditId ? `${API_BASE}/v1/models/${modelEditId}` : `${API_BASE}/v1/models`,
        {
          method: modelEditId ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
            ...buildAuthHeaders(token, userId),
          },
          body: JSON.stringify(payload),
        }
      );
      updateTokenFromResponse(response);
      if (!response.ok) {
        throw new Error(await parseErrorMessage(response, "保存模型失败"));
      }
      const data = (await response.json()) as BackendResult<unknown>;
      if (data.code !== 0) {
        throw new Error(data.message || "保存模型失败");
      }
      resetModelForm();
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
      const response = await fetch(`${API_BASE}/v1/model-providers/${providerId}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(response);
      if (!response.ok) {
        setError(await parseErrorMessage(response, "删除提供商失败"));
        return;
      }
      const data = (await response.json()) as BackendResult<unknown>;
      if (data.code !== 0) {
        setError(data.message || "删除提供商失败");
        return;
      }
      if (providerEditId === providerId) {
        resetProviderForm();
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
      const response = await fetch(`${API_BASE}/v1/models/${modelId}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(response);
      if (!response.ok) {
        setError(await parseErrorMessage(response, "删除模型失败"));
        return;
      }
      const data = (await response.json()) as BackendResult<unknown>;
      if (data.code !== 0) {
        setError(data.message || "删除模型失败");
        return;
      }
      if (modelEditId === modelId) {
        resetModelForm();
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
      const response = await fetch(`${API_BASE}/v1/models/default`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...buildAuthHeaders(token, userId),
        },
        body: JSON.stringify({ model_id: defaultModelId }),
      });
      updateTokenFromResponse(response);
      if (!response.ok) {
        throw new Error(await parseErrorMessage(response, "保存默认模型失败"));
      }
      const data = (await response.json()) as BackendResult<unknown>;
      if (data.code !== 0) {
        throw new Error(data.message || "保存默认模型失败");
      }
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
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-6">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="workspace-toolbar-surface rounded-[var(--radius-lg)] border p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                模型服务管理
              </div>
              <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                管理模型提供商、模型目录以及系统默认模型。
              </div>
            </div>
            <UiButton
              type="button"
              variant="secondary"
              onClick={fetchAll}
              loading={loading}
            >
              刷新
            </UiButton>
          </div>
        </div>

        {error && (
          <div className="mt-4 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            {error}
          </div>
        )}

        <div className="mt-6 grid gap-6 lg:grid-cols-2">
          <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                  模型提供商
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  存储 API 类型、地址、密钥和提供商级头像。
                </div>
              </div>
              {providerEditId && (
                <UiButton type="button" variant="ghost" size="sm" onClick={resetProviderForm}>
                  取消编辑
                </UiButton>
              )}
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <UiInput
                placeholder="提供商名称"
                value={providerForm.name}
                onChange={(e) =>
                  setProviderForm((prev) => ({ ...prev, name: e.target.value }))
                }
              />
              <UiSelect
                value={providerForm.api_type}
                onChange={(e) =>
                  setProviderForm((prev) => ({ ...prev, api_type: e.target.value }))
                }
              >
                <option value="ark">ark</option>
                <option value="deepseek">deepseek</option>
                <option value="openai_compatible">openai_compatible</option>
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
                onChange={(e) =>
                  setProviderForm((prev) => ({ ...prev, api_key: e.target.value }))
                }
              />
              <UiInput
                placeholder="提供商头像 URL（可选）"
                value={providerForm.avatar}
                onChange={(e) =>
                  setProviderForm((prev) => ({ ...prev, avatar: e.target.value }))
                }
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

            <label className="mt-3 flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
              <input
                type="checkbox"
                checked={providerForm.enabled}
                onChange={(e) =>
                  setProviderForm((prev) => ({ ...prev, enabled: e.target.checked }))
                }
              />
              启用提供商
            </label>

            <div className="mt-4">
              <UiButton type="button" onClick={handleSaveProvider} loading={savingProvider}>
                {providerEditId ? "更新提供商" : "新增提供商"}
              </UiButton>
            </div>

            <div className="mt-4 space-y-3">
              {providers.length === 0 ? (
                <div className="text-sm text-[var(--color-text-muted)]">暂无提供商</div>
              ) : (
                providers.map((item) => (
                  <div
                    key={item.provider_id}
                    className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] p-3"
                  >
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          {item.avatar ? (
                            <img
                              src={item.avatar}
                              alt={item.name}
                              className="h-8 w-8 rounded-full border border-[var(--color-border-default)] object-cover"
                            />
                          ) : (
                            <div className="flex h-8 w-8 items-center justify-center rounded-full border border-[var(--color-border-default)] text-xs text-[var(--color-text-muted)]">
                              {item.name.slice(0, 1)}
                            </div>
                          )}
                          <div className="min-w-0">
                            <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
                              {item.name}
                            </div>
                            <div className="truncate text-xs text-[var(--color-text-muted)]">
                              {item.api_type}
                            </div>
                          </div>
                        </div>
                        <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                          Base URL: {item.base_url || "-"}
                        </div>
                        <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                          API Key: {item.api_key_masked || "-"}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <UiButton
                          type="button"
                          variant="secondary"
                          size="sm"
                          onClick={() => {
                            setProviderEditId(item.provider_id);
                            setProviderForm({
                              name: item.name || "",
                              api_type: item.api_type || "ark",
                              base_url: item.base_url || "",
                              api_key: "",
                              avatar: item.avatar || "",
                              extra_config_json: item.extra_config_json || "",
                              enabled: !!item.enabled,
                            });
                          }}
                        >
                          编辑
                        </UiButton>
                        <UiButton
                          type="button"
                          variant="danger"
                          size="sm"
                          onClick={() => void handleDeleteProvider(item.provider_id)}
                        >
                          删除
                        </UiButton>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </section>

          <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                  模型目录
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  独立维护模型标识、模型头像以及系统默认模型。
                </div>
              </div>
              {modelEditId && (
                <UiButton type="button" variant="ghost" size="sm" onClick={resetModelForm}>
                  取消编辑
                </UiButton>
              )}
            </div>

            <div className="mt-4 rounded-[var(--radius-md)] border border-[var(--color-border-default)] p-3">
              <div className="text-xs font-semibold text-[var(--color-text-secondary)]">
                系统默认模型
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <UiSelect
                  className="min-w-[240px] flex-1"
                  value={defaultModelId}
                  onChange={(e) => setDefaultModelId(e.target.value)}
                >
                  <option value="">请选择默认模型</option>
                  {models.map((item) => (
                    <option key={item.model_id} value={item.model_id}>
                      {item.provider_name || "未知提供商"} / {item.name}
                    </option>
                  ))}
                </UiSelect>
                <UiButton
                  type="button"
                  onClick={handleSaveDefaultModel}
                  loading={savingDefault}
                >
                  保存默认模型
                </UiButton>
              </div>
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <UiSelect
                value={modelForm.provider_id}
                onChange={(e) =>
                  setModelForm((prev) => ({ ...prev, provider_id: e.target.value }))
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
                placeholder="模型标识，例如 deepseek-chat"
                value={modelForm.model_key}
                onChange={(e) =>
                  setModelForm((prev) => ({ ...prev, model_key: e.target.value }))
                }
              />
              <UiInput
                placeholder="模型展示名称"
                value={modelForm.name}
                onChange={(e) =>
                  setModelForm((prev) => ({ ...prev, name: e.target.value }))
                }
              />
              <UiInput
                placeholder="模型头像 URL（可选）"
                value={modelForm.avatar}
                onChange={(e) =>
                  setModelForm((prev) => ({ ...prev, avatar: e.target.value }))
                }
              />
              <UiInput
                placeholder="排序"
                value={modelForm.sort}
                onChange={(e) =>
                  setModelForm((prev) => ({ ...prev, sort: e.target.value }))
                }
              />
              <div className="flex items-center text-sm text-[var(--color-text-secondary)]">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={modelForm.enabled}
                    onChange={(e) =>
                      setModelForm((prev) => ({ ...prev, enabled: e.target.checked }))
                    }
                  />
                  启用模型
                </label>
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

            <div className="mt-4">
              <UiButton type="button" onClick={handleSaveModel} loading={savingModel}>
                {modelEditId ? "更新模型" : "新增模型"}
              </UiButton>
            </div>

            <div className="mt-4 space-y-3">
              {models.length === 0 ? (
                <div className="text-sm text-[var(--color-text-muted)]">暂无模型</div>
              ) : (
                models.map((item) => (
                  <div
                    key={item.model_id}
                    className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] p-3"
                  >
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          {item.avatar ? (
                            <img
                              src={item.avatar}
                              alt={item.name}
                              className="h-8 w-8 rounded-full border border-[var(--color-border-default)] object-cover"
                            />
                          ) : (
                            <div className="flex h-8 w-8 items-center justify-center rounded-full border border-[var(--color-border-default)] text-xs text-[var(--color-text-muted)]">
                              {item.name.slice(0, 1)}
                            </div>
                          )}
                          <div className="min-w-0">
                            <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
                              {item.name}
                            </div>
                            <div className="truncate text-xs text-[var(--color-text-muted)]">
                              {item.provider_name || "未知提供商"} / {item.model_key}
                            </div>
                          </div>
                          {item.is_system_default && (
                            <span className="rounded-full border border-[rgba(37,99,255,0.16)] bg-[rgba(37,99,255,0.08)] px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--color-action-primary)]">
                              default
                            </span>
                          )}
                        </div>
                        <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                          API 类型: {item.api_type || "-"}，排序: {item.sort}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <UiButton
                          type="button"
                          variant="secondary"
                          size="sm"
                          onClick={() => {
                            setModelEditId(item.model_id);
                            setModelForm({
                              provider_id: item.provider_id || "",
                              model_key: item.model_key || "",
                              name: item.name || "",
                              avatar: item.avatar || "",
                              capabilities_json: item.capabilities_json || "",
                              enabled: !!item.enabled,
                              sort: String(item.sort ?? 0),
                            });
                          }}
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
                  </div>
                ))
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
