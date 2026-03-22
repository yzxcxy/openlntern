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
} from "@douyinfe/semi-ui-19";
import type { Message as SemiMessage } from "@douyinfe/semi-ui-19/lib/es/aiChatDialogue/interface";
import type { Message as AguiMessage } from "@ag-ui/client";
import type {
  Content as AIChatInputContent,
  MessageContent,
} from "@douyinfe/semi-ui-19/lib/es/aiChatInput/interface";
import { ChatComposerAssist } from "./ChatComposerAssist";
import { ChatComposerExpandModal } from "./ChatComposerExpandModal";
import { ChatInputActionArea } from "./ChatInputActionArea";
import { ChatModeConfigureArea } from "./ChatModeConfigureArea";
import { PluginSelectionModal } from "./PluginSelectionModal";
import { useChatDialogueRenderers } from "./useChatDialogueRenderers";
import { useMentionComposer } from "./useMentionComposer";
import { usePendingUploads } from "./usePendingUploads";
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
import {
  type BackendEnvelope,
  buildAuthHeaders,
  fetchBackend,
  getUserIdFromToken,
  readStoredUser,
  readValidToken,
  requestBackend,
  type StoredUser,
} from "../auth";
import { dispatchThreadHistoryUpsert } from "../thread-history-events";
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
  extractText,
  extractToolResultText,
  groupAssistantProcessItems,
  mapAguiUserContentToSemi,
  mapActivityContent,
  mapHistoryMessages,
  toEditorParagraphHtml,
  type BackendMessagePage,
  type BackendThreadItem,
  type KnowledgeBaseOption,
  type MentionTargetOption,
  type ModelCatalogOption,
  type SkillCatalogItem,
} from "./chat-helpers";

const A2UI_MESSAGE_RENDERER = createA2UIMessageRenderer({ theme });
// 需要稳定引用，避免 renderActivityMessages 触发不稳定数组报错
const ACTIVITY_RENDERERS = [A2UI_MESSAGE_RENDERER];

type ChatContentProps = {
  token: string;
  userId: string;
  userName: string;
  userAvatar: string;
};

function ChatContent({ token, userId, userName, userAvatar }: ChatContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const inputRef = useRef<{ setContent: (content: string) => void } | null>(null);
  const fallbackThreadIdRef = useRef<string>("");
  const latestThreadIdRef = useRef("");
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
  const [composerDraft, setComposerDraft] = useState("");
  const [composerExpandOpen, setComposerExpandOpen] = useState(false);
  const [composerExpandDraft, setComposerExpandDraft] = useState("");
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

  const setComposerText = useCallback((text: string) => {
    setComposerDraft(text);
    inputRef.current?.setContent(toEditorParagraphHtml(text));
  }, []);

  const {
    selectedMentions,
    mentionOpen,
    mentionActiveIndex,
    mentionTriggerSymbol,
    mentionCandidates,
    handleInputContentChange,
    handleMentionSelect,
    handleMentionKeyDownCapture,
    removeMentionSelection,
    closeMentionMenu,
    clearComposerDraft,
    resetMentionComposer,
    setMentionActiveIndex,
  } = useMentionComposer({
    mentionOptions,
    setComposerText,
  });

  const handleComposerContentChange = useCallback(
    (contents: AIChatInputContent[]) => {
      const text = extractInputPlainText(contents ?? []);
      setComposerDraft(text);
      handleInputContentChange(contents);
    },
    [handleInputContentChange]
  );

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
    uploadsBlocked: agent.isRunning,
  });

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
          const data = await requestBackend<BackendThreadItem>(`/v1/threads/${targetThreadId}`, {
            fallbackMessage: "获取会话信息失败",
            router,
            userId,
          });
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
    latestThreadIdRef.current = threadId;
  }, [threadId]);

  useEffect(() => {
    let active = true;
    const loadModelCatalog = async () => {
      if (!token) return;
      try {
        const data = await requestBackend<ModelCatalogOption[]>("/v1/models/catalog", {
          fallbackMessage: "模型目录加载失败",
          router,
          userId,
        });
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
        const [skillsData, kbsData] = await Promise.all([
          requestBackend<{
            data?: SkillCatalogItem[];
          }>("/v1/skills/meta?page=1&page_size=500", {
            fallbackMessage: "技能列表加载失败",
            router,
            userId,
          }),
          requestBackend<KnowledgeBaseOption[]>("/v1/kbs", {
            fallbackMessage: "知识库列表加载失败",
            router,
            userId,
          }),
        ]);

        const options: MentionTargetOption[] = [];
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
        const data = await requestBackend<ChatPluginOption[]>("/v1/plugins/available-for-chat", {
          fallbackMessage: "插件列表加载失败",
          router,
          userId,
        });
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
      setComposerDraft("");
      setComposerExpandDraft("");
      setComposerExpandOpen(false);
      setSemiMessages([]);
      setHistoryLoading(false);
      // 新会话不继承上一会话的 @知识库 / #skill 选择，避免上下文串线。
      resetMentionComposer();
      resetUploads();
      currentRunIdRef.current = "";
      textMessageMapRef.current.clear();
      toolCallMapRef.current.clear();
      reasoningMessageMapRef.current.clear();
      activityMessageMapRef.current.clear();
    }
  }, [
    agent,
    clearThreadTitleSync,
    resetMentionComposer,
    resetUploads,
    resolvedThreadId,
    threadId,
  ]);

  // 拉取历史消息并映射为 SemiMessage[]
  const loadHistory = useCallback(async () => {
    if (!token || !threadId) return;
    setHistoryLoading(true);
    setInputError("");
    try {
      const response = await fetchBackend(`/v1/threads/${threadId}/messages`, {
        fallbackMessage: "历史消息加载失败",
        router,
        userId,
      });
      const data = (await response
        .json()
        .catch(() => null)) as BackendEnvelope<BackendMessagePage> | null;
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
            completeReasoningItems(runId);
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

        if (eventType === "REASONING_END") {
          const runId = resolveRunId(rawEvent) ?? currentRunIdRef.current;
          if (runId) {
            completeReasoningItems(runId);
          }
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
            id: createMessageId(),
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
  }, [agent, completeReasoningItems, startThreadTitleSync, threadId, updateRunMessage]);

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
  const showHistorySkeleton = historyLoading && chats.length === 0;
  const showEmptyState =
    !historyLoading && !agent.isRunning && chats.length === 0 && !inputError;
  const emptyStatePrompts = [
    "介绍一下你能做什么",
    "帮我总结一份文档",
    "帮我分析一个报错",
    "给我写一个接口设计",
  ];
  const openComposerExpandModal = useCallback(() => {
    setComposerExpandDraft(composerDraft);
    setComposerExpandOpen(true);
  }, [composerDraft]);
  const closeComposerExpandModal = useCallback(() => {
    setComposerExpandOpen(false);
  }, []);
  const applyExpandedComposerDraft = useCallback(() => {
    setComposerText(composerExpandDraft);
    setComposerExpandOpen(false);
  }, [composerExpandDraft, setComposerText]);

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

  const renderDialogueContentItem = useChatDialogueRenderers(renderActivityMessage);

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
      <ChatInputActionArea
        className={props.className}
        menuItem={props.menuItem}
        conversationMode={conversationMode}
        selectedModelOption={selectedModelOption}
        availableModels={availableModels}
        selectedModelId={selectedModelId}
        onModelChange={(nextModelId) => {
          const nextItem = availableModels.find((item) => item.model_id === nextModelId);
          setSelectedModelId(nextModelId);
          setSelectedProviderId(nextItem?.provider_id ?? "");
        }}
        onOpenUploadPicker={handleOpenUploadPicker}
        uploadDisabled={agent.isRunning || uploading}
        pendingUploadCount={pendingUploads.length}
      />
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
      const messageThreadId = threadId;
      const messageContent = buildAguiUserContent(text, uploadedAssets);
      setInputError("");
      clearUploadError();
      closeMentionMenu();
      clearComposerDraft();
      setComposerDraft("");
      setComposerExpandDraft("");
      setComposerExpandOpen(false);
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
      clearPendingUploads();
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
          if (
            uploadedAssets.length > 0 &&
            latestThreadIdRef.current === messageThreadId
          ) {
            restorePendingUploads(uploadedAssets);
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
      clearComposerDraft,
      clearPendingUploads,
      clearUploadError,
      closeMentionMenu,
      conversationMode,
      pluginMode,
      restorePendingUploads,
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
      <div className="chat-page chat-stage-surface flex h-full w-full flex-col p-0">
        <div className="motion-safe-fade-in relative z-[1] flex h-full min-h-0 w-full flex-col gap-0">
          <div className="chat-shell-surface motion-safe-lift flex min-h-0 flex-1 flex-col overflow-hidden rounded-[28px] border backdrop-blur-sm">
            <div className="flex-1 overflow-hidden bg-[linear-gradient(180deg,rgba(255,255,255,0.36),rgba(248,250,252,0.18))] px-2 py-2 md:px-3 md:py-3">
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
                  <div className="max-w-2xl rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,250,252,0.94))] p-6 text-center shadow-[var(--shadow-sm)]">
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
                    <div className="mt-6 flex flex-wrap justify-center gap-3">
                      {emptyStatePrompts.map((prompt) => (
                        <button
                          key={prompt}
                          type="button"
                          onClick={() => setComposerText(prompt)}
                          className="motion-safe-highlight rounded-full border border-[rgba(191,219,254,0.6)] bg-[rgba(255,255,255,0.88)] px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)] hover:border-[rgba(59,130,246,0.28)] hover:bg-[rgba(239,246,255,0.9)] hover:text-[var(--color-text-primary)]"
                        >
                          {prompt}
                        </button>
                      ))}
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
            <div className="border-t border-[rgba(226,232,240,0.88)] bg-[linear-gradient(180deg,rgba(255,255,255,0.9),rgba(248,250,252,0.98))] px-4 py-3 md:px-5 md:py-4">
              {agent.isRunning && (
                <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-[rgba(37,99,255,0.14)] bg-[rgba(37,99,255,0.06)] px-3 py-1 text-xs font-medium text-[var(--color-action-primary)]">
                  <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-current" />
                  AI 正在生成回复
                </div>
              )}
              <div className="relative" onKeyDownCapture={handleMentionKeyDownCapture}>
                <button
                  type="button"
                  onClick={openComposerExpandModal}
                  className="absolute right-3 top-3 z-10 inline-flex h-7 w-7 items-center justify-center rounded-full border border-[rgba(203,213,225,0.92)] bg-[rgba(248,250,252,0.92)] text-[var(--color-text-muted)] shadow-sm transition hover:border-[rgba(148,163,184,0.9)] hover:bg-[rgba(255,255,255,0.98)] hover:text-[var(--color-text-secondary)]"
                  aria-label="展开输入框编辑"
                  title="展开编辑"
                >
                  <svg
                    viewBox="0 0 24 24"
                    className="h-3.5 w-3.5"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.8"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M8 3H3v5" />
                    <path d="M16 3h5v5" />
                    <path d="M3 16v5h5" />
                    <path d="M21 16v5h-5" />
                    <path d="M8 8 3 3" />
                    <path d="m16 8 5-5" />
                    <path d="m8 16-5 5" />
                    <path d="m16 16 5 5" />
                  </svg>
                </button>
                <ChatComposerAssist
                  mentionOpen={mentionOpen}
                  mentionCandidates={mentionCandidates}
                  mentionTriggerSymbol={mentionTriggerSymbol}
                  mentionActiveIndex={mentionActiveIndex}
                  onMentionHover={setMentionActiveIndex}
                  onMentionSelect={handleMentionSelect}
                  selectedMentions={selectedMentions}
                  onRemoveMention={removeMentionSelection}
                  uploadInputRef={uploadInputRef}
                  onUploadInputChange={handleUploadInputChange}
                  pendingUploads={pendingUploads}
                  uploading={uploading}
                  onRemovePendingUpload={removePendingUpload}
                />
                <AIChatInput
                  ref={inputRef as any}
                  className="chat-composer-input"
                  keepSkillAfterSend={false}
                  placeholder="输入消息；@ 选择知识库，# 选择 Skill"
                  onContentChange={handleComposerContentChange}
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
      <ChatComposerExpandModal
        open={composerExpandOpen}
        value={composerExpandDraft}
        onChange={setComposerExpandDraft}
        onClose={closeComposerExpandModal}
        onApply={applyExpandedComposerDraft}
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
