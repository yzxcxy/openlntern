import { Collapsible } from "@douyinfe/semi-ui-19";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiModal as Modal } from "../../components/ui/UiModal";
import {
  getChatPluginKey,
  uniqueStringList,
  type ChatPluginOption,
} from "./chat-plugin-config";

type SelectedToolItem = {
  toolId: string;
  pluginName: string;
  pluginIcon: string;
  toolName: string;
  toolDescription: string;
};

type PluginSelectionModalProps = {
  open: boolean;
  maxSelectedTools: number;
  selectedToolIds: string[];
  selectedPluginIds: string[];
  selectedToolIdSet: Set<string>;
  pluginLoading: boolean;
  pluginSearchKeyword: string;
  pluginSourceFilter: string;
  pluginTypeFilter: string;
  pluginPage: number;
  pluginPageSize: number;
  pluginPageSizeOptions: number[];
  pluginTotalPages: number;
  availableToolCount: number;
  availableRuntimeTypes: string[];
  filteredPlugins: ChatPluginOption[];
  paginatedPlugins: ChatPluginOption[];
  expandedPluginKeys: string[];
  selectedToolItems: SelectedToolItem[];
  pluginError: string;
  onClose: () => void;
  onReset: () => void;
  onPluginSearchKeywordChange: (value: string) => void;
  onPluginSourceFilterChange: (value: string) => void;
  onPluginTypeFilterChange: (value: string) => void;
  onPluginPageChange: (page: number) => void;
  onPluginPageSizeChange: (size: number) => void;
  onTogglePluginExpanded: (pluginKey: string) => void;
  onTogglePluginSelection: (plugin: ChatPluginOption, checked: boolean) => void;
  onToggleToolSelection: (toolId: string, checked: boolean) => void;
};

const getRuntimeTypeLabel = (runtimeType: string) => {
  const normalized = runtimeType.trim().toLowerCase();
  switch (normalized) {
    case "api":
      return "API";
    case "mcp":
      return "MCP";
    case "code":
      return "Code";
    case "builtin":
      return "内建";
    default:
      return normalized ? normalized.toUpperCase() : "PLUGIN";
  }
};

export function PluginSelectionModal({
  open,
  maxSelectedTools,
  selectedToolIds,
  selectedPluginIds,
  selectedToolIdSet,
  pluginLoading,
  pluginSearchKeyword,
  pluginSourceFilter,
  pluginTypeFilter,
  pluginPage,
  pluginPageSize,
  pluginPageSizeOptions,
  pluginTotalPages,
  availableToolCount,
  availableRuntimeTypes,
  filteredPlugins,
  paginatedPlugins,
  expandedPluginKeys,
  selectedToolItems,
  pluginError,
  onClose,
  onReset,
  onPluginSearchKeywordChange,
  onPluginSourceFilterChange,
  onPluginTypeFilterChange,
  onPluginPageChange,
  onPluginPageSizeChange,
  onTogglePluginExpanded,
  onTogglePluginSelection,
  onToggleToolSelection,
}: PluginSelectionModalProps) {
  return (
    <Modal
      open={open}
      title="添加工具"
      onClose={onClose}
      footer={
        <>
          <div className="mr-auto text-xs text-[var(--color-text-muted)]">
            已选 {selectedToolIds.length}/{maxSelectedTools} 个工具，覆盖{" "}
            {selectedPluginIds.length} 个插件
          </div>
          <UiButton
            type="button"
            variant="secondary"
            onClick={onReset}
            disabled={pluginLoading}
          >
            恢复默认
          </UiButton>
          <UiButton type="button" onClick={onClose}>
            完成
          </UiButton>
        </>
      }
    >
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1.6fr)_minmax(280px,0.9fr)]">
        <div className="min-w-0">
          <div className="grid gap-2 md:grid-cols-[minmax(0,1fr)_136px_136px] md:items-center">
            <UiInput
              value={pluginSearchKeyword}
              onChange={(event) => onPluginSearchKeywordChange(event.target.value)}
              placeholder="搜索插件名、工具名或描述"
              className="min-w-0 bg-[rgba(255,255,255,0.9)]"
            />
            <UiSelect
              className="min-w-0"
              value={pluginSourceFilter}
              onChange={(event) => onPluginSourceFilterChange(event.target.value)}
            >
              <option value="all">全部来源</option>
              <option value="builtin">内建插件</option>
              <option value="custom">自定义插件</option>
            </UiSelect>
            <UiSelect
              className="min-w-0"
              value={pluginTypeFilter}
              onChange={(event) => onPluginTypeFilterChange(event.target.value)}
            >
              <option value="all">全部类型</option>
              {availableRuntimeTypes.map((runtimeType) => (
                <option key={runtimeType} value={runtimeType}>
                  {getRuntimeTypeLabel(runtimeType)}
                </option>
              ))}
            </UiSelect>
          </div>
          {pluginLoading ? (
            <div className="mt-3 text-xs text-[var(--color-text-muted)]">
              正在加载可用插件...
            </div>
          ) : filteredPlugins.length === 0 ? (
            <div className="mt-3 rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] px-3 py-3 text-xs text-[var(--color-text-muted)]">
              {pluginSearchKeyword.trim()
                ? "没有匹配的插件或工具。"
                : "当前没有可用于聊天的启用插件。"}
            </div>
          ) : (
            <div className="mt-3 max-h-[58vh] space-y-2 overflow-y-auto pr-1">
              {paginatedPlugins.map((plugin, pluginIndex) => {
                const pluginKey = getChatPluginKey(plugin, pluginIndex);
                const pluginToolIds = uniqueStringList(
                  (plugin.tools ?? []).map((tool) =>
                    typeof tool.tool_id === "string" ? tool.tool_id : ""
                  )
                );
                const selectedCount = pluginToolIds.filter((toolId) =>
                  selectedToolIdSet.has(toolId)
                ).length;
                const pluginDescriptionHtml =
                  typeof plugin.description === "string" ? plugin.description.trim() : "";
                const isExpanded =
                  pluginSearchKeyword.trim().length > 0 ||
                  expandedPluginKeys.includes(pluginKey);
                const allSelected =
                  pluginToolIds.length > 0 && selectedCount === pluginToolIds.length;

                return (
                  <div
                    key={pluginKey}
                    className="overflow-hidden rounded-[var(--radius-lg)] border border-[rgba(148,163,184,0.18)] bg-[rgba(255,255,255,0.86)]"
                  >
                    <div className="flex flex-col gap-4 px-4 py-4 md:flex-row md:items-start md:justify-between">
                      <div className="flex min-w-0 flex-1 gap-4">
                        <div
                          className="flex h-16 w-16 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.14)] text-lg font-semibold text-[var(--color-text-secondary)]"
                          style={
                            plugin.icon
                              ? {
                                  backgroundImage: `url(${plugin.icon})`,
                                  backgroundPosition: "center",
                                  backgroundSize: "cover",
                                }
                              : undefined
                          }
                          aria-label={plugin.name || "plugin"}
                          title={plugin.name || "plugin"}
                        >
                          {plugin.icon ? "" : (plugin.name || "P").slice(0, 1)}
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <div className="truncate text-[15px] font-semibold text-[var(--color-text-primary)]">
                              {plugin.name || "未命名插件"}
                            </div>
                            <span className="rounded-[var(--radius-sm)] border border-[rgba(148,163,184,0.18)] px-2 py-0.5 text-[11px] font-medium tracking-[0.04em] text-[var(--color-text-muted)]">
                              {getRuntimeTypeLabel(plugin.runtime_type || "plugin")}
                            </span>
                            <span className="rounded-[var(--radius-sm)] border border-[rgba(148,163,184,0.18)] px-2 py-0.5 text-[11px] font-medium text-[var(--color-text-muted)]">
                              已选 {selectedCount}/{pluginToolIds.length}
                            </span>
                          </div>
                          {pluginDescriptionHtml ? (
                            <div
                              className="mt-2 line-clamp-2 break-words text-sm text-[var(--color-text-secondary)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                              dangerouslySetInnerHTML={{
                                __html: pluginDescriptionHtml,
                              }}
                            />
                          ) : null}
                          <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-[var(--color-text-muted)]">
                            <span>{pluginToolIds.length} 个工具</span>
                            <UiButton
                              onClick={() =>
                                onTogglePluginSelection(plugin, !allSelected)
                              }
                              disabled={pluginToolIds.length === 0}
                              variant="ghost"
                              size="sm"
                              className="rounded-full border-[var(--color-border-default)] px-2 text-[var(--color-action-primary)]"
                            >
                              {allSelected ? "全部移除" : "全部添加"}
                            </UiButton>
                          </div>
                        </div>
                      </div>
                      <UiButton
                        onClick={() => onTogglePluginExpanded(pluginKey)}
                        variant="ghost"
                        size="sm"
                        className="self-start rounded-full border-[var(--color-border-default)] px-2 text-[var(--color-text-muted)]"
                      >
                        {isExpanded ? "收起" : "展开"}
                      </UiButton>
                    </div>
                    <div className="border-t border-[rgba(148,163,184,0.14)]">
                      <Collapsible isOpen={isExpanded}>
                        {pluginToolIds.length > 0 && (
                          <div className="divide-y divide-[rgba(148,163,184,0.12)]">
                            {(plugin.tools ?? []).map((tool, toolIndex) => {
                              const toolId =
                                typeof tool.tool_id === "string" ? tool.tool_id : "";
                              if (!toolId) {
                                return null;
                              }
                              const checked = selectedToolIdSet.has(toolId);
                              const toolDescriptionHtml =
                                typeof tool.description === "string"
                                  ? tool.description.trim()
                                  : "";

                              return (
                                <div
                                  key={`${toolId}-${toolIndex}`}
                                  className="flex flex-col gap-3 px-4 py-4 md:flex-row md:items-start md:justify-between"
                                >
                                  <div className="min-w-0 flex-1">
                                    <div className="flex flex-wrap items-center gap-2">
                                      <div className="break-words text-[15px] font-semibold text-[var(--color-text-primary)]">
                                        {tool.tool_name || toolId}
                                      </div>
                                    </div>
                                    <div
                                      className="mt-2 line-clamp-2 break-words text-sm text-[var(--color-text-muted)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                                      dangerouslySetInnerHTML={{
                                        __html: toolDescriptionHtml || "暂无工具描述",
                                      }}
                                    />
                                  </div>
                                  <UiButton
                                    onClick={() =>
                                      onToggleToolSelection(toolId, !checked)
                                    }
                                    variant={checked ? "secondary" : "primary"}
                                    size="sm"
                                    className="self-start rounded-[var(--radius-md)] px-4"
                                  >
                                    {checked ? "移除" : "添加"}
                                  </UiButton>
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </Collapsible>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
          {!pluginLoading && filteredPlugins.length > 0 && (
            <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[rgba(148,163,184,0.14)] bg-[rgba(248,250,252,0.9)] px-3 py-2">
              <div className="min-w-0 text-xs text-[var(--color-text-muted)]">
                共 {filteredPlugins.length} 个插件，可用 {availableToolCount} 个工具
              </div>
              <div className="flex shrink-0 items-center justify-end gap-2 whitespace-nowrap">
                <UiSelect
                  className="h-9 w-[102px] min-w-[102px] rounded-full py-1.5 pl-3 pr-9 text-sm"
                  value={String(pluginPageSize)}
                  onChange={(event) =>
                    onPluginPageSizeChange(
                      Number.parseInt(event.target.value, 10) || pluginPageSize
                    )
                  }
                >
                  {pluginPageSizeOptions.map((size) => (
                    <option key={size} value={size}>
                      每页 {size} 个
                    </option>
                  ))}
                </UiSelect>
                <UiButton
                  onClick={() => onPluginPageChange(pluginPage - 1)}
                  variant="ghost"
                  size="sm"
                  disabled={pluginPage <= 1}
                  className="rounded-full border-[var(--color-border-default)] px-3 text-[var(--color-text-muted)]"
                >
                  上一页
                </UiButton>
                <span className="text-xs text-[var(--color-text-muted)]">
                  第 {pluginPage}/{pluginTotalPages} 页
                </span>
                <UiButton
                  onClick={() => onPluginPageChange(pluginPage + 1)}
                  variant="ghost"
                  size="sm"
                  disabled={pluginPage >= pluginTotalPages}
                  className="rounded-full border-[var(--color-border-default)] px-3 text-[var(--color-text-muted)]"
                >
                  下一页
                </UiButton>
              </div>
            </div>
          )}
        </div>
        <div className="rounded-[var(--radius-lg)] border border-[rgba(148,163,184,0.18)] bg-[rgba(255,255,255,0.86)] px-3 py-3">
          <div className="text-sm font-semibold text-[var(--color-text-primary)]">
            已选工具
          </div>
          <div className="mt-1 text-xs text-[var(--color-text-muted)]">
            共 {selectedToolItems.length} 个
          </div>
          {selectedToolItems.length === 0 ? (
            <div className="mt-3 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] px-3 py-3 text-xs text-[var(--color-text-muted)]">
              暂无已选工具。
            </div>
          ) : (
            <div className="mt-3 max-h-[58vh] space-y-2 overflow-y-auto pr-1">
              {selectedToolItems.map((item) => (
                <div
                  key={item.toolId}
                  className="flex items-center gap-3 rounded-[var(--radius-md)] border border-[rgba(148,163,184,0.16)] bg-[rgba(248,250,252,0.92)] px-3 py-2"
                >
                  <div
                    className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[rgba(148,163,184,0.18)] text-[11px] font-semibold text-[var(--color-text-secondary)]"
                    style={
                      item.pluginIcon
                        ? {
                            backgroundImage: `url(${item.pluginIcon})`,
                            backgroundPosition: "center",
                            backgroundSize: "cover",
                          }
                        : undefined
                    }
                    aria-label={item.pluginName}
                    title={item.pluginName}
                  >
                    {item.pluginIcon ? "" : item.pluginName.slice(0, 1)}
                  </div>
                  <div className="min-w-0 flex-1 text-xs text-[var(--color-text-primary)]">
                    <div className="truncate font-medium">
                      {item.pluginName}/{item.toolName}
                    </div>
                    <div
                      className="mt-1 line-clamp-2 text-[11px] text-[var(--color-text-muted)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                      dangerouslySetInnerHTML={{
                        __html: item.toolDescription || "暂无工具描述",
                      }}
                    />
                  </div>
                  <UiButton
                    onClick={() => onToggleToolSelection(item.toolId, false)}
                    variant="ghost"
                    size="sm"
                    className="rounded-full border-[var(--color-border-default)] px-2 text-[var(--color-text-muted)]"
                  >
                    移除
                  </UiButton>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
      {pluginError && (
        <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.18)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-xs text-[var(--color-state-error)]">
          {pluginError}
        </div>
      )}
    </Modal>
  );
}
