export const MAX_SELECTED_TOOLS = 40;
export const PLUGIN_PAGE_SIZE = 6;
export const PLUGIN_PAGE_SIZE_OPTIONS = [6, 12, 24];
export const KNOWN_PLUGIN_RUNTIME_TYPES = ["api", "mcp", "code"];

export type ChatPluginToolOption = {
  tool_id?: string;
  tool_name?: string;
  description?: string;
  tool_response_mode?: string;
};

export type ChatPluginOption = {
  plugin_id?: string;
  name?: string;
  description?: string;
  icon?: string;
  source?: string;
  runtime_type?: string;
  tools?: ChatPluginToolOption[];
};

export const getChatPluginKey = (plugin: ChatPluginOption, index: number) =>
  plugin.plugin_id || `${plugin.name || "plugin"}-${index}`;

export const uniqueStringList = (values: string[]) =>
  Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));

export const collectToolIdsFromPlugins = (plugins: ChatPluginOption[]) =>
  uniqueStringList(
    plugins.flatMap((plugin) =>
      (plugin.tools ?? []).map((tool) =>
        typeof tool.tool_id === "string" ? tool.tool_id : ""
      )
    )
  );

export const sanitizeDescriptionText = (value?: string) =>
  String(value || "")
    .replace(/<[^>]*>/g, " ")
    .replace(/&nbsp;/gi, " ")
    .replace(/\s+/g, " ")
    .trim();

export const normalizePluginSource = (value?: string) => {
  const normalized = String(value || "").trim().toLowerCase();
  if (normalized === "builtin" || normalized === "built_in" || normalized === "built-in") {
    return "builtin";
  }
  if (
    normalized === "custom" ||
    normalized === "user" ||
    normalized === "custom_plugin" ||
    normalized === "custom-plugin"
  ) {
    return "custom";
  }
  return normalized;
};

export const getPluginSourceFilterValue = (value?: string) =>
  normalizePluginSource(value) === "builtin" ? "builtin" : "custom";

export const normalizePluginRuntimeType = (value?: string) =>
  String(value || "").trim().toLowerCase();
