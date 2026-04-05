"use client";

import { useEffect, useState } from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";
import { UiSelect } from "../../../components/ui/UiSelect";

type SummaryLLMConfig = {
  model?: string;
  api_key?: string;
  base_url?: string;
  provider?: string;
};

type MinIOConfig = {
  endpoint?: string;
  access_key?: string;
  secret_key?: string;
  bucket?: string;
  use_ssl?: boolean;
  public_base_url?: string;
};

type APMPlusConfig = {
  host?: string;
  app_key?: string;
  service_name?: string;
  release?: string;
};

type Props = {
  summaryLLM?: SummaryLLMConfig;
  minio?: MinIOConfig;
  apmplus?: APMPlusConfig;
  onSave: (section: string, updates: Record<string, unknown>) => void;
  saving: boolean;
};

export function SystemSettings({ summaryLLM, minio, apmplus, onSave, saving }: Props) {
  // SummaryLLM state
  const [summaryModel, setSummaryModel] = useState("");
  const [summaryApiKey, setSummaryApiKey] = useState("");
  const [summaryBaseUrl, setSummaryBaseUrl] = useState("");
  const [summaryProvider, setSummaryProvider] = useState("deepseek");

  // MinIO state
  const [minioEndpoint, setMinioEndpoint] = useState("");
  const [minioAccessKey, setMinioAccessKey] = useState("");
  const [minioSecretKey, setMinioSecretKey] = useState("");
  const [minioBucket, setMinioBucket] = useState("");
  const [minioUseSSL, setMinioUseSSL] = useState(false);
  const [minioPublicBaseUrl, setMinioPublicBaseUrl] = useState("");

  // APMPlus state
  const [apmHost, setApmHost] = useState("");
  const [apmAppKey, setApmAppKey] = useState("");
  const [apmServiceName, setApmServiceName] = useState("");
  const [apmRelease, setApmRelease] = useState("");

  // Sync props to state
  useEffect(() => {
    if (summaryLLM) {
      if (summaryLLM.model !== undefined) setSummaryModel(summaryLLM.model);
      if (summaryLLM.api_key !== undefined) setSummaryApiKey(summaryLLM.api_key);
      if (summaryLLM.base_url !== undefined) setSummaryBaseUrl(summaryLLM.base_url);
      if (summaryLLM.provider !== undefined) setSummaryProvider(summaryLLM.provider);
    }
    if (minio) {
      if (minio.endpoint !== undefined) setMinioEndpoint(minio.endpoint);
      if (minio.bucket !== undefined) setMinioBucket(minio.bucket);
      if (minio.use_ssl !== undefined) setMinioUseSSL(minio.use_ssl);
      if (minio.public_base_url !== undefined) setMinioPublicBaseUrl(minio.public_base_url);
    }
    setMinioAccessKey("");
    setMinioSecretKey("");
    if (apmplus) {
      if (apmplus.host !== undefined) setApmHost(apmplus.host);
      if (apmplus.app_key !== undefined) setApmAppKey(apmplus.app_key);
      if (apmplus.service_name !== undefined) setApmServiceName(apmplus.service_name);
      if (apmplus.release !== undefined) setApmRelease(apmplus.release);
    }
  }, [summaryLLM, minio, apmplus]);

  const handleSaveSummaryLLM = () => {
    const updates: Record<string, unknown> = {
      model: summaryModel,
      provider: summaryProvider,
      base_url: summaryBaseUrl,
    };
    if (summaryApiKey.trim()) {
      updates.api_key = summaryApiKey.trim();
    }
    onSave("summary_llm", updates);
  };

  const handleSaveMinIO = () => {
    const updates: Record<string, unknown> = {
      endpoint: minioEndpoint,
      bucket: minioBucket,
      use_ssl: minioUseSSL,
      public_base_url: minioPublicBaseUrl,
    };
    if (minioAccessKey.trim()) {
      updates.access_key = minioAccessKey.trim();
    }
    if (minioSecretKey.trim()) {
      updates.secret_key = minioSecretKey.trim();
    }
    onSave("minio", updates);
  };

  const handleSaveAPMPlus = () => {
    const updates: Record<string, unknown> = {
      host: apmHost,
      service_name: apmServiceName,
      release: apmRelease,
    };
    if (apmAppKey.trim()) {
      updates.app_key = apmAppKey.trim();
    }
    onSave("apmplus", updates);
  };

  return (
    <div className="space-y-8">
      {/* SummaryLLM Section */}
      <div className="space-y-4">
        <div>
          <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
            摘要模型配置
          </h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            用于上下文压缩时生成摘要的模型配置
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              提供商
            </label>
            <div className="mt-2 max-w-md">
              <UiSelect
                value={summaryProvider}
                onChange={(e) => setSummaryProvider(e.target.value)}
              >
                <option value="deepseek">DeepSeek</option>
                <option value="openai">OpenAI</option>
                <option value="ark">火山引擎 ARK</option>
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              模型名称
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={summaryModel}
                onChange={(e) => setSummaryModel(e.target.value)}
                placeholder="deepseek-chat"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Base URL
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              可选，留空使用默认地址
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                value={summaryBaseUrl}
                onChange={(e) => setSummaryBaseUrl(e.target.value)}
                placeholder="https://api.deepseek.com"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {summaryLLM?.api_key || "未设置"}。输入新值以更新，留空保持不变。
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={summaryApiKey}
                onChange={(e) => setSummaryApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <UiButton onClick={handleSaveSummaryLLM} disabled={saving}>
            {saving ? "保存中..." : "保存摘要模型配置"}
          </UiButton>
        </div>
      </div>

      {/* MinIO Section */}
      <div className="space-y-4 border-t border-[var(--color-border-default)] pt-6">
        <div>
          <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
            对象存储配置 (MinIO)
          </h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            MinIO 对象存储服务配置，用于文件上传和存储
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Endpoint
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={minioEndpoint}
                onChange={(e) => setMinioEndpoint(e.target.value)}
                placeholder="minio.example.com:9000"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Access Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {minio?.access_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={minioAccessKey}
                onChange={(e) => setMinioAccessKey(e.target.value)}
                placeholder="输入新的 Access Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Secret Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {minio?.secret_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={minioSecretKey}
                onChange={(e) => setMinioSecretKey(e.target.value)}
                placeholder="输入新的 Secret Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Bucket
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={minioBucket}
                onChange={(e) => setMinioBucket(e.target.value)}
                placeholder="openintern"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Use SSL
            </label>
            <div className="mt-2 max-w-md">
              <UiSelect
                value={minioUseSSL ? "true" : "false"}
                onChange={(e) => setMinioUseSSL(e.target.value === "true")}
              >
                <option value="true">是</option>
                <option value="false">否</option>
              </UiSelect>
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Public Base URL
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              可选，用于拼接公开访问链接
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                value={minioPublicBaseUrl}
                onChange={(e) => setMinioPublicBaseUrl(e.target.value)}
                placeholder="https://cdn.example.com"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <UiButton onClick={handleSaveMinIO} disabled={saving}>
            {saving ? "保存中..." : "保存存储配置"}
          </UiButton>
        </div>
      </div>

      {/* APMPlus Section */}
      <div className="space-y-4 border-t border-[var(--color-border-default)] pt-6">
        <div>
          <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
            APM Plus 配置
          </h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            应用性能监控服务配置，用于链路追踪和性能分析
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Host
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={apmHost}
                onChange={(e) => setApmHost(e.target.value)}
                placeholder="apmplus-cn-beijing.volces.com:4317"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              App Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {apmplus?.app_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={apmAppKey}
                onChange={(e) => setApmAppKey(e.target.value)}
                placeholder="输入新的 App Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Service Name
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={apmServiceName}
                onChange={(e) => setApmServiceName(e.target.value)}
                placeholder="openintern-backend"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Release
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={apmRelease}
                onChange={(e) => setApmRelease(e.target.value)}
                placeholder="1.0.0"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <UiButton onClick={handleSaveAPMPlus} disabled={saving}>
            {saving ? "保存中..." : "保存监控配置"}
          </UiButton>
        </div>
      </div>
    </div>
  );
}
