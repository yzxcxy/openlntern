"use client";

import { useEffect, useState } from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";
import { UiSelect } from "../../../components/ui/UiSelect";

type OpenVikingServiceConfig = {
  storage?: {
    workspace?: string;
    vectordb?: {
      name?: string;
      backend?: string;
    };
    agfs?: {
      port?: number;
      log_level?: string;
      backend?: string;
    };
  };
  log?: {
    level?: string;
    output?: string;
  };
  embedding?: {
    dense?: {
      api_base?: string;
      api_key?: string;
      provider?: string;
      dimension?: number;
      model?: string;
      input?: string;
      batch_size?: number;
      query_param?: string;
      document_param?: string;
      extra_headers?: Record<string, string>;
      ak?: string;
      sk?: string;
      region?: string;
    };
    sparse?: {
      api_base?: string;
      api_key?: string;
      provider?: string;
      model?: string;
    };
    hybrid?: {
      api_base?: string;
      api_key?: string;
      provider?: string;
      model?: string;
      dimension?: number;
    };
    max_concurrent?: number;
  };
  vlm?: {
    api_base?: string;
    api_key?: string;
    provider?: string;
    model?: string;
    max_concurrent?: number;
    thinking?: boolean;
    stream?: boolean;
    extra_headers?: Record<string, string>;
  };
  parsers?: {
    code?: {
      code_summary_mode?: string;
    };
  };
  rerank?: {
    api_base?: string;
    api_key?: string;
    provider?: string;
    model?: string;
    threshold?: number;
  };
};

type Props = {
  config?: OpenVikingServiceConfig;
  onSave: (updates: Record<string, unknown>) => void;
  saving: boolean;
};

// 可折叠区域组件
function CollapsibleSection({
  title,
  defaultOpen = false,
  children,
}: {
  title: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  return (
    <div className="border border-[var(--color-border-default)] rounded-[var(--radius-lg)]">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full px-4 py-3 flex items-center justify-between text-left hover:bg-[var(--color-bg-page)]"
      >
        <span className="font-medium text-[var(--color-text-primary)]">
          {title}
        </span>
        <svg
          className={`h-5 w-5 transition-transform ${isOpen ? "rotate-180" : ""}`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {isOpen && <div className="px-4 pb-4 space-y-4">{children}</div>}
    </div>
  );
}

// Select options helper
function SelectOptions({ options }: { options: { value: string; label: string }[] }) {
  return (
    <>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </>
  );
}

export function OpenVikingServiceSettings({
  config,
  onSave,
  saving,
}: Props) {
  // Storage
  const [workspace, setWorkspace] = useState(config?.storage?.workspace ?? "");
  const [vectordbName, setVectordbName] = useState(
    config?.storage?.vectordb?.name ?? ""
  );
  const [vectordbBackend, setVectordbBackend] = useState(
    config?.storage?.vectordb?.backend ?? ""
  );
  const [agfsPort, setAgfsPort] = useState(
    config?.storage?.agfs?.port?.toString() ?? ""
  );
  const [agfsLogLevel, setAgfsLogLevel] = useState(
    config?.storage?.agfs?.log_level ?? ""
  );
  const [agfsBackend, setAgfsBackend] = useState(
    config?.storage?.agfs?.backend ?? ""
  );

  // Log
  const [logLevel, setLogLevel] = useState(config?.log?.level ?? "INFO");
  const [logOutput, setLogOutput] = useState(config?.log?.output ?? "stdout");

  // Dense Embedding
  const [denseApiBase, setDenseApiBase] = useState(
    config?.embedding?.dense?.api_base ?? ""
  );
  const [denseApiKey, setDenseApiKey] = useState("");
  const [denseProvider, setDenseProvider] = useState(
    config?.embedding?.dense?.provider ?? ""
  );
  const [denseDimension, setDenseDimension] = useState(
    config?.embedding?.dense?.dimension?.toString() ?? ""
  );
  const [denseModel, setDenseModel] = useState(
    config?.embedding?.dense?.model ?? ""
  );
  const [denseInput, setDenseInput] = useState(
    config?.embedding?.dense?.input ?? ""
  );
  const [denseBatchSize, setDenseBatchSize] = useState(
    config?.embedding?.dense?.batch_size?.toString() ?? ""
  );
  const [denseQueryParam, setDenseQueryParam] = useState(
    config?.embedding?.dense?.query_param ?? ""
  );
  const [denseDocumentParam, setDenseDocumentParam] = useState(
    config?.embedding?.dense?.document_param ?? ""
  );
  const [denseAk, setDenseAk] = useState("");
  const [denseSk, setDenseSk] = useState("");
  const [denseRegion, setDenseRegion] = useState(
    config?.embedding?.dense?.region ?? ""
  );

  // Sparse Embedding
  const [sparseApiBase, setSparseApiBase] = useState(
    config?.embedding?.sparse?.api_base ?? ""
  );
  const [sparseApiKey, setSparseApiKey] = useState("");
  const [sparseProvider, setSparseProvider] = useState(
    config?.embedding?.sparse?.provider ?? ""
  );
  const [sparseModel, setSparseModel] = useState(
    config?.embedding?.sparse?.model ?? ""
  );

  // Hybrid Embedding
  const [hybridApiBase, setHybridApiBase] = useState(
    config?.embedding?.hybrid?.api_base ?? ""
  );
  const [hybridApiKey, setHybridApiKey] = useState("");
  const [hybridProvider, setHybridProvider] = useState(
    config?.embedding?.hybrid?.provider ?? ""
  );
  const [hybridModel, setHybridModel] = useState(
    config?.embedding?.hybrid?.model ?? ""
  );
  const [hybridDimension, setHybridDimension] = useState(
    config?.embedding?.hybrid?.dimension?.toString() ?? ""
  );

  const [embeddingMaxConcurrent, setEmbeddingMaxConcurrent] = useState(
    config?.embedding?.max_concurrent?.toString() ?? "10"
  );

  // VLM
  const [vlmApiBase, setVlmApiBase] = useState(config?.vlm?.api_base ?? "");
  const [vlmApiKey, setVlmApiKey] = useState("");
  const [vlmProvider, setVlmProvider] = useState(config?.vlm?.provider ?? "");
  const [vlmModel, setVlmModel] = useState(config?.vlm?.model ?? "");
  const [vlmMaxConcurrent, setVlmMaxConcurrent] = useState(
    config?.vlm?.max_concurrent?.toString() ?? "100"
  );
  const [vlmThinking, setVlmThinking] = useState(config?.vlm?.thinking ?? false);
  const [vlmStream, setVlmStream] = useState(config?.vlm?.stream ?? false);

  // Parsers
  const [codeSummaryMode, setCodeSummaryMode] = useState(
    config?.parsers?.code?.code_summary_mode ?? "ast"
  );

  // Rerank
  const [rerankApiBase, setRerankApiBase] = useState(
    config?.rerank?.api_base ?? ""
  );
  const [rerankApiKey, setRerankApiKey] = useState("");
  const [rerankProvider, setRerankProvider] = useState(
    config?.rerank?.provider ?? ""
  );
  const [rerankModel, setRerankModel] = useState(config?.rerank?.model ?? "");
  const [rerankThreshold, setRerankThreshold] = useState(
    config?.rerank?.threshold?.toString() ?? ""
  );

  // 当 config prop 更新时，同步到所有 state
  useEffect(() => {
    if (config) {
      // Storage
      if (config.storage?.workspace !== undefined) setWorkspace(config.storage.workspace);
      if (config.storage?.vectordb?.name !== undefined) setVectordbName(config.storage.vectordb.name);
      if (config.storage?.vectordb?.backend !== undefined) setVectordbBackend(config.storage.vectordb.backend);
      if (config.storage?.agfs?.port !== undefined) setAgfsPort(config.storage.agfs.port.toString());
      if (config.storage?.agfs?.log_level !== undefined) setAgfsLogLevel(config.storage.agfs.log_level);
      if (config.storage?.agfs?.backend !== undefined) setAgfsBackend(config.storage.agfs.backend);

      // Log
      if (config.log?.level !== undefined) setLogLevel(config.log.level);
      if (config.log?.output !== undefined) setLogOutput(config.log.output);

      // Dense Embedding
      if (config.embedding?.dense?.api_base !== undefined) setDenseApiBase(config.embedding.dense.api_base);
      if (config.embedding?.dense?.provider !== undefined) setDenseProvider(config.embedding.dense.provider);
      if (config.embedding?.dense?.dimension !== undefined) setDenseDimension(config.embedding.dense.dimension.toString());
      if (config.embedding?.dense?.model !== undefined) setDenseModel(config.embedding.dense.model);
      if (config.embedding?.dense?.input !== undefined) setDenseInput(config.embedding.dense.input);
      if (config.embedding?.dense?.batch_size !== undefined) setDenseBatchSize(config.embedding.dense.batch_size.toString());
      if (config.embedding?.dense?.query_param !== undefined) setDenseQueryParam(config.embedding.dense.query_param);
      if (config.embedding?.dense?.document_param !== undefined) setDenseDocumentParam(config.embedding.dense.document_param);
      if (config.embedding?.dense?.region !== undefined) setDenseRegion(config.embedding.dense.region);

      // Sparse Embedding
      if (config.embedding?.sparse?.api_base !== undefined) setSparseApiBase(config.embedding.sparse.api_base);
      if (config.embedding?.sparse?.provider !== undefined) setSparseProvider(config.embedding.sparse.provider);
      if (config.embedding?.sparse?.model !== undefined) setSparseModel(config.embedding.sparse.model);

      // Hybrid Embedding
      if (config.embedding?.hybrid?.api_base !== undefined) setHybridApiBase(config.embedding.hybrid.api_base);
      if (config.embedding?.hybrid?.provider !== undefined) setHybridProvider(config.embedding.hybrid.provider);
      if (config.embedding?.hybrid?.model !== undefined) setHybridModel(config.embedding.hybrid.model);
      if (config.embedding?.hybrid?.dimension !== undefined) setHybridDimension(config.embedding.hybrid.dimension.toString());

      if (config.embedding?.max_concurrent !== undefined) setEmbeddingMaxConcurrent(config.embedding.max_concurrent.toString());

      // VLM
      if (config.vlm?.api_base !== undefined) setVlmApiBase(config.vlm.api_base);
      if (config.vlm?.provider !== undefined) setVlmProvider(config.vlm.provider);
      if (config.vlm?.model !== undefined) setVlmModel(config.vlm.model);
      if (config.vlm?.max_concurrent !== undefined) setVlmMaxConcurrent(config.vlm.max_concurrent.toString());
      if (config.vlm?.thinking !== undefined) setVlmThinking(config.vlm.thinking);
      if (config.vlm?.stream !== undefined) setVlmStream(config.vlm.stream);

      // Parsers
      if (config.parsers?.code?.code_summary_mode !== undefined) setCodeSummaryMode(config.parsers.code.code_summary_mode);

      // Rerank
      if (config.rerank?.api_base !== undefined) setRerankApiBase(config.rerank.api_base);
      if (config.rerank?.provider !== undefined) setRerankProvider(config.rerank.provider);
      if (config.rerank?.model !== undefined) setRerankModel(config.rerank.model);
      if (config.rerank?.threshold !== undefined) setRerankThreshold(config.rerank.threshold.toString());
    }
  }, [config]);

  const handleSave = () => {
    const updates: Record<string, unknown> = {
      storage: {
        workspace,
        vectordb: { name: vectordbName, backend: vectordbBackend },
        agfs: {
          port: parseInt(agfsPort, 10) || undefined,
          log_level: agfsLogLevel || undefined,
          backend: agfsBackend || undefined,
        },
      },
      log: { level: logLevel, output: logOutput },
      embedding: {
        dense: {
          api_base: denseApiBase,
          provider: denseProvider,
          dimension: parseInt(denseDimension, 10) || undefined,
          model: denseModel,
          input: denseInput || undefined,
          batch_size: parseInt(denseBatchSize, 10) || undefined,
          query_param: denseQueryParam || undefined,
          document_param: denseDocumentParam || undefined,
          region: denseRegion || undefined,
        },
        sparse: {
          api_base: sparseApiBase,
          provider: sparseProvider,
          model: sparseModel,
        },
        hybrid: {
          api_base: hybridApiBase,
          provider: hybridProvider,
          model: hybridModel,
          dimension: parseInt(hybridDimension, 10) || undefined,
        },
        max_concurrent: parseInt(embeddingMaxConcurrent, 10) || 10,
      },
      vlm: {
        api_base: vlmApiBase,
        provider: vlmProvider,
        model: vlmModel,
        max_concurrent: parseInt(vlmMaxConcurrent, 10) || 100,
        thinking: vlmThinking,
        stream: vlmStream,
      },
      parsers: {
        code: { code_summary_mode: codeSummaryMode },
      },
      rerank: {
        api_base: rerankApiBase,
        provider: rerankProvider,
        model: rerankModel,
        threshold: parseFloat(rerankThreshold) || undefined,
      },
    };

    // 只有当用户输入了新的敏感值时才更新
    if (denseApiKey.trim()) {
      const emb = updates.embedding as { dense?: Record<string, unknown> };
      if (emb.dense) {
        emb.dense = { ...emb.dense, api_key: denseApiKey.trim() };
      }
    }
    if (denseAk.trim() || denseSk.trim()) {
      const emb = updates.embedding as { dense?: Record<string, unknown> };
      if (emb.dense) {
        if (denseAk.trim()) emb.dense.ak = denseAk.trim();
        if (denseSk.trim()) emb.dense.sk = denseSk.trim();
      }
    }
    if (sparseApiKey.trim()) {
      const emb = updates.embedding as { sparse?: Record<string, unknown> };
      if (emb.sparse) {
        emb.sparse = { ...emb.sparse, api_key: sparseApiKey.trim() };
      }
    }
    if (hybridApiKey.trim()) {
      const emb = updates.embedding as { hybrid?: Record<string, unknown> };
      if (emb.hybrid) {
        emb.hybrid = { ...emb.hybrid, api_key: hybridApiKey.trim() };
      }
    }
    if (vlmApiKey.trim()) {
      (updates.vlm as Record<string, unknown>).api_key = vlmApiKey.trim();
    }
    if (rerankApiKey.trim()) {
      (updates.rerank as Record<string, unknown>).api_key = rerankApiKey.trim();
    }

    onSave(updates);
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
          OpenViking 服务配置
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          配置 OpenViking 服务的内部参数（存储在 ov.conf 文件中）
        </p>
      </div>

      {/* Storage */}
      <CollapsibleSection title="存储配置" defaultOpen>
        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            工作空间路径
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            OpenViking 索引数据存储的目录
          </p>
          <div className="mt-2 max-w-md">
            <UiInput
              value={workspace}
              onChange={(e) => setWorkspace(e.target.value)}
              placeholder="./data"
            />
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              VectorDB 名称
            </label>
            <div className="mt-2">
              <UiInput
                value={vectordbName}
                onChange={(e) => setVectordbName(e.target.value)}
                placeholder="context"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              VectorDB 后端
            </label>
            <div className="mt-2">
              <UiSelect
                value={vectordbBackend}
                onChange={(e) => setVectordbBackend(e.target.value)}
              >
                <SelectOptions options={[{ value: "", label: "默认" }, { value: "local", label: "local" }]} />
              </UiSelect>
            </div>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              AGFS 端口
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={agfsPort}
                onChange={(e) => setAgfsPort(e.target.value)}
                placeholder="1833"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              AGFS 日志级别
            </label>
            <div className="mt-2">
              <UiSelect
                value={agfsLogLevel}
                onChange={(e) => setAgfsLogLevel(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "", label: "默认" },
                  { value: "debug", label: "debug" },
                  { value: "info", label: "info" },
                  { value: "warn", label: "warn" },
                  { value: "error", label: "error" },
                ]} />
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              AGFS 后端
            </label>
            <div className="mt-2">
              <UiSelect
                value={agfsBackend}
                onChange={(e) => setAgfsBackend(e.target.value)}
              >
                <SelectOptions options={[{ value: "", label: "默认" }, { value: "local", label: "local" }]} />
              </UiSelect>
            </div>
          </div>
        </div>
      </CollapsibleSection>

      {/* Log */}
      <CollapsibleSection title="日志配置">
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              日志级别
            </label>
            <div className="mt-2">
              <UiSelect
                value={logLevel}
                onChange={(e) => setLogLevel(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "DEBUG", label: "DEBUG" },
                  { value: "INFO", label: "INFO" },
                  { value: "WARN", label: "WARN" },
                  { value: "ERROR", label: "ERROR" },
                ]} />
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              输出目标
            </label>
            <div className="mt-2">
              <UiSelect
                value={logOutput}
                onChange={(e) => setLogOutput(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "stdout", label: "标准输出" },
                  { value: "stderr", label: "标准错误" },
                ]} />
              </UiSelect>
            </div>
          </div>
        </div>
      </CollapsibleSection>

      {/* Dense Embedding */}
      <CollapsibleSection title="Dense Embedding 配置" defaultOpen>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API 地址
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={denseApiBase}
                onChange={(e) => setDenseApiBase(e.target.value)}
                placeholder="https://api.example.com/v1"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {config?.embedding?.dense?.api_key || "未设置"}。输入新值以更新。
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={denseApiKey}
                onChange={(e) => setDenseApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Provider
            </label>
            <div className="mt-2">
              <UiSelect
                value={denseProvider}
                onChange={(e) => setDenseProvider(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "", label: "选择 Provider" },
                  { value: "volcengine", label: "volcengine" },
                  { value: "openai", label: "openai" },
                  { value: "vikingdb", label: "vikingdb" },
                  { value: "jina", label: "jina" },
                  { value: "voyage", label: "voyage" },
                  { value: "minimax", label: "minimax" },
                  { value: "gemini", label: "gemini" },
                ]} />
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Model
            </label>
            <div className="mt-2">
              <UiInput
                value={denseModel}
                onChange={(e) => setDenseModel(e.target.value)}
                placeholder="doubao-embedding-vision-250615"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              向量维度
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={denseDimension}
                onChange={(e) => setDenseDimension(e.target.value)}
                placeholder="1024"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              输入类型
            </label>
            <div className="mt-2">
              <UiSelect
                value={denseInput}
                onChange={(e) => setDenseInput(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "", label: "默认" },
                  { value: "text", label: "text" },
                  { value: "multimodal", label: "multimodal" },
                ]} />
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              批量大小
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={denseBatchSize}
                onChange={(e) => setDenseBatchSize(e.target.value)}
                placeholder="32"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              最大并发数
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={embeddingMaxConcurrent}
                onChange={(e) => setEmbeddingMaxConcurrent(e.target.value)}
                min={1}
              />
            </div>
          </div>
        </div>

        {/* 非对称检索参数 */}
        <div className="mt-4 pt-4 border-t border-[var(--color-border-default)]">
          <p className="text-sm font-medium text-[var(--color-text-secondary)] mb-2">
            非对称检索参数（可选）
          </p>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm text-[var(--color-text-muted)]">
                Query Param
              </label>
              <div className="mt-1">
                <UiInput
                  value={denseQueryParam}
                  onChange={(e) => setDenseQueryParam(e.target.value)}
                  placeholder="RETRIEVAL_QUERY"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm text-[var(--color-text-muted)]">
                Document Param
              </label>
              <div className="mt-1">
                <UiInput
                  value={denseDocumentParam}
                  onChange={(e) => setDenseDocumentParam(e.target.value)}
                  placeholder="RETRIEVAL_DOCUMENT"
                />
              </div>
            </div>
          </div>
        </div>

        {/* VikingDB 专用参数 */}
        {denseProvider === "vikingdb" && (
          <div className="mt-4 pt-4 border-t border-[var(--color-border-default)]">
            <p className="text-sm font-medium text-[var(--color-text-secondary)] mb-2">
              VikingDB 专用参数
            </p>
            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <label className="block text-sm text-[var(--color-text-muted)]">
                  Access Key (AK)
                </label>
                <p className="text-xs text-[var(--color-text-muted)]">
                  当前: {config?.embedding?.dense?.ak || "未设置"}
                </p>
                <div className="mt-1">
                  <UiInput
                    type="password"
                    value={denseAk}
                    onChange={(e) => setDenseAk(e.target.value)}
                    placeholder="AK"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm text-[var(--color-text-muted)]">
                  Secret Key (SK)
                </label>
                <p className="text-xs text-[var(--color-text-muted)]">
                  当前: {config?.embedding?.dense?.sk || "未设置"}
                </p>
                <div className="mt-1">
                  <UiInput
                    type="password"
                    value={denseSk}
                    onChange={(e) => setDenseSk(e.target.value)}
                    placeholder="SK"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm text-[var(--color-text-muted)]">
                  Region
                </label>
                <div className="mt-1">
                  <UiInput
                    value={denseRegion}
                    onChange={(e) => setDenseRegion(e.target.value)}
                    placeholder="cn-beijing"
                  />
                </div>
              </div>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* Sparse Embedding */}
      <CollapsibleSection title="Sparse Embedding 配置">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API 地址
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={sparseApiBase}
                onChange={(e) => setSparseApiBase(e.target.value)}
                placeholder="https://api.example.com/v1"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {config?.embedding?.sparse?.api_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={sparseApiKey}
                onChange={(e) => setSparseApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Provider
            </label>
            <div className="mt-2">
              <UiInput
                value={sparseProvider}
                onChange={(e) => setSparseProvider(e.target.value)}
                placeholder="volcengine"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Model
            </label>
            <div className="mt-2">
              <UiInput
                value={sparseModel}
                onChange={(e) => setSparseModel(e.target.value)}
                placeholder="doubao-embedding-vision-250615"
              />
            </div>
          </div>
        </div>
      </CollapsibleSection>

      {/* Hybrid Embedding */}
      <CollapsibleSection title="Hybrid Embedding 配置">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API 地址
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={hybridApiBase}
                onChange={(e) => setHybridApiBase(e.target.value)}
                placeholder="https://api.example.com/v1"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {config?.embedding?.hybrid?.api_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={hybridApiKey}
                onChange={(e) => setHybridApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Provider
            </label>
            <div className="mt-2">
              <UiInput
                value={hybridProvider}
                onChange={(e) => setHybridProvider(e.target.value)}
                placeholder="volcengine"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Model
            </label>
            <div className="mt-2">
              <UiInput
                value={hybridModel}
                onChange={(e) => setHybridModel(e.target.value)}
                placeholder="doubao-embedding-hybrid"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              向量维度
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={hybridDimension}
                onChange={(e) => setHybridDimension(e.target.value)}
                placeholder="1024"
              />
            </div>
          </div>
        </div>
      </CollapsibleSection>

      {/* VLM */}
      <CollapsibleSection title="VLM 配置" defaultOpen>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API 地址
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={vlmApiBase}
                onChange={(e) => setVlmApiBase(e.target.value)}
                placeholder="https://api.example.com/v1"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {config?.vlm?.api_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={vlmApiKey}
                onChange={(e) => setVlmApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Provider
            </label>
            <div className="mt-2">
              <UiInput
                value={vlmProvider}
                onChange={(e) => setVlmProvider(e.target.value)}
                placeholder="volcengine"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Model
            </label>
            <div className="mt-2">
              <UiInput
                value={vlmModel}
                onChange={(e) => setVlmModel(e.target.value)}
                placeholder="doubao-seed-2-0-pro-260215"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              最大并发数
            </label>
            <div className="mt-2">
              <UiInput
                type="number"
                value={vlmMaxConcurrent}
                onChange={(e) => setVlmMaxConcurrent(e.target.value)}
                min={1}
              />
            </div>
          </div>
          <div className="flex items-center gap-6 pt-6">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="vlm-thinking"
                checked={vlmThinking}
                onChange={(e) => setVlmThinking(e.target.checked)}
                className="h-4 w-4 rounded border-[var(--color-border-default)]"
              />
              <label
                htmlFor="vlm-thinking"
                className="text-sm text-[var(--color-text-secondary)]"
              >
                Thinking 模式
              </label>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="vlm-stream"
                checked={vlmStream}
                onChange={(e) => setVlmStream(e.target.checked)}
                className="h-4 w-4 rounded border-[var(--color-border-default)]"
              />
              <label
                htmlFor="vlm-stream"
                className="text-sm text-[var(--color-text-secondary)]"
              >
                Stream 模式
              </label>
            </div>
          </div>
        </div>
      </CollapsibleSection>

      {/* Parsers */}
      <CollapsibleSection title="解析器配置">
        <div>
          <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
            代码摘要模式
          </label>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            代码文件的摘要生成方式
          </p>
          <div className="mt-2 max-w-xs">
            <UiSelect
              value={codeSummaryMode}
              onChange={(e) => setCodeSummaryMode(e.target.value)}
            >
              <SelectOptions options={[
                { value: "ast", label: "AST（推荐，快速）" },
                { value: "llm", label: "纯 LLM（成本高）" },
                { value: "ast_llm", label: "AST + LLM（质量最高）" },
              ]} />
            </UiSelect>
          </div>
        </div>
      </CollapsibleSection>

      {/* Rerank */}
      <CollapsibleSection title="Rerank 配置">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API 地址
            </label>
            <div className="mt-2 max-w-md">
              <UiInput
                value={rerankApiBase}
                onChange={(e) => setRerankApiBase(e.target.value)}
                placeholder="https://api.example.com/v1/reranks"
              />
            </div>
          </div>
          <div className="sm:col-span-2">
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              API Key
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              当前: {config?.rerank?.api_key || "未设置"}
            </p>
            <div className="mt-2 max-w-md">
              <UiInput
                type="password"
                value={rerankApiKey}
                onChange={(e) => setRerankApiKey(e.target.value)}
                placeholder="输入新的 API Key"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Provider
            </label>
            <div className="mt-2">
              <UiSelect
                value={rerankProvider}
                onChange={(e) => setRerankProvider(e.target.value)}
              >
                <SelectOptions options={[
                  { value: "", label: "选择 Provider" },
                  { value: "volcengine", label: "volcengine" },
                  { value: "openai", label: "openai" },
                ]} />
              </UiSelect>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              Model
            </label>
            <div className="mt-2">
              <UiInput
                value={rerankModel}
                onChange={(e) => setRerankModel(e.target.value)}
                placeholder="doubao-rerank-250615"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)]">
              阈值
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              重排序阈值（可选）
            </p>
            <div className="mt-2">
              <UiInput
                type="number"
                step="0.01"
                value={rerankThreshold}
                onChange={(e) => setRerankThreshold(e.target.value)}
                placeholder="0.1"
              />
            </div>
          </div>
        </div>
      </CollapsibleSection>

      <div className="flex justify-end">
        <UiButton onClick={handleSave} disabled={saving}>
          {saving ? "保存中..." : "保存"}
        </UiButton>
      </div>
    </div>
  );
}