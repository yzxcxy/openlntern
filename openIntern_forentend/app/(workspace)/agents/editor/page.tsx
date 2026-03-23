"use client";

import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import {
  CopilotKitProvider,
  useRenderActivityMessage,
} from "@copilotkit/react-core/v2";
import type { Message as AguiMessage } from "@ag-ui/client";
import { AIChatDialogue, AIChatInput } from "@douyinfe/semi-ui-19";
import type { Message as SemiMessage } from "@douyinfe/semi-ui-19/lib/es/aiChatDialogue/interface";
import type {
  Content as AIChatInputContent,
  MessageContent,
} from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import { useRouter, useSearchParams } from "next/navigation";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type Ref,
  type ReactNode,
} from "react";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";
import { UiModal } from "../../../components/ui/UiModal";
import { UiSelect } from "../../../components/ui/UiSelect";
import { UiTextarea } from "../../../components/ui/UiTextarea";
import { theme } from "../../../theme";
import {
  buildAuthHeaders,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  type StoredUser,
} from "../../auth";
import { ChatComposerAssist } from "../../chat/ChatComposerAssist";
import { ChatInputActionArea } from "../../chat/ChatInputActionArea";
import {
  ACTIVITY_CONTENT_TYPE,
  ACTIVITY_EVENT_DELTA,
  ACTIVITY_EVENT_SNAPSHOT,
  TOOL_RESULT_TYPE,
  buildAguiUserContent,
  buildTextAvatarDataUrl,
  createMessageId,
  createThreadId,
  extractInputPlainText,
  extractToolResultText,
  groupAssistantProcessItems,
  mapActivityContent,
  mapAguiUserContentToSemi,
  toEditorParagraphHtml,
} from "../../chat/chat-helpers";
import { useChatDialogueRenderers } from "../../chat/useChatDialogueRenderers";
import { usePendingUploads } from "../../chat/usePendingUploads";
import {
  type AgentDetail,
  type AgentPayload,
  type AgentType,
  createAgent,
  getAgent,
  listEnabledAgentOptions,
  listKnowledgeBaseOptions,
  listModelOptions,
  listPluginOptions,
  listSkillOptions,
  runAgentDebugSession,
  updateAgent,
  uploadAgentImage,
} from "../agent-api";

type ResourceOption = {
  id: string;
  label: string;
  description?: string;
};

type AgentFormState = {
  name: string;
  description: string;
  agent_type: AgentType;
  system_prompt: string;
  example_questions_text: string;
  default_model_id: string;
  agent_memory_enabled: boolean;
  avatar_url: string;
  background_image_url: string;
  tool_ids: string[];
  skill_names: string[];
  knowledge_base_names: string[];
  sub_agent_ids: string[];
};

type BindingPickerType =
  | "tool_ids"
  | "skill_names"
  | "knowledge_base_names"
  | "sub_agent_ids";

type AgentEditorContentProps = {
  token: string;
  userId: string;
  userName: string;
  userAvatar: string;
};

type DebugEventPayload = Record<string, unknown> & {
  type?: string;
  delta?: string;
  id?: unknown;
  toolCallId?: unknown;
  toolCall?: { id?: unknown } | null;
  toolCallName?: string;
  encryptedValue?: unknown;
  activityType?: string;
  content?: unknown;
  result?: unknown;
  patch?: unknown;
  replace?: boolean;
  status?: string;
  timestamp?: number;
};

type MessageItemRecord = Record<string, unknown> & {
  content?: Array<Record<string, unknown>>;
  summary?: Array<Record<string, unknown>>;
  arguments?: string;
};

const SITE_DEFAULT_AVATAR_URL = "/OpenIntern.png";
const A2UI_MESSAGE_RENDERER = createA2UIMessageRenderer({ theme });
const ACTIVITY_RENDERERS = [A2UI_MESSAGE_RENDERER];

const EMPTY_FORM: AgentFormState = {
  name: "",
  description: "",
  agent_type: "single",
  system_prompt: "",
  example_questions_text: "",
  default_model_id: "",
  agent_memory_enabled: false,
  avatar_url: "",
  background_image_url: "",
  tool_ids: [],
  skill_names: [],
  knowledge_base_names: [],
  sub_agent_ids: [],
};

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

const parseBackgroundImageURL = (rawJSON: string | undefined): string => {
  const trimmed = (rawJSON || "").trim();
  if (!trimmed) {
    return "";
  }
  try {
    const parsed = JSON.parse(trimmed) as { image_url?: string; url?: string };
    if (typeof parsed.image_url === "string" && parsed.image_url.trim()) {
      return parsed.image_url.trim();
    }
    if (typeof parsed.url === "string" && parsed.url.trim()) {
      return parsed.url.trim();
    }
  } catch {
    return "";
  }
  return "";
};

const normalizeEventID = (value: unknown) =>
  typeof value === "string" || typeof value === "number"
    ? String(value).trim()
    : "";

const resolveRunId = (event: unknown) => {
  const payload = event as
    | {
        runId?: unknown;
        run_id?: unknown;
        data?: { runId?: unknown; run_id?: unknown };
      }
    | null;
  return normalizeEventID(
    payload?.runId ?? payload?.run_id ?? payload?.data?.runId ?? payload?.data?.run_id
  );
};

const resolveMessageId = (event: unknown) => {
  const payload = event as
    | {
        messageId?: unknown;
        message_id?: unknown;
        data?: { messageId?: unknown; message_id?: unknown };
      }
    | null;
  return normalizeEventID(
    payload?.messageId ??
      payload?.message_id ??
      payload?.data?.messageId ??
      payload?.data?.message_id
  );
};

const resolveActivityMessageId = (event: unknown) => {
  const payload = event as
    | {
        id?: unknown;
        messageId?: unknown;
        message_id?: unknown;
        data?: { messageId?: unknown; message_id?: unknown };
      }
    | null;
  return normalizeEventID(
    payload?.messageId ??
      payload?.message_id ??
      payload?.id ??
      payload?.data?.messageId ??
      payload?.data?.message_id
  );
};

// toPayload converts editor-only form fields into backend agent payload shape.
const toPayload = (form: AgentFormState): AgentPayload => ({
  name: form.name.trim(),
  description: form.description.trim(),
  agent_type: form.agent_type,
  system_prompt: form.system_prompt.trim(),
  avatar_url: form.avatar_url.trim(),
  chat_background_json: form.background_image_url.trim()
    ? JSON.stringify({ image_url: form.background_image_url.trim() })
    : "",
  example_questions: form.example_questions_text
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean),
  default_model_id: form.default_model_id.trim(),
  agent_memory_enabled: form.agent_memory_enabled,
  tool_ids: form.tool_ids,
  skill_names: form.skill_names,
  knowledge_base_names: form.knowledge_base_names,
  sub_agent_ids: form.sub_agent_ids,
});

// fromDetail converts persisted backend detail into editor form state.
const fromDetail = (detail: AgentDetail): AgentFormState => ({
  name: detail.name ?? "",
  description: detail.description ?? "",
  agent_type: detail.agent_type,
  system_prompt: detail.system_prompt ?? "",
  example_questions_text: Array.isArray(detail.example_questions)
    ? detail.example_questions.join("\n")
    : "",
  default_model_id: detail.default_model_id ?? "",
  agent_memory_enabled: Boolean(detail.agent_memory_enabled),
  avatar_url: detail.avatar_url ?? "",
  background_image_url: parseBackgroundImageURL(detail.chat_background_json),
  tool_ids: Array.isArray(detail.tool_ids) ? detail.tool_ids : [],
  skill_names: Array.isArray(detail.skill_names) ? detail.skill_names : [],
  knowledge_base_names: Array.isArray(detail.knowledge_base_names)
    ? detail.knowledge_base_names
    : [],
  sub_agent_ids: Array.isArray(detail.sub_agent_ids) ? detail.sub_agent_ids : [],
});

function AgentEditorContent({
  token,
  userId,
  userName,
  userAvatar,
}: AgentEditorContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const editingAgentId = (searchParams.get("agent_id") || "").trim();
  const { renderActivityMessage } = useRenderActivityMessage();

  const requestContext = useMemo(
    () => ({
      router,
      userId,
    }),
    [router, userId]
  );

  const [form, setForm] = useState<AgentFormState>(EMPTY_FORM);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pageError, setPageError] = useState("");

  const [modelOptions, setModelOptions] = useState<ResourceOption[]>([]);
  const [toolOptions, setToolOptions] = useState<ResourceOption[]>([]);
  const [skillOptions, setSkillOptions] = useState<ResourceOption[]>([]);
  const [kbOptions, setKbOptions] = useState<ResourceOption[]>([]);
  const [subAgentOptions, setSubAgentOptions] = useState<ResourceOption[]>([]);

  const [bindingPickerType, setBindingPickerType] =
    useState<BindingPickerType | null>(null);
  const [bindingSearch, setBindingSearch] = useState("");
  const [avatarModalOpen, setAvatarModalOpen] = useState(false);
  const [moreOptionsOpen, setMoreOptionsOpen] = useState(false);

  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [uploadingBackground, setUploadingBackground] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);
  const backgroundInputRef = useRef<HTMLInputElement | null>(null);

  const inputRef = useRef<{ setContent: (content: string) => void } | null>(null);
  const dialogueWrapperRef = useRef<HTMLDivElement | null>(null);
  const currentRunIdRef = useRef("");
  const textMessageMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const toolCallMapRef = useRef(new Map<string, { runId: string; index: number }>());
  const reasoningMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );
  const activityMessageMapRef = useRef(
    new Map<string, { runId: string; index: number }>()
  );

  const [threadId, setThreadId] = useState(() => createThreadId());
  const [conversationMessages, setConversationMessages] = useState<AguiMessage[]>([]);
  const [semiMessages, setSemiMessages] = useState<SemiMessage[]>([]);
  const [chatRunning, setChatRunning] = useState(false);
  const [chatError, setChatError] = useState("");

  const effectiveAvatarURL =
    (form.avatar_url || "").trim() || SITE_DEFAULT_AVATAR_URL;
  const chatBackgroundStyle = useMemo(() => {
    const bg = (form.background_image_url || "").trim();
    if (!bg) {
      return undefined;
    }
    return {
      backgroundImage: `linear-gradient(rgba(255,255,255,0.80), rgba(255,255,255,0.84)), url(${bg})`,
      backgroundSize: "cover",
      backgroundPosition: "center",
    } as CSSProperties;
  }, [form.background_image_url]);

  const setComposerText = useCallback((text: string) => {
    inputRef.current?.setContent(toEditorParagraphHtml(text));
  }, []);

  const {
    uploadInputRef,
    pendingUploads,
    uploading,
    uploadError,
    handleOpenUploadPicker,
    handleUploadInputChange,
    removePendingUpload,
    clearUploadError,
    clearPendingUploads,
    resetUploads,
    restorePendingUploads,
  } = usePendingUploads({
    threadId,
    router,
    userId,
    uploadsBlocked: chatRunning,
  });

  const handleComposerContentChange = useCallback(() => {
    if (chatError) {
      setChatError("");
    }
  }, [chatError]);

  const clearRuntimeMaps = useCallback(() => {
    currentRunIdRef.current = "";
    textMessageMapRef.current.clear();
    toolCallMapRef.current.clear();
    reasoningMessageMapRef.current.clear();
    activityMessageMapRef.current.clear();
  }, []);

  const updateRunMessage = useCallback(
    (runId: string, updater: (message: SemiMessage) => SemiMessage) => {
      if (!runId) {
        return;
      }
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

  // 调试历史只保留前端会话快照，下一轮把已完成的 user/assistant 消息重新送回后端。
  const upsertAssistantHistoryText = useCallback(
    (
      runId: string,
      updater: (previousText: string) => string,
      options?: { removeIfEmpty?: boolean }
    ) => {
      if (!runId) {
        return;
      }
      setConversationMessages((current) => {
        const next = [...current];
        const index = next.findIndex(
          (message) => message.id === runId && message.role === "assistant"
        );
        const currentText =
          index >= 0 && typeof next[index]?.content === "string"
            ? next[index].content
            : "";
        const nextText = updater(currentText);
        if (!nextText && options?.removeIfEmpty) {
          if (index >= 0) {
            next.splice(index, 1);
          }
          return next;
        }
        const nextMessage: AguiMessage = {
          id: runId,
          role: "assistant",
          content: nextText,
        };
        if (index >= 0) {
          next[index] = nextMessage;
        } else {
          next.push(nextMessage);
        }
        return next;
      });
    },
    []
  );

  const completeReasoningItems = useCallback(
    (runId: string) => {
      if (!runId) return;
      updateRunMessage(runId, (message) => {
        const content = Array.isArray(message.content) ? [...message.content] : [];
        let changed = false;
        const nextContent = content.map((item) => {
          if (item?.type !== "reasoning" || item.status === "completed") {
            return item;
          }
          changed = true;
          return {
            ...item,
            status: "completed",
          };
        });
        if (!changed) {
          return message;
        }
        return {
          ...message,
          content: nextContent,
        };
      });
      reasoningMessageMapRef.current.forEach((mapping, messageId) => {
        if (mapping.runId === runId) {
          reasoningMessageMapRef.current.delete(messageId);
        }
      });
    },
    [updateRunMessage]
  );

  const handleDebugEvent = useCallback(
    ({ event }: { event: unknown }) => {
      const rawEvent = event as DebugEventPayload;
      const eventType = String(rawEvent?.type ?? "");
      if (!eventType) {
        return;
      }

      if (eventType === "RUN_STARTED") {
        const runId = resolveRunId(rawEvent);
        if (runId) {
          currentRunIdRef.current = runId;
          updateRunMessage(runId, (message) => message);
        }
        return;
      }

      if (eventType === "RUN_FINISHED") {
        const runId = resolveRunId(rawEvent) || currentRunIdRef.current;
        if (runId) {
          completeReasoningItems(runId);
          updateRunMessage(runId, (message) => ({
            ...message,
            status: "completed",
          }));
          upsertAssistantHistoryText(runId, (text) => text, {
            removeIfEmpty: true,
          });
        }
        clearRuntimeMaps();
        return;
      }

      if (eventType === "REASONING_END") {
        const runId = resolveRunId(rawEvent) || currentRunIdRef.current;
        if (runId) {
          completeReasoningItems(runId);
        }
        return;
      }

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
        upsertAssistantHistoryText(runId, (text) => text);
        return;
      }

      if (eventType === "TEXT_MESSAGE_CONTENT") {
        const messageId = resolveMessageId(rawEvent);
        if (!messageId) return;
        const mapping = textMessageMapRef.current.get(messageId);
        if (!mapping) return;
        const delta = String(rawEvent?.delta ?? "");
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as MessageItemRecord | undefined;
          if (!target || !Array.isArray(target.content)) {
            return message;
          }
          const items = [...target.content];
          const first = items[0] as { text?: unknown } | undefined;
          if (typeof first?.text === "string") {
            items[0] = { ...first, text: `${first.text}${delta}` };
          }
          content[mapping.index] = {
            ...target,
            content: items,
          };
          return { ...message, content };
        });
        upsertAssistantHistoryText(mapping.runId, (text) => `${text}${delta}`);
        return;
      }

      if (eventType === "TEXT_MESSAGE_END") {
        const messageId = resolveMessageId(rawEvent);
        if (!messageId) return;
        const mapping = textMessageMapRef.current.get(messageId);
        if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index];
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
        textMessageMapRef.current.delete(messageId);
        return;
      }

      if (eventType === "REASONING_MESSAGE_START") {
        const runId = currentRunIdRef.current;
        const messageId = resolveMessageId(rawEvent);
        if (!runId || !messageId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "reasoning",
            id: messageId,
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
        const delta = String(rawEvent?.delta ?? "");
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as MessageItemRecord | undefined;
          if (!target || !Array.isArray(target.summary)) {
            return message;
          }
          const items = [...target.summary];
          const first = items[0] as { text?: unknown } | undefined;
          if (typeof first?.text === "string") {
            items[0] = { ...first, text: `${first.text}${delta}` };
          }
          content[mapping.index] = {
            ...target,
            summary: items,
          };
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
          const target = content[mapping.index];
          if (!target) {
            return message;
          }
          content[mapping.index] = { ...target, status: "completed" };
          return { ...message, content };
        });
        reasoningMessageMapRef.current.delete(messageId);
        return;
      }

      if (eventType === "REASONING_ENCRYPTED_VALUE") {
        const messageId = resolveMessageId(rawEvent);
        if (!messageId) return;
        const mapping = reasoningMessageMapRef.current.get(messageId);
        if (!mapping) return;
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as MessageItemRecord | undefined;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            encryptedValue: rawEvent?.encryptedValue,
          };
          return { ...message, content };
        });
        return;
      }

      if (eventType === "TOOL_CALL_START") {
        const runId = currentRunIdRef.current;
        const toolCallId = normalizeEventID(
          rawEvent?.toolCallId ?? rawEvent?.toolCall?.id ?? rawEvent?.id
        );
        if (!runId || !toolCallId) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const index = content.length;
          content.push({
            type: "function_call",
            id: toolCallId,
            call_id: toolCallId,
            name: rawEvent?.toolCallName,
            status: "in_progress",
            arguments: "",
          });
          toolCallMapRef.current.set(toolCallId, { runId, index });
          return { ...message, content };
        });
        return;
      }

      if (eventType === "TOOL_CALL_ARGS") {
        const toolCallId = normalizeEventID(
          rawEvent?.toolCallId ?? rawEvent?.toolCall?.id ?? rawEvent?.id
        );
        if (!toolCallId) return;
        const mapping = toolCallMapRef.current.get(toolCallId);
        if (!mapping) return;
        const delta = String(rawEvent?.delta ?? "");
        updateRunMessage(mapping.runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const target = content[mapping.index] as MessageItemRecord | undefined;
          if (!target) return message;
          content[mapping.index] = {
            ...target,
            arguments: `${target.arguments ?? ""}${delta}`,
          };
          return { ...message, content };
        });
        return;
      }

      if (eventType === "TOOL_CALL_END") {
        const toolCallId = normalizeEventID(
          rawEvent?.toolCallId ?? rawEvent?.toolCall?.id ?? rawEvent?.id
        );
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
          rawEvent?.result ?? rawEvent?.content
        );
        if (!outputText) return;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          content.push({
            type: TOOL_RESULT_TYPE,
            id: createMessageId(),
            text: outputText,
            status: "completed",
          });
          return { ...message, content };
        });
        return;
      }

      if (
        eventType === ACTIVITY_EVENT_SNAPSHOT ||
        eventType === ACTIVITY_EVENT_DELTA
      ) {
        const runId = resolveRunId(rawEvent) || currentRunIdRef.current;
        const activityType =
          typeof rawEvent?.activityType === "string" ? rawEvent.activityType : "";
        const activityMessageId = resolveActivityMessageId(rawEvent);
        if (!runId || !activityType || !activityMessageId) {
          return;
        }
        const activityEventType =
          eventType === ACTIVITY_EVENT_DELTA
            ? ACTIVITY_EVENT_DELTA
            : ACTIVITY_EVENT_SNAPSHOT;
        updateRunMessage(runId, (message) => {
          const content = Array.isArray(message.content) ? [...message.content] : [];
          const mapped = activityMessageMapRef.current.get(activityMessageId);
          let index =
            mapped && mapped.runId === runId ? mapped.index : -1;
          if (
            index < 0 ||
            index >= content.length ||
            (content[index] as { activityMessageId?: string })?.activityMessageId !==
              activityMessageId
          ) {
            index = content.findIndex(
              (item) =>
                (item as { type?: string }).type === ACTIVITY_CONTENT_TYPE &&
                (item as { activityMessageId?: string }).activityMessageId ===
                  activityMessageId
            );
          }
          const previousItem =
            index === -1 ? null : (content[index] as MessageItemRecord);
          const mappedContent = mapActivityContent({
            activityType,
            eventType: activityEventType,
            content: rawEvent?.content,
            patch: rawEvent?.patch,
            replace: rawEvent?.replace,
            previousContent: previousItem?.content,
          });
          const activityItem = {
            ...(previousItem ?? {}),
            id: activityMessageId,
            type: ACTIVITY_CONTENT_TYPE,
            activityMessageId,
            activityType,
            activityEventType,
            content: mappedContent,
            status:
              rawEvent?.status ??
              (activityEventType === ACTIVITY_EVENT_DELTA
                ? "in_progress"
                : "completed"),
            timestamp: rawEvent?.timestamp ?? Date.now(),
          };
          if (index === -1) {
            index = content.length;
            content.push(activityItem);
          } else {
            content[index] = activityItem;
          }
          activityMessageMapRef.current.set(activityMessageId, {
            runId,
            index,
          });
          return { ...message, content };
        });
      }
    },
    [
      clearRuntimeMaps,
      completeReasoningItems,
      updateRunMessage,
      upsertAssistantHistoryText,
    ]
  );

  useEffect(() => {
    let cancelled = false;
    const loadOptions = async () => {
      try {
        const [modelsRes, pluginsRes, skillsRes, kbsRes, subAgentsRes] =
          await Promise.all([
            listModelOptions(requestContext),
            listPluginOptions(requestContext),
            listSkillOptions(requestContext),
            listKnowledgeBaseOptions(requestContext),
            listEnabledAgentOptions(requestContext),
          ]);
        if (cancelled) {
          return;
        }
        setModelOptions(
          Array.isArray(modelsRes.data)
            ? modelsRes.data.map((item) => ({
                id: item.model_id,
                label: item.model_name || item.model_id,
                description: item.provider_name || "",
              }))
            : []
        );
        setToolOptions(
          Array.isArray(pluginsRes.data)
            ? pluginsRes.data.flatMap((plugin) =>
                (Array.isArray(plugin.tools) ? plugin.tools : [])
                  .filter((tool) => typeof tool.tool_id === "string" && tool.tool_id)
                  .map((tool) => ({
                    id: tool.tool_id as string,
                    label: tool.tool_name || (tool.tool_id as string),
                    description: plugin.name || tool.description || "",
                  }))
              )
            : []
        );
        const skills = Array.isArray(skillsRes.data?.data)
          ? skillsRes.data.data
          : [];
        setSkillOptions(
          skills
            .map((item) => {
              const name =
                typeof item.name === "string" && item.name.trim()
                  ? item.name.trim()
                  : typeof item.skillName === "string" && item.skillName.trim()
                    ? item.skillName.trim()
                    : typeof item.path === "string" && item.path.trim()
                      ? item.path.trim().split("/").filter(Boolean).pop() || ""
                      : "";
              return name
                ? {
                    id: name,
                    label: name,
                  }
                : null;
            })
            .filter(Boolean) as ResourceOption[]
        );
        setKbOptions(
          Array.isArray(kbsRes.data)
            ? kbsRes.data
                .filter((item) => typeof item.name === "string" && item.name.trim())
                .map((item) => ({
                  id: item.name.trim(),
                  label: item.name.trim(),
                }))
            : []
        );
        const currentID = editingAgentId;
        setSubAgentOptions(
          Array.isArray(subAgentsRes.data)
            ? subAgentsRes.data
                .filter((item) => item.agent_id !== currentID)
                .map((item) => ({
                  id: item.agent_id,
                  label: item.name,
                  description:
                    item.agent_type === "supervisor" ? "Supervisor" : "Single",
                }))
            : []
        );
      } catch (error) {
        setPageError(error instanceof Error ? error.message : "加载资源候选失败");
      }
    };
    void loadOptions();
    return () => {
      cancelled = true;
    };
  }, [editingAgentId, requestContext, token]);

  useEffect(() => {
    if (!editingAgentId) {
      return;
    }
    let cancelled = false;
    const loadDetail = async () => {
      setLoadingDetail(true);
      setPageError("");
      try {
        const detail = await getAgent(editingAgentId, requestContext);
        if (cancelled) {
          return;
        }
        setForm(fromDetail(detail.data as AgentDetail));
      } catch (error) {
        if (!cancelled) {
          setPageError(error instanceof Error ? error.message : "加载 Agent 详情失败");
        }
      } finally {
        if (!cancelled) {
          setLoadingDetail(false);
        }
      }
    };
    void loadDetail();
    return () => {
      cancelled = true;
    };
  }, [editingAgentId, requestContext]);

  const chats = useMemo<SemiMessage[]>(
    () =>
      semiMessages.map((message, index) => ({
        ...message,
        content:
          message.role === "assistant"
            ? groupAssistantProcessItems(message.content)
            : message.content,
        id: message.id ?? `${message.role}-${index}`,
      })),
    [semiMessages]
  );

  useLayoutEffect(() => {
    if (chats.length === 0) {
      return;
    }
    const dialogueContainer =
      dialogueWrapperRef.current?.querySelector<HTMLElement>(
        ".semi-ai-chat-dialogue-list"
      ) ?? null;
    if (!dialogueContainer) {
      return;
    }
    dialogueContainer.scrollTop = dialogueContainer.scrollHeight;
    let secondFrameId = 0;
    const firstFrameId = window.requestAnimationFrame(() => {
      dialogueContainer.scrollTop = dialogueContainer.scrollHeight;
      secondFrameId = window.requestAnimationFrame(() => {
        dialogueContainer.scrollTop = dialogueContainer.scrollHeight;
      });
    });
    return () => {
      window.cancelAnimationFrame(firstFrameId);
      if (secondFrameId) {
        window.cancelAnimationFrame(secondFrameId);
      }
    };
  }, [chats.length]);

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
        name: form.name.trim() || "当前 Agent",
        avatar:
          effectiveAvatarURL ||
          buildTextAvatarDataUrl(form.name.trim() || "AI", {
            background: "#6366F1",
          }),
        color: "indigo",
      },
    }),
    [effectiveAvatarURL, form.name, userAvatar, userName]
  );

  const renderDialogueContentItem = useChatDialogueRenderers(renderActivityMessage);

  const renderActionArea = useCallback(
    (props: { menuItem: ReactNode[]; className: string }) => (
      <ChatInputActionArea
        className={props.className}
        menuItem={props.menuItem}
        conversationMode="agent"
        selectedModelOption={null}
        availableModels={[]}
        selectedModelId=""
        onModelChange={() => undefined}
        onOpenUploadPicker={handleOpenUploadPicker}
        uploadDisabled={chatRunning || uploading}
        pendingUploadCount={pendingUploads.length}
      />
    ),
    [chatRunning, handleOpenUploadPicker, pendingUploads.length, uploading]
  );

  const emptyStatePrompts = useMemo(
    () =>
      form.example_questions_text
        .split("\n")
        .map((item) => item.trim())
        .filter(Boolean)
        .slice(0, 4),
    [form.example_questions_text]
  );

  const showEmptyState = !chatRunning && chats.length === 0;

  const handleSave = useCallback(async () => {
    setSaving(true);
    setPageError("");
    try {
      const payload = toPayload(form);
      if (editingAgentId) {
        await updateAgent(editingAgentId, payload, requestContext);
      } else {
        await createAgent(payload, requestContext);
      }
      router.push("/agents");
    } catch (error) {
      setPageError(error instanceof Error ? error.message : "保存 Agent 失败");
    } finally {
      setSaving(false);
    }
  }, [editingAgentId, form, requestContext, router]);

  const uploadAvatar = useCallback(
    async (file: File) => {
      setUploadingAvatar(true);
      setPageError("");
      try {
        const uploaded = await uploadAgentImage(file, requestContext);
        setForm((current) => ({
          ...current,
          avatar_url: uploaded.url,
        }));
      } catch (error) {
        setPageError(error instanceof Error ? error.message : "上传头像失败");
      } finally {
        setUploadingAvatar(false);
      }
    },
    [requestContext]
  );

  const uploadBackground = useCallback(
    async (file: File) => {
      setUploadingBackground(true);
      setPageError("");
      try {
        const uploaded = await uploadAgentImage(file, requestContext);
        setForm((current) => ({
          ...current,
          background_image_url: uploaded.url,
        }));
      } catch (error) {
        setPageError(error instanceof Error ? error.message : "上传聊天背景失败");
      } finally {
        setUploadingBackground(false);
      }
    },
    [requestContext]
  );

  const handleClearChat = useCallback(() => {
    clearRuntimeMaps();
    setConversationMessages([]);
    setSemiMessages([]);
    setChatError("");
    clearUploadError();
    resetUploads();
    setThreadId(createThreadId());
    setComposerText("");
  }, [clearRuntimeMaps, clearUploadError, resetUploads, setComposerText]);

  const handleMessageSend = useCallback(
    (payload: MessageContent) => {
      if (chatRunning) {
        return;
      }
      if (uploading) {
        setChatError("附件上传中，请稍候");
        return;
      }

      const inputContents = payload?.inputContents ?? [];
      const text = extractInputPlainText(inputContents as AIChatInputContent[]);
      const textExists = text.trim().length > 0;
      if (!textExists && pendingUploads.length === 0) {
        setChatError("请输入内容或上传附件");
        return;
      }

      const uploadedAssets = [...pendingUploads];
      const userMessage: AguiMessage = {
        id: createMessageId(),
        role: "user",
        content: buildAguiUserContent(text, uploadedAssets),
      };
      const nextConversationMessages = [...conversationMessages, userMessage];

      setChatRunning(true);
      setChatError("");
      clearUploadError();
      clearRuntimeMaps();
      setConversationMessages(nextConversationMessages);
      setSemiMessages((prev) => [
        ...prev,
        {
          id: userMessage.id,
          role: "user",
          content: mapAguiUserContentToSemi(userMessage.content),
          status: "completed",
          createdAt: Date.now(),
        },
      ]);
      clearPendingUploads();
      setComposerText("");

      void runAgentDebugSession(
        toPayload(form),
        nextConversationMessages,
        requestContext,
        {
          onEvent: handleDebugEvent,
        }
      )
        .catch((error) => {
          if (uploadedAssets.length > 0) {
            restorePendingUploads(uploadedAssets);
          }
          setChatError(error instanceof Error ? error.message : "测试运行失败");
        })
        .finally(() => {
          setChatRunning(false);
        });
    },
    [
      chatRunning,
      clearPendingUploads,
      clearRuntimeMaps,
      clearUploadError,
      conversationMessages,
      form,
      handleDebugEvent,
      pendingUploads,
      requestContext,
      restorePendingUploads,
      setComposerText,
      uploading,
    ]
  );

  const bindingOptions = useMemo(() => {
    switch (bindingPickerType) {
      case "tool_ids":
        return toolOptions;
      case "skill_names":
        return skillOptions;
      case "knowledge_base_names":
        return kbOptions;
      case "sub_agent_ids":
        return subAgentOptions;
      default:
        return [];
    }
  }, [bindingPickerType, kbOptions, skillOptions, subAgentOptions, toolOptions]);

  const filteredBindingOptions = useMemo(() => {
    const keyword = bindingSearch.trim().toLowerCase();
    if (!keyword) {
      return bindingOptions;
    }
    return bindingOptions.filter((item) => {
      const haystack = `${item.label} ${item.description || ""}`.toLowerCase();
      return haystack.includes(keyword);
    });
  }, [bindingOptions, bindingSearch]);

  const toggleBinding = useCallback((type: BindingPickerType, value: string) => {
    setForm((current) => {
      const exists = current[type].includes(value);
      return {
        ...current,
        [type]: exists
          ? current[type].filter((item) => item !== value)
          : [...current[type], value],
      };
    });
  }, []);

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-0">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <UiButton variant="secondary" onClick={() => router.push("/agents")}>
              返回市场
            </UiButton>
            <div className="text-sm font-semibold text-[var(--color-text-secondary)]">
              {editingAgentId ? "编辑 Agent" : "创建 Agent"}
            </div>
          </div>
          <UiButton onClick={() => void handleSave()} disabled={saving || loadingDetail}>
            {saving ? "保存中..." : "保存"}
          </UiButton>
        </div>

        <div className="mt-4 flex items-start gap-4 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.62)] p-4">
          <button
            type="button"
            className="group relative h-20 w-20 shrink-0 overflow-hidden rounded-[18px] border border-[var(--color-border-default)]"
            onClick={() => setAvatarModalOpen(true)}
            title="点击更换头像"
          >
            <img src={effectiveAvatarURL} alt="Agent Avatar" className="h-full w-full object-cover" />
            <span className="absolute inset-0 hidden items-center justify-center bg-[rgba(15,23,42,0.38)] text-xs text-white group-hover:flex">
              更换
            </span>
          </button>
          <div className="min-w-0 flex-1">
            <UiInput
              className="h-12 border-0 bg-transparent px-0 text-4xl font-semibold leading-none text-[var(--color-text-primary)] shadow-none focus:ring-0"
              value={form.name}
              onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
              placeholder="输入智能体名称"
            />
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className="rounded-[8px] border border-[var(--color-border-default)] bg-white px-2 py-1 text-xs text-[var(--color-text-secondary)]">
                {form.agent_type === "supervisor" ? "Supervisor 模式" : "Single 模式"}
              </span>
              <span className="rounded-[8px] border border-[var(--color-border-default)] bg-white px-2 py-1 text-xs text-[var(--color-text-secondary)]">
                测试会话仅保存在当前页面
              </span>
            </div>
          </div>
        </div>

        {pageError ? (
          <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(255,255,255,0.7)] px-3 py-2 text-sm text-[var(--color-danger)]">
            {pageError}
          </div>
        ) : null}

        <div className="mt-4 grid items-start gap-4 md:grid-cols-[minmax(360px,1fr)_minmax(420px,1.2fr)]">
          <section className="chat-page workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div className="text-sm font-semibold text-[var(--color-text-primary)]">基本信息</div>
              <UiButton variant="secondary" size="sm" onClick={() => setMoreOptionsOpen(true)}>
                更多选项
              </UiButton>
            </div>
            <div className="grid gap-3">
              <UiSelect
                value={form.agent_type}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    agent_type: (
                      event.target.value === "supervisor" ? "supervisor" : "single"
                    ) as AgentType,
                    sub_agent_ids:
                      event.target.value === "supervisor" ? current.sub_agent_ids : [],
                  }))
                }
              >
                <option value="single">Single Agent</option>
                <option value="supervisor">Supervisor Agent</option>
              </UiSelect>
              <UiTextarea
                rows={2}
                value={form.description}
                onChange={(event) =>
                  setForm((current) => ({ ...current, description: event.target.value }))
                }
                placeholder="智能体描述"
              />
              <UiTextarea
                rows={8}
                value={form.system_prompt}
                onChange={(event) =>
                  setForm((current) => ({ ...current, system_prompt: event.target.value }))
                }
                placeholder="系统提示词"
              />
              <UiTextarea
                rows={3}
                value={form.example_questions_text}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    example_questions_text: event.target.value,
                  }))
                }
                placeholder="示例问法，每行一条"
              />
              <UiSelect
                value={form.default_model_id}
                onChange={(event) =>
                  setForm((current) => ({ ...current, default_model_id: event.target.value }))
                }
              >
                <option value="">系统默认模型</option>
                {modelOptions.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.label}
                  </option>
                ))}
              </UiSelect>
            </div>

            <input
              ref={avatarInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0];
                event.target.value = "";
                if (file) {
                  void uploadAvatar(file);
                }
              }}
            />
            <input
              ref={backgroundInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0];
                event.target.value = "";
                if (file) {
                  void uploadBackground(file);
                }
              }}
            />

            <div className="mt-4 space-y-3">
              <BindingSection
                title="工具"
                values={form.tool_ids}
                options={toolOptions}
                onRemove={(value) => toggleBinding("tool_ids", value)}
                onAdd={() => {
                  setBindingSearch("");
                  setBindingPickerType("tool_ids");
                }}
              />
              <BindingSection
                title="Skill"
                values={form.skill_names}
                options={skillOptions}
                onRemove={(value) => toggleBinding("skill_names", value)}
                onAdd={() => {
                  setBindingSearch("");
                  setBindingPickerType("skill_names");
                }}
              />
              <BindingSection
                title="知识库"
                values={form.knowledge_base_names}
                options={kbOptions}
                onRemove={(value) => toggleBinding("knowledge_base_names", value)}
                onAdd={() => {
                  setBindingSearch("");
                  setBindingPickerType("knowledge_base_names");
                }}
              />
              {form.agent_type === "supervisor" ? (
                <BindingSection
                  title="Sub Agent"
                  values={form.sub_agent_ids}
                  options={subAgentOptions}
                  onRemove={(value) => toggleBinding("sub_agent_ids", value)}
                  onAdd={() => {
                    setBindingSearch("");
                    setBindingPickerType("sub_agent_ids");
                  }}
                />
              ) : null}
            </div>
          </section>

          <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="text-base font-semibold text-[var(--color-text-primary)]">
                智能体对话测试
              </div>
              <UiButton
                variant="ghost"
                size="sm"
                onClick={handleClearChat}
                disabled={chatRunning || (chats.length === 0 && pendingUploads.length === 0)}
              >
                清空会话
              </UiButton>
            </div>
            <div
              className="mt-3 h-[520px] overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.66)]"
              style={chatBackgroundStyle}
            >
              {showEmptyState ? (
                <div className="flex h-full items-center justify-center px-6 py-8">
                  <div className="max-w-xl text-center">
                    <div className="mx-auto flex h-11 w-11 items-center justify-center rounded-[18px] border border-[rgba(199,104,67,0.14)] bg-[rgba(255,250,245,0.72)] text-[var(--color-action-primary)]">
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
                    <div className="mt-4">
                      <h2 className="text-[24px] font-semibold leading-none tracking-[-0.04em] text-[var(--color-text-primary)]">
                        {form.name.trim() || "当前 Agent"}
                      </h2>
                      {form.description.trim() ? (
                        <p className="mx-auto mt-3 max-w-lg text-sm leading-6 text-[var(--color-text-secondary)]">
                          {form.description.trim()}
                        </p>
                      ) : null}
                    </div>
                    {emptyStatePrompts.length > 0 ? (
                      <div className="mt-5 flex flex-wrap justify-center gap-2.5">
                        {emptyStatePrompts.map((prompt) => (
                          <button
                            key={prompt}
                            type="button"
                            onClick={() => setComposerText(prompt)}
                            className="rounded-full border border-[rgba(126,96,69,0.16)] bg-[rgba(255,252,247,0.82)] px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)] transition hover:border-[rgba(199,104,67,0.24)] hover:bg-[rgba(255,247,240,0.9)] hover:text-[var(--color-text-primary)]"
                          >
                            {prompt}
                          </button>
                        ))}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : (
                <div ref={dialogueWrapperRef} className="h-full px-2 py-2 md:px-3 md:py-3">
                  <AIChatDialogue
                    align="leftRight"
                    mode="bubble"
                    chats={chats}
                    renderDialogueContentItem={renderDialogueContentItem}
                    roleConfig={roleConfig}
                    className="h-full"
                  />
                </div>
              )}
            </div>
            <div className="mt-3 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,252,247,0.9),rgba(247,237,227,0.96))] p-3">
              {chatRunning ? (
                <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.06)] px-3 py-1 text-xs font-medium text-[var(--color-action-primary)]">
                  <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-current" />
                  AI 正在生成回复
                </div>
              ) : null}
              <div className="relative">
                <ChatComposerAssist
                  mentionOpen={false}
                  mentionCandidates={[]}
                  mentionTriggerSymbol={null}
                  mentionActiveIndex={0}
                  onMentionHover={() => undefined}
                  onMentionSelect={() => undefined}
                  selectedMentions={[]}
                  onRemoveMention={() => undefined}
                  uploadInputRef={uploadInputRef}
                  onUploadInputChange={handleUploadInputChange}
                  pendingUploads={pendingUploads}
                  uploading={uploading}
                  onRemovePendingUpload={removePendingUpload}
                />
                <AIChatInput
                  ref={inputRef as unknown as Ref<unknown>}
                  className="chat-composer-input"
                  keepSkillAfterSend={false}
                  placeholder="输入消息进行调试测试"
                  onContentChange={handleComposerContentChange}
                  onMessageSend={handleMessageSend}
                  generating={chatRunning}
                  canSend={!chatRunning && !uploading}
                  showUploadButton={false}
                  showUploadFile={false}
                  showReference={false}
                  round
                  immediatelyRender={false}
                  renderActionArea={renderActionArea}
                />
              </div>
              {chatError ? (
                <div
                  role="alert"
                  aria-live="polite"
                  className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]"
                >
                  {chatError}
                </div>
              ) : null}
              {uploadError ? (
                <div
                  role="alert"
                  aria-live="polite"
                  className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]"
                >
                  {uploadError}
                </div>
              ) : null}
            </div>
          </section>
        </div>
      </div>

      <UiModal
        open={avatarModalOpen}
        title="更改头像"
        onClose={() => setAvatarModalOpen(false)}
        footer={
          <UiButton variant="secondary" onClick={() => setAvatarModalOpen(false)}>
            关闭
          </UiButton>
        }
      >
        <div className="grid gap-3">
          <div className="mx-auto h-28 w-28 overflow-hidden rounded-[20px] border border-[var(--color-border-default)]">
            <img src={effectiveAvatarURL} alt="头像预览" className="h-full w-full object-cover" />
          </div>
          <div className="flex flex-wrap justify-center gap-2">
            <UiButton
              variant="secondary"
              size="sm"
              onClick={() => avatarInputRef.current?.click()}
              disabled={uploadingAvatar}
            >
              {uploadingAvatar ? "上传中..." : "上传头像"}
            </UiButton>
            <UiButton
              variant="ghost"
              size="sm"
              onClick={() => {
                setForm((current) => ({
                  ...current,
                  avatar_url: SITE_DEFAULT_AVATAR_URL,
                }));
              }}
            >
              使用默认头像
            </UiButton>
          </div>
        </div>
      </UiModal>

      <UiModal
        open={moreOptionsOpen}
        title="更多选项"
        onClose={() => setMoreOptionsOpen(false)}
        footer={
          <UiButton variant="secondary" onClick={() => setMoreOptionsOpen(false)}>
            完成
          </UiButton>
        }
      >
        <div className="grid gap-4">
          <label className="flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--color-border-default)] px-3 py-2 text-sm text-[var(--color-text-secondary)]">
            <input
              type="checkbox"
              checked={form.agent_memory_enabled}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  agent_memory_enabled: event.target.checked,
                }))
              }
            />
            开启 Agent 级长期记忆
          </label>
          <ImageUploadCard
            title="聊天背景"
            imageURL={form.background_image_url}
            uploading={uploadingBackground}
            onPick={() => backgroundInputRef.current?.click()}
            onClear={() => setForm((current) => ({ ...current, background_image_url: "" }))}
          />
        </div>
      </UiModal>

      <UiModal
        open={Boolean(bindingPickerType)}
        title="添加绑定资源"
        onClose={() => setBindingPickerType(null)}
        footer={
          <UiButton variant="secondary" onClick={() => setBindingPickerType(null)}>
            关闭
          </UiButton>
        }
      >
        <UiInput
          value={bindingSearch}
          onChange={(event) => setBindingSearch(event.target.value)}
          placeholder="搜索资源名称"
        />
        <div className="mt-3 max-h-80 space-y-2 overflow-auto pr-1">
          {filteredBindingOptions.length === 0 ? (
            <div className="text-xs text-[var(--color-text-muted)]">没有匹配项</div>
          ) : (
            filteredBindingOptions.map((item) => {
              const type = bindingPickerType as BindingPickerType;
              const checked = form[type]?.includes(item.id);
              return (
                <label
                  key={item.id}
                  className={joinClasses(
                    "flex cursor-pointer items-start gap-3 rounded-[var(--radius-md)] border px-3 py-2 text-sm",
                    checked
                      ? "border-[rgba(37,99,255,0.24)] bg-[rgba(37,99,255,0.06)]"
                      : "border-[var(--color-border-default)]"
                  )}
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggleBinding(type, item.id)}
                    className="mt-1"
                  />
                  <span className="min-w-0">
                    <span className="block font-medium text-[var(--color-text-primary)]">
                      {item.label}
                    </span>
                    {item.description ? (
                      <span className="mt-0.5 block text-xs text-[var(--color-text-muted)]">
                        {item.description}
                      </span>
                    ) : null}
                  </span>
                </label>
              );
            })
          )}
        </div>
      </UiModal>
    </div>
  );
}

export default function AgentEditorPage() {
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
      <AgentEditorContent
        token={token}
        userId={userId}
        userName={userName}
        userAvatar={userAvatar}
      />
    </CopilotKitProvider>
  );
}

function ImageUploadCard({
  title,
  imageURL,
  uploading,
  onPick,
  onClear,
}: {
  title: string;
  imageURL: string;
  uploading: boolean;
  onPick: () => void;
  onClear: () => void;
}) {
  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] p-3">
      <div className="text-xs font-semibold text-[var(--color-text-secondary)]">{title}</div>
      <div className="mt-2">
        {imageURL ? (
          <img src={imageURL} alt={title} className="h-24 w-full rounded-[var(--radius-sm)] border object-cover" />
        ) : (
          <div className="flex h-24 items-center justify-center rounded-[var(--radius-sm)] border border-dashed text-xs text-[var(--color-text-muted)]">
            暂无图片
          </div>
        )}
      </div>
      <div className="mt-2 flex gap-2">
        <UiButton variant="secondary" size="sm" onClick={onPick} disabled={uploading}>
          {uploading ? "上传中..." : "上传图片"}
        </UiButton>
        {imageURL ? (
          <UiButton variant="ghost" size="sm" onClick={onClear}>
            清空
          </UiButton>
        ) : null}
      </div>
    </div>
  );
}

function BindingSection({
  title,
  values,
  options,
  onAdd,
  onRemove,
}: {
  title: string;
  values: string[];
  options: ResourceOption[];
  onAdd: () => void;
  onRemove: (value: string) => void;
}) {
  const optionMap = useMemo(() => {
    const map = new Map<string, ResourceOption>();
    for (const option of options) {
      map.set(option.id, option);
    }
    return map;
  }, [options]);

  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="text-sm font-semibold text-[var(--color-text-primary)]">
          {title}（{values.length}）
        </div>
        <UiButton variant="secondary" size="sm" onClick={onAdd}>
          添加
        </UiButton>
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        {values.length === 0 ? (
          <span className="text-xs text-[var(--color-text-muted)]">尚未绑定</span>
        ) : (
          values.map((value) => (
            <span
              key={value}
              className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.75)] px-3 py-1 text-xs"
            >
              <span>{optionMap.get(value)?.label || value}</span>
              <button
                type="button"
                onClick={() => onRemove(value)}
                className="text-[var(--color-text-muted)] hover:text-[var(--color-danger)]"
              >
                ×
              </button>
            </span>
          ))
        )}
      </div>
    </div>
  );
}
