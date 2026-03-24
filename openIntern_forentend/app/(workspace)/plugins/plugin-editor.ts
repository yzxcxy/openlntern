// 插件编辑页的纯类型和纯函数集中到这里，避免页面文件继续堆叠领域逻辑。

export type RuntimeType = "" | "api" | "mcp" | "code" | "builtin";
export type ResponseMode = "" | "streaming" | "non_streaming";
export type RequestType = "GET" | "POST";
export type FieldType =
  | "string"
  | "number"
  | "integer"
  | "boolean"
  | "object"
  | "array";
export type MCPProtocol = "sse" | "streamableHttp";

export type PluginField = {
  id: string;
  name: string;
  type: FieldType;
  required: boolean;
  description: string;
  defaultValue: string;
  enumText: string;
  children: PluginField[];
  item: PluginField | null;
};

export type PluginTool = {
  tool_id?: string;
  tool_name?: string;
  description?: string;
  tool_response_mode?: ResponseMode;
  api_request_type?: RequestType;
  request_url?: string;
  auth_config_ref?: string;
  timeout_ms?: number;
  query_fields?: Array<Record<string, unknown>>;
  header_fields?: Array<Record<string, unknown>>;
  body_fields?: Array<Record<string, unknown>>;
  code_language?: string;
  code?: string;
  input_schema_json?: string;
  output_schema_json?: string;
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
};

export type PluginRecord = {
  plugin_id?: string;
  name?: string;
  description?: string;
  icon?: string;
  source?: string;
  runtime_type?: RuntimeType;
  status?: "enabled" | "disabled";
  mcp_url?: string;
  mcp_protocol?: string;
  timeout_ms?: number;
  last_sync_at?: string | null;
  tool_count?: number;
  tools?: PluginTool[];
  created_at?: string;
  updated_at?: string;
};

export type ToolDraft = {
  toolId?: string;
  toolName: string;
  description: string;
  toolResponseMode: ResponseMode;
  apiRequestType: RequestType;
  requestURL: string;
  authConfigRef: string;
  timeoutMS: number;
  queryFields: PluginField[];
  headerFields: PluginField[];
  bodyFields: PluginField[];
  outputFields: PluginField[];
  codeLanguage: string;
  code: string;
};

export type PluginDraft = {
  pluginId?: string;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
  runtimeType: RuntimeType;
  mcpURL: string;
  mcpProtocol: MCPProtocol;
  timeoutMS: number;
  tools: ToolDraft[];
};

export type DetailFieldSection = {
  key: string;
  label: string;
  fields: PluginField[];
};

export type FlatFieldRow = {
  id: string;
  name: string;
  type: FieldType;
  required: boolean;
  description: string;
  defaultValue: string;
  enumText: string;
  depth: number;
};

const TOOL_NAME_PATTERN = /^[a-zA-Z0-9_-]+$/;

const createId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const isValidToolName = (value: string) => TOOL_NAME_PATTERN.test(value.trim());

export const createField = (type: FieldType = "string"): PluginField => ({
  id: createId(),
  name: "",
  type,
  required: false,
  description: "",
  defaultValue: "",
  enumText: "",
  children: [],
  item: type === "array" ? createField("string") : null,
});

const sanitizeOutputField = (field: PluginField): PluginField => ({
  ...field,
  required: false,
  defaultValue: "",
  enumText: "",
  children: field.children.map((child) => sanitizeOutputField(child)),
  item: field.item ? sanitizeOutputField(field.item) : null,
});

export const sanitizeOutputFields = (fields: PluginField[]): PluginField[] =>
  fields.map((field) => sanitizeOutputField(field));

export const createToolDraft = (): ToolDraft => ({
  toolName: "",
  description: "",
  toolResponseMode: "non_streaming",
  apiRequestType: "GET",
  requestURL: "",
  authConfigRef: "",
  timeoutMS: 30,
  queryFields: [],
  headerFields: [],
  bodyFields: [],
  outputFields: [],
  codeLanguage: "javascript",
  code: "",
});

export const createPluginDraft = (): PluginDraft => ({
  name: "",
  description: "",
  icon: "",
  enabled: true,
  runtimeType: "",
  mcpURL: "",
  mcpProtocol: "sse",
  timeoutMS: 30,
  tools: [createToolDraft()],
});

export const runtimeLabel: Record<Exclude<RuntimeType, "">, string> = {
  api: "API",
  mcp: "MCP",
  code: "Code",
  builtin: "内建",
};

export const responseModeLabel: Record<Exclude<ResponseMode, "">, string> = {
  streaming: "流式",
  non_streaming: "非流式",
};

export const mcpProtocolLabel: Record<MCPProtocol, string> = {
  sse: "服务器发送事件（SSE）",
  streamableHttp: "可流式传输的 HTTP（streamableHttp）",
};

export const getSourceBadgeClassName = (source?: string) => {
  if (source === "builtin") {
    return "border-[rgba(217,119,6,0.18)] bg-[rgba(245,158,11,0.12)] text-[rgb(180,83,9)]";
  }
  return "border-[rgba(14,116,144,0.18)] bg-[rgba(6,182,212,0.12)] text-[rgb(14,116,144)]";
};

export const getRuntimeBadgeClassName = (runtimeType?: RuntimeType) => {
  switch (runtimeType) {
    case "api":
      return "border-[rgba(37,99,235,0.18)] bg-[rgba(59,130,246,0.12)] text-[rgb(29,78,216)]";
    case "mcp":
      return "border-[rgba(13,148,136,0.18)] bg-[rgba(20,184,166,0.12)] text-[rgb(15,118,110)]";
    case "code":
      return "border-[rgba(234,88,12,0.18)] bg-[rgba(249,115,22,0.12)] text-[rgb(194,65,12)]";
    case "builtin":
      return "border-[rgba(202,138,4,0.18)] bg-[rgba(234,179,8,0.12)] text-[rgb(161,98,7)]";
    default:
      return "border-[var(--color-border-default)] bg-white text-[var(--color-text-secondary)]";
  }
};

export const normalizeMCPProtocolValue = (value: unknown): MCPProtocol =>
  value === "streamableHttp" ? "streamableHttp" : "sse";

export const getToolKey = (tool: PluginTool, index: number) =>
  tool.tool_id || tool.tool_name || `tool-${index}`;

export const formatTime = (value?: string | null) => {
  if (!value) return "未记录";
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(parsed));
};

export const parseFields = (value: unknown): PluginField[] => {
  if (!Array.isArray(value)) return [];
  return value.map((item) => parseField(item)).filter(Boolean) as PluginField[];
};

export const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const normalizeFieldType = (value: unknown): FieldType => {
  if (
    typeof value === "string" &&
    ["string", "number", "integer", "boolean", "object", "array"].includes(
      value
    )
  ) {
    return value as FieldType;
  }
  return "string";
};

const stringifyDefaultValue = (value: unknown): string => {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return "";
  }
};

const parseField = (value: unknown): PluginField | null => {
  if (!value || typeof value !== "object") return null;
  const record = value as Record<string, unknown>;
  const rawType = typeof record.type === "string" ? record.type : "string";
  const type = (
    ["string", "number", "integer", "boolean", "object", "array"].includes(
      rawType
    )
      ? rawType
      : "string"
  ) as FieldType;
  const enumValues = Array.isArray(record.enum_values)
    ? record.enum_values.filter((item) => typeof item === "string")
    : [];
  const children = parseFields(record.children);
  const item = record.items ? parseField(record.items) : null;
  return {
    id: createId(),
    name: typeof record.name === "string" ? record.name : "",
    type,
    required: Boolean(record.required),
    description:
      typeof record.description === "string" ? record.description : "",
    defaultValue: stringifyDefaultValue(record.default_value),
    enumText: enumValues.join(", "),
    children,
    item: type === "array" ? item ?? createField("string") : null,
  };
};

const parseSchemaField = (
  name: string,
  value: unknown,
  required = false,
  isArrayItem = false
): PluginField | null => {
  if (!isRecord(value)) return null;

  const hasProperties = isRecord(value.properties);
  const rawType =
    typeof value.type === "string"
      ? value.type
      : hasProperties
        ? "object"
        : value.items
          ? "array"
          : "string";
  const type = normalizeFieldType(rawType);
  const requiredFields = new Set(
    Array.isArray(value.required)
      ? value.required.filter(
          (item): item is string => typeof item === "string"
        )
      : []
  );
  const enumValues = Array.isArray(value.enum)
    ? value.enum.filter((item): item is string => typeof item === "string")
    : [];

  const children =
    type === "object" && hasProperties
      ? Object.entries(value.properties as Record<string, unknown>)
          .map(([childName, childValue]) =>
            parseSchemaField(childName, childValue, requiredFields.has(childName))
          )
          .filter(Boolean) as PluginField[]
      : [];

  const item =
    type === "array"
      ? parseSchemaField("", value.items, false, true) ?? createField("string")
      : null;

  return {
    id: createId(),
    name: isArrayItem ? "" : name,
    type,
    required,
    description: typeof value.description === "string" ? value.description : "",
    defaultValue: stringifyDefaultValue(value.default),
    enumText: enumValues.join(", "),
    children,
    item,
  };
};

const parseSchemaFieldsFromJSON = (
  raw: string | undefined,
  fallbackName: string
): PluginField[] => {
  const trimmed = raw?.trim();
  if (!trimmed) return [];

  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (isRecord(parsed) && isRecord(parsed.properties)) {
      const requiredFields = new Set(
        Array.isArray(parsed.required)
          ? parsed.required.filter(
              (item): item is string => typeof item === "string"
            )
          : []
      );
      return Object.entries(parsed.properties as Record<string, unknown>)
        .map(([name, value]) =>
          parseSchemaField(name, value, requiredFields.has(name))
        )
        .filter(Boolean) as PluginField[];
    }
    const rootField = parseSchemaField(fallbackName, parsed);
    return rootField ? [rootField] : [];
  } catch {
    return [];
  }
};

export const getInputSections = (
  runtimeType: RuntimeType | undefined,
  tool: PluginTool
): DetailFieldSection[] => {
  if (runtimeType === "api") {
    return [
      { key: "header", label: "Header", fields: parseFields(tool.header_fields) },
      { key: "query", label: "Query", fields: parseFields(tool.query_fields) },
      { key: "body", label: "Body", fields: parseFields(tool.body_fields) },
    ].filter((section) => section.fields.length > 0);
  }

  if (runtimeType === "code") {
    const bodyFields = parseFields(tool.body_fields);
    return bodyFields.length > 0
      ? [{ key: "body", label: "输入参数", fields: bodyFields }]
      : [];
  }

  const schemaFields = parseSchemaFieldsFromJSON(tool.input_schema_json, "input");
  if (schemaFields.length === 0) return [];

  const splitSections = schemaFields
    .filter((field) => field.type === "object")
    .map((field) => ({
      key: field.name || field.id,
      label: field.name || "输入参数",
      fields: field.children,
    }))
    .filter((section) => section.fields.length > 0);

  if (splitSections.length > 0) {
    return splitSections;
  }

  return [{ key: "input", label: "输入参数", fields: schemaFields }];
};

export const getOutputFields = (tool: PluginTool) =>
  parseSchemaFieldsFromJSON(tool.output_schema_json, "result");

export const flattenFields = (fields: PluginField[]): FlatFieldRow[] => {
  const rows: FlatFieldRow[] = [];

  const walk = (field: PluginField, depth: number, label?: string) => {
    const name = (label ?? field.name) || "item";
    rows.push({
      id: `${field.id}-${depth}-${name}`,
      name,
      type: field.type,
      required: field.required,
      description: field.description,
      defaultValue: field.defaultValue,
      enumText: field.enumText,
      depth,
    });

    if (field.type === "object") {
      field.children.forEach((child) => walk(child, depth + 1));
    }
    if (field.type === "array" && field.item) {
      walk(field.item, depth + 1, `${name}[]`);
    }
  };

  fields.forEach((field) => walk(field, 0));
  return rows;
};

const toFieldPayload = (field: PluginField, isArrayItem = false): Record<string, unknown> => ({
  ...(isArrayItem ? {} : { name: field.name.trim() }),
  type: field.type,
  required: field.required,
  description: field.description.trim(),
  ...(field.defaultValue.trim()
    ? { default_value: field.defaultValue.trim() }
    : {}),
  enum_values:
    field.type === "string"
      ? field.enumText
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean)
      : [],
  children:
    field.type === "object"
      ? field.children.map((child) => toFieldPayload(child))
      : [],
  items:
    field.type === "array" && field.item
      ? toFieldPayload(field.item, true)
      : undefined,
});

export const getCodeTemplate = (language: string) => {
  if (language === "python") {
    return [
      "# 请将入口函数命名为 main，入参为 params: dict，返回 dict",
      "def main(params: dict) -> dict:",
      "    return {",
      '        "result": params.get("input"),',
      "    }",
    ].join("\n");
  }

  return [
    "// 请将入口函数命名为 main，入参为 params，返回普通对象",
    "function main(params) {",
    "  return {",
    "    result: params?.input ?? null,",
    "  };",
    "}",
  ].join("\n");
};

export const getDefaultCodeBodyFields = (): PluginField[] => [
  {
    ...createField("string"),
    name: "input",
    description: "传入 main(params) 的默认示例参数",
  },
];

export const getEffectiveCodeBodyFields = (tool: ToolDraft): PluginField[] => {
  if (tool.bodyFields.length > 0) {
    return tool.bodyFields;
  }

  const codeLanguage =
    tool.codeLanguage.trim() === "python" ? "python" : "javascript";
  const defaultTemplate = getCodeTemplate(codeLanguage).trim();
  if (tool.code.trim() === defaultTemplate) {
    return getDefaultCodeBodyFields();
  }

  return [];
};

export const applyCodeToolDefaults = (tool: ToolDraft): ToolDraft => ({
  ...tool,
  toolResponseMode: "non_streaming",
  code: tool.code.trim() ? tool.code : getCodeTemplate(tool.codeLanguage),
  bodyFields: getEffectiveCodeBodyFields(tool),
});

export const requiresRuntimeTools = (runtimeType: RuntimeType) =>
  runtimeType === "api" || runtimeType === "code";

// 保证 API/Code 插件至少有一个工具，避免表单进入空状态。
export const ensureRuntimeTools = (
  tools: ToolDraft[],
  runtimeType: RuntimeType
): ToolDraft[] => {
  if (!requiresRuntimeTools(runtimeType)) {
    return tools;
  }
  if (tools.length === 0) {
    const nextTool = createToolDraft();
    return runtimeType === "code"
      ? [applyCodeToolDefaults(nextTool)]
      : [nextTool];
  }
  if (runtimeType === "code") {
    return tools.map((tool) => applyCodeToolDefaults(tool));
  }
  return tools;
};

const buildFieldSchemaObject = (field: PluginField): Record<string, unknown> => {
  const schema: Record<string, unknown> = {
    type: field.type,
  };

  if (field.description.trim()) {
    schema.description = field.description.trim();
  }

  if (field.defaultValue.trim()) {
    if (field.type === "string") {
      schema.default = field.defaultValue.trim();
    } else if (field.type === "number" || field.type === "integer") {
      schema.default = Number(field.defaultValue);
    } else if (field.type === "boolean") {
      schema.default = /^(true|1)$/i.test(field.defaultValue.trim());
    } else {
      try {
        schema.default = JSON.parse(field.defaultValue);
      } catch {
        schema.default = field.defaultValue.trim();
      }
    }
  }

  if (field.type === "string") {
    const enumValues = field.enumText
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
    if (enumValues.length > 0) {
      schema.enum = enumValues;
    }
  }

  if (field.type === "object") {
    const properties: Record<string, unknown> = {};
    const required: string[] = [];
    field.children.forEach((child) => {
      properties[child.name.trim()] = buildFieldSchemaObject(child);
      if (child.required) {
        required.push(child.name.trim());
      }
    });
    schema.properties = properties;
    schema.additionalProperties = false;
    if (required.length > 0) {
      schema.required = required;
    }
  }

  if (field.type === "array" && field.item) {
    schema.items = buildFieldSchemaObject(field.item);
  }

  return schema;
};

const buildObjectSchemaJSON = (fields: PluginField[]) => {
  if (fields.length === 0) return "";

  const properties: Record<string, unknown> = {};
  const required: string[] = [];

  fields.forEach((field) => {
    const fieldName = field.name.trim();
    if (!fieldName) return;
    properties[fieldName] = buildFieldSchemaObject(field);
    if (field.required) {
      required.push(fieldName);
    }
  });

  return JSON.stringify(
    {
      $schema: "https://json-schema.org/draft/2020-12/schema",
      type: "object",
      properties,
      additionalProperties: false,
      ...(required.length > 0 ? { required } : {}),
    },
    null,
    2
  );
};

const validateRuntimeValue = (
  value: unknown,
  field: PluginField,
  path: string
): string => {
  if (value === undefined || value === null) {
    return `${path}不能为空`;
  }

  switch (field.type) {
    case "string": {
      if (typeof value !== "string") return `${path}必须为 string`;
      const enumValues = field.enumText
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean);
      if (enumValues.length > 0 && !enumValues.includes(value)) {
        return `${path}必须命中枚举值`;
      }
      return "";
    }
    case "number":
      return typeof value === "number" && Number.isFinite(value)
        ? ""
        : `${path}必须为 number`;
    case "integer":
      return typeof value === "number" && Number.isInteger(value)
        ? ""
        : `${path}必须为 integer`;
    case "boolean":
      return typeof value === "boolean" ? "" : `${path}必须为 boolean`;
    case "object":
      if (!isRecord(value)) return `${path}必须为 object`;
      return validateRuntimePayload(value, field.children, path);
    case "array":
      if (!Array.isArray(value)) return `${path}必须为 array`;
      if (!field.item) return `${path}缺少数组元素定义`;
      for (let index = 0; index < value.length; index += 1) {
        const itemError = validateRuntimeValue(
          value[index],
          field.item,
          `${path}[${index}]`
        );
        if (itemError) return itemError;
      }
      return "";
    default:
      return "";
  }
};

const validateRuntimePayload = (
  payload: Record<string, unknown>,
  fields: PluginField[],
  scopeLabel = "输入参数"
) => {
  const fieldMap = new Map(fields.map((field) => [field.name.trim(), field]));

  for (const field of fields) {
    const fieldName = field.name.trim();
    if (!fieldName) continue;
    if (field.required && !(fieldName in payload)) {
      return `${scopeLabel}.${fieldName}为必填项`;
    }
  }

  for (const [key, value] of Object.entries(payload)) {
    const field = fieldMap.get(key);
    if (!field) {
      return `${scopeLabel}.${key}未在输入参数中声明`;
    }
    const valueError = validateRuntimeValue(value, field, `${scopeLabel}.${key}`);
    if (valueError) return valueError;
  }

  return "";
};

export const buildPayload = (draft: PluginDraft): Record<string, unknown> => {
  const payload: Record<string, unknown> = {
    name: draft.name.trim(),
    description: draft.description.trim(),
    icon: draft.icon.trim(),
    source: "custom",
    runtime_type: draft.runtimeType,
    enabled: draft.enabled,
  };

  if (draft.runtimeType === "mcp") {
    payload.mcp_url = draft.mcpURL.trim();
    payload.mcp_protocol = draft.mcpProtocol;
    payload.timeout_ms =
      Number.isFinite(draft.timeoutMS) && draft.timeoutMS >= 1
        ? draft.timeoutMS * 1000
        : 30000;
    payload.tools = [];
    return payload;
  }

  const nextTools = ensureRuntimeTools(draft.tools, draft.runtimeType);
  payload.tools = nextTools.map((tool) => {
    const toolPayload: Record<string, unknown> = {
      ...(tool.toolId ? { tool_id: tool.toolId } : {}),
      tool_name: tool.toolName.trim(),
      description: tool.description.trim(),
      enabled: true,
    };

    if (draft.runtimeType === "api") {
      toolPayload.tool_response_mode = tool.toolResponseMode;
      toolPayload.api_request_type = tool.apiRequestType;
      toolPayload.request_url = tool.requestURL.trim();
      toolPayload.auth_config_ref = "";
      toolPayload.timeout_ms =
        Number.isFinite(tool.timeoutMS) && tool.timeoutMS >= 1
          ? tool.timeoutMS * 1000
          : 30000;
      toolPayload.query_fields = tool.queryFields.map((field) =>
        toFieldPayload(field)
      );
      toolPayload.header_fields = tool.headerFields.map((field) =>
        toFieldPayload(field)
      );
      toolPayload.body_fields =
        tool.apiRequestType === "POST"
          ? tool.bodyFields.map((field) => toFieldPayload(field))
          : [];
    }

    if (draft.runtimeType === "code") {
      const codeBodyFields = getEffectiveCodeBodyFields(tool);
      const outputFields = sanitizeOutputFields(tool.outputFields);
      toolPayload.tool_response_mode = "non_streaming";
      toolPayload.timeout_ms =
        Number.isFinite(tool.timeoutMS) && tool.timeoutMS >= 1
          ? tool.timeoutMS * 1000
          : 30000;
      toolPayload.code_language = tool.codeLanguage.trim();
      toolPayload.code = tool.code;
      toolPayload.body_fields = codeBodyFields.map((field) =>
        toFieldPayload(field)
      );
      toolPayload.output_schema_json = buildObjectSchemaJSON(outputFields);
    }
    return toolPayload;
  });
  return payload;
};

const validateURL = (value: string) => {
  try {
    const target = new URL(value);
    return target.protocol === "http:" || target.protocol === "https:";
  } catch {
    return false;
  }
};

const validateDefaultValue = (field: PluginField, sectionLabel: string) => {
  const raw = field.defaultValue.trim();
  if (!raw) return "";

  if (field.type === "string") {
    return "";
  }
  if (field.type === "number" && Number.isFinite(Number(raw))) {
    return "";
  }
  if (field.type === "integer" && /^[-+]?\\d+$/.test(raw)) {
    return "";
  }
  if (field.type === "boolean" && /^(true|false|1|0)$/i.test(raw)) {
    return "";
  }

  if (field.type === "object" || field.type === "array") {
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (field.type === "object" && isRecord(parsed)) {
        return "";
      }
      if (field.type === "array" && Array.isArray(parsed)) {
        return "";
      }
    } catch {
      return `${sectionLabel}的默认值格式不正确`;
    }
  }

  return `${sectionLabel}的默认值与字段类型不匹配`;
};

const validateFieldList = (
  fields: PluginField[],
  sectionLabel: string,
  requireName = true
): string => {
  const names = new Set<string>();
  for (const field of fields) {
    const name = field.name.trim();
    if (requireName && !name) {
      return `${sectionLabel}存在未填写字段名的项`;
    }
    if (requireName) {
      if (names.has(name)) {
        return `${sectionLabel}存在重复字段名：${name}`;
      }
      names.add(name);
    }
    const defaultValueError = validateDefaultValue(
      field,
      `${sectionLabel}/${name || "字段"}`
    );
    if (defaultValueError) return defaultValueError;
    if (field.type === "string" && field.enumText.trim()) {
      const enums = field.enumText
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean);
      if (new Set(enums).size !== enums.length) {
        return `${sectionLabel}的枚举值不能重复`;
      }
    }
    if (field.type === "object") {
      const childError = validateFieldList(field.children, `${sectionLabel}/${name}`);
      if (childError) return childError;
    }
    if (field.type === "array") {
      if (!field.item) {
        return `${sectionLabel}/${name || "数组字段"}缺少数组元素定义`;
      }
      const itemError = validateFieldList(
        [field.item],
        `${sectionLabel}/${name || "数组字段"}[item]`,
        false
      );
      if (itemError) return itemError;
    }
  }
  return "";
};

export const validateDraft = (draft: PluginDraft) => {
  if (!draft.runtimeType) return "请选择插件运行方式";
  if (!draft.name.trim()) return "请输入插件名称";
  if (draft.runtimeType === "api") {
    const tools = ensureRuntimeTools(draft.tools, draft.runtimeType);
    if (tools.length === 0) return "请至少添加一个工具";
    const toolNameSet = new Set<string>();
    for (const [index, tool] of tools.entries()) {
      const label = `工具 ${index + 1}`;
      const toolName = tool.toolName.trim();
      if (!toolName) return `${label}：请输入工具名称`;
      if (!isValidToolName(toolName)) {
        return `${label}：工具名称仅支持字母、数字、下划线和中划线`;
      }
      if (toolNameSet.has(toolName)) {
        return `工具名称不能重复：${toolName}`;
      }
      toolNameSet.add(toolName);
      if (!tool.toolResponseMode) return `${label}：请选择响应模式`;
      if (!tool.requestURL.trim() || !validateURL(tool.requestURL.trim())) {
        return `${label}：请输入合法的 RequestURL`;
      }
      const queryError = validateFieldList(tool.queryFields, `${label}/Query 参数`);
      if (queryError) return queryError;
      const headerError = validateFieldList(tool.headerFields, `${label}/Header 参数`);
      if (headerError) return headerError;
      if (tool.apiRequestType === "GET" && tool.bodyFields.length > 0) {
        return `${label}：GET 类型不支持 body`;
      }
      if (tool.apiRequestType === "POST") {
        const bodyError = validateFieldList(tool.bodyFields, `${label}/Body 参数`);
        if (bodyError) return bodyError;
      }
    }
  }
  if (draft.runtimeType === "mcp") {
    if (!draft.mcpURL.trim() || !validateURL(draft.mcpURL.trim())) {
      return "请输入合法的 MCP URL";
    }
  }
  if (draft.runtimeType === "code") {
    const tools = ensureRuntimeTools(draft.tools, draft.runtimeType);
    if (tools.length === 0) return "请至少添加一个工具";
    const toolNameSet = new Set<string>();
    for (const [index, tool] of tools.entries()) {
      const label = `工具 ${index + 1}`;
      const toolName = tool.toolName.trim();
      if (!toolName) return `${label}：请输入工具名称`;
      if (!isValidToolName(toolName)) {
        return `${label}：工具名称仅支持字母、数字、下划线和中划线`;
      }
      if (toolNameSet.has(toolName)) {
        return `工具名称不能重复：${toolName}`;
      }
      toolNameSet.add(toolName);
      if (!["python", "javascript"].includes(tool.codeLanguage.trim())) {
        return `${label}：代码语言仅支持 python 或 javascript`;
      }
      if (!tool.code.trim()) return `${label}：请输入代码内容`;
      const inputError = validateFieldList(tool.bodyFields, `${label}/输入参数`);
      if (inputError) return inputError;
      const outputError = validateFieldList(
        sanitizeOutputFields(tool.outputFields),
        `${label}/输出参数`
      );
      if (outputError) return outputError;
    }
  }
  return "";
};
