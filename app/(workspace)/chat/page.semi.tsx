"use client";

import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import {
  CopilotKitProvider,
  UseAgentUpdate,
  useAgent,
  useRenderActivityMessage,
} from "@copilotkit/react-core/v2";
import {
  AIChatDialogue,
  AIChatInput,
  Collapsible,
  MarkdownRender,
} from "@douyinfe/semi-ui-19";
import {
  IconChevronDown,
  IconChevronUp,
  IconWrench,
} from "@douyinfe/semi-icons";
import type { Message as SemiMessage } from "@douyinfe/semi-ui-19/lib/es/aiChatDialogue/interface";
import type {
  ActivityMessage,
  Message as AguiMessage,
} from "@ag-ui/client";
import type {
  Content as AIChatInputContent,
  MessageContent,
} from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import { ChatModeConfigureArea } from "./ChatModeConfigureArea";
import { PluginSelectionModal } from "./PluginSelectionModal";
import {
  KNOWN_PLUGIN_RUNTIME_TYPES,
  MAX_SELECTED_TOOLS,
  PLUGIN_PAGE_SIZE,
  PLUGIN_PAGE_SIZE_OPTIONS,
  collectToolIdsFromPlugins,
  getChatPluginKey,
  getPluginSourceFilterValue,
  normalizePluginRuntimeType,
  sanitizeDescriptionText,
  uniqueStringList,
  type ChatPluginOption,
} from "./chat-plugin-config";
import { theme } from "../../theme";
import { UiSelect } from "../../components/ui/UiSelect";
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
  type StoredUser,
} from "../auth";
import {
  dispatchThreadHistoryUpsert,
  type ThreadHistoryItem,
} from "../thread-history-events";

const A2UI_MESSAGE_RENDERER = createA2UIMessageRenderer({ theme });
// 需要稳定引用，避免 renderActivityMessages 触发不稳定数组报错
const ACTIVITY_RENDERERS = [A2UI_MESSAGE_RENDERER];
const TOOL_RESULT_TYPE = "tool_result_text";
const ACTIVITY_CONTENT_TYPE = "activity_message";
const ACTIVITY_EVENT_SNAPSHOT = "ACTIVITY_SNAPSHOT";
const ACTIVITY_EVENT_DELTA = "ACTIVITY_DELTA";
const A2UI_SURFACE_ACTIVITY_TYPE = "a2ui-surface";

type ToolResultCollapseProps = {
  text: string;
};

function ToolResultCollapse({ text }: ToolResultCollapseProps) {
  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        toggleOpen();
      }
    },
    [toggleOpen]
  );

  return (
    <div className="motion-safe-slide-up">
      <div
        className="semi-ai-chat-dialogue-content-tool-call motion-safe-highlight"
        onClick={toggleOpen}
        role="button"
        tabIndex={0}
        onKeyDown={handleKeyDown}
      >
        <IconWrench />
        <span>工具执行结果</span>
        {isOpen ? <IconChevronUp /> : <IconChevronDown />}
      </div>
      <Collapsible isOpen={isOpen}>
        <div className="semi-ai-chat-dialogue-content-bubble px-3 py-3">
          <MarkdownRender format="md" raw={text} />
        </div>
      </Collapsible>
    </div>
  );
}

const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const createMessageId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

// 基于文字生成头像，避免无图时对话头信息空白
const buildTextAvatarDataUrl = (
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

type BackendMessageItem = {
  msg_id: string;
  thread_id: string;
  run_id: string;
  type: string;
  content: string;
  status?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
};

type BackendMessagePage = {
  data: BackendMessageItem[];
  total: number;
  page?: number;
  size?: number;
};

type BackendResult<T> = {
  code: number;
  message: string;
  data?: T;
};

type BackendThreadItem = ThreadHistoryItem;

type ModelCatalogOption = {
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

type SkillCatalogItem = {
  skill_id?: string;
  name?: string;
  path?: string;
};

type KnowledgeBaseOption = {
  name?: string;
  uri?: string;
};

type MentionTargetType = "skill" | "kb";
type MentionTriggerSymbol = "@" | "#";

type MentionTargetOption = {
  type: MentionTargetType;
  id: string;
  name: string;
  displayName: string;
  keyword: string;
};

type MentionSelectionItem = {
  type: MentionTargetType;
  id: string;
  name: string;
};

type UploadAssetKind = "image" | "audio" | "video" | "file";

type UploadAssetItem = {
  id: string;
  key: string;
  url: string;
  mimeType: string;
  fileName: string;
  size: number;
  mediaKind: UploadAssetKind;
};

type BackendChatUploadAsset = {
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

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

// 后端消息中的 content 为 JSON 字符串，解析失败时返回 null
const safeParseJson = <T,>(value: string): T | null => {
  try {
    return JSON.parse(value) as T;
  } catch {
    return null;
  }
};

// 从不同形态的 content 中提取文本
const extractText = (content: unknown) => {
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
const normalizeMimeType = (value: unknown) =>
  typeof value === "string" ? value.trim().toLowerCase() : "";

// 根据 MIME 推导附件类型，和后端返回保持一致。
const inferUploadKind = (mimeType: string): UploadAssetKind => {
  if (mimeType.startsWith("image/")) return "image";
  if (mimeType.startsWith("audio/")) return "audio";
  if (mimeType.startsWith("video/")) return "video";
  return "file";
};

// 字节大小格式化为更易读的单位文本。
const formatFileSize = (size: number) => {
  if (!Number.isFinite(size) || size <= 0) return "";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

// 将输入文本与上传附件组合成 AGUI user content 结构。
const buildAguiUserContent = (
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
const mapAguiUserContentToSemi = (content: unknown): string | Array<Record<string, any>> => {
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
const extractToolResultText = (content: unknown) => {
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
const extractInputPlainText = (contents: AIChatInputContent[]) =>
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
const toEditorParagraphHtml = (value: string) => {
  if (!value) {
    return "";
  }
  return value
    .split("\n")
    .map((line) => `<p>${escapeHtml(line)}</p>`)
    .join("");
};

// 匹配输入结尾的触发器：@ 用于知识库，# 用于 skill。
const matchMentionTrigger = (
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
const stripMentionSuffix = (text: string, symbol: MentionTriggerSymbol | null) => {
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

const toRecord = (value: unknown): Record<string, any> => {
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

const mapActivityContent = (params: {
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
const mapHistoryMessages = (items: BackendMessageItem[]) => {
  const sorted = [...items].sort((a, b) => {
    const aTime = toTimestamp(a.updated_at) ?? toTimestamp(a.created_at) ?? 0;
    const bTime = toTimestamp(b.updated_at) ?? toTimestamp(b.created_at) ?? 0;
    return aTime - bTime;
  });
  const result: SemiMessage[] = [];
  const runIndexMap = new Map<string, number>();

  const ensureAssistantMessage = (
    runId: string,
    status?: string,
    createdAt?: number,
    updatedAt?: number
  ) => {
    let runIndex = runIndexMap.get(runId);
    if (runIndex === undefined) {
      result.push({
        id: runId,
        role: "assistant",
        content: [],
        status,
        createdAt,
        updatedAt,
      });
      runIndex = result.length - 1;
      runIndexMap.set(runId, runIndex);
    }
    return runIndex;
  };

  sorted.forEach((item) => {
    const aguiMessage = safeParseJson<any>(item.content);
    const role = aguiMessage?.role;
    const createdAt =
      toTimestamp(item.created_at) ?? toTimestamp(item.updated_at);
    const updatedAt =
      toTimestamp(item.updated_at) ?? toTimestamp(item.created_at);

    // user 消息直接映射为独立的 SemiMessage
    if (role === "user") {
      result.push({
        id: item.msg_id,
        role: "user",
        content: mapAguiUserContentToSemi(aguiMessage?.content),
        status: item.status,
        createdAt,
        updatedAt,
      });
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
      const runIndex = ensureAssistantMessage(runId, item.status, createdAt, updatedAt);
      const target = result[runIndex] as SemiMessage;
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
        updatedAt: updatedAt ?? target.updatedAt,
      } as SemiMessage;
      return;
    }

    // 其他消息按 run_id 归并到同一个 assistant SemiMessage
    if (!item.run_id) {
      return;
    }

    const runIndex = ensureAssistantMessage(
      item.run_id,
      item.status,
      createdAt,
      updatedAt
    );
    const target = result[runIndex] as SemiMessage;
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
        status: item.status ?? "completed",
        summary: text ? [{ type: "summary_text", text }] : [],
        encryptedValue: aguiMessage?.encryptedValue,
      });
    }

    result[runIndex] = {
      ...target,
      content: nextContent,
      updatedAt: updatedAt ?? target.updatedAt,
    } as SemiMessage;
  });

  return result;
};

type ChatContentProps = {
  token: string;
  userId: string;
  userName: string;
  userAvatar: string;
};

function ChatContent({ token, userId, userName, userAvatar }: ChatContentProps) {
  const searchParams = useSearchParams();
  const inputRef = useRef<{ setContent: (content: string) => void } | null>(null);
  const uploadInputRef = useRef<HTMLInputElement | null>(null);
  const fallbackThreadIdRef = useRef<string>("");
  const currentRunIdRef = useRef<string>("");
  const titleSyncTokenRef = useRef(0);
  const titleSyncTimerRef = useRef<number | null>(null);
  const textMessageMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const toolCallMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const reasoningMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );
  const activityMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );
  const { agent } = useAgent({
    updates: [
      UseAgentUpdate.OnRunStatusChanged,
      UseAgentUpdate.OnStateChanged,
    ],
  });
  const { renderActivityMessage } = useRenderActivityMessage();
  const [inputError, setInputError] = useState("");
  const [threadId, setThreadId] = useState("");
  const [semiMessages, setSemiMessages] = useState<SemiMessage[]>([]);
  const [conversationMode, setConversationMode] = useState<"chat" | "agent">("chat");
  const [availableModels, setAvailableModels] = useState<ModelCatalogOption[]>([]);
  const [selectedProviderId, setSelectedProviderId] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");
  const [pluginMode, setPluginMode] = useState<"select" | "search">("select");
  const [availablePlugins, setAvailablePlugins] = useState<ChatPluginOption[]>([]);
  const [selectedToolIds, setSelectedToolIds] = useState<string[]>([]);
  const [defaultToolIds, setDefaultToolIds] = useState<string[]>([]);
  const [pluginPanelOpen, setPluginPanelOpen] = useState(false);
  const [pluginSearchKeyword, setPluginSearchKeyword] = useState("");
  const [pluginSourceFilter, setPluginSourceFilter] = useState("all");
  const [pluginTypeFilter, setPluginTypeFilter] = useState("all");
  const [pluginPage, setPluginPage] = useState(1);
  const [pluginPageSize, setPluginPageSize] = useState(PLUGIN_PAGE_SIZE);
  const [expandedPluginKeys, setExpandedPluginKeys] = useState<string[]>([]);
  const [pluginLoading, setPluginLoading] = useState(false);
  const [pluginError, setPluginError] = useState("");
  const pluginSelectionInitializedRef = useRef(false);
  const [mentionOptions, setMentionOptions] = useState<MentionTargetOption[]>([]);
  const [selectedMentions, setSelectedMentions] = useState<MentionSelectionItem[]>([]);
  const [mentionKeyword, setMentionKeyword] = useState("");
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionActiveIndex, setMentionActiveIndex] = useState(0);
  const [mentionTriggerSymbol, setMentionTriggerSymbol] =
    useState<MentionTriggerSymbol | null>(null);
  const [pendingUploads, setPendingUploads] = useState<UploadAssetItem[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState("");
  const [, setComposerTextValue] = useState("");

  const clearThreadTitleSync = useCallback(() => {
    titleSyncTokenRef.current += 1;
    if (titleSyncTimerRef.current !== null) {
      window.clearTimeout(titleSyncTimerRef.current);
      titleSyncTimerRef.current = null;
    }
  }, []);

  const startThreadTitleSync = useCallback(
    (targetThreadId: string) => {
      if (!targetThreadId || !token) {
        return;
      }

      clearThreadTitleSync();
      const syncToken = titleSyncTokenRef.current;

      const pollTitle = async (attempt: number) => {
        if (titleSyncTokenRef.current !== syncToken) {
          return;
        }

        try {
          const response = await fetch(`/api/backend/v1/threads/${targetThreadId}`, {
            headers: buildAuthHeaders(token, userId),
          });
          updateTokenFromResponse(response);
          const data = (await response
            .json()
            .catch(() => null)) as BackendResult<BackendThreadItem> | null;

          if (response.ok && data?.code === 0) {
            const thread = data.data ?? {};
            const title =
              typeof thread.title === "string" ? thread.title.trim() : "";

            dispatchThreadHistoryUpsert({
              thread_id: targetThreadId,
              title,
              created_at: thread.created_at,
              updated_at: thread.updated_at,
              pending_title: !title,
            });

            if (title) {
              titleSyncTimerRef.current = null;
              return;
            }
          }
        } catch {
          // 标题同步失败不阻断主对话流程，按下一个轮询周期继续尝试。
        }

        if (titleSyncTokenRef.current !== syncToken) {
          return;
        }

        const delay = Math.min(1000 + attempt * 500, 5000);
        titleSyncTimerRef.current = window.setTimeout(() => {
          void pollTitle(attempt + 1);
        }, delay);
      };

      void pollTitle(0);
    },
    [clearThreadTitleSync, token, userId]
  );

  useEffect(() => () => clearThreadTitleSync(), [clearThreadTitleSync]);

  useEffect(() => {
    let active = true;
    const loadModelCatalog = async () => {
      if (!token) return;
      try {
        const response = await fetch("/api/backend/v1/models/catalog", {
          headers: buildAuthHeaders(token, userId),
        });
        updateTokenFromResponse(response);
        const data = (await response
          .json()
          .catch(() => null)) as BackendResult<ModelCatalogOption[]> | null;
        if (!response.ok || !data || data.code !== 0) {
          if (!active) return;
          setAvailableModels([]);
          setSelectedModelId("");
          setSelectedProviderId("");
          return;
        }
        const nextItems = Array.isArray(data.data) ? data.data : [];
        if (!active) return;
        setAvailableModels(nextItems);
        const defaultItem =
          nextItems.find((item) => item.is_system_default) ?? nextItems[0] ?? null;
        setSelectedModelId((prev) => {
          const matched = nextItems.find((item) => item.model_id === prev);
          return matched?.model_id ?? defaultItem?.model_id ?? "";
        });
        setSelectedProviderId(defaultItem?.provider_id ?? "");
      } catch {
        if (!active) return;
        setAvailableModels([]);
        setSelectedModelId("");
        setSelectedProviderId("");
      }
    };
    loadModelCatalog();
    return () => {
      active = false;
    };
  }, [token, userId]);

  useEffect(() => {
    let active = true;
    const loadMentionOptions = async () => {
      if (!token) return;
      try {
        const [skillsResponse, kbsResponse] = await Promise.all([
          fetch("/api/backend/v1/skills/meta?page=1&page_size=500", {
            headers: buildAuthHeaders(token, userId),
          }),
          fetch("/api/backend/v1/kbs", {
            headers: buildAuthHeaders(token, userId),
          }),
        ]);
        updateTokenFromResponse(skillsResponse);
        updateTokenFromResponse(kbsResponse);

        const skillsData = (await skillsResponse
          .json()
          .catch(() => null)) as BackendResult<{
          data?: SkillCatalogItem[];
        }> | null;
        const kbsData = (await kbsResponse
          .json()
          .catch(() => null)) as BackendResult<KnowledgeBaseOption[]> | null;

        const options: MentionTargetOption[] = [];
        if (skillsResponse.ok && skillsData?.code === 0) {
          const skillItems = Array.isArray(skillsData.data?.data)
            ? skillsData.data?.data
            : [];
          skillItems.forEach((item) => {
            const pathName =
              typeof item.path === "string" && item.path.includes("/")
                ? item.path.split("/").filter(Boolean).pop()
                : item.path;
            const name =
              (typeof item.name === "string" && item.name.trim()) ||
              (typeof pathName === "string" && pathName.trim()) ||
              "";
            const id =
              (typeof item.skill_id === "string" && item.skill_id.trim()) ||
              name;
            if (!id || !name) {
              return;
            }
            options.push({
              type: "skill",
              id,
              name,
              displayName: name,
              keyword: `${id} ${name}`.toLowerCase(),
            });
          });
        }
        if (kbsResponse.ok && kbsData?.code === 0) {
          const kbItems = Array.isArray(kbsData.data) ? kbsData.data : [];
          kbItems.forEach((item) => {
            const name =
              typeof item.name === "string" ? item.name.trim() : "";
            if (!name) {
              return;
            }
            options.push({
              type: "kb",
              id: name,
              name,
              displayName: name,
              keyword: name.toLowerCase(),
            });
          });
        }
        if (!active) return;
        const deduped = options.filter((item, index, list) => {
          const key = `${item.type}:${item.id}`;
          return list.findIndex((option) => `${option.type}:${option.id}` === key) === index;
        });
        setMentionOptions(deduped);
      } catch {
        if (!active) return;
        setMentionOptions([]);
      }
    };
    void loadMentionOptions();
    return () => {
      active = false;
    };
  }, [token, userId]);

  useEffect(() => {
    if (!availableModels.length) {
      setSelectedModelId("");
      setSelectedProviderId("");
      return;
    }
    const selected =
      availableModels.find((item) => item.model_id === selectedModelId) ??
      availableModels.find((item) => item.is_system_default) ??
      availableModels[0];
    if (!selected) {
      return;
    }
    if (selected.model_id !== selectedModelId) {
      setSelectedModelId(selected.model_id);
    }
    if (selected.provider_id !== selectedProviderId) {
      setSelectedProviderId(selected.provider_id);
    }
  }, [availableModels, selectedModelId, selectedProviderId]);

  useEffect(() => {
    let active = true;
    const loadAvailablePlugins = async () => {
      if (!token) return;
      setPluginLoading(true);
      setPluginError("");
      try {
        const response = await fetch("/api/backend/v1/plugins/available-for-chat", {
          headers: buildAuthHeaders(token, userId),
        });
        updateTokenFromResponse(response);
        const data = (await response
          .json()
          .catch(() => null)) as BackendResult<ChatPluginOption[]> | null;
        if (!response.ok || !data || data.code !== 0) {
          if (!active) return;
          setAvailablePlugins([]);
          setDefaultToolIds([]);
          if (!pluginSelectionInitializedRef.current) {
            setSelectedToolIds([]);
            pluginSelectionInitializedRef.current = true;
          }
          setPluginError(data?.message || "插件列表加载失败");
          return;
        }
        const nextItems = (Array.isArray(data.data) ? data.data : []).map((item) => ({
          ...item,
          source: getPluginSourceFilterValue(item.source),
          runtime_type: normalizePluginRuntimeType(item.runtime_type),
          tools: Array.isArray(item.tools) ? item.tools : [],
        }));
        const nextDefaultToolIds = collectToolIdsFromPlugins(nextItems).slice(
          0,
          MAX_SELECTED_TOOLS
        );
        if (!active) return;
        setAvailablePlugins(nextItems);
        setDefaultToolIds(nextDefaultToolIds);
        const nextPluginKeySet = new Set(
          nextItems.map((plugin, index) => getChatPluginKey(plugin, index))
        );
        setExpandedPluginKeys((current) => {
          if (current.length === 0) {
            return current;
          }
          return current.filter((pluginKey) => nextPluginKeySet.has(pluginKey));
        });
        setSelectedToolIds((current) => {
          if (!pluginSelectionInitializedRef.current) {
            pluginSelectionInitializedRef.current = true;
            return nextDefaultToolIds;
          }
          if (current.length === 0) {
            return current;
          }
          const nextAvailable = new Set(nextDefaultToolIds);
          return current.filter((toolId) => nextAvailable.has(toolId));
        });
      } catch (error) {
        if (!active) return;
        setAvailablePlugins([]);
        setDefaultToolIds([]);
        setExpandedPluginKeys([]);
        if (!pluginSelectionInitializedRef.current) {
          setSelectedToolIds([]);
          pluginSelectionInitializedRef.current = true;
        }
        if (error instanceof Error && error.message) {
          setPluginError(error.message);
        } else {
          setPluginError("插件列表加载失败");
        }
      } finally {
        if (active) {
          setPluginLoading(false);
        }
      }
    };
    void loadAvailablePlugins();
    return () => {
      active = false;
    };
  }, [token, userId]);

  const selectedModelOption = useMemo(
    () => availableModels.find((item) => item.model_id === selectedModelId) ?? null,
    [availableModels, selectedModelId]
  );
  const selectedToolIdSet = useMemo(
    () => new Set(selectedToolIds),
    [selectedToolIds]
  );
  const selectedPluginIds = useMemo(
    () =>
      uniqueStringList(
        availablePlugins.map((plugin) => {
          const hasSelectedTool = (plugin.tools ?? []).some(
            (tool) =>
              typeof tool.tool_id === "string" && selectedToolIdSet.has(tool.tool_id)
          );
          return hasSelectedTool && typeof plugin.plugin_id === "string"
            ? plugin.plugin_id
            : "";
        })
      ),
    [availablePlugins, selectedToolIdSet]
  );
  const availableToolCount = useMemo(
    () => collectToolIdsFromPlugins(availablePlugins).length,
    [availablePlugins]
  );
  const availableRuntimeTypes = useMemo(
    () =>
      uniqueStringList(
        [
          ...KNOWN_PLUGIN_RUNTIME_TYPES,
          ...availablePlugins.map((plugin) =>
            normalizePluginRuntimeType(plugin.runtime_type)
          ),
        ]
      ),
    [availablePlugins]
  );
  const filteredPlugins = useMemo(() => {
    const keyword = pluginSearchKeyword.trim().toLowerCase();
    const matchesFilter = (plugin: ChatPluginOption) => {
      const sourceMatched =
        pluginSourceFilter === "all" ||
        getPluginSourceFilterValue(plugin.source) === pluginSourceFilter;
      const typeMatched =
        pluginTypeFilter === "all" ||
        normalizePluginRuntimeType(plugin.runtime_type) === pluginTypeFilter;
      return sourceMatched && typeMatched;
    };
    if (!keyword) {
      return availablePlugins.filter(matchesFilter);
    }
    return availablePlugins
      .filter(matchesFilter)
      .map((plugin) => {
        const pluginName = (plugin.name || "").toLowerCase();
        const pluginDescription = sanitizeDescriptionText(plugin.description).toLowerCase();
        const pluginMatched =
          pluginName.includes(keyword) || pluginDescription.includes(keyword);
        const matchedTools = (plugin.tools ?? []).filter((tool) => {
          const toolName = (tool.tool_name || "").toLowerCase();
          const toolDescription = sanitizeDescriptionText(tool.description).toLowerCase();
          const toolId = (tool.tool_id || "").toLowerCase();
          return (
            pluginMatched ||
            toolName.includes(keyword) ||
            toolDescription.includes(keyword) ||
            toolId.includes(keyword)
          );
        });
        if (pluginMatched) {
          return plugin;
        }
        if (matchedTools.length === 0) {
          return null;
        }
        return {
          ...plugin,
          tools: matchedTools,
        };
      })
      .filter(Boolean) as ChatPluginOption[];
  }, [availablePlugins, pluginSearchKeyword, pluginSourceFilter, pluginTypeFilter]);
  const paginatedPlugins = useMemo(
    () =>
      filteredPlugins.slice(
        (pluginPage - 1) * pluginPageSize,
        pluginPage * pluginPageSize
      ),
    [filteredPlugins, pluginPage, pluginPageSize]
  );
  const pluginTotalPages = useMemo(
    () => Math.max(1, Math.ceil(filteredPlugins.length / pluginPageSize)),
    [filteredPlugins.length, pluginPageSize]
  );
  const selectedToolItems = useMemo(() => {
    const toolMap = new Map<
      string,
      {
        pluginName: string;
        pluginIcon: string;
        toolName: string;
        toolDescription: string;
      }
    >();
    availablePlugins.forEach((plugin) => {
      const pluginName = plugin.name || "未命名插件";
      const pluginIcon = plugin.icon || "";
      (plugin.tools ?? []).forEach((tool) => {
        if (!tool.tool_id) {
          return;
        }
        toolMap.set(tool.tool_id, {
          pluginName,
          pluginIcon,
          toolName: tool.tool_name || tool.tool_id,
          toolDescription:
            typeof tool.description === "string" ? tool.description.trim() : "",
        });
      });
    });
    return selectedToolIds
      .map((toolId) => {
        const item = toolMap.get(toolId);
        if (!item) {
          return null;
        }
        return {
          toolId,
          ...item,
        };
      })
      .filter(Boolean) as Array<{
      toolId: string;
      pluginName: string;
      pluginIcon: string;
      toolName: string;
      toolDescription: string;
    }>;
  }, [availablePlugins, selectedToolIds]);
  useEffect(() => {
    setPluginPage(1);
  }, [pluginSearchKeyword, pluginSourceFilter, pluginTypeFilter, pluginPageSize]);
  useEffect(() => {
    if (pluginPage > pluginTotalPages) {
      setPluginPage(pluginTotalPages);
    }
  }, [pluginPage, pluginTotalPages]);
  const [historyLoading, setHistoryLoading] = useState(false);

  // 根据 query 参数或生成新 thread_id
  const resolvedThreadId = useMemo(() => {
    const paramThreadId = searchParams.get("threadId");
    if (paramThreadId) {
      return paramThreadId;
    }
    if (!fallbackThreadIdRef.current) {
      fallbackThreadIdRef.current = createThreadId();
    }
    return fallbackThreadIdRef.current;
  }, [searchParams]);

  // 切换 thread 时重置消息与事件映射表
  useEffect(() => {
    if (!agent) return;
    if (resolvedThreadId !== threadId) {
      clearThreadTitleSync();
      setThreadId(resolvedThreadId);
      agent.threadId = resolvedThreadId;
      agent.setMessages([]);
      setInputError("");
      setSemiMessages([]);
      setHistoryLoading(false);
      // 新会话不继承上一会话的 @知识库 / #skill 选择，避免上下文串线。
      setSelectedMentions([]);
      setMentionOpen(false);
      setMentionKeyword("");
      setMentionTriggerSymbol(null);
      setMentionActiveIndex(0);
      setPendingUploads([]);
      setUploadError("");
      setUploading(false);
      currentRunIdRef.current = "";
      textMessageMapRef.current.clear();
      toolCallMapRef.current.clear();
      reasoningMessageMapRef.current.clear();
      activityMessageMapRef.current.clear();
    }
  }, [agent, clearThreadTitleSync, resolvedThreadId, threadId]);

  // 拉取历史消息并映射为 SemiMessage[]
  const loadHistory = useCallback(async () => {
    if (!token || !threadId) return;
    setHistoryLoading(true);
    setInputError("");
    try {
      const response = await fetch(`/api/backend/v1/threads/${threadId}/messages`, {
        headers: buildAuthHeaders(token, userId),
      });
      updateTokenFromResponse(response);
      const data = (await response
        .json()
        .catch(() => null)) as BackendResult<BackendMessagePage> | null;
      if (!response.ok) {
        if (data?.code === 1004) {
          // 新建会话暂无历史，保持空消息列表
          setSemiMessages([]);
          return;
        }
        setInputError(data?.message || "历史消息加载失败");
        return;
      }
      if (!data || data.code !== 0) {
        if (data?.code === 1004) {
          // 新建会话暂无历史，保持空消息列表
          setSemiMessages([]);
          return;
        }
        setInputError(data?.message || "历史消息加载失败");
        return;
      }
      // 后端返回 data.data 才是消息列表
      const mapped = mapHistoryMessages(data.data?.data ?? []);
      setSemiMessages(mapped);
    } catch (error) {
      if (error instanceof Error && error.message) {
        setInputError(error.message);
      } else {
        setInputError("历史消息加载失败");
      }
    } finally {
      setHistoryLoading(false);
    }
  }, [threadId, token, userId]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // 按 run_id 定位或创建 assistant SemiMessage
  const updateRunMessage = useCallback(
    (runId: string, updater: (message: SemiMessage) => SemiMessage) => {
      setSemiMessages((prev) => {
        const next = [...prev];
        const index = next.findIndex(
          (message) => message.id === runId && message.role === "assistant"
        );
        const baseMessage =
          index === -1
            ? {
                id: runId,
                role: "assistant",
                content: [],
                status: "in_progress",
                createdAt: Date.now(),
              }
            : (next[index] as SemiMessage);
        const normalizedMessage = {
          ...baseMessage,
          content: Array.isArray(baseMessage.content)
            ? [...baseMessage.content]
            : [],
        } as SemiMessage;
        const updated = updater(normalizedMessage);
        if (index === -1) {
          next.push(updated);
        } else {
          next[index] = updated;
        }
        return next;
      });
    },
    []
  );

  const resolveRunId = (event: any) =>
    event?.runId ?? event?.run_id ?? event?.data?.runId ?? event?.data?.run_id;

  const resolveMessageId = (event: any) =>
    event?.messageId ??
    event?.message_id ??
    event?.data?.messageId ??
    event?.data?.message_id;

  const resolveActivityMessageId = (event: any) =>
    event?.messageId ??
    event?.message_id ??
    event?.id ??
    event?.data?.messageId ??
    event?.data?.message_id;

  // 订阅 AG-UI 事件流，增量拼装 ContentItem[]
  useEffect(() => {
    if (!agent) return;
    const subscription = agent.subscribe({
      onEvent: ({ event }) => {
        const rawEvent = event as any;
        const eventType = String(rawEvent?.type ?? "");
        if (!eventType) return;
        // 调试日志：确认事件类型与关键信息是否完整
        console.debug("[chat][event]", {
          type: eventType,
          runId: resolveRunId(rawEvent),
          messageId: resolveMessageId(rawEvent),
          threadId: rawEvent?.threadId ?? rawEvent?.thread_id,
        });

        // RUN_STARTED 用来确立 runId
        if (eventType === "RUN_STARTED") {
          const runId = resolveRunId(rawEvent);
          const startedThreadId =
            rawEvent?.threadId ?? rawEvent?.thread_id ?? "";
          // 关键日志：runId 为空会导致后续流式消息无法归档
          console.debug("[chat][RUN_STARTED]", { runId, threadId: startedThreadId });
          if (runId) {
            currentRunIdRef.current = runId;
            updateRunMessage(runId, (message) => message);
          }
          if (startedThreadId) {
            dispatchThreadHistoryUpsert({
              thread_id: startedThreadId,
              replace_thread_id:
                threadId && startedThreadId !== threadId ? threadId : undefined,
              updated_at: new Date().toISOString(),
              pending_title: true,
            });
          }
          return;
        }

      // RUN_FINISHED 标记 run 结束，并清空 AG-UI 消息结构
        if (eventType === "RUN_FINISHED") {
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          const finishedThreadId =
            rawEvent?.threadId ?? rawEvent?.thread_id ?? threadId;
          if (finishedThreadId) {
            dispatchThreadHistoryUpsert({
              thread_id: finishedThreadId,
              replace_thread_id:
                threadId && finishedThreadId !== threadId ? threadId : undefined,
              updated_at: new Date().toISOString(),
              pending_title: true,
            });
          }
          if (runId) {
            updateRunMessage(runId, (message) => ({
              ...message,
              status: "completed",
            }));
          }
          agent.setMessages([]);
          currentRunIdRef.current = "";
          textMessageMapRef.current.clear();
          toolCallMapRef.current.clear();
          reasoningMessageMapRef.current.clear();
          activityMessageMapRef.current.clear();
          startThreadTitleSync(finishedThreadId);
          return;
        }

      // TEXT_MESSAGE_* 流式拼接 output_text
        if (eventType === "TEXT_MESSAGE_START") {
          const runId = currentRunIdRef.current;
          const messageId = resolveMessageId(rawEvent);
          if (!runId || !messageId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "message",
            content: [{ type: "output_text", text: "" }],
            status: "in_progress",
          });
            textMessageMapRef.current.set(messageId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TEXT_MESSAGE_CONTENT") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = textMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target || !Array.isArray(target.content)) {
            return message;
          }
          const nextTarget = { ...target };
          const items = [...target.content];
          const first = items[0];
          if (first && typeof first.text === "string") {
            items[0] = { ...first, text: `${first.text}${event.delta ?? ""}` };
          }
          nextTarget.content = items;
          content[mapping.index] = nextTarget;
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TEXT_MESSAGE_END") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = textMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
          textMessageMapRef.current.delete(messageId);
        return;
        }

      // REASONING_MESSAGE_* 流式拼接 summary_text
        if (eventType === "REASONING_MESSAGE_START") {
          const runId = currentRunIdRef.current;
          const messageId = resolveMessageId(rawEvent);
          if (!runId || !messageId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "reasoning",
            summary: [{ type: "summary_text", text: "" }],
            status: "in_progress",
          });
            reasoningMessageMapRef.current.set(messageId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "REASONING_MESSAGE_CONTENT") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target || !Array.isArray(target.summary)) {
            return message;
          }
          const nextTarget = { ...target };
          const items = [...target.summary];
          const first = items[0];
          if (first && typeof first.text === "string") {
            items[0] = { ...first, text: `${first.text}${event.delta ?? ""}` };
          }
          nextTarget.summary = items;
          content[mapping.index] = nextTarget;
          return { ...message, content };
        });
        return;
        }

        if (eventType === "REASONING_MESSAGE_END") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
          reasoningMessageMapRef.current.delete(messageId);
        return;
        }

      // REASONING_ENCRYPTED_VALUE 绑定到 reasoning 消息
        if (eventType === "REASONING_ENCRYPTED_VALUE") {
          const messageId = resolveMessageId(rawEvent);
          if (!messageId) return;
          const mapping = reasoningMessageMapRef.current.get(messageId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            encryptedValue: event.encryptedValue,
          };
          return { ...message, content };
        });
        return;
        }

      // TOOL_CALL_* 处理 function_call 的参数流
        if (eventType === "TOOL_CALL_START") {
          const runId = currentRunIdRef.current;
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!runId || !toolCallId) return;
          // 实时 SSE 事件仅从 toolCallName 获取工具名
          const name = rawEvent.toolCallName;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "function_call",
            id: toolCallId,
            call_id: toolCallId,
            name,
            status: "in_progress",
            arguments: "",
          });
          toolCallMapRef.current.set(toolCallId, { runId, index });
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_ARGS") {
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!toolCallId) return;
          const mapping = toolCallMapRef.current.get(toolCallId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as any;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            arguments: `${target.arguments ?? ""}${rawEvent.delta ?? ""}`,
          };
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_END") {
          const toolCallId =
            rawEvent.toolCallId ?? rawEvent.toolCall?.id ?? rawEvent.id;
          if (!toolCallId) return;
          const mapping = toolCallMapRef.current.get(toolCallId);
          if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index];
          if (!target) return message;
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
        return;
        }

        if (eventType === "TOOL_CALL_RESULT") {
          const runId = currentRunIdRef.current;
          if (!runId) return;
          const outputText = extractToolResultText(
            rawEvent.result ?? rawEvent.content
          );
        if (!outputText) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          content.push({
            type: TOOL_RESULT_TYPE,
            text: outputText,
            status: "completed",
          });
          return { ...message, content };
        });
        return;
        }

      // activity 归并到 assistant.content 中，由 content renderer 交给 A2UI
        if (
          eventType === ACTIVITY_EVENT_SNAPSHOT ||
          eventType === ACTIVITY_EVENT_DELTA
        ) {
          const activityEvent = rawEvent;
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          const activityType =
            typeof activityEvent?.activityType === "string"
              ? activityEvent.activityType
              : "";
          const activityMessageId = resolveActivityMessageId(activityEvent);
          if (!runId || !activityType || !activityMessageId) return;
          const activityMessageKey = String(activityMessageId);
          const activityEventType =
            eventType === ACTIVITY_EVENT_DELTA
              ? ACTIVITY_EVENT_DELTA
              : ACTIVITY_EVENT_SNAPSHOT;
          updateRunMessage(runId, (message) => {
            const content = Array.isArray(message.content) ? [...message.content] : [];
            const mapped = activityMessageMapRef.current.get(activityMessageKey);
            let index =
              mapped && mapped.runId === runId ? mapped.index : -1;
            if (
              index < 0 ||
              index >= content.length ||
              (content[index] as { activityMessageId?: string })?.activityMessageId !==
                activityMessageKey
            ) {
              index = content.findIndex(
                (contentItem) =>
                  (contentItem as { type?: string }).type === ACTIVITY_CONTENT_TYPE &&
                  (contentItem as { activityMessageId?: string }).activityMessageId ===
                    activityMessageKey
              );
            }
            const previousItem =
              index === -1 ? null : (content[index] as Record<string, any>);
            const mappedContent = mapActivityContent({
              activityType,
              eventType: activityEventType,
              content: activityEvent?.content,
              patch: activityEvent?.patch,
              replace: activityEvent?.replace,
              previousContent: previousItem?.content,
            });
            const activityContentItem = {
              ...(previousItem ?? {}),
              id: activityMessageKey,
              type: ACTIVITY_CONTENT_TYPE,
              activityMessageId: activityMessageKey,
              activityType,
              activityEventType,
              content: mappedContent,
              status:
                activityEvent?.status ??
                (activityEventType === ACTIVITY_EVENT_DELTA
                  ? "in_progress"
                  : "completed"),
              timestamp:
                activityEvent?.timestamp ??
                rawEvent?.timestamp ??
                Date.now(),
            };
            if (index === -1) {
              index = content.length;
              content.push(activityContentItem);
            } else {
              content[index] = activityContentItem;
            }
            activityMessageMapRef.current.set(activityMessageKey, {
              runId,
              index,
            });
            return {
              ...message,
              content,
            };
          });
          return;
        }
      },
    });

    return () => {
      subscription?.unsubscribe?.();
    };
  }, [agent, startThreadTitleSync, threadId, updateRunMessage]);

  const chats = useMemo<SemiMessage[]>(
    () =>
      semiMessages.map((message, index) => ({
        ...message,
        id: message.id ?? `${message.role}-${index}`,
      })),
    [semiMessages]
  );
  const showHistorySkeleton = historyLoading && chats.length === 0;
  const showEmptyState =
    !historyLoading && !agent.isRunning && chats.length === 0 && !inputError;
  const mentionCandidates = useMemo(() => {
    if (!mentionOpen || !mentionTriggerSymbol) {
      return [];
    }
    const keyword = mentionKeyword.trim().toLowerCase();
    const targetType: MentionTargetType =
      mentionTriggerSymbol === "@" ? "kb" : "skill";
    return mentionOptions
      .filter((item) => item.type === targetType)
      .filter((item) => {
        if (!keyword) {
          return true;
        }
        return item.keyword.includes(keyword);
      })
      .filter((item) => {
        const key = `${item.type}:${item.id}`;
        return !selectedMentions.some(
          (selection) => `${selection.type}:${selection.id}` === key
        );
      })
      .slice(0, 8);
  }, [
    mentionKeyword,
    mentionOpen,
    mentionOptions,
    mentionTriggerSymbol,
    selectedMentions,
  ]);

  useEffect(() => {
    if (!mentionOpen || mentionCandidates.length === 0) {
      setMentionActiveIndex(0);
      return;
    }
    setMentionActiveIndex((current) =>
      Math.min(current, mentionCandidates.length - 1)
    );
  }, [mentionCandidates, mentionOpen]);

  const setComposerText = useCallback((text: string) => {
    inputRef.current?.setContent(toEditorParagraphHtml(text));
  }, []);

  const handleInputContentChange = useCallback(
    (contents: AIChatInputContent[]) => {
      const text = extractInputPlainText(contents ?? []);
      setComposerTextValue(text);
      const trigger = matchMentionTrigger(text);
      if (!trigger) {
        setMentionKeyword("");
        setMentionTriggerSymbol(null);
        setMentionOpen(false);
        return;
      }
      setMentionTriggerSymbol(trigger.symbol);
      setMentionKeyword(trigger.keyword);
      setMentionOpen(true);
      setMentionActiveIndex(0);
    },
    []
  );

  const handleMentionSelect = useCallback(
    (target: MentionTargetOption) => {
      setSelectedMentions((current) => {
        const key = `${target.type}:${target.id}`;
        if (current.some((item) => `${item.type}:${item.id}` === key)) {
          return current;
        }
        return [
          ...current,
          {
            type: target.type,
            id: target.id,
            name: target.name,
          },
        ];
      });
      setMentionOpen(false);
      setMentionKeyword("");
      setMentionTriggerSymbol(null);
      setMentionActiveIndex(0);
      setComposerTextValue((current) => {
        const next = stripMentionSuffix(current, mentionTriggerSymbol);
        setComposerText(next);
        return next;
      });
    },
    [mentionTriggerSymbol, setComposerText]
  );

  // mention 下拉打开时拦截方向键与回车，支持纯键盘选择目标。
  const handleMentionKeyDownCapture = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (!mentionOpen || mentionCandidates.length === 0) {
        return;
      }
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setMentionActiveIndex((current) =>
          (current + 1) % mentionCandidates.length
        );
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setMentionActiveIndex((current) =>
          (current - 1 + mentionCandidates.length) % mentionCandidates.length
        );
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        const target =
          mentionCandidates[
            Math.max(0, Math.min(mentionActiveIndex, mentionCandidates.length - 1))
          ];
        if (target) {
          handleMentionSelect(target);
        }
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setMentionOpen(false);
        setMentionKeyword("");
        setMentionTriggerSymbol(null);
        setMentionActiveIndex(0);
      }
    },
    [handleMentionSelect, mentionActiveIndex, mentionCandidates, mentionOpen]
  );

  const removeMentionSelection = useCallback(
    (target: MentionSelectionItem) => {
      setSelectedMentions((current) =>
        current.filter(
          (item) =>
            item.type !== target.type ||
            item.id !== target.id
        )
      );
    },
    []
  );

  // 打开本地文件选择器，用于上传聊天附件。
  const handleOpenUploadPicker = useCallback(() => {
    if (agent.isRunning || uploading) {
      return;
    }
    uploadInputRef.current?.click();
  }, [agent.isRunning, uploading]);

  // 从待发送附件列表中删除指定项。
  const removePendingUpload = useCallback((assetId: string) => {
    setPendingUploads((current) => current.filter((item) => item.id !== assetId));
  }, []);

  // 上传单个附件并返回规范化后的上传结果。
  const uploadSingleAsset = useCallback(
    async (file: File): Promise<UploadAssetItem> => {
      const formData = new FormData();
      formData.append("file", file);
      if (threadId) {
        formData.append("thread_id", threadId);
      }
      const response = await fetch("/api/backend/v1/chat/uploads", {
        method: "POST",
        headers: buildAuthHeaders(token, userId),
        body: formData,
      });
      updateTokenFromResponse(response);
      const result = (await response
        .json()
        .catch(() => null)) as BackendResult<BackendChatUploadAsset> | null;
      if (!response.ok || !result || result.code !== 0 || !result.data?.url) {
        const backendMessage =
          result?.message && typeof result.message === "string"
            ? result.message
            : "";
        throw new Error(backendMessage || `上传失败：${file.name}`);
      }

      const normalizedMimeType = normalizeMimeType(
        result.data.mime_type || file.type || "application/octet-stream"
      );
      const normalizedFileName =
        (typeof result.data.file_name === "string" && result.data.file_name.trim()) ||
        file.name ||
        "attachment";
      const normalizedSize =
        typeof result.data.size === "number" && Number.isFinite(result.data.size)
          ? result.data.size
          : file.size;
      return {
        id: createMessageId(),
        key:
          (typeof result.data.key === "string" && result.data.key.trim()) ||
          normalizedFileName,
        url: String(result.data.url),
        mimeType: normalizedMimeType,
        fileName: normalizedFileName,
        size: normalizedSize,
        mediaKind:
          result.data.media_kind ||
          inferUploadKind(normalizedMimeType || "application/octet-stream"),
      };
    },
    [threadId, token, userId]
  );

  // 处理文件选择并批量上传，成功项加入待发送附件区。
  const handleUploadInputChange = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = Array.from(event.target.files ?? []);
      event.target.value = "";
      if (selectedFiles.length === 0) {
        return;
      }

      setUploadError("");
      setUploading(true);
      const successItems: UploadAssetItem[] = [];
      let firstError = "";
      for (const file of selectedFiles) {
        try {
          const uploaded = await uploadSingleAsset(file);
          successItems.push(uploaded);
        } catch (error) {
          if (!firstError) {
            firstError = error instanceof Error ? error.message : `上传失败：${file.name}`;
          }
        }
      }
      setUploading(false);

      if (successItems.length > 0) {
        setPendingUploads((current) => {
          const merged = [...current];
          successItems.forEach((item) => {
            const existed = merged.some((existing) => existing.url === item.url);
            if (!existed) {
              merged.push(item);
            }
          });
          return merged;
        });
      }
      if (firstError) {
        setUploadError(firstError);
      }
    },
    [uploadSingleAsset]
  );

  const roleConfig = useMemo(
    () => ({
      user: {
        name: userName || "用户",
        avatar:
          userAvatar ||
          buildTextAvatarDataUrl(userName || "用户", {
            background: "#2DD4BF",
          }),
        color: "teal",
      },
      assistant: {
        name: "AI助手",
        avatar: buildTextAvatarDataUrl("AI", { background: "#6366F1" }),
        color: "indigo",
      },
    }),
    [userAvatar, userName]
  );

  const renderDialogueContentItem = useMemo(
    () => ({
      [TOOL_RESULT_TYPE]: (item: { text?: string }) => {
        if (!item?.text) return null;
        return <ToolResultCollapse text={item.text} />;
      },
      [ACTIVITY_CONTENT_TYPE]: (item: {
        id?: string;
        activityMessageId?: string;
        activityType?: string;
        content?: unknown;
      }) => {
        if (!item?.activityType) return null;
        const activityMessage: ActivityMessage = {
          id: String(item.activityMessageId ?? item.id ?? createMessageId()),
          role: "activity",
          activityType: item.activityType,
          content: toRecord(item.content),
        };
        const activityNode = renderActivityMessage(activityMessage);
        if (!activityNode) return null;
        return (
          <div className="motion-safe-slide-up my-2 overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-md)]">
            <div className="border-b border-[var(--color-border-default)] bg-[linear-gradient(90deg,rgba(37,99,255,0.08),rgba(14,165,233,0.08))] px-3 py-2">
              <span className="text-xs font-medium text-[var(--color-text-secondary)]">
                可视化内容
              </span>
            </div>
            <div className="p-3">{activityNode}</div>
          </div>
        );
      },
    }),
    [renderActivityMessage]
  );

  const toggleToolSelection = useCallback((toolId: string, checked: boolean) => {
    setSelectedToolIds((current) => {
      if (!toolId) {
        return current;
      }
      if (checked) {
        if (current.includes(toolId)) {
          return current;
        }
        if (current.length >= MAX_SELECTED_TOOLS) {
          setPluginError(`最多只能选择 ${MAX_SELECTED_TOOLS} 个工具`);
          return current;
        }
        setPluginError("");
        return uniqueStringList([...current, toolId]);
      }
      setPluginError("");
      return current.filter((item) => item !== toolId);
    });
  }, []);

  const togglePluginSelection = useCallback(
    (plugin: ChatPluginOption, checked: boolean) => {
      const pluginToolIds = uniqueStringList(
        (plugin.tools ?? []).map((tool) =>
          typeof tool.tool_id === "string" ? tool.tool_id : ""
        )
      );
      setSelectedToolIds((current) => {
        if (pluginToolIds.length === 0) {
          return current;
        }
        if (checked) {
          const pendingToolIds = pluginToolIds.filter(
            (toolId) => !current.includes(toolId)
          );
          if (pendingToolIds.length === 0) {
            return current;
          }
          const remaining = MAX_SELECTED_TOOLS - current.length;
          if (remaining <= 0) {
            setPluginError(`最多只能选择 ${MAX_SELECTED_TOOLS} 个工具`);
            return current;
          }
          const appended = pendingToolIds.slice(0, remaining);
          if (appended.length < pendingToolIds.length) {
            setPluginError(`最多只能选择 ${MAX_SELECTED_TOOLS} 个工具，已按上限截断`);
          } else {
            setPluginError("");
          }
          return uniqueStringList([...current, ...appended]);
        }
        const blockedToolSet = new Set(pluginToolIds);
        setPluginError("");
        return current.filter((toolId) => !blockedToolSet.has(toolId));
      });
    },
    []
  );

  const resetPluginSelection = useCallback(() => {
    setSelectedToolIds(defaultToolIds.slice(0, MAX_SELECTED_TOOLS));
    setPluginError("");
  }, [defaultToolIds]);

  const closePluginPanel = useCallback(() => {
    setPluginPanelOpen(false);
    setExpandedPluginKeys([]);
  }, []);

  const openPluginPanel = useCallback(() => {
    setExpandedPluginKeys([]);
    setPluginPanelOpen(true);
  }, []);

  const togglePluginExpanded = useCallback((pluginKey: string) => {
    setExpandedPluginKeys((current) =>
      current.includes(pluginKey)
        ? current.filter((item) => item !== pluginKey)
        : [...current, pluginKey]
    );
  }, []);

  const handleConversationModeChange = useCallback(
    (nextMode: "chat" | "agent") => {
      setConversationMode(nextMode);
      if (nextMode !== "chat") {
        closePluginPanel();
      }
    },
    [closePluginPanel]
  );

  const handlePluginModeChange = useCallback(
    (nextMode: "select" | "search") => {
      setPluginMode(nextMode);
      if (nextMode !== "select") {
        closePluginPanel();
      }
    },
    [closePluginPanel]
  );

  const handlePluginPageChange = useCallback(
    (nextPage: number) => {
      setPluginPage(Math.max(1, Math.min(pluginTotalPages, nextPage)));
    },
    [pluginTotalPages]
  );

  const handlePluginPageSizeChange = useCallback((nextSize: number) => {
    if (!PLUGIN_PAGE_SIZE_OPTIONS.includes(nextSize)) {
      return;
    }
    setPluginPageSize(nextSize);
  }, []);

  const renderConfigureArea = useCallback(
    () => (
      <ChatModeConfigureArea
        conversationMode={conversationMode}
        pluginMode={pluginMode}
        selectedToolCount={selectedToolIds.length}
        onConversationModeChange={handleConversationModeChange}
        onPluginModeChange={handlePluginModeChange}
        onOpenPluginPanel={openPluginPanel}
      />
    ),
    [
      conversationMode,
      handleConversationModeChange,
      handlePluginModeChange,
      openPluginPanel,
      pluginMode,
      selectedToolIds.length,
    ]
  );

  const renderActionArea = useCallback(
    (props: { menuItem: ReactNode[]; className: string }) => (
      <div
        className={joinClasses(props.className, "flex items-center gap-2")}
        onMouseDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
      >
        {conversationMode === "chat" && (
          <div
            className="ui-select-control--glass relative w-[280px] max-w-[70vw] rounded-full border border-transparent px-4 py-2 focus-within:border-[var(--color-action-primary)]"
            title={
              selectedModelOption
                ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
                : "请先配置模型"
            }
          >
            <span className="pointer-events-none block overflow-hidden pr-11 text-ellipsis whitespace-nowrap text-sm font-medium text-[var(--color-text-primary)]">
              {selectedModelOption
                ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
                : "请先配置模型"}
            </span>
            <UiSelect
              value={selectedModelId}
              onChange={(event) => {
                const nextModelId = event.target.value;
                const nextItem = availableModels.find((item) => item.model_id === nextModelId);
                setSelectedModelId(nextModelId);
                setSelectedProviderId(nextItem?.provider_id ?? "");
              }}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
              title={
                selectedModelOption
                  ? `${selectedModelOption.provider_name} / ${selectedModelOption.model_name}`
                  : "请先配置模型"
              }
              className="absolute inset-0 h-full w-full cursor-pointer rounded-full opacity-0"
            >
              {availableModels.length === 0 ? (
                <option value="">请先配置模型</option>
              ) : (
                availableModels.map((item) => (
                  <option key={item.model_id} value={item.model_id}>
                    {item.provider_name} / {item.model_name}
                  </option>
                ))
              )}
            </UiSelect>
          </div>
        )}
        <button
          type="button"
          onClick={handleOpenUploadPicker}
          disabled={agent.isRunning || uploading}
          className="inline-flex items-center gap-1 rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)] transition hover:bg-[var(--color-bg-page)] disabled:cursor-not-allowed disabled:opacity-50"
          title="上传图片/文件/音频/视频"
        >
          <span>上传</span>
          {pendingUploads.length > 0 && (
            <span className="rounded-full bg-[var(--color-bg-page)] px-1.5 py-0.5 text-[10px] text-[var(--color-text-muted)]">
              {pendingUploads.length}
            </span>
          )}
        </button>
        {props.menuItem}
      </div>
    ),
    [
      agent.isRunning,
      availableModels,
      conversationMode,
      handleOpenUploadPicker,
      pendingUploads.length,
      selectedModelId,
      selectedModelOption,
      uploading,
    ]
  );

  // 发送：先回显 user 消息，再触发 run
  const handleMessageSend = useCallback(
    (payload: MessageContent) => {
      if (agent.isRunning) return;
      if (uploading) {
        setInputError("附件上传中，请稍候");
        return;
      }
      if (conversationMode === "agent") {
        setInputError("Agent 模式暂未开放");
        return;
      }
      if (!selectedModelOption) {
        setInputError("请先配置模型服务并选择模型");
        return;
      }
      const inputContents = payload?.inputContents ?? [];
      const text = extractInputPlainText(inputContents as AIChatInputContent[]);
      const textExists = text.trim().length > 0;
      if (!textExists && pendingUploads.length === 0) {
        setInputError("请输入内容或上传附件");
        return;
      }
      const uploadedAssets = [...pendingUploads];
      const messageContent = buildAguiUserContent(text, uploadedAssets);
      setInputError("");
      setUploadError("");
      setMentionOpen(false);
      setMentionKeyword("");
      setMentionTriggerSymbol(null);
      setComposerTextValue("");
      const now = new Date().toISOString();
      dispatchThreadHistoryUpsert({
        thread_id: threadId,
        updated_at: now,
        pending_title: true,
      });
      const message: AguiMessage = {
        id: createMessageId(),
        role: "user",
        content: messageContent,
      };
      setSemiMessages((prev) => [
        ...prev,
        {
          id: message.id,
          role: "user",
          content: mapAguiUserContentToSemi(message.content),
          status: "completed",
          createdAt: Date.now(),
        },
      ]);
      setPendingUploads([]);
      agent.setMessages([message]);
      agent
        .runAgent({
          forwardedProps: {
            contextSelections: {
              skills: selectedMentions
                .filter((item) => item.type === "skill")
                .map((item) => ({
                  id: item.id,
                  name: item.name,
                })),
              knowledgeBases: selectedMentions
                .filter((item) => item.type === "kb")
                .map((item) => ({
                  id: item.id,
                  name: item.name,
                })),
            },
            agentConfig: {
              conversation: {
                mode: conversationMode,
              },
              model: {
                providerId: selectedProviderId || selectedModelOption.provider_id,
                modelId: selectedModelOption.model_id,
              },
              plugins: {
                mode: pluginMode,
                selectedToolIds: pluginMode === "select" ? selectedToolIds : [],
              },
              features: {},
            },
          },
        })
        .catch((error) => {
          if (uploadedAssets.length > 0) {
            setPendingUploads(uploadedAssets);
          }
          if (error instanceof Error && error.message) {
            setInputError(error.message);
          } else {
            setInputError("发送失败");
          }
        });
    },
    [
      agent,
      conversationMode,
      pluginMode,
      selectedModelOption,
      selectedMentions,
      selectedProviderId,
      selectedToolIds,
      threadId,
      pendingUploads,
      uploading,
    ]
  );

  const handleStopGenerate = useCallback(() => {
    agent.abortRun();
  }, [agent]);

  return (
    <>
      <div className="chat-page workspace-gradient-surface workspace-gradient-surface--chat flex h-full w-full flex-col p-3 md:p-4">
        <div className="motion-safe-fade-in flex h-full min-h-0 flex-col gap-3">
          <div className="motion-safe-lift flex min-h-0 flex-1 flex-col overflow-hidden rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.92)] shadow-[var(--shadow-sm)] backdrop-blur-sm">
            <div className="flex-1 overflow-hidden px-1 py-1">
              {showHistorySkeleton ? (
                <div className="flex h-full flex-col gap-4 px-4 py-5">
                  <div className="flex justify-start">
                    <div className="w-full max-w-[68%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(226,232,240,0.8)] bg-[rgba(248,250,252,0.82)] p-4">
                      <div className="h-3 w-20 animate-pulse rounded-full bg-[rgba(148,163,184,0.16)]" />
                      <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(148,163,184,0.12)]" />
                      <div className="h-3 w-4/5 animate-pulse rounded-full bg-[rgba(148,163,184,0.1)]" />
                    </div>
                  </div>
                  <div className="flex justify-end">
                    <div className="w-full max-w-[52%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(37,99,255,0.12)] bg-[rgba(37,99,255,0.06)] p-4">
                      <div className="h-3 w-16 animate-pulse rounded-full bg-[rgba(37,99,255,0.12)]" />
                      <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(37,99,255,0.08)]" />
                    </div>
                  </div>
                  <div className="flex justify-start">
                    <div className="w-full max-w-[74%] space-y-2 rounded-[var(--radius-xl)] border border-[rgba(226,232,240,0.8)] bg-[rgba(248,250,252,0.82)] p-4">
                      <div className="h-3 w-24 animate-pulse rounded-full bg-[rgba(148,163,184,0.14)]" />
                      <div className="h-3 w-full animate-pulse rounded-full bg-[rgba(148,163,184,0.1)]" />
                      <div className="h-3 w-3/4 animate-pulse rounded-full bg-[rgba(148,163,184,0.08)]" />
                    </div>
                  </div>
                </div>
              ) : showEmptyState ? (
                <div className="motion-safe-fade-in flex h-full items-center justify-center px-6 py-8">
                  <div className="max-w-md rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,250,252,0.94))] p-6 text-center shadow-[var(--shadow-sm)]">
                    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.08)] text-[var(--color-action-primary)]">
                      <svg
                        viewBox="0 0 24 24"
                        className="h-6 w-6"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.8"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <path d="M8 10h8" />
                        <path d="M8 14h5" />
                        <path d="M6 4h12a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H9l-5 3V6a2 2 0 0 1 2-2z" />
                      </svg>
                    </div>
                    <div className="mt-4 text-sm font-semibold text-[var(--color-text-primary)]">
                      开始新的对话
                    </div>
                    <div className="mt-2 text-sm text-[var(--color-text-muted)]">
                      输入你的问题，系统会实时展示回复、工具调用和可视化内容。
                    </div>
                  </div>
                </div>
              ) : (
                <AIChatDialogue
                  align="leftRight"
                  mode="bubble"
                  chats={chats}
                  renderDialogueContentItem={renderDialogueContentItem}
                  roleConfig={roleConfig}
                  className="h-full"
                />
              )}
            </div>
            <div className="border-t border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,250,252,0.96))] px-4 py-3">
              {agent.isRunning && (
                <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.06)] px-3 py-1 text-xs font-medium text-[var(--color-action-primary)]">
                  <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-current" />
                  AI 正在生成回复
                </div>
              )}
              <div className="relative" onKeyDownCapture={handleMentionKeyDownCapture}>
                {mentionOpen && mentionCandidates.length > 0 && (
                  <div className="absolute bottom-full left-0 right-0 z-20 mb-2 overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-md)]">
                    <div className="border-b border-[var(--color-border-default)] px-3 py-2 text-xs text-[var(--color-text-muted)]">
                      {mentionTriggerSymbol === "@"
                        ? "当前选择：知识库（@）"
                        : "当前选择：Skill（#）"}
                    </div>
                    {mentionCandidates.map((item, index) => (
                      <button
                        key={`${item.type}:${item.id}`}
                        type="button"
                        className={joinClasses(
                          "flex w-full items-center justify-between px-3 py-2 text-left text-sm transition",
                          mentionActiveIndex === index
                            ? "bg-[var(--color-bg-page)]"
                            : "hover:bg-[var(--color-bg-page)]"
                        )}
                        onMouseEnter={() => setMentionActiveIndex(index)}
                        onClick={() => handleMentionSelect(item)}
                      >
                        <span className="truncate text-[var(--color-text-primary)]">
                          {item.displayName}
                        </span>
                        <span className="ml-3 shrink-0 rounded-full border border-[var(--color-border-default)] px-2 py-0.5 text-[10px] text-[var(--color-text-muted)]">
                          {item.type === "skill" ? "Skill" : "知识库"}
                        </span>
                      </button>
                    ))}
                  </div>
                )}
                {selectedMentions.length > 0 && (
                  <div className="mb-2 flex flex-wrap gap-2">
                    {selectedMentions.map((item) => (
                      <span
                        key={`${item.type}:${item.id}`}
                        className="inline-flex items-center gap-1 rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-2 py-1 text-xs text-[var(--color-text-secondary)]"
                      >
                        <span className="font-medium text-[var(--color-text-primary)]">
                          {item.type === "skill" ? "Skill" : "知识库"}
                        </span>
                        <span className="max-w-[220px] truncate">{item.name || item.id}</span>
                        <button
                          type="button"
                          className="rounded px-1 leading-none text-[var(--color-text-muted)] hover:bg-[var(--color-bg-overlay)]"
                          onClick={() => removeMentionSelection(item)}
                          aria-label="删除已选项"
                        >
                          ×
                        </button>
                      </span>
                    ))}
                  </div>
                )}
                <input
                  ref={uploadInputRef}
                  type="file"
                  className="hidden"
                  multiple
                  onChange={handleUploadInputChange}
                />
                {(pendingUploads.length > 0 || uploading) && (
                  <div className="mb-2 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-2">
                    <div className="mb-2 flex items-center justify-between text-xs text-[var(--color-text-muted)]">
                      <span>待发送附件</span>
                      {uploading && <span>上传中...</span>}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {pendingUploads.map((asset) => (
                        <span
                          key={asset.id}
                          className="inline-flex max-w-full items-center gap-1 rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-2 py-1 text-xs text-[var(--color-text-secondary)]"
                        >
                          <span className="font-medium text-[var(--color-text-primary)]">
                            {asset.mediaKind}
                          </span>
                          <span className="max-w-[220px] truncate">{asset.fileName}</span>
                          <span className="text-[10px] text-[var(--color-text-muted)]">
                            {formatFileSize(asset.size)}
                          </span>
                          <button
                            type="button"
                            className="rounded px-1 leading-none text-[var(--color-text-muted)] hover:bg-[var(--color-bg-overlay)]"
                            onClick={() => removePendingUpload(asset.id)}
                            aria-label="删除附件"
                          >
                            ×
                          </button>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
                <AIChatInput
                  ref={inputRef as any}
                  keepSkillAfterSend={false}
                  placeholder="输入消息；@ 选择知识库，# 选择 Skill"
                  onContentChange={handleInputContentChange}
                  onMessageSend={handleMessageSend}
                  onStopGenerate={handleStopGenerate}
                  generating={agent.isRunning}
                  canSend={!agent.isRunning && !uploading}
                  showUploadButton={false}
                  showUploadFile={false}
                  showReference={false}
                  round
                  immediatelyRender={false}
                  renderConfigureArea={renderConfigureArea}
                  renderActionArea={renderActionArea}
                />
              </div>
              {inputError && (
                <div
                  role="alert"
                  aria-live="polite"
                  className="motion-safe-slide-up mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]"
                >
                  {inputError}
                </div>
              )}
              {uploadError && (
                <div
                  role="alert"
                  aria-live="polite"
                  className="motion-safe-slide-up mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]"
                >
                  {uploadError}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      <PluginSelectionModal
        open={conversationMode === "chat" && pluginMode === "select" && pluginPanelOpen}
        maxSelectedTools={MAX_SELECTED_TOOLS}
        selectedToolIds={selectedToolIds}
        selectedPluginIds={selectedPluginIds}
        selectedToolIdSet={selectedToolIdSet}
        pluginLoading={pluginLoading}
        pluginSearchKeyword={pluginSearchKeyword}
        pluginSourceFilter={pluginSourceFilter}
        pluginTypeFilter={pluginTypeFilter}
        pluginPage={pluginPage}
        pluginPageSize={pluginPageSize}
        pluginPageSizeOptions={PLUGIN_PAGE_SIZE_OPTIONS}
        pluginTotalPages={pluginTotalPages}
        availableToolCount={availableToolCount}
        availableRuntimeTypes={availableRuntimeTypes}
        filteredPlugins={filteredPlugins}
        paginatedPlugins={paginatedPlugins}
        expandedPluginKeys={expandedPluginKeys}
        selectedToolItems={selectedToolItems}
        pluginError={pluginError}
        onClose={closePluginPanel}
        onReset={resetPluginSelection}
        onPluginSearchKeywordChange={setPluginSearchKeyword}
        onPluginSourceFilterChange={setPluginSourceFilter}
        onPluginTypeFilterChange={setPluginTypeFilter}
        onPluginPageChange={handlePluginPageChange}
        onPluginPageSizeChange={handlePluginPageSizeChange}
        onTogglePluginExpanded={togglePluginExpanded}
        onTogglePluginSelection={togglePluginSelection}
        onToggleToolSelection={toggleToolSelection}
      />
    </>
  );
}

export default function ChatPage() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [userInfo, setUserInfo] = useState<StoredUser | null>(null);
  const refreshUserInfo = useCallback(() => {
    setUserInfo(readStoredUser());
  }, []);

  useEffect(() => {
    const currentToken = readValidToken(router);
    if (!currentToken) {
      router.push("/login");
      return;
    }
    setToken(currentToken);
    refreshUserInfo();
  }, [refreshUserInfo, router]);

  // 同步 localStorage 用户信息变化
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handleStorage = (event: StorageEvent) => {
      if (event.key === "user") {
        refreshUserInfo();
      }
    };
    const handleFocus = () => {
      refreshUserInfo();
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("focus", handleFocus);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("focus", handleFocus);
    };
  }, [refreshUserInfo]);

  const userName = useMemo(() => {
    const name =
      typeof userInfo?.username === "string" ? userInfo.username.trim() : "";
    return name || "用户";
  }, [userInfo]);

  const userAvatar = useMemo(() => {
    const avatar =
      typeof userInfo?.avatar === "string" ? userInfo.avatar.trim() : "";
    return avatar;
  }, [userInfo]);

  const userId = useMemo(() => {
    if (!token) return "";
    const storedId = userInfo?.user_id;
    if (storedId !== undefined && storedId !== null) {
      return String(storedId);
    }
    return getUserIdFromToken(token);
  }, [token, userInfo]);

  const copilotHeaders = useMemo(
    () => buildAuthHeaders(token, userId),
    [token, userId]
  );

  if (!token) {
    return null;
  }

  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      renderActivityMessages={ACTIVITY_RENDERERS}
      headers={copilotHeaders}
    >
      <ChatContent
        token={token}
        userId={userId}
        userName={userName}
        userAvatar={userAvatar}
      />
    </CopilotKitProvider>
  );
}
