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

type COSConfig = {
  secret_id?: string;
  secret_key?: string;
  bucket?: string;
  region?: string;
};

type APMPlusConfig = {
  host?: string;
  app_key?: string;
  service_name?: string;
  release?: string;
};

type Props = {
  summaryLLM?: SummaryLLMConfig;
  cos?: COSConfig;
  apmplus?: APMPlusConfig;
  onSave: (section: string, updates: Record<string, unknown>) => void;
  saving: boolean;
};

export function SystemSettings({ summaryLLM, cos, apmplus, onSave, saving }: Props) {
  // SummaryLLM state
  const [summaryModel, setSummaryModel] = useState("");
  const [summaryApiKey, setSummaryApiKey] = useState("");
  const [summaryBaseUrl, setSummaryBaseUrl] = useState("");
  const [summaryProvider, setSummaryProvider] = useState("deepseek");

  // COS state
  const [cosSecretId, setCosSecretId] = useState("");
  const [cosSecretKey, setCosSecretKey] = useState("");
  const [cosBucket, setCosBucket] = useState("");
  const [cosRegion, setCosRegion] = useState("");

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
    if (cos) {
      if (cos.secret_id !== undefined) setCosSecretId(cos.secret_id);
      if (cos.secret_key !== undefined) setCosSecretKey(cos.secret_key);
      if (cos.bucket !== undefined) setCosBucket(cos.bucket);
      if (cos.region !== undefined) setCosRegion(cos.region);
    }
    if (apmplus) {
      if (apmplus.host !== undefined) setApmHost(apmplus.host);
      if (apmplus.app_key !== undefined) setApmAppKey(apmplus.app_key);
      if (apmplus.service_name !== undefined) setApmServiceName(apmplus.service_name);
      if (apmplus.release !== undefined) setApmRelease(apmplus.release);
    }
  }, [summaryLLM, cos, apmplus]);

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

  const handleSaveCOS = () => {
    const updates: Record<string, unknown> = {
      bucket: cosBucket,
      region: cosRegion,
    };
    if (cosSecretId.trim()) {
      updates.secret_id = cosSecretId.trim();
    }
    if (cosSecretKey.trim()) {
      updates.secret_key = cosSecretKey.trim();
    }
    onSave("cos", updates);
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

      {/* COS Section */}
      <div className="space-y-4 border-t border-[var(--color-border-default)] pt-6">
        <div>
          <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
            对象存储配置 (COS)
          </h3>
          <p className="mt-1 text-sm text-[var(--color-text-muted)]">
            腾讯云对象存储服务配置，用于文件上传和存储
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Secret ID
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {cos?.secret_id || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={cosSecretId}
                onChange={(e) => setCosSecretId(e.target.value)}
                placeholder="输入新的 Secret ID"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Secret Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {cos?.secret_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={cosSecretKey}
                onChange={(e) => setCosSecretKey(e.target.value)}
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
                value={cosBucket}
                onChange={(e) => setCosBucket(e.target.value)}
                placeholder="my-bucket-1234567890"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Region
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={cosRegion}
                onChange={(e) => setCosRegion(e.target.value)}
                placeholder="ap-guangzhou"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <UiButton onClick={handleSaveCOS} disabled={saving}>
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