import { UiButton } from "../../components/ui/UiButton";
import { UiSelect } from "../../components/ui/UiSelect";

type ConversationMode = "chat" | "agent";
type PluginMode = "select" | "search";
type AgentOption = {
  agent_id: string;
  name: string;
  agent_type: "single" | "supervisor";
};

type ChatModeConfigureAreaProps = {
  conversationMode: ConversationMode;
  pluginMode: PluginMode;
  selectedAgentId: string;
  agentOptions: AgentOption[];
  selectedToolCount: number;
  onConversationModeChange: (mode: ConversationMode) => void;
  onAgentChange: (agentId: string) => void;
  onPluginModeChange: (mode: PluginMode) => void;
  onOpenPluginPanel: () => void;
};

export function ChatModeConfigureArea({
  conversationMode,
  pluginMode,
  selectedAgentId,
  agentOptions,
  selectedToolCount,
  onConversationModeChange,
  onAgentChange,
  onPluginModeChange,
  onOpenPluginPanel,
}: ChatModeConfigureAreaProps) {
  // 统一使用可点击胶囊壳层，再叠加透明 select，避免底部配置区出现多套控件视觉。
  const pillClass =
    "relative min-w-[116px] rounded-full border border-[rgba(126,96,69,0.16)] bg-[rgba(255,252,247,0.82)] px-4 py-2.5 shadow-[inset_0_1px_0_rgba(255,255,255,0.56)] transition focus-within:border-[rgba(199,104,67,0.28)] focus-within:bg-[rgba(255,250,243,0.96)]";

  return (
    <div
      className="chat-configure-rail flex flex-wrap items-center gap-2"
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => event.stopPropagation()}
    >
      <div className={pillClass}>
        <span className="pointer-events-none block pr-8 text-sm font-medium text-[var(--color-text-primary)]">
          {conversationMode === "agent" ? "Agent 模式" : "聊天模式"}
        </span>
        <UiSelect
          value={conversationMode}
          onChange={(event) => {
            onConversationModeChange(
              event.target.value === "agent" ? "agent" : "chat"
            );
          }}
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
          className="absolute inset-0 h-full w-full cursor-pointer rounded-full opacity-0"
        >
          <option value="chat">Chat</option>
          <option value="agent">Agent</option>
        </UiSelect>
      </div>
      {conversationMode === "agent" && (
        <div className={`${pillClass} min-w-[220px]`}>
          <span className="pointer-events-none block pr-8 text-sm font-medium text-[var(--color-text-primary)]">
            {agentOptions.find((item) => item.agent_id === selectedAgentId)?.name ||
              "选择 Agent"}
          </span>
          <UiSelect
            value={selectedAgentId}
            onChange={(event) => onAgentChange(event.target.value)}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={(event) => event.stopPropagation()}
            className="absolute inset-0 h-full w-full cursor-pointer rounded-full opacity-0"
          >
            <option value="">选择 Agent</option>
            {agentOptions.map((item) => (
              <option key={item.agent_id} value={item.agent_id}>
                {item.name} ({item.agent_type === "supervisor" ? "Supervisor" : "Single"})
              </option>
            ))}
          </UiSelect>
        </div>
      )}
      {conversationMode === "chat" && (
        <>
          <div className={`${pillClass} min-w-[132px]`}>
            <span className="pointer-events-none block pr-8 text-sm font-medium text-[var(--color-text-primary)]">
              {pluginMode === "select" ? "选择工具" : "搜索工具"}
            </span>
            <UiSelect
              value={pluginMode}
              onChange={(event) => {
                onPluginModeChange(
                  event.target.value === "search" ? "search" : "select"
                );
              }}
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => event.stopPropagation()}
              className="absolute inset-0 h-full w-full cursor-pointer rounded-full opacity-0"
            >
              <option value="select">Select</option>
              <option value="search">Search</option>
            </UiSelect>
          </div>
          {pluginMode === "select" && (
            <UiButton
              onClick={onOpenPluginPanel}
              variant="secondary"
              size="sm"
              className="h-10 whitespace-nowrap rounded-full px-4 text-[13px]"
            >
              <span>工具 {selectedToolCount}</span>
            </UiButton>
          )}
        </>
      )}
    </div>
  );
}
