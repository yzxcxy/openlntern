"use client";

import { HttpAgent, type AgentSubscriber, type Message as AguiMessage } from "@ag-ui/client";
import type { RouterLike } from "../auth";
import { buildAuthHeaders, fetchBackend, readValidToken, requestBackend } from "../auth";

export type AgentStatus = "draft" | "enabled" | "disabled";
export type AgentType = "single" | "supervisor";

export type AgentListItem = {
  agent_id: string;
  owner_id: string;
  name: string;
  description: string;
  agent_type: AgentType;
  status: AgentStatus;
  avatar_url: string;
  default_model_id: string;
  default_model_name: string;
  agent_memory_enabled: boolean;
  tool_count: number;
  skill_count: number;
  knowledge_base_count: number;
  sub_agent_count: number;
  created_at: string;
  updated_at: string;
};

export type AgentDetail = AgentListItem & {
  system_prompt: string;
  chat_background_json: string;
  example_questions: string[];
  tool_ids: string[];
  skill_names: string[];
  knowledge_base_names: string[];
  sub_agent_ids: string[];
};

export type EnabledAgentOption = {
  agent_id: string;
  name: string;
  description: string;
  agent_type: AgentType;
  status: AgentStatus;
  avatar_url: string;
  default_model_id: string;
  default_model_name: string;
};

export type AgentPayload = {
  name: string;
  description: string;
  agent_type: AgentType;
  system_prompt: string;
  avatar_url: string;
  chat_background_json: string;
  example_questions: string[];
  default_model_id: string;
  agent_memory_enabled: boolean;
  tool_ids: string[];
  skill_names: string[];
  knowledge_base_names: string[];
  sub_agent_ids: string[];
};

export type AgentDebugMessage = {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
};

export type AgentPage<T> = {
  data: T[];
  total: number;
  page?: number;
  size?: number;
};

export type ModelCatalogOption = {
  model_id: string;
  model_name: string;
  provider_name?: string;
};

export type PluginOption = {
  plugin_id: string;
  name: string;
  tools?: Array<{
    tool_id?: string;
    tool_name?: string;
    description?: string;
  }>;
};

export type SkillMetaOption = {
  skillName?: string;
  name?: string;
  path?: string;
};

export type KnowledgeBaseOption = {
  name: string;
};

export type UploadedImageAsset = {
  key: string;
  url: string;
  mimeType: string;
  fileName: string;
};

type BackendChatUploadAsset = {
  key?: string;
  url?: string;
  mime_type?: string;
  file_name?: string;
};

type RequestContext = {
  router: RouterLike;
  userId?: string;
};

export const listAgents = async (
  query: URLSearchParams,
  ctx: RequestContext
) =>
  requestBackend<AgentPage<AgentListItem>>(`/v1/agents?${query.toString()}`, {
    fallbackMessage: "获取 Agent 列表失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const getAgent = async (agentId: string, ctx: RequestContext) =>
  requestBackend<AgentDetail>(`/v1/agents/${agentId}`, {
    fallbackMessage: "获取 Agent 详情失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const createAgent = async (payload: AgentPayload, ctx: RequestContext) =>
  requestBackend<AgentDetail>("/v1/agents", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    fallbackMessage: "创建 Agent 失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const updateAgent = async (
  agentId: string,
  payload: AgentPayload,
  ctx: RequestContext
) =>
  requestBackend<AgentDetail>(`/v1/agents/${agentId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    fallbackMessage: "更新 Agent 失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const enableAgent = async (agentId: string, ctx: RequestContext) =>
  requestBackend<AgentDetail>(`/v1/agents/${agentId}/enable`, {
    method: "POST",
    fallbackMessage: "启用 Agent 失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const disableAgent = async (agentId: string, ctx: RequestContext) =>
  requestBackend<AgentDetail>(`/v1/agents/${agentId}/disable`, {
    method: "POST",
    fallbackMessage: "停用 Agent 失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const removeAgent = async (agentId: string, ctx: RequestContext) =>
  requestBackend(`/v1/agents/${agentId}`, {
    method: "DELETE",
    fallbackMessage: "删除 Agent 失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const listEnabledAgentOptions = async (ctx: RequestContext) =>
  requestBackend<EnabledAgentOption[]>("/v1/agents/enabled-options", {
    fallbackMessage: "获取 Agent 候选失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const listModelOptions = async (ctx: RequestContext) =>
  requestBackend<ModelCatalogOption[]>("/v1/models/catalog", {
    fallbackMessage: "获取模型列表失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const listPluginOptions = async (ctx: RequestContext) =>
  requestBackend<PluginOption[]>("/v1/plugins/available-for-chat", {
    fallbackMessage: "获取工具列表失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const listSkillOptions = async (ctx: RequestContext) =>
  requestBackend<AgentPage<SkillMetaOption>>("/v1/skills/meta?page=1&page_size=500", {
    fallbackMessage: "获取 Skill 列表失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const listKnowledgeBaseOptions = async (ctx: RequestContext) =>
  requestBackend<KnowledgeBaseOption[]>("/v1/kbs", {
    fallbackMessage: "获取知识库列表失败",
    router: ctx.router,
    userId: ctx.userId,
  });

export const runAgentDebugSession = async (
  definition: AgentPayload,
  messages: AgentDebugMessage[],
  ctx: RequestContext,
  subscriber?: AgentSubscriber
) => {
  const token = readValidToken(ctx.router);
  if (!token) {
    throw new Error("未登录");
  }
  const agent = new HttpAgent({
    url: "/api/backend/v1/agents/debug/sse",
    headers: buildAuthHeaders(token, ctx.userId),
  });
  agent.setMessages(messages as AguiMessage[]);
  return agent.runAgent(
    {
      forwardedProps: {
        debugDefinition: definition,
      },
    },
    subscriber
  );
};

export const uploadAgentImage = async (file: File, ctx: RequestContext): Promise<UploadedImageAsset> => {
  if (!file.type.startsWith("image/")) {
    throw new Error("仅支持上传图片");
  }
  const formData = new FormData();
  formData.append("file", file);
  const response = await fetchBackend("/v1/chat/uploads", {
    method: "POST",
    body: formData,
    fallbackMessage: `上传失败：${file.name}`,
    router: ctx.router,
    userId: ctx.userId,
  });
  const payload = (await response.json().catch(() => null)) as
    | {
        code?: number;
        message?: string;
        data?: BackendChatUploadAsset;
      }
    | null;
  if (!response.ok || payload?.code !== 0 || !payload.data?.url) {
    throw new Error(payload?.message || `上传失败：${file.name}`);
  }
  return {
    key: payload.data.key || file.name,
    url: String(payload.data.url),
    mimeType: payload.data.mime_type || file.type || "image/png",
    fileName: payload.data.file_name || file.name,
  };
};
