"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";
import { UiModal } from "../../../components/ui/UiModal";
import { UiSelect } from "../../../components/ui/UiSelect";
import { UiTextarea } from "../../../components/ui/UiTextarea";
import { getUserIdFromToken, readValidToken } from "../../auth";
import {
  type AgentDebugMessage,
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

type BindingPickerType = "tool_ids" | "skill_names" | "knowledge_base_names" | "sub_agent_ids";
const SITE_DEFAULT_AVATAR_URL = "/OpenIntern.png";

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

const resolveDebugEventMessageID = (event: unknown): string => {
  const payload = event as { messageId?: string; message_id?: string } | null;
  if (!payload) {
    return "";
  }
  return String(payload.messageId || payload.message_id || "").trim();
};

const extractRuntimeMessageText = (content: unknown): string => {
  if (typeof content === "string") {
    return content;
  }
  if (Array.isArray(content)) {
    return content
      .map((part) => {
        if (typeof part === "string") {
          return part;
        }
        if (!part || typeof part !== "object") {
          return "";
        }
        const typedPart = part as {
          type?: unknown;
          text?: unknown;
          content?: unknown;
        };
        const partType = String(typedPart.type || "").toLowerCase();
        if (typeof typedPart.text === "string" && (!partType || partType.includes("text"))) {
          return typedPart.text;
        }
        if (typedPart.content !== undefined) {
          return extractRuntimeMessageText(typedPart.content);
        }
        return "";
      })
      .join("");
  }
  if (content && typeof content === "object") {
    const typedContent = content as { text?: unknown; content?: unknown };
    if (typeof typedContent.text === "string") {
      return typedContent.text;
    }
    if (typedContent.content !== undefined) {
      return extractRuntimeMessageText(typedContent.content);
    }
  }
  return "";
};

const mapAgentRuntimeMessagesToDebug = (
  messages: Array<{ id?: string; role?: string; content?: unknown }>
): AgentDebugMessage[] => {
  const seen = new Set<string>();
  const result: AgentDebugMessage[] = [];
  for (const item of messages) {
    if (!item) {
      continue;
    }
    const role = String(item.role || "").trim() as AgentDebugMessage["role"];
    if (!["user", "assistant", "system"].includes(role)) {
      continue;
    }
    const id = String(item.id || `${role}-${Date.now()}`).trim();
    if (!id || seen.has(id)) {
      continue;
    }
    const text = extractRuntimeMessageText(item.content).trim();
    if (!text) {
      continue;
    }
    seen.add(id);
    result.push({
      id,
      role,
      content: text,
    });
  }
  return result;
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
  knowledge_base_names: Array.isArray(detail.knowledge_base_names) ? detail.knowledge_base_names : [],
  sub_agent_ids: Array.isArray(detail.sub_agent_ids) ? detail.sub_agent_ids : [],
});

export default function AgentEditorPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = readValidToken(router);
  const userId = getUserIdFromToken(token);
  const editingAgentId = (searchParams.get("agent_id") || "").trim();

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

  const [bindingPickerType, setBindingPickerType] = useState<BindingPickerType | null>(null);
  const [bindingSearch, setBindingSearch] = useState("");
  const [avatarModalOpen, setAvatarModalOpen] = useState(false);
  const [moreOptionsOpen, setMoreOptionsOpen] = useState(false);

  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [uploadingBackground, setUploadingBackground] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);
  const backgroundInputRef = useRef<HTMLInputElement | null>(null);

  const [chatMessages, setChatMessages] = useState<AgentDebugMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [chatRunning, setChatRunning] = useState(false);
  const [chatError, setChatError] = useState("");
  const chatScrollRef = useRef<HTMLDivElement | null>(null);
  const visibleChatMessages = useMemo(
    () =>
      chatMessages.filter((message) =>
        message.role === "assistant" ? message.content.trim().length > 0 : true
      ),
    [chatMessages]
  );

  const effectiveAvatarURL = (form.avatar_url || "").trim() || SITE_DEFAULT_AVATAR_URL;
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

  useEffect(() => {
    if (!token) {
      return;
    }
    let cancelled = false;
    const loadOptions = async () => {
      // Load all binding candidates once, then use a searchable picker for selection.
      try {
        const [modelsRes, pluginsRes, skillsRes, kbsRes, subAgentsRes] = await Promise.all([
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
        const skills = Array.isArray(skillsRes.data?.data) ? skillsRes.data.data : [];
        setSkillOptions(
          skills
            .map((item) => {
              const name = typeof item.name === "string" && item.name.trim()
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
                  description: item.agent_type === "supervisor" ? "Supervisor" : "Single",
                }))
            : []
        );
      } catch (err) {
        setPageError(err instanceof Error ? err.message : "加载资源候选失败");
      }
    };
    void loadOptions();
    return () => {
      cancelled = true;
    };
  }, [editingAgentId, requestContext, token]);

  useEffect(() => {
    if (!token || !editingAgentId) {
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
      } catch (err) {
        if (!cancelled) {
          setPageError(err instanceof Error ? err.message : "加载 Agent 详情失败");
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
  }, [editingAgentId, requestContext, token]);

  useEffect(() => {
    if (!chatScrollRef.current) {
      return;
    }
    chatScrollRef.current.scrollTop = chatScrollRef.current.scrollHeight;
  }, [chatMessages]);

  const handleSave = useCallback(async () => {
    if (!token) {
      return;
    }
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
    } catch (err) {
      setPageError(err instanceof Error ? err.message : "保存 Agent 失败");
    } finally {
      setSaving(false);
    }
  }, [editingAgentId, form, requestContext, router, token]);

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
      } catch (err) {
        setPageError(err instanceof Error ? err.message : "上传头像失败");
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
      } catch (err) {
        setPageError(err instanceof Error ? err.message : "上传聊天背景失败");
      } finally {
        setUploadingBackground(false);
      }
    },
    [requestContext]
  );

  const runTestChat = useCallback(async () => {
    const inputText = chatInput.trim();
    if (!inputText) {
      return;
    }
    setChatError("");
    setChatRunning(true);
    const userMessage: AgentDebugMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content: inputText,
    };
    const nextMessages = [...chatMessages, userMessage];
    setChatMessages(nextMessages);
    setChatInput("");

    try {
      // Keep test history entirely on the client and only send plain chat messages to backend.
      await runAgentDebugSession(toPayload(form), nextMessages, requestContext, {
        onTextMessageStartEvent: ({ event }) => {
          const messageID = resolveDebugEventMessageID(event);
          if (!messageID) {
            return;
          }
          setChatMessages((current) => {
            if (current.some((item) => item.id === messageID)) {
              return current;
            }
            return [
              ...current,
              {
                id: messageID,
                role: "assistant",
                content: "",
              },
            ];
          });
        },
        onTextMessageContentEvent: ({ event, textMessageBuffer }) => {
          const messageID = resolveDebugEventMessageID(event);
          if (!messageID) {
            return;
          }
          setChatMessages((current) => {
            if (!current.some((item) => item.id === messageID)) {
              return [
                ...current,
                {
                  id: messageID,
                  role: "assistant",
                  content: textMessageBuffer,
                },
              ];
            }
            return current.map((item) =>
              item.id === messageID
                ? {
                    ...item,
                    content: textMessageBuffer,
                  }
                : item
            );
          });
        },
        onMessagesChanged: ({ messages }) => {
          const normalized = mapAgentRuntimeMessagesToDebug(
            messages as Array<{ id?: string; role?: string; content?: unknown }>
          );
          if (normalized.length === 0) {
            return;
          }
          setChatMessages((current) => {
            const currentMap = new Map(current.map((item) => [item.id, item] as const));
            for (const message of normalized) {
              currentMap.set(message.id, message);
            }
            return Array.from(currentMap.values());
          });
        },
        onRunErrorEvent: ({ event }) => {
          setChatError(event.message || "测试运行失败");
        },
      });
    } catch (err) {
      setChatError(err instanceof Error ? err.message : "测试运行失败");
    } finally {
      setChatRunning(false);
    }
  }, [chatInput, chatMessages, form, requestContext]);

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
        [type]: exists ? current[type].filter((item) => item !== value) : [...current[type], value],
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
                自动保存于本地草稿
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
          <section className="workspace-item-surface rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4">
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
                    agent_type: (event.target.value === "supervisor" ? "supervisor" : "single") as AgentType,
                    sub_agent_ids: event.target.value === "supervisor" ? current.sub_agent_ids : [],
                  }))
                }
              >
                <option value="single">Single Agent</option>
                <option value="supervisor">Supervisor Agent</option>
              </UiSelect>
              <UiTextarea
                rows={2}
                value={form.description}
                onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
                placeholder="智能体描述"
              />
              <UiTextarea
                rows={8}
                value={form.system_prompt}
                onChange={(event) => setForm((current) => ({ ...current, system_prompt: event.target.value }))}
                placeholder="系统提示词"
              />
              <UiTextarea
                rows={3}
                value={form.example_questions_text}
                onChange={(event) =>
                  setForm((current) => ({ ...current, example_questions_text: event.target.value }))
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
              <div className="text-base font-semibold text-[var(--color-text-primary)]">智能体对话测试</div>
              <UiButton
                variant="ghost"
                size="sm"
                onClick={() => {
                  setChatMessages([]);
                  setChatError("");
                }}
                disabled={chatRunning || chatMessages.length === 0}
              >
                清空会话
              </UiButton>
            </div>
            <div
              ref={chatScrollRef}
              className="mt-3 h-[520px] overflow-auto rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.66)] p-3"
              style={chatBackgroundStyle}
            >
              {visibleChatMessages.length === 0 ? (
                <div className="text-sm text-[var(--color-text-muted)]">这里是完整测试聊天区，支持多轮对话。</div>
              ) : (
                <div className="space-y-3">
                  {visibleChatMessages.map((message) => (
                    <div
                      key={message.id}
                      className={joinClasses(
                        "max-w-[88%] rounded-[var(--radius-md)] border px-3 py-2 text-sm whitespace-pre-wrap",
                        message.role === "user"
                          ? "ml-auto border-[rgba(37,99,255,0.24)] bg-[rgba(37,99,255,0.08)]"
                          : "mr-auto border-[var(--color-border-default)] bg-white"
                      )}
                    >
                      {message.content}
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="mt-3 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-white p-3">
              <UiTextarea
                rows={4}
                value={chatInput}
                onChange={(event) => setChatInput(event.target.value)}
                placeholder="输入消息进行调试测试"
                disabled={chatRunning}
              />
              <div className="mt-2 flex justify-end">
                <UiButton onClick={() => void runTestChat()} disabled={chatRunning}>
                  {chatRunning ? "测试中..." : "发送"}
                </UiButton>
              </div>
            </div>
            {chatError ? (
              <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(255,255,255,0.7)] px-3 py-2 text-sm text-[var(--color-danger)]">
                {chatError}
              </div>
            ) : null}
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
                setForm((current) => ({ ...current, avatar_url: SITE_DEFAULT_AVATAR_URL }));
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
                  <input type="checkbox" checked={checked} onChange={() => toggleBinding(type, item.id)} className="mt-1" />
                  <span className="min-w-0">
                    <span className="block font-medium text-[var(--color-text-primary)]">{item.label}</span>
                    {item.description ? (
                      <span className="mt-0.5 block text-xs text-[var(--color-text-muted)]">{item.description}</span>
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
              <button type="button" onClick={() => onRemove(value)} className="text-[var(--color-text-muted)] hover:text-[var(--color-danger)]">
                ×
              </button>
            </span>
          ))
        )}
      </div>
    </div>
  );
}
