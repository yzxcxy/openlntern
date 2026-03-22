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
  return (
    <div
      className="flex items-center gap-2"
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => event.stopPropagation()}
    >
      <UiSelect
        value={conversationMode}
        onChange={(event) => {
          onConversationModeChange(
            event.target.value === "agent" ? "agent" : "chat"
          );
        }}
        onMouseDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
        className="ui-select-control--compact ui-select-control--glass rounded-full border-transparent px-4 py-2 text-sm font-medium text-[var(--color-text-primary)] outline-none focus:border-[var(--color-action-primary)]"
      >
        <option value="chat">Chat</option>
        <option value="agent">Agent</option>
      </UiSelect>
      {conversationMode === "agent" && (
        <div className="ui-select-control--glass relative min-w-[196px] rounded-full border border-transparent px-4 py-2 focus-within:border-[var(--color-action-primary)]">
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
          <div className="ui-select-control--glass relative min-w-[128px] rounded-full border border-transparent px-4 py-2 focus-within:border-[var(--color-action-primary)]">
            <span className="pointer-events-none block pr-8 text-sm font-medium text-[var(--color-text-primary)]">
              {pluginMode === "select" ? "Select" : "Search"}
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
              variant="ghost"
              size="sm"
              className="rounded-full border-[var(--color-border-default)] text-[var(--color-text-secondary)]"
            >
              <span>选工具 {selectedToolCount}</span>
            </UiButton>
          )}
        </>
      )}
    </div>
  );
}
