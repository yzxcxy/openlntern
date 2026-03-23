import type { Message as SemiMessage } from "@douyinfe/semi-ui-19/lib/es/aiChatDialogue/interface";
import type { Content as AIChatInputContent } from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import type { ThreadHistoryItem } from "../thread-history-events";

// 聊天页的协议类型和纯转换逻辑集中在这里，页面主体只保留交互和状态编排。

export const TOOL_RESULT_TYPE = "tool_result_text";
export const ACTIVITY_CONTENT_TYPE = "activity_message";
export const PROCESS_PANEL_TYPE = "process_panel";
export const ACTIVITY_EVENT_SNAPSHOT = "ACTIVITY_SNAPSHOT";
export const ACTIVITY_EVENT_DELTA = "ACTIVITY_DELTA";
export const A2UI_SURFACE_ACTIVITY_TYPE = "a2ui-surface";
export const CHAT_ASSISTANT_KEY = "chat";
export const AGENT_ASSISTANT_KEY_PREFIX = "agent:";

export type ChatMessage = SemiMessage & {
  assistantAvatar?: string;
  assistantKey?: string;
  assistantName?: string;
};

export type BackendMessageItem = {
  msg_id: string;
  thread_id: string;
  run_id: string;
  sequence?: number;
  type: string;
  content: string;
  status?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
};

type BackendMessageMetadata = {
  assistant_key?: string;
};

export type BackendMessagePage = {
  data: BackendMessageItem[];
  total: number;
  page?: number;
  size?: number;
};

export type BackendThreadItem = ThreadHistoryItem;

export type ModelCatalogOption = {
  provider_id: string;
  provider_name: string;
  provider_avatar?: string;
  api_type: string;
  model_id: string;
  model_key: string;
  model_name: string;
  model_avatar?: string;
  is_system_default?: boolean;
};

export type SkillCatalogItem = {
  skill_id?: string;
  name?: string;
  path?: string;
};

export type KnowledgeBaseOption = {
  name?: string;
  uri?: string;
};

export type MentionTargetType = "skill" | "kb";
export type MentionTriggerSymbol = "@" | "#";

export type MentionTargetOption = {
  type: MentionTargetType;
  id: string;
  name: string;
  displayName: string;
  keyword: string;
};

export type MentionSelectionItem = {
  type: MentionTargetType;
  id: string;
  name: string;
};

export type UploadAssetKind = "image" | "audio" | "video" | "file";

export type UploadAssetItem = {
  id: string;
  key: string;
  url: string;
  mimeType: string;
  fileName: string;
  size: number;
  mediaKind: UploadAssetKind;
};

export type BackendChatUploadAsset = {
  key?: string;
  url?: string;
  mime_type?: string;
  file_name?: string;
  size?: number;
  media_kind?: UploadAssetKind;
};

type AguiUserContentPart =
  | { type: "text"; text: string }
  | {
      type: "binary";
      mimeType: string;
      id?: string;
      url?: string;
      data?: string;
      filename?: string;
    };

export const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

export const createMessageId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

// assistant_key 只保存稳定身份，便于历史消息在切换 agent 后仍能正确归属。
export const buildAssistantMessageKey = (
  conversationMode: "chat" | "agent",
  agentId?: string
) => {
  const normalizedAgentID = typeof agentId === "string" ? agentId.trim() : "";
  if (conversationMode === "agent" && normalizedAgentID) {
    return `${AGENT_ASSISTANT_KEY_PREFIX}${normalizedAgentID}`;
  }
  return CHAT_ASSISTANT_KEY;
};

export const parseAssistantKey = (value: unknown) => {
  const normalized = typeof value === "string" ? value.trim() : "";
  if (!normalized) {
    return CHAT_ASSISTANT_KEY;
  }
  return normalized;
};

// 基于文字生成头像，避免无图时对话头信息空白
export const buildTextAvatarDataUrl = (
  text: string,
  options?: { background?: string; color?: string }
) => {
  const safeText = Array.from(text || "")
    .filter(Boolean)
    .slice(0, 2)
    .join("");
  const label = safeText || "AI";
  const background = options?.background ?? "#94A3B8";
  const color = options?.color ?? "#FFFFFF";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64">
    <rect width="64" height="64" rx="16" fill="${background}" />
    <text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle"
      font-family="Arial, sans-serif" font-size="24" fill="${color}">${label}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
};

export const joinClasses = (
  ...classes: Array<string | false | null | undefined>
) => classes.filter(Boolean).join(" ");

const isProcessContentItem = (item: unknown) => {
  const type = typeof (item as { type?: unknown })?.type === "string"
    ? String((item as { type?: string }).type)
    : "";
  return type === "reasoning" || type === "function_call" || type === TOOL_RESULT_TYPE;
};

export const groupAssistantProcessItems = (
  content: SemiMessage["content"]
): SemiMessage["content"] => {
  if (!Array.isArray(content)) {
    return content;
  }
  const nextContent: Array<Record<string, any>> = [];
  let processItems: Array<Record<string, any>> = [];
  const flushProcessItems = (options?: { collapseOnOutput?: boolean }) => {
    if (processItems.length === 0) {
      return;
    }
    const panelStatus = processItems.some((item) => item?.status !== "completed")
      ? "in_progress"
      : "completed";
    nextContent.push({
      type: PROCESS_PANEL_TYPE,
      status: panelStatus,
      collapseOnOutput: Boolean(options?.collapseOnOutput),
      items: processItems,
    });
    processItems = [];
  };

  content.forEach((item) => {
    if (!item || typeof item !== "object") {
      flushProcessItems();
      return;
    }
    const normalizedItem = item as Record<string, any>;
    if (isProcessContentItem(normalizedItem)) {
      processItems.push(normalizedItem);
      return;
    }
    // 只有后面跟着真正的助手文本时，才触发外层面板自动收起。
    flushProcessItems({ collapseOnOutput: normalizedItem.type === "message" });
    nextContent.push(normalizedItem);
  });
  flushProcessItems();
  return nextContent;
};

// 后端消息中的 content 为 JSON 字符串，解析失败时返回 null
const safeParseJson = <T,>(value: string): T | null => {
  try {
    return JSON.parse(value) as T;
  } catch {
    return null;
  }
};

// 从不同形态的 content 中提取文本
export const extractText = (content: unknown) => {
  if (typeof content === "string") {
    return content;
  }
  if (Array.isArray(content)) {
    return content
      .map((item) =>
        typeof (item as { text?: unknown }).text === "string"
          ? String((item as { text?: string }).text)
          : ""
      )
      .filter(Boolean)
      .join("");
  }
  return "";
};

// 归一化 MIME 字符串，统一转小写并去掉空白。
export const normalizeMimeType = (value: unknown) =>
  typeof value === "string" ? value.trim().toLowerCase() : "";

// 根据 MIME 推导附件类型，和后端返回保持一致。
export const inferUploadKind = (mimeType: string): UploadAssetKind => {
  if (mimeType.startsWith("image/")) return "image";
  if (mimeType.startsWith("audio/")) return "audio";
  if (mimeType.startsWith("video/")) return "video";
  return "file";
};

// 字节大小格式化为更易读的单位文本。
export const formatFileSize = (size: number) => {
  if (!Number.isFinite(size) || size <= 0) return "";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

// 将输入文本与上传附件组合成 AGUI user content 结构。
export const buildAguiUserContent = (
  text: string,
  uploads: UploadAssetItem[]
): string | AguiUserContentPart[] => {
  const normalizedText = text.trim();
  const normalizedUploads = uploads.filter(
    (item) => item.url && item.mimeType && item.fileName
  );
  if (normalizedUploads.length === 0) {
    return normalizedText;
  }
  const parts: AguiUserContentPart[] = [];
  if (normalizedText) {
    parts.push({ type: "text", text: normalizedText });
  }
  normalizedUploads.forEach((item) => {
    parts.push({
      type: "binary",
      mimeType: item.mimeType,
      url: item.url,
      filename: item.fileName,
    });
  });
  return parts;
};

// 将 AGUI user content 映射为 Semi 可展示的输入内容结构。
export const mapAguiUserContentToSemi = (
  content: unknown
): string | Array<Record<string, any>> => {
  if (typeof content === "string") {
    return content;
  }
  if (!Array.isArray(content)) {
    return "";
  }
  const items: Array<Record<string, any>> = [];
  content.forEach((part) => {
    const entry = part as Record<string, any>;
    if (!entry || typeof entry !== "object") {
      return;
    }
    const partType = String(entry.type ?? "");
    if (partType === "text") {
      const text = typeof entry.text === "string" ? entry.text : "";
      if (text) {
        items.push({ type: "input_text", text });
      }
      return;
    }
    if (partType === "binary") {
      const mimeType = normalizeMimeType(entry.mimeType ?? entry.mime_type);
      const url = typeof entry.url === "string" ? entry.url : "";
      const fileName =
        typeof entry.filename === "string" ? entry.filename : "attachment";
      if (!url) return;
      if (mimeType.startsWith("image/")) {
        items.push({
          type: "input_image",
          image_url: url,
          file_id: fileName,
        });
        return;
      }
      items.push({
        type: "input_file",
        file_url: url,
        filename: fileName,
      });
    }
  });
  if (items.length === 0) {
    return extractText(content);
  }
  return [{ type: "message", content: items }];
};

// tool_result 的 content 可能是 JSON 字符串，优先解析出 text 聚合
export const extractToolResultText = (content: unknown) => {
  if (typeof content === "string") {
    const parsed = safeParseJson<{ content?: Array<{ text?: string }> }>(content);
    if (parsed?.content?.length) {
      return parsed.content
        .map((item) => (typeof item.text === "string" ? item.text : ""))
        .filter(Boolean)
        .join("");
    }
    return content;
  }
  if (content && typeof content === "object") {
    const items = (content as { content?: Array<{ text?: string }> }).content;
    if (Array.isArray(items)) {
      return items
        .map((item) => (typeof item.text === "string" ? item.text : ""))
        .filter(Boolean)
        .join("");
    }
  }
  return "";
};

// 统一提取输入内容中的纯文本，忽略非文本 slot。
export const extractInputPlainText = (contents: AIChatInputContent[]) =>
  contents
    .map((item) =>
      typeof item?.text === "string" && item.type === "text" ? item.text : ""
    )
    .filter(Boolean)
    .join("");

// 转义文本为安全 HTML。
const escapeHtml = (value: string) =>
  value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");

// 将纯文本转换为 AIChatInput 可接受的基础段落 HTML。
export const toEditorParagraphHtml = (value: string) => {
  if (!value) {
    return "";
  }
  return value
    .split("\n")
    .map((line) => `<p>${escapeHtml(line)}</p>`)
    .join("");
};

// 匹配输入结尾的触发器：@ 用于知识库，# 用于 skill。
export const matchMentionTrigger = (
  text: string
): { symbol: MentionTriggerSymbol; keyword: string } | null => {
  const match = text.match(/(?:^|\s)([@#])([^\s@#]*)$/);
  if (!match) {
    return null;
  }
  const symbol = match[1] as MentionTriggerSymbol;
  const keyword = match[2] ?? "";
  return { symbol, keyword };
};

// 去掉输入末尾的 mention 触发串，避免把 @ 关键词留在正文中。
export const stripMentionSuffix = (
  text: string,
  symbol: MentionTriggerSymbol | null
) => {
  if (!symbol) {
    return text;
  }
  const pattern = new RegExp(`(^|\\s)\\${symbol}[^\\s@#]*$`);
  return text.replace(pattern, "$1").replace(/\s+$/, " ");
};

// ISO 时间字符串转为时间戳，解析失败返回 undefined
const toTimestamp = (value?: string) => {
  if (!value) return undefined;
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return undefined;
  return parsed;
};

// 聚合历史消息时保留最早的创建时间，避免 assistant 容器被后续片段覆盖时间线。
const mergeCreatedAt = (current?: number, incoming?: number) => {
  if (current === undefined) return incoming;
  if (incoming === undefined) return current;
  return Math.min(current, incoming);
};

// 历史回放优先使用后端持久化的 sequence；旧数据无该字段时再回退到时间排序。
const toSequence = (value?: number) =>
  typeof value === "number" && Number.isFinite(value) && value > 0 ? value : undefined;

export const toRecord = (value: unknown): Record<string, any> => {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, any>;
  }
  return {};
};

const cloneValue = <T,>(value: T): T => {
  if (typeof structuredClone === "function") {
    try {
      return structuredClone(value);
    } catch {
      // fallback to JSON clone
    }
  }
  try {
    return JSON.parse(JSON.stringify(value)) as T;
  } catch {
    return value;
  }
};

const applyA2UISurfacePatch = (previousContent: unknown, patch: unknown) => {
  const base = toRecord(cloneValue(previousContent));
  const operations = Array.isArray(base.operations) ? [...base.operations] : [];
  const patchItems = Array.isArray(patch) ? patch : [];
  patchItems.forEach((item) => {
    if (!item || typeof item !== "object") return;
    const op = String((item as { op?: unknown }).op ?? "");
    const path = String((item as { path?: unknown }).path ?? "");
    const value = (item as { value?: unknown }).value;

    if (op === "add" && path === "/operations/-") {
      operations.push(value);
      return;
    }

    const match = path.match(/^\/operations\/(\d+)$/);
    if (!match) return;
    const index = Number(match[1]);
    if (!Number.isInteger(index) || index < 0) return;

    if (op === "add" || op === "replace") {
      operations[index] = value;
      return;
    }
    if (op === "remove") {
      operations.splice(index, 1);
    }
  });
  return {
    ...base,
    operations,
  };
};

export const mapActivityContent = (params: {
  activityType?: string;
  eventType?: string;
  content?: unknown;
  patch?: unknown;
  replace?: boolean;
  previousContent?: unknown;
}) => {
  const {
    activityType,
    eventType,
    content,
    patch,
    replace,
    previousContent,
  } = params;
  const normalizedEventType =
    eventType === ACTIVITY_EVENT_DELTA
      ? ACTIVITY_EVENT_DELTA
      : ACTIVITY_EVENT_SNAPSHOT;

  if (normalizedEventType === ACTIVITY_EVENT_DELTA) {
    if (activityType === A2UI_SURFACE_ACTIVITY_TYPE) {
      return applyA2UISurfacePatch(previousContent, patch);
    }
    return {
      ...toRecord(previousContent),
      patch: Array.isArray(patch) ? patch : [],
    };
  }

  const nextContent = toRecord(content);
  if (replace || !previousContent) {
    return nextContent;
  }
  return {
    ...toRecord(previousContent),
    ...nextContent,
  };
};

// 历史消息：按时间排序后聚合到 SemiMessage[]
export const mapHistoryMessages = (items: BackendMessageItem[]) => {
  const sorted = items
    .map((item, index) => ({ item, index }))
    .sort((left, right) => {
      const leftSequence = toSequence(left.item.sequence);
      const rightSequence = toSequence(right.item.sequence);
      if (leftSequence !== undefined && rightSequence !== undefined) {
        return leftSequence - rightSequence;
      }
      // 历史时间线以 created_at 为准，避免 updated_at 把旧消息重新挪位。
      const leftTime =
        toTimestamp(left.item.created_at) ?? toTimestamp(left.item.updated_at) ?? 0;
      const rightTime =
        toTimestamp(right.item.created_at) ?? toTimestamp(right.item.updated_at) ?? 0;
      if (leftTime !== rightTime) {
        return leftTime - rightTime;
      }
      // 后端列表接口按倒序返回；同时间戳时反转原始索引，恢复正向时间线。
      return right.index - left.index;
    });
  const result: ChatMessage[] = [];
  const runIndexMap = new Map<string, number>();
  const messageOrderKeys: number[] = [];
  const runAssistantKeyMap = new Map<string, string>();

  const resolveAssistantKey = (item: BackendMessageItem) => {
    const metadata = safeParseJson<BackendMessageMetadata>(item.metadata || "");
    const assistantKey = parseAssistantKey(metadata?.assistant_key);
    if (item.run_id && assistantKey) {
      runAssistantKeyMap.set(item.run_id, assistantKey);
    }
    return item.run_id
      ? runAssistantKeyMap.get(item.run_id) ?? assistantKey
      : assistantKey;
  };

  const ensureAssistantMessage = (
    runId: string,
    assistantKey: string,
    status?: string,
    createdAt?: number,
    updatedAt?: number,
    orderKey?: number
  ) => {
    let runIndex = runIndexMap.get(runId);
    if (runIndex === undefined) {
      result.push({
        id: runId,
        role: "assistant",
        assistantKey,
        content: [],
        status,
        createdAt,
        updatedAt,
      });
      runIndex = result.length - 1;
      runIndexMap.set(runId, runIndex);
      messageOrderKeys[runIndex] = orderKey ?? Number.MAX_SAFE_INTEGER;
    } else if (orderKey !== undefined) {
      messageOrderKeys[runIndex] = Math.min(
        messageOrderKeys[runIndex] ?? Number.MAX_SAFE_INTEGER,
        orderKey
      );
    }
    const existingMessage = result[runIndex];
    if (assistantKey && existingMessage && !existingMessage.assistantKey) {
      result[runIndex] = {
        ...existingMessage,
        assistantKey,
      };
    }
    return runIndex;
  };

  sorted.forEach(({ item, index }) => {
    const aguiMessage = safeParseJson<any>(item.content);
    const role = aguiMessage?.role;
    const assistantKey = resolveAssistantKey(item);
    const createdAt =
      toTimestamp(item.created_at) ?? toTimestamp(item.updated_at);
    const updatedAt =
      toTimestamp(item.updated_at) ?? toTimestamp(item.created_at);

    // user 消息直接映射为独立的 SemiMessage
    if (role === "user") {
      result.push({
        id: item.msg_id,
        role: "user",
        assistantKey,
        content: mapAguiUserContentToSemi(aguiMessage?.content),
        status: item.status,
        createdAt,
        updatedAt,
      });
      messageOrderKeys.push(toSequence(item.sequence) ?? index);
      return;
    }

    // activity 消息归并到 assistant.content 中，交由 A2UI 渲染
    if (role === "activity" || aguiMessage?.activityType) {
      const activityType =
        typeof aguiMessage?.activityType === "string"
          ? aguiMessage.activityType
          : "";
      if (!activityType) return;
      const activityEventType =
        aguiMessage?.type === ACTIVITY_EVENT_DELTA
          ? ACTIVITY_EVENT_DELTA
          : ACTIVITY_EVENT_SNAPSHOT;
      const runId = item.run_id || `history-activity-${item.msg_id}`;
      const runIndex = ensureAssistantMessage(
        runId,
        assistantKey,
        item.status,
        createdAt,
        updatedAt,
        toSequence(item.sequence) ?? index
      );
      const target = result[runIndex] as ChatMessage;
      const content = Array.isArray(target.content) ? [...target.content] : [];
      const existingIndex = content.findIndex(
        (contentItem) =>
          (contentItem as { type?: string }).type === ACTIVITY_CONTENT_TYPE &&
          (contentItem as { activityMessageId?: string }).activityMessageId ===
            item.msg_id
      );
      const previousItem =
        existingIndex === -1 ? null : (content[existingIndex] as Record<string, any>);
      const mappedContent = mapActivityContent({
        activityType,
        eventType: activityEventType,
        content: aguiMessage?.content,
        patch: aguiMessage?.patch,
        replace: aguiMessage?.replace,
        previousContent: previousItem?.content,
      });
      const activityItem = {
        ...(previousItem ?? {}),
        id: item.msg_id,
        type: ACTIVITY_CONTENT_TYPE,
        activityMessageId: item.msg_id,
        activityType,
        activityEventType,
        content: mappedContent,
        status: item.status ?? "completed",
        timestamp: createdAt ?? updatedAt ?? Date.now(),
      };
      if (existingIndex === -1) {
        content.push(activityItem);
      } else {
        content[existingIndex] = activityItem;
      }
      result[runIndex] = {
        ...target,
        content,
        createdAt: mergeCreatedAt(target.createdAt, createdAt),
        updatedAt: updatedAt ?? target.updatedAt,
      } as ChatMessage;
      return;
    }

    // 其他消息按 run_id 归并到同一个 assistant SemiMessage
    if (!item.run_id) {
      return;
    }

    const runIndex = ensureAssistantMessage(
      item.run_id,
      assistantKey,
      item.status,
      createdAt,
      updatedAt,
      toSequence(item.sequence) ?? index
    );
    const target = result[runIndex] as ChatMessage;
    const content = Array.isArray(target.content) ? target.content : [];
    const nextContent = [...content];

    // text -> output_text
    if (item.type === "text") {
      const text = extractText(aguiMessage?.content);
      if (text) {
        nextContent.push({
          type: "message",
          content: [{ type: "output_text", text }],
          status: item.status ?? "completed",
        });
      }
    }

    // tool_call -> function_call
    if (item.type === "tool_call") {
      const toolCall = aguiMessage?.toolCalls?.[0];
      if (toolCall) {
        nextContent.push({
          type: "function_call",
          id: toolCall.id,
          call_id: toolCall.id,
          name: toolCall.function?.name,
          status: item.status ?? "completed",
          arguments: toolCall.function?.arguments ?? "",
        });
      }
    }

    if (item.type === "tool_result") {
      const text = extractToolResultText(aguiMessage?.content);
      if (text) {
        nextContent.push({
          type: TOOL_RESULT_TYPE,
          id: item.msg_id,
          text,
          status: item.status ?? "completed",
        });
      }
    }

    // reasoning -> summary_text
    if (item.type === "reasoning" || item.type === "reasoning_message") {
      const text = extractText(aguiMessage?.content);
      nextContent.push({
        type: "reasoning",
        id: item.msg_id,
        status: item.status ?? "completed",
        summary: text ? [{ type: "summary_text", text }] : [],
        encryptedValue: aguiMessage?.encryptedValue,
      });
    }

    result[runIndex] = {
      ...target,
      content: nextContent,
      createdAt: mergeCreatedAt(target.createdAt, createdAt),
      updatedAt: updatedAt ?? target.updatedAt,
    } as ChatMessage;
  });

  return result
    .map((message, index) => ({
      message,
      orderKey: messageOrderKeys[index] ?? Number.MAX_SAFE_INTEGER,
      timestamp:
        message.createdAt ?? message.updatedAt ?? Number.MAX_SAFE_INTEGER,
    }))
    .sort((left, right) => {
      if (left.orderKey !== right.orderKey) {
        return left.orderKey - right.orderKey;
      }
      return left.timestamp - right.timestamp;
    })
    .map(({ message }) => message);
};
