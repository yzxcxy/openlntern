"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiMonacoEditor } from "../../components/ui/UiMonacoEditor";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import {
  buildAuthHeaders,
  readValidToken,
  updateTokenFromResponse,
} from "../auth";

type RuntimeType = "" | "api" | "mcp" | "code";
type ResponseMode = "" | "streaming" | "non_streaming";
type RequestType = "GET" | "POST";
type FieldType = "string" | "number" | "integer" | "boolean" | "object" | "array";
type MCPProtocol = "sse" | "streamableHttp";

type PluginField = {
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

type PluginTool = {
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

type PluginRecord = {
  plugin_id?: string;
  name?: string;
  description?: string;
  icon?: string;
  source?: string;
  runtime_type?: RuntimeType;
  status?: "enabled" | "disabled";
  mcp_url?: string;
  mcp_protocol?: string;
  last_sync_at?: string | null;
  tool_count?: number;
  tools?: PluginTool[];
  created_at?: string;
  updated_at?: string;
};

type ToolDraft = {
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

type PluginDraft = {
  pluginId?: string;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
  runtimeType: RuntimeType;
  mcpURL: string;
  mcpProtocol: MCPProtocol;
  tool: ToolDraft;
};

type DetailFieldSection = {
  key: string;
  label: string;
  fields: PluginField[];
};

type FlatFieldRow = {
  id: string;
  name: string;
  type: FieldType;
  required: boolean;
  description: string;
  defaultValue: string;
  enumText: string;
  depth: number;
};

const API_BASE = "/api/backend";
const TOOL_NAME_PATTERN = /^[a-zA-Z0-9_-]+$/;

const createId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const isValidToolName = (value: string) => TOOL_NAME_PATTERN.test(value.trim());

const createField = (type: FieldType = "string"): PluginField => ({
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

const sanitizeOutputFields = (fields: PluginField[]): PluginField[] =>
  fields.map((field) => sanitizeOutputField(field));

const createToolDraft = (): ToolDraft => ({
  toolName: "",
  description: "",
  toolResponseMode: "non_streaming",
  apiRequestType: "GET",
  requestURL: "",
  authConfigRef: "",
  timeoutMS: 30000,
  queryFields: [],
  headerFields: [],
  bodyFields: [],
  outputFields: [],
  codeLanguage: "javascript",
  code: "",
});

const createPluginDraft = (): PluginDraft => ({
  name: "",
  description: "",
  icon: "",
  enabled: true,
  runtimeType: "",
  mcpURL: "",
  mcpProtocol: "sse",
  tool: createToolDraft(),
});

const runtimeLabel: Record<Exclude<RuntimeType, "">, string> = {
  api: "API",
  mcp: "MCP",
  code: "Code",
};

const responseModeLabel: Record<Exclude<ResponseMode, "">, string> = {
  streaming: "流式",
  non_streaming: "非流式",
};

const mcpProtocolLabel: Record<MCPProtocol, string> = {
  sse: "服务器发送事件（SSE）",
  streamableHttp: "可流式传输的 HTTP（streamableHttp）",
};

const getSourceBadgeClassName = (source?: string) => {
  if (source === "builtin") {
    return "border-[rgba(217,119,6,0.18)] bg-[rgba(245,158,11,0.12)] text-[rgb(180,83,9)]";
  }
  return "border-[rgba(14,116,144,0.18)] bg-[rgba(6,182,212,0.12)] text-[rgb(14,116,144)]";
};

const getRuntimeBadgeClassName = (runtimeType?: RuntimeType) => {
  switch (runtimeType) {
    case "api":
      return "border-[rgba(37,99,235,0.18)] bg-[rgba(59,130,246,0.12)] text-[rgb(29,78,216)]";
    case "mcp":
      return "border-[rgba(13,148,136,0.18)] bg-[rgba(20,184,166,0.12)] text-[rgb(15,118,110)]";
    case "code":
      return "border-[rgba(234,88,12,0.18)] bg-[rgba(249,115,22,0.12)] text-[rgb(194,65,12)]";
    default:
      return "border-[var(--color-border-default)] bg-white text-[var(--color-text-secondary)]";
  }
};

const normalizeMCPProtocolValue = (value: unknown): MCPProtocol =>
  value === "streamableHttp" ? "streamableHttp" : "sse";

const getToolKey = (tool: PluginTool, index: number) =>
  tool.tool_id || tool.tool_name || `tool-${index}`;

const formatTime = (value?: string | null) => {
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

function PluginAvatar({
  src,
  name,
  fallbackSrc,
  className = "",
}: {
  src?: string;
  name?: string;
  fallbackSrc?: string;
  className?: string;
}) {
  const imageURL = src?.trim() || fallbackSrc?.trim() || "";
  const fallbackLabel = (name?.trim().slice(0, 1) || "P").toUpperCase();

  if (imageURL) {
    return (
      <div
        className={`overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-white ${className}`}
      >
        <img src={imageURL} alt={name || "plugin"} className="h-full w-full object-cover" />
      </div>
    );
  }

  return (
    <div
      className={`flex items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] text-sm font-semibold text-[var(--color-text-secondary)] ${className}`}
    >
      {fallbackLabel}
    </div>
  );
}

const parseFields = (value: unknown): PluginField[] => {
  if (!Array.isArray(value)) return [];
  return value.map((item) => parseField(item)).filter(Boolean) as PluginField[];
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const normalizeFieldType = (value: unknown): FieldType => {
  if (typeof value === "string" && ["string", "number", "integer", "boolean", "object", "array"].includes(value)) {
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
  const type = (["string", "number", "integer", "boolean", "object", "array"].includes(
    rawType
  )
    ? rawType
    : "string") as FieldType;
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
    description: typeof record.description === "string" ? record.description : "",
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
      ? value.required.filter((item): item is string => typeof item === "string")
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

const parseSchemaFieldsFromJSON = (raw: string | undefined, fallbackName: string): PluginField[] => {
  const trimmed = raw?.trim();
  if (!trimmed) return [];

  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (isRecord(parsed) && isRecord(parsed.properties)) {
      const requiredFields = new Set(
        Array.isArray(parsed.required)
          ? parsed.required.filter((item): item is string => typeof item === "string")
          : []
      );
      return Object.entries(parsed.properties as Record<string, unknown>)
        .map(([name, value]) => parseSchemaField(name, value, requiredFields.has(name)))
        .filter(Boolean) as PluginField[];
    }
    const rootField = parseSchemaField(fallbackName, parsed);
    return rootField ? [rootField] : [];
  } catch {
    return [];
  }
};

const getInputSections = (
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
    return bodyFields.length > 0 ? [{ key: "body", label: "输入参数", fields: bodyFields }] : [];
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

const getOutputFields = (tool: PluginTool) => parseSchemaFieldsFromJSON(tool.output_schema_json, "result");

const flattenFields = (fields: PluginField[]): FlatFieldRow[] => {
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

const toFieldPayload = (
  field: PluginField,
  isArrayItem = false
): Record<string, unknown> => ({
  ...(isArrayItem ? {} : { name: field.name.trim() }),
  type: field.type,
  required: field.required,
  description: field.description.trim(),
  ...(field.defaultValue.trim() ? { default_value: field.defaultValue.trim() } : {}),
  enum_values:
    field.type === "string"
      ? field.enumText
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean)
      : [],
  children:
    field.type === "object" ? field.children.map((child) => toFieldPayload(child)) : [],
  items:
    field.type === "array" && field.item ? toFieldPayload(field.item, true) : undefined,
});

const getCodeTemplate = (language: string) => {
  if (language === "python") {
    return [
      "# 请将入口函数命名为 main，入参为 params: dict，返回 dict",
      "def main(params: dict) -> dict:",
      "    return {",
      "        \"result\": params.get(\"input\"),",
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

const getDefaultCodeBodyFields = (): PluginField[] => [
  {
    ...createField("string"),
    name: "input",
    description: "传入 main(params) 的默认示例参数",
  },
];

const getEffectiveCodeBodyFields = (tool: ToolDraft): PluginField[] => {
  if (tool.bodyFields.length > 0) {
    return tool.bodyFields;
  }

  const codeLanguage = tool.codeLanguage.trim() === "python" ? "python" : "javascript";
  const defaultTemplate = getCodeTemplate(codeLanguage).trim();
  if (tool.code.trim() === defaultTemplate) {
    return getDefaultCodeBodyFields();
  }

  return [];
};

const applyCodeToolDefaults = (tool: ToolDraft): ToolDraft => ({
  ...tool,
  toolResponseMode: "non_streaming",
  code: tool.code.trim() ? tool.code : getCodeTemplate(tool.codeLanguage),
  bodyFields: getEffectiveCodeBodyFields(tool),
});

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

const validateRuntimeValue = (value: unknown, field: PluginField, path: string): string => {
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
      return typeof value === "number" && Number.isFinite(value) ? "" : `${path}必须为 number`;
    case "integer":
      return typeof value === "number" && Number.isInteger(value) ? "" : `${path}必须为 integer`;
    case "boolean":
      return typeof value === "boolean" ? "" : `${path}必须为 boolean`;
    case "object":
      if (!isRecord(value)) return `${path}必须为 object`;
      return validateRuntimePayload(value, field.children, path);
    case "array":
      if (!Array.isArray(value)) return `${path}必须为 array`;
      if (!field.item) return `${path}缺少数组元素定义`;
      for (let index = 0; index < value.length; index += 1) {
        const itemError = validateRuntimeValue(value[index], field.item, `${path}[${index}]`);
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

const buildPayload = (draft: PluginDraft): Record<string, unknown> => {
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
    payload.tools = [];
    return payload;
  }

  const toolPayload: Record<string, unknown> = {
    ...(draft.tool.toolId ? { tool_id: draft.tool.toolId } : {}),
    tool_name: draft.tool.toolName.trim(),
    description: draft.tool.description.trim(),
    enabled: true,
  };

  if (draft.runtimeType === "api") {
    toolPayload.tool_response_mode = draft.tool.toolResponseMode;
    toolPayload.api_request_type = draft.tool.apiRequestType;
    toolPayload.request_url = draft.tool.requestURL.trim();
    toolPayload.auth_config_ref = "";
    toolPayload.timeout_ms =
      Number.isFinite(draft.tool.timeoutMS) && draft.tool.timeoutMS >= 1
        ? draft.tool.timeoutMS
        : 30000;
    toolPayload.query_fields = draft.tool.queryFields.map((field) => toFieldPayload(field));
    toolPayload.header_fields = draft.tool.headerFields.map((field) => toFieldPayload(field));
    toolPayload.body_fields =
      draft.tool.apiRequestType === "POST"
        ? draft.tool.bodyFields.map((field) => toFieldPayload(field))
        : [];
  }

  if (draft.runtimeType === "code") {
    const codeBodyFields = getEffectiveCodeBodyFields(draft.tool);
    const outputFields = sanitizeOutputFields(draft.tool.outputFields);
    toolPayload.tool_response_mode = "non_streaming";
    toolPayload.code_language = draft.tool.codeLanguage.trim();
    toolPayload.code = draft.tool.code;
    toolPayload.body_fields = codeBodyFields.map((field) => toFieldPayload(field));
    toolPayload.output_schema_json = buildObjectSchemaJSON(outputFields);
  }

  payload.tools = [toolPayload];
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
  if (field.type === "integer" && /^[-+]?\d+$/.test(raw)) {
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
    const defaultValueError = validateDefaultValue(field, `${sectionLabel}/${name || "字段"}`);
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

const validateDraft = (draft: PluginDraft) => {
  if (!draft.runtimeType) return "请选择插件运行方式";
  if (!draft.name.trim()) return "请输入插件名称";
  if (draft.runtimeType === "api") {
    if (!draft.tool.toolName.trim()) return "请输入工具名称";
    if (!isValidToolName(draft.tool.toolName)) {
      return "工具名称仅支持字母、数字、下划线和中划线";
    }
    if (!draft.tool.toolResponseMode) return "请选择响应模式";
    if (!draft.tool.requestURL.trim() || !validateURL(draft.tool.requestURL.trim())) {
      return "请输入合法的 RequestURL";
    }
    const queryError = validateFieldList(draft.tool.queryFields, "Query 参数");
    if (queryError) return queryError;
    const headerError = validateFieldList(draft.tool.headerFields, "Header 参数");
    if (headerError) return headerError;
    if (draft.tool.apiRequestType === "GET" && draft.tool.bodyFields.length > 0) {
      return "GET 类型不支持 body";
    }
    if (draft.tool.apiRequestType === "POST") {
      const bodyError = validateFieldList(draft.tool.bodyFields, "Body 参数");
      if (bodyError) return bodyError;
    }
  }
  if (draft.runtimeType === "mcp") {
    if (!draft.mcpURL.trim() || !validateURL(draft.mcpURL.trim())) {
      return "请输入合法的 MCP URL";
    }
  }
  if (draft.runtimeType === "code") {
    if (!draft.tool.toolName.trim()) return "请输入工具名称";
    if (!isValidToolName(draft.tool.toolName)) {
      return "工具名称仅支持字母、数字、下划线和中划线";
    }
    if (!["python", "javascript"].includes(draft.tool.codeLanguage.trim())) {
      return "代码语言仅支持 python 或 javascript";
    }
    if (!draft.tool.code.trim()) return "请输入代码内容";
    const inputError = validateFieldList(draft.tool.bodyFields, "输入参数");
    if (inputError) return inputError;
    const outputError = validateFieldList(sanitizeOutputFields(draft.tool.outputFields), "输出参数");
    if (outputError) return outputError;
  }
  return "";
};

function FieldListEditor({
  label,
  fields,
  onChange,
  mode = "input",
}: {
  label: string;
  fields: PluginField[];
  onChange: (next: PluginField[]) => void;
  mode?: "input" | "output";
}) {
  const isOutput = mode === "output";
  const displayFields = isOutput ? sanitizeOutputFields(fields) : fields;
  const gridClassName = isOutput
    ? "grid grid-cols-[minmax(112px,1fr)_minmax(136px,1.2fr)_96px_40px] gap-2"
    : "grid grid-cols-[minmax(112px,1fr)_minmax(136px,1.2fr)_96px_52px_minmax(112px,1fr)_minmax(112px,1fr)_40px] gap-2";
  const minWidthClassName = isOutput ? "min-w-[440px]" : "min-w-[760px]";
  const commitChange = (next: PluginField[]) => onChange(isOutput ? sanitizeOutputFields(next) : next);

  return (
    <div className="overflow-hidden rounded-[14px] border border-[var(--color-border-default)] bg-white">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border-default)] px-3 py-2.5">
        <div className="text-sm font-semibold text-[var(--color-text-primary)]">{label}</div>
        <UiButton
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => commitChange([...displayFields, createField()])}
        >
          + 添加
        </UiButton>
      </div>
      <div className="overflow-x-auto">
        <div className={minWidthClassName}>
          <div
            className={`${gridClassName} border-b border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-3 py-2 text-xs font-semibold text-[var(--color-text-muted)]`}
          >
            <span>参数名称</span>
            <span>参数描述</span>
            <span>参数类型</span>
            {!isOutput && <span>必填</span>}
            {!isOutput && <span>默认值</span>}
            {!isOutput && <span>枚举值</span>}
            <span />
          </div>
          {displayFields.length === 0 ? (
            <div className="px-3 py-4 text-sm text-[var(--color-text-muted)]">暂无字段</div>
          ) : (
            <div className="divide-y divide-[var(--color-border-default)]">
              <FieldRowsEditor fields={displayFields} depth={0} onChange={commitChange} mode={mode} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function FieldRowsEditor({
  fields,
  depth,
  onChange,
  mode,
}: {
  fields: PluginField[];
  depth: number;
  onChange: (next: PluginField[]) => void;
  mode: "input" | "output";
}) {
  return (
    <>
      {fields.map((field, index) => (
        <FieldEditor
          key={field.id}
          field={field}
          depth={depth}
          onChange={(nextField) =>
            onChange(fields.map((item, itemIndex) => (itemIndex === index ? nextField : item)))
          }
          onRemove={() => onChange(fields.filter((_, itemIndex) => itemIndex !== index))}
          mode={mode}
        />
      ))}
    </>
  );
}

function FieldEditor({
  field,
  depth,
  onChange,
  onRemove,
  isArrayItem = false,
  mode,
}: {
  field: PluginField;
  depth: number;
  onChange: (next: PluginField) => void;
  onRemove?: () => void;
  isArrayItem?: boolean;
  mode: "input" | "output";
}) {
  const isOutput = mode === "output";
  const showEnumEditor = !isOutput && field.type === "string";
  const showObjectChildren = field.type === "object";
  const showArrayItem = field.type === "array" && field.item;
  const gridClassName = isOutput
    ? "grid grid-cols-[minmax(112px,1fr)_minmax(136px,1.2fr)_96px_40px] gap-2"
    : "grid grid-cols-[minmax(112px,1fr)_minmax(136px,1.2fr)_96px_52px_minmax(112px,1fr)_minmax(112px,1fr)_40px] gap-2";

  return (
    <>
      <div className={`${gridClassName} px-3 py-2`}>
        <div style={{ paddingLeft: `${depth * 24}px` }}>
          {isArrayItem ? (
            <div className="flex h-9 items-center rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-3 text-sm text-[var(--color-text-secondary)]">
              数组元素
            </div>
          ) : (
            <UiInput
              placeholder="参数名称"
              value={field.name}
              onChange={(event) => onChange({ ...field, name: event.target.value })}
            />
          )}
        </div>
        <UiInput
          placeholder="参数描述"
          value={field.description}
          onChange={(event) => onChange({ ...field, description: event.target.value })}
        />
        <UiSelect
          value={field.type}
          onChange={(event) => {
            const nextType = event.target.value as FieldType;
            onChange({
              ...field,
              type: nextType,
              children: nextType === "object" ? field.children : [],
              item: nextType === "array" ? field.item ?? createField("string") : null,
              enumText: nextType === "string" ? field.enumText : "",
            });
          }}
        >
          <option value="string">string</option>
          <option value="number">number</option>
          <option value="integer">integer</option>
          <option value="boolean">boolean</option>
              <option value="object">object</option>
              <option value="array">array</option>
            </UiSelect>
        {!isOutput && (
          <label className="flex h-10 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] px-2 text-sm text-[var(--color-text-secondary)]">
            <input
              type="checkbox"
              checked={field.required}
              onChange={(event) => onChange({ ...field, required: event.target.checked })}
            />
          </label>
        )}
        {!isOutput && (
          <UiInput
            placeholder={
              field.type === "object" || field.type === "array"
                ? "JSON 默认值（可选）"
                : "默认值（可选）"
            }
            value={field.defaultValue}
            onChange={(event) => onChange({ ...field, defaultValue: event.target.value })}
          />
        )}
        {!isOutput &&
          (showEnumEditor ? (
            <UiInput
              placeholder="逗号分隔，可选"
              value={field.enumText}
              onChange={(event) => onChange({ ...field, enumText: event.target.value })}
            />
          ) : (
            <div className="flex h-10 items-center rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] bg-[rgba(148,163,184,0.05)] px-3 text-xs text-[var(--color-text-muted)]">
              仅 string
            </div>
          ))}
        <div className="flex items-center justify-end gap-1">
          {showObjectChildren && (
            <UiButton
              type="button"
              variant="ghost"
              size="sm"
              onClick={() =>
                onChange({
                  ...field,
                  children: [...field.children, createField()],
                })
              }
            >
              子项
            </UiButton>
          )}
          {onRemove ? (
            <UiButton type="button" variant="ghost" size="sm" onClick={onRemove}>
              删
            </UiButton>
          ) : (
            <div className="h-8" />
          )}
        </div>
      </div>
      {showObjectChildren && field.children.length === 0 && (
        <div className="px-3 pb-2">
          <div
            className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-3 py-2 text-xs text-[var(--color-text-muted)]"
            style={{ marginLeft: `${(depth + 1) * 24}px` }}
          >
            当前对象还没有子字段，点击“添加子项”继续配置。
          </div>
        </div>
      )}
      {showObjectChildren && field.children.length > 0 && (
        <FieldRowsEditor
          fields={field.children}
          depth={depth + 1}
          onChange={(nextChildren) => onChange({ ...field, children: nextChildren })}
          mode={mode}
        />
      )}
      {showArrayItem && (
        <div className="border-t border-dashed border-[var(--color-border-default)] bg-[rgba(148,163,184,0.04)]">
          <FieldEditor
            field={field.item!}
            depth={depth + 1}
            isArrayItem
            onChange={(nextItem) => onChange({ ...field, item: nextItem })}
            mode={mode}
          />
        </div>
      )}
    </>
  );
}

function DetailSectionTitle({ title }: { title: string }) {
  return (
    <div className="flex items-center gap-3">
      <span className="h-6 w-1 rounded-full bg-[var(--color-action-primary)]" />
      <div className="text-base font-semibold text-[var(--color-text-primary)]">{title}</div>
    </div>
  );
}

function FormFieldRow({
  label,
  required = false,
  children,
}: {
  label: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="grid items-center gap-3 md:grid-cols-[140px_minmax(0,1fr)]">
      <div className="text-sm font-medium text-[var(--color-text-secondary)]">
        {label}
        {required && <span className="ml-1 text-[var(--color-state-error)]">*</span>}
      </div>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

function FieldTable({
  fields,
  emptyText = "暂无参数",
  showConstraints = true,
}: {
  fields: PluginField[];
  emptyText?: string;
  showConstraints?: boolean;
}) {
  const rows = flattenFields(fields);
  const gridClassName = showConstraints
    ? "grid grid-cols-[minmax(180px,2fr)_120px_100px_minmax(180px,1.4fr)_minmax(220px,2fr)_minmax(140px,1.2fr)] gap-3"
    : "grid grid-cols-[minmax(180px,2fr)_120px_minmax(220px,2fr)] gap-3";

  if (rows.length === 0) {
    return (
      <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] px-4 py-5 text-sm text-[var(--color-text-muted)]">
        {emptyText}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--color-border-default)]">
      <div
        className={`${gridClassName} bg-[var(--color-bg-page)] px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)]`}
      >
        <span>参数名称</span>
        <span>参数类型</span>
        {showConstraints && <span>必填</span>}
        {showConstraints && <span>默认值</span>}
        <span>参数描述</span>
        {showConstraints && <span>枚举值</span>}
      </div>
      <div className="divide-y divide-[var(--color-border-default)] bg-white">
        {rows.map((row) => (
          <div key={row.id} className={`${gridClassName} px-4 py-3 text-sm text-[var(--color-text-secondary)]`}>
            <span
              className="truncate text-[var(--color-text-primary)]"
              style={{ paddingLeft: `${row.depth * 16}px` }}
              title={row.name}
            >
              {row.name}
            </span>
            <span>{row.type}</span>
            {showConstraints && <span>{row.required ? "是" : "否"}</span>}
            {showConstraints && <span>{row.defaultValue || "-"}</span>}
            <span>{row.description || "-"}</span>
            {showConstraints && <span>{row.enumText || "-"}</span>}
          </div>
        ))}
      </div>
    </div>
  );
}

function CodeToolEditor({
  tool,
  onChange,
  onDebugRun,
}: {
  tool: ToolDraft;
  onChange: (next: ToolDraft) => void;
  onDebugRun: (tool: ToolDraft, input: Record<string, unknown>) => Promise<unknown>;
}) {
  const [testInput, setTestInput] = useState('{\n  "input": "A"\n}');
  const [testOutput, setTestOutput] = useState("");
  const [testError, setTestError] = useState("");
  const [debugging, setDebugging] = useState(false);
  const previousLanguageRef = useRef(tool.codeLanguage);

  useEffect(() => {
    const previousLanguage = previousLanguageRef.current;
    if (tool.codeLanguage === previousLanguage) return;

    previousLanguageRef.current = tool.codeLanguage;
    const previousTemplate = getCodeTemplate(previousLanguage);
    if (!tool.code.trim() || tool.code.trim() === previousTemplate.trim()) {
      onChange({
        ...tool,
        code: getCodeTemplate(tool.codeLanguage),
      });
    }
  }, [onChange, tool]);

  useEffect(() => {
    if (tool.bodyFields.length > 0) return;
    if (getEffectiveCodeBodyFields(tool).length === 0) return;

    onChange({
      ...tool,
      bodyFields: getDefaultCodeBodyFields(),
    });
  }, [onChange, tool]);

  const lineCount = Math.max(6, tool.code.split("\n").length);

  const runLocalCheck = async () => {
    setTestError("");

    let payload: unknown;
    try {
      payload = JSON.parse(testInput);
    } catch {
      setTestError("测试输入必须是合法的 JSON");
      setTestOutput("");
      return;
    }

    if (!isRecord(payload)) {
      setTestError("测试输入必须是 JSON 对象");
      setTestOutput("");
      return;
    }

    setDebugging(true);
    try {
      const result = await onDebugRun(tool, payload);
      setTestOutput(JSON.stringify(result, null, 2));
    } catch (error) {
      setTestOutput("");
      setTestError(error instanceof Error ? error.message : "调试执行失败");
    } finally {
      setDebugging(false);
    }
  };

  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,0.82fr)_minmax(280px,0.78fr)]">
      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2 px-1 py-1">
          <div>
            <div className="text-sm font-semibold text-white">
              {tool.codeLanguage === "python" ? "Python" : "JavaScript"}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <UiSelect
              value={tool.codeLanguage}
              onChange={(event) =>
                onChange({
                  ...tool,
                  codeLanguage: event.target.value.trim() === "python" ? "python" : "javascript",
                })
              }
            >
              <option value="javascript">javascript</option>
              <option value="python">python</option>
            </UiSelect>
            <UiButton
              type="button"
              variant="secondary"
              size="sm"
              onClick={() =>
                onChange({
                  ...tool,
                  code: getCodeTemplate(tool.codeLanguage),
                })
              }
            >
              重置模板
            </UiButton>
          </div>
        </div>

        <div className="grid min-h-[240px] grid-cols-[32px_minmax(0,1fr)] overflow-hidden rounded-[14px] bg-[rgb(15,23,42)]">
          <div className="border-r border-[rgba(148,163,184,0.1)] bg-[rgba(2,6,23,0.55)] px-1 py-2 text-right text-[10px] leading-4 text-[rgba(148,163,184,0.6)]">
            {Array.from({ length: lineCount }).map((_, index) => (
              <div key={index + 1}>{index + 1}</div>
            ))}
          </div>
          <UiMonacoEditor
            value={tool.code}
            language={tool.codeLanguage === "python" ? "python" : "javascript"}
            minHeight={240}
            fontSize={11}
            placeholder="在这里输入代码"
            className="w-full"
            onChange={(event) =>
              onChange({
                ...tool,
                code: event,
              })
            }
          />
        </div>

        <div className="grid gap-2 md:grid-cols-2">
          <div className="rounded-[12px] bg-[rgba(2,6,23,0.78)] px-3 py-2">
            <div className="mb-1.5 flex items-center justify-between gap-2">
              <div className="text-xs font-semibold text-white">输入测试</div>
              <UiButton type="button" size="sm" onClick={() => void runLocalCheck()} loading={debugging}>
                {debugging ? "执行中..." : "运行调试"}
              </UiButton>
            </div>
            <UiTextarea
              className="min-h-[76px] w-full resize-y rounded-[10px] border-[rgba(148,163,184,0.16)] bg-[rgba(15,23,42,0.72)] font-mono text-[10px] leading-4 text-[rgb(226,232,240)]"
              spellCheck={false}
              value={testInput}
              onChange={(event) => setTestInput(event.target.value)}
            />
            {testError ? (
              <div className="mt-1.5 rounded-[10px] border border-[rgba(248,113,113,0.26)] bg-[rgba(127,29,29,0.32)] px-2 py-1.5 text-[10px] text-[rgb(254,202,202)]">
                {testError}
              </div>
            ) : (
              <div className="mt-1.5 text-[10px] text-[rgba(148,163,184,0.82)]">
                前端仅校验 JSON 格式，随后直接调用后端调试接口执行沙箱代码。
              </div>
            )}
          </div>
          <div className="rounded-[12px] bg-[rgba(2,6,23,0.78)] px-3 py-2">
            <div className="mb-1.5 text-xs font-semibold text-white">输出结果</div>
            <pre className="min-h-[76px] overflow-auto rounded-[10px] border border-[rgba(148,163,184,0.16)] bg-[rgba(15,23,42,0.72)] px-2 py-2 font-mono text-[10px] leading-4 text-[rgb(226,232,240)]">
              {testOutput || "// 调试执行后，这里会展示后端返回结果"}
            </pre>
          </div>
        </div>
      </div>

      <div className="space-y-3">
        <div className="rounded-[18px] border border-[var(--color-border-default)] bg-white p-3.5 shadow-[var(--shadow-sm)]">
          <DetailSectionTitle title="基础信息" />
          <div className="mt-4 space-y-4">
            <FormFieldRow label="工具名称" required>
              <UiInput
                placeholder="请输入工具名称"
                value={tool.toolName}
                onChange={(event) =>
                  onChange({
                    ...tool,
                    toolName: event.target.value,
                  })
                }
              />
            </FormFieldRow>
            <FormFieldRow label="工具描述">
                <UiInput
                placeholder="请输入工具描述"
                value={tool.description}
                onChange={(event) =>
                  onChange({
                    ...tool,
                    description: event.target.value,
                  })
                }
              />
            </FormFieldRow>
          </div>
        </div>

        <FieldListEditor
          label="输入参数"
          fields={tool.bodyFields}
          onChange={(nextFields) =>
            onChange({
              ...tool,
              bodyFields: nextFields,
            })
          }
          mode="input"
        />

        <FieldListEditor
          label="输出参数"
          fields={sanitizeOutputFields(tool.outputFields)}
          onChange={(nextFields) =>
            onChange({
              ...tool,
              outputFields: sanitizeOutputFields(nextFields),
            })
          }
          mode="output"
        />
      </div>
    </div>
  );
}

export default function PluginsPage() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [searchKeyword, setSearchKeyword] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [runtimeFilter, setRuntimeFilter] = useState("");
  const [items, setItems] = useState<PluginRecord[]>([]);
  const [selectedPlugin, setSelectedPlugin] = useState<PluginRecord | null>(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(9);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [isWizardOpen, setIsWizardOpen] = useState(false);
  const [wizardStep, setWizardStep] = useState(1);
  const [draft, setDraft] = useState<PluginDraft>(createPluginDraft());
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [selectedToolKey, setSelectedToolKey] = useState("");
  const [uploadingIcon, setUploadingIcon] = useState(false);
  const [defaultPluginIconURL, setDefaultPluginIconURL] = useState("");
  const iconUploadInputRef = useRef<HTMLInputElement | null>(null);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const previewJSON = useMemo(() => JSON.stringify(buildPayload(draft), null, 2), [draft]);
  const activeTool = useMemo(() => {
    const tools = selectedPlugin?.tools ?? [];
    if (tools.length === 0) return null;
    return (
      tools.find((tool, index) => getToolKey(tool, index) === selectedToolKey) ?? tools[0] ?? null
    );
  }, [selectedPlugin, selectedToolKey]);
  const activeToolInputSections = useMemo(
    () =>
      activeTool ? getInputSections(selectedPlugin?.runtime_type, activeTool) : [],
    [activeTool, selectedPlugin?.runtime_type]
  );
  const activeToolOutputFields = useMemo(
    () => (activeTool ? getOutputFields(activeTool) : []),
    [activeTool]
  );

  const getToken = useCallback(() => readValidToken(router), [router]);

  useEffect(() => {
    const tools = selectedPlugin?.tools ?? [];
    if (tools.length === 0) {
      setSelectedToolKey("");
      return;
    }

    const nextKeys = tools.map((tool, index) => getToolKey(tool, index));
    setSelectedToolKey((current) =>
      current && nextKeys.includes(current) ? current : (nextKeys[0] ?? "")
    );
  }, [selectedPlugin]);

  const fetchList = useCallback(async () => {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", String(pageSize));
      if (searchKeyword.trim()) params.set("keyword", searchKeyword.trim());
      if (sourceFilter) params.set("source", sourceFilter);
      if (runtimeFilter) params.set("runtime_type", runtimeFilter);
      const res = await fetch(`${API_BASE}/v1/plugins?${params.toString()}`, {
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取插件列表失败");
      }
      setItems(Array.isArray(data.data?.data) ? data.data.data : []);
      setTotal(typeof data.data?.total === "number" ? data.data.total : 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取插件列表失败");
    } finally {
      setLoading(false);
    }
  }, [
    getToken,
    page,
    pageSize,
    runtimeFilter,
    searchKeyword,
    sourceFilter,
  ]);

  useEffect(() => {
    void fetchList();
  }, [fetchList]);

  const fetchPluginDefaults = useCallback(async () => {
    const token = getToken();
    if (!token) return;
    try {
      const res = await fetch(`${API_BASE}/v1/plugins/defaults`, {
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取插件默认配置失败");
      }
      const nextURL =
        typeof data.data?.default_icon_url === "string" ? data.data.default_icon_url.trim() : "";
      setDefaultPluginIconURL(nextURL);
    } catch {
      setDefaultPluginIconURL("");
    }
  }, [getToken]);

  useEffect(() => {
    void fetchPluginDefaults();
  }, [fetchPluginDefaults]);

  const fetchPluginDetail = async (pluginId: string) => {
    const token = getToken();
    if (!token) return null;
    const res = await fetch(`${API_BASE}/v1/plugins/${pluginId}`, {
      headers: buildAuthHeaders(token),
    });
    updateTokenFromResponse(res);
    const data = await res.json();
    if (!res.ok || data.code !== 0) {
      throw new Error(data.message || "获取插件详情失败");
    }
    return (data.data ?? null) as PluginRecord | null;
  };

  const fillDraftFromPlugin = (plugin: PluginRecord) => {
    const tool = plugin.tools?.[0];
    setDraft({
      pluginId: plugin.plugin_id,
      name: plugin.name ?? "",
      description: plugin.description ?? "",
      icon: plugin.icon ?? "",
      enabled: plugin.status !== "disabled",
      runtimeType: plugin.runtime_type ?? "",
      mcpURL: plugin.mcp_url ?? "",
      mcpProtocol: normalizeMCPProtocolValue(plugin.mcp_protocol),
      tool: {
        toolId: tool?.tool_id,
        toolName: tool?.tool_name ?? "",
        description: tool?.description ?? "",
        toolResponseMode:
          tool?.tool_response_mode === "streaming" ? "streaming" : "non_streaming",
        apiRequestType: tool?.api_request_type ?? "GET",
        requestURL: tool?.request_url ?? "",
        authConfigRef: tool?.auth_config_ref ?? "",
        timeoutMS:
          typeof tool?.timeout_ms === "number" && tool.timeout_ms >= 1
            ? tool.timeout_ms
            : 30000,
        queryFields: parseFields(tool?.query_fields),
        headerFields: parseFields(tool?.header_fields),
        bodyFields: parseFields(tool?.body_fields),
        outputFields: getOutputFields(tool ?? {}),
        codeLanguage: tool?.code_language === "python" ? "python" : "javascript",
        code:
          tool?.code && tool.code.trim()
            ? tool.code
            : getCodeTemplate(tool?.code_language === "python" ? "python" : "javascript"),
      },
    });
  };

  const openCreate = () => {
    if (sourceFilter === "builtin") {
      return;
    }
    setDraft(createPluginDraft());
    setWizardStep(1);
    setFormError("");
    setIsWizardOpen(true);
  };

  const openEdit = async (pluginId?: string) => {
    if (!pluginId) return;
    setLoadingDetail(true);
    setFormError("");
    try {
      const detail = await fetchPluginDetail(pluginId);
      if (!detail) return;
      fillDraftFromPlugin(detail);
      setWizardStep(1);
      setIsWizardOpen(true);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "获取插件详情失败");
    } finally {
      setLoadingDetail(false);
    }
  };

  const openDetail = async (pluginId?: string) => {
    if (!pluginId) return;
    setLoadingDetail(true);
    setError("");
    try {
      const detail = await fetchPluginDetail(pluginId);
      setSelectedPlugin(detail);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取插件详情失败");
    } finally {
      setLoadingDetail(false);
    }
  };

  const handleSearch = () => {
    setPage(1);
    setSearchKeyword(keyword);
  };

  const closeWizard = () => {
    setIsWizardOpen(false);
    setSaving(false);
  };

  const closeDetail = () => {
    setSelectedPlugin(null);
  };

  const openIconUpload = () => {
    setFormError("");
    iconUploadInputRef.current?.click();
  };

  const handleIconFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    const token = getToken();
    if (!token) return;
    setUploadingIcon(true);
    setFormError("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(`${API_BASE}/v1/plugins/icon`, {
        method: "POST",
        headers: buildAuthHeaders(token),
        body: formData,
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "上传头像失败");
      }
      const url = typeof data.data?.url === "string" ? data.data.url : "";
      if (!url) {
        throw new Error("上传头像失败");
      }
      setDraft((current) => ({ ...current, icon: url }));
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "上传头像失败");
    } finally {
      setUploadingIcon(false);
    }
  };

  const debugCodeTool = useCallback(
    async (tool: ToolDraft, input: Record<string, unknown>) => {
      const token = getToken();
      if (!token) {
        throw new Error("登录已失效，请重新登录后再试");
      }

      const res = await fetch(`${API_BASE}/v1/plugins/code/debug`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...buildAuthHeaders(token),
        },
        body: JSON.stringify({
          code: tool.code,
          code_language: tool.codeLanguage.trim(),
          input,
          timeout_ms:
            Number.isFinite(tool.timeoutMS) && tool.timeoutMS >= 1
              ? tool.timeoutMS
              : 30000,
        }),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "调试执行失败");
      }
      return data.data ?? null;
    },
    [getToken]
  );

  const goNext = async () => {
    if (wizardStep === 1 && !draft.runtimeType) {
      setFormError("请选择插件运行方式");
      return;
    }
    if (wizardStep === 2 && !draft.name.trim()) {
      setFormError("请输入插件名称");
      return;
    }
    if (wizardStep === 3) {
      const message = validateDraft(draft);
      if (message) {
        setFormError(message);
        return;
      }
    }
    setFormError("");
    if (wizardStep < 4) {
      setWizardStep((current) => current + 1);
      return;
    }
    const token = getToken();
    if (!token) return;
    setSaving(true);
    try {
      const payload = buildPayload(draft);
      const isEditing = Boolean(draft.pluginId);
      const res = await fetch(
        isEditing
          ? `${API_BASE}/v1/plugins/${draft.pluginId}`
          : `${API_BASE}/v1/plugins`,
        {
          method: isEditing ? "PUT" : "POST",
          headers: {
            "Content-Type": "application/json",
            ...buildAuthHeaders(token),
          },
          body: JSON.stringify(payload),
        }
      );
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "保存插件失败");
      }
      closeWizard();
      const savedPlugin = (data.data ?? null) as PluginRecord | null;
      setSelectedPlugin((current) =>
        current?.plugin_id && current.plugin_id === savedPlugin?.plugin_id ? savedPlugin : current
      );
      await fetchList();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "保存插件失败");
    } finally {
      setSaving(false);
    }
  };

  const changeStatus = async (plugin: PluginRecord, enable: boolean) => {
    if (!plugin.plugin_id) return;
    const token = getToken();
    if (!token) return;
    setError("");
    try {
      const res = await fetch(
        `${API_BASE}/v1/plugins/${plugin.plugin_id}/${enable ? "enable" : "disable"}`,
        {
          method: "POST",
          headers: buildAuthHeaders(token),
        }
      );
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "更新插件状态失败");
      }
      const nextPlugin = data.data as PluginRecord;
      setItems((current) =>
        current.map((item) => (item.plugin_id === nextPlugin.plugin_id ? nextPlugin : item))
      );
      setSelectedPlugin((current) =>
        current?.plugin_id === nextPlugin.plugin_id ? nextPlugin : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新插件状态失败");
    }
  };

  const syncPlugin = async (plugin: PluginRecord) => {
    if (!plugin.plugin_id) return;
    const token = getToken();
    if (!token) return;
    setError("");
    try {
      const res = await fetch(`${API_BASE}/v1/plugins/${plugin.plugin_id}/sync`, {
        method: "POST",
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "同步失败");
      }
      const nextPlugin = data.data as PluginRecord;
      setItems((current) =>
        current.map((item) => (item.plugin_id === nextPlugin.plugin_id ? nextPlugin : item))
      );
      setSelectedPlugin((current) =>
        current?.plugin_id === nextPlugin.plugin_id ? nextPlugin : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "同步失败");
    }
  };

  const removePlugin = async (plugin: PluginRecord) => {
    if (!plugin.plugin_id) return;
    if (!window.confirm(`确认删除插件「${plugin.name || plugin.plugin_id}」吗？`)) {
      return;
    }
    const token = getToken();
    if (!token) return;
    setError("");
    try {
      const res = await fetch(`${API_BASE}/v1/plugins/${plugin.plugin_id}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "删除插件失败");
      }
      setItems((current) => current.filter((item) => item.plugin_id !== plugin.plugin_id));
      setTotal((current) => Math.max(0, current - 1));
      setSelectedPlugin((current) =>
        current?.plugin_id === plugin.plugin_id ? null : current
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除插件失败");
    }
  };

  const isBuiltinFilterActive = sourceFilter === "builtin";

  return (
    <div className="workspace-gradient-surface workspace-gradient-surface--panel h-full overflow-auto p-6">
      <div className="workspace-panel-card rounded-[var(--radius-xl)] border border-[var(--color-border-default)] p-5">
        {!selectedPlugin && (
          <>
            <div className="workspace-toolbar-surface rounded-[var(--radius-lg)] border p-3">
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(260px,2.2fr)_160px_160px_auto_auto]">
                <UiInput
                  className="min-w-0"
                  placeholder="搜索插件名称或描述"
                  value={keyword}
                  onChange={(event) => setKeyword(event.target.value)}
                />
                <UiSelect
                  className="min-w-0"
                  value={sourceFilter}
                  onChange={(event) => {
                    setSourceFilter(event.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">全部来源</option>
                  <option value="custom">自定义</option>
                  <option value="builtin">内建</option>
                </UiSelect>
                <UiSelect
                  className="min-w-0"
                  value={runtimeFilter}
                  onChange={(event) => {
                    setRuntimeFilter(event.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">全部类型</option>
                  <option value="api">API</option>
                  <option value="mcp">MCP</option>
                  <option value="code">Code</option>
                </UiSelect>
                <UiButton
                  type="button"
                  variant="secondary"
                  className="w-full xl:w-auto"
                  onClick={handleSearch}
                >
                  搜索
                </UiButton>
                {!isBuiltinFilterActive && (
                  <UiButton type="button" className="w-full xl:w-auto" onClick={openCreate}>
                    新增插件
                  </UiButton>
                )}
              </div>
            </div>

            <div className="mt-4 flex items-center justify-between gap-3">
              <div className="text-sm text-[var(--color-text-muted)]">共 {total} 条</div>
              {loadingDetail && (
                <div className="text-xs text-[var(--color-text-muted)]">详情加载中...</div>
              )}
            </div>
          </>
        )}

        {error && (
          <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            {error}
          </div>
        )}
        {formError && !isWizardOpen && (
          <div className="mt-3 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
            {formError}
          </div>
        )}

        {!selectedPlugin ? (
          <>
            <div className="mt-4">
              {loading ? (
                <div className="text-sm text-[var(--color-text-muted)]">加载中...</div>
              ) : items.length === 0 ? (
                <div className="rounded-[var(--radius-lg)] border border-dashed border-[var(--color-border-default)] p-6 text-center">
                  <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                    {isBuiltinFilterActive ? "暂无内建插件" : "还没有自定义插件"}
                  </div>
                  <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                    {isBuiltinFilterActive
                      ? "当前筛选条件下没有可用的内建插件。"
                      : "从 API、MCP 或 Code 向导开始创建。"}
                  </div>
                  {!isBuiltinFilterActive && (
                    <div className="mt-4 flex flex-wrap justify-center gap-2">
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "api";
                          next.tool.toolResponseMode = "non_streaming";
                          setDraft(next);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 API 插件
                      </UiButton>
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "mcp";
                          setDraft(next);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 MCP 插件
                      </UiButton>
                      <UiButton
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          const next = createPluginDraft();
                          next.runtimeType = "code";
                          next.tool = applyCodeToolDefaults(next.tool);
                          setDraft(next);
                          setWizardStep(2);
                          setFormError("");
                          setIsWizardOpen(true);
                        }}
                      >
                        新建 Code 插件
                      </UiButton>
                    </div>
                  )}
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  {items.map((item) => {
                    const toolCount = item.tool_count ?? item.tools?.length ?? 0;
                    return (
                      <button
                        key={item.plugin_id || item.name}
                        type="button"
                        onClick={() => void openDetail(item.plugin_id)}
                        className="workspace-item-surface workspace-item-hover-lift flex flex-col rounded-[var(--radius-lg)] border border-[var(--color-border-default)] p-4 text-left shadow-[var(--shadow-sm)]"
                      >
                        <div className="flex items-start gap-3">
                          <PluginAvatar
                            src={item.icon}
                            name={item.name}
                            fallbackSrc={defaultPluginIconURL}
                            className="h-11 w-11 shrink-0"
                          />
                          <div className="min-w-0 flex-1">
                            <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
                              {item.name || "未命名插件"}
                            </div>
                          </div>
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2 text-xs">
                          <span
                            className={`rounded-full border px-2 py-1 ${getSourceBadgeClassName(item.source)}`}
                          >
                            {item.source === "builtin" ? "内建" : "自定义"}
                          </span>
                          <span
                            className={`rounded-full border px-2 py-1 ${getRuntimeBadgeClassName(
                              item.runtime_type
                            )}`}
                          >
                            {item.runtime_type
                              ? runtimeLabel[item.runtime_type as Exclude<RuntimeType, "">]
                              : "-"}
                          </span>
                          <span
                            className={`rounded-full px-2 py-1 ${
                              item.status === "enabled"
                                ? "bg-[rgba(22,163,74,0.12)] text-[var(--color-state-success)]"
                                : "bg-[rgba(148,163,184,0.14)] text-[var(--color-text-muted)]"
                            }`}
                          >
                            {item.status || "disabled"}
                          </span>
                          <span className="rounded-full bg-[rgba(37,99,255,0.08)] px-2 py-1 text-[var(--color-action-primary)]">
                            {toolCount} 个工具
                          </span>
                        </div>
                        <div
                          className="mt-3 line-clamp-2 text-xs text-[var(--color-text-muted)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                          dangerouslySetInnerHTML={{
                            __html: item.description?.trim() || "暂无描述",
                          }}
                        />
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            <div className="mt-5 flex flex-wrap items-center justify-end gap-3 text-sm text-[var(--color-text-secondary)]">
              <div className="flex shrink-0 items-center gap-2">
                <span className="shrink-0 whitespace-nowrap">每页</span>
                <UiSelect
                  className="w-24"
                  value={pageSize}
                  onChange={(event) => {
                    setPageSize(Number(event.target.value));
                    setPage(1);
                  }}
                >
                  <option value={9}>9</option>
                  <option value={18}>18</option>
                  <option value={36}>36</option>
                </UiSelect>
              </div>
              <UiButton
                type="button"
                variant="secondary"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                上一页
              </UiButton>
              <span>
                {page} / {totalPages}
              </span>
              <UiButton
                type="button"
                variant="secondary"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
              >
                下一页
              </UiButton>
            </div>
          </>
        ) : (
          <div className="mt-4 rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-5">
            {(() => {
              const detailToolCount =
                selectedPlugin.tool_count ?? selectedPlugin.tools?.length ?? 0;
              return (
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex min-w-0 items-start gap-4">
                    <button
                      type="button"
                      onClick={closeDetail}
                      className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-white text-[var(--color-text-secondary)]"
                      aria-label="返回插件列表"
                    >
                      <svg
                        className="h-4 w-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.8"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <path d="M15 18l-6-6 6-6" />
                      </svg>
                    </button>
                    <PluginAvatar
                      src={selectedPlugin.icon}
                      name={selectedPlugin.name}
                      fallbackSrc={defaultPluginIconURL}
                      className="h-14 w-14 shrink-0"
                    />
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                        <div className="truncate text-lg font-semibold text-[var(--color-text-primary)]">
                          {selectedPlugin.name || "未命名插件"}
                        </div>
                        <div className="text-xs text-[var(--color-text-muted)]">
                          更新时间：{formatTime(selectedPlugin.updated_at)}
                        </div>
                      </div>
                      <div className="mt-1 flex flex-wrap items-center gap-2 text-xs">
                        <span
                          className={`rounded-full border px-2 py-1 ${getSourceBadgeClassName(
                            selectedPlugin.source
                          )}`}
                        >
                          {selectedPlugin.source === "builtin" ? "内建" : "自定义"}
                        </span>
                        <span
                          className={`rounded-full border px-2 py-1 ${getRuntimeBadgeClassName(
                            selectedPlugin.runtime_type
                          )}`}
                        >
                          {selectedPlugin.runtime_type
                            ? runtimeLabel[selectedPlugin.runtime_type as Exclude<RuntimeType, "">]
                            : "-"}
                        </span>
                        <span
                          className={`rounded-full border px-2 py-1 ${
                            selectedPlugin.status === "enabled"
                              ? "border-[rgba(22,163,74,0.2)] bg-[rgba(22,163,74,0.12)] text-[var(--color-state-success)]"
                              : "border-[rgba(148,163,184,0.2)] bg-[rgba(148,163,184,0.14)] text-[var(--color-text-muted)]"
                          }`}
                        >
                          {selectedPlugin.status || "disabled"}
                        </span>
                        <span className="rounded-full border border-[rgba(37,99,255,0.16)] bg-[rgba(37,99,255,0.08)] px-2 py-1 text-[var(--color-action-primary)]">
                          {detailToolCount} 个工具
                        </span>
                      </div>
                      <div
                        className="mt-2 text-sm text-[var(--color-text-secondary)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                        dangerouslySetInnerHTML={{
                          __html: selectedPlugin.description?.trim() || "暂无描述",
                        }}
                      />
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => void openEdit(selectedPlugin.plugin_id)}
                    >
                      编辑
                    </UiButton>
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() =>
                        void changeStatus(selectedPlugin, selectedPlugin.status !== "enabled")
                      }
                    >
                      {selectedPlugin.status === "enabled" ? "停用" : "启用"}
                    </UiButton>
                    {selectedPlugin.runtime_type === "mcp" && (
                      <UiButton
                        type="button"
                        variant="secondary"
                        size="sm"
                        onClick={() => void syncPlugin(selectedPlugin)}
                      >
                        手动同步
                      </UiButton>
                    )}
                    <UiButton
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => void removePlugin(selectedPlugin)}
                    >
                      删除
                    </UiButton>
                  </div>
                </div>
              );
            })()}

            {selectedPlugin.tools && selectedPlugin.tools.length > 0 && activeTool ? (
              <div className="mt-5 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-white">
                <div className="border-b border-[var(--color-border-default)] px-5 py-4">
                  <div className="flex min-w-full gap-2 overflow-x-auto pb-1">
                    {selectedPlugin.tools.map((tool, index) => {
                      const key = getToolKey(tool, index);
                      const active = key === selectedToolKey;
                      return (
                        <button
                          key={key}
                          type="button"
                          onClick={() => setSelectedToolKey(key)}
                          className={`shrink-0 whitespace-nowrap rounded-[var(--radius-md)] border px-4 py-2 text-sm transition ${
                            active
                              ? "border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.08)] font-semibold text-[var(--color-action-primary)]"
                              : "border-[var(--color-border-default)] text-[var(--color-text-secondary)]"
                          }`}
                        >
                          {tool.tool_name || `工具 ${index + 1}`}
                        </button>
                      );
                    })}
                  </div>
                </div>

                <div className="space-y-6 px-5 py-5">
                  <section className="space-y-4">
                    <DetailSectionTitle title="基础信息" />
                    <div className="grid gap-3 md:grid-cols-2">
                      <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                        <div className="text-xs text-[var(--color-text-muted)]">调用方式</div>
                        <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                          {activeTool.tool_response_mode
                            ? responseModeLabel[
                                activeTool.tool_response_mode as Exclude<ResponseMode, "">
                              ] || activeTool.tool_response_mode
                            : "-"}
                        </div>
                      </div>
                      {selectedPlugin.runtime_type === "api" && (
                        <>
                          <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                            <div className="text-xs text-[var(--color-text-muted)]">请求方式</div>
                            <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                              {activeTool.api_request_type || "-"}
                            </div>
                          </div>
                          <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                            <div className="text-xs text-[var(--color-text-muted)]">请求地址</div>
                            <div className="mt-1 break-all text-sm font-semibold text-[var(--color-text-primary)]">
                              {activeTool.request_url || "-"}
                            </div>
                          </div>
                        </>
                      )}
                      {selectedPlugin.runtime_type === "code" && (
                        <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3">
                          <div className="text-xs text-[var(--color-text-muted)]">代码语言</div>
                          <div className="mt-1 text-sm font-semibold text-[var(--color-text-primary)]">
                            {activeTool.code_language || "-"}
                          </div>
                        </div>
                      )}
                      {selectedPlugin.runtime_type === "mcp" && (
                        <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                          <div className="text-xs text-[var(--color-text-muted)]">MCP 地址</div>
                          <div className="mt-1 break-all text-sm font-semibold text-[var(--color-text-primary)]">
                            {selectedPlugin.mcp_url || "-"}
                          </div>
                        </div>
                      )}
                      <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-page)] px-4 py-3 md:col-span-2">
                        <div className="text-xs text-[var(--color-text-muted)]">工具描述</div>
                        <div
                          className="mt-1 text-sm text-[var(--color-text-primary)] [&_a]:break-all [&_a]:font-medium [&_a]:text-[var(--color-action-primary)] [&_a]:underline [&_a]:underline-offset-2 [&_a:hover]:text-[var(--color-action-primary-hover)]"
                          dangerouslySetInnerHTML={{
                            __html: activeTool.description?.trim() || "暂无描述",
                          }}
                        />
                      </div>
                    </div>
                  </section>

                  <section className="space-y-4">
                    <DetailSectionTitle title="入参数" />
                    {activeToolInputSections.length > 0 ? (
                      <div className="space-y-4">
                        {activeToolInputSections.map((section) => (
                          <div key={section.key} className="space-y-2">
                            <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                              {section.label}
                            </div>
                            <FieldTable fields={section.fields} />
                          </div>
                        ))}
                      </div>
                    ) : (
                      <FieldTable fields={[]} emptyText="暂无入参数配置" />
                    )}
                  </section>

                  <section className="space-y-4">
                    <DetailSectionTitle title="出参数" />
                    <FieldTable
                      fields={activeToolOutputFields}
                      emptyText="暂无结构化出参数配置"
                      showConstraints={false}
                    />
                  </section>
                </div>
              </div>
            ) : (
              <div className="mt-5 text-sm text-[var(--color-text-muted)]">暂无工具</div>
            )}
          </div>
        )}
      </div>

      {isWizardOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(15,23,42,0.42)] p-2 md:p-6">
          <div className="flex h-[94vh] w-full max-w-[1440px] flex-col overflow-hidden rounded-[28px] border border-[var(--color-border-default)] bg-white shadow-[var(--shadow-lg)]">
            <div className="flex items-center justify-between gap-3 border-b border-[var(--color-border-default)] px-8 py-5">
              <div>
                <div className="text-xl font-semibold text-[var(--color-text-primary)]">
                  {draft.pluginId ? "编辑插件" : "新增插件"}
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  步骤 {wizardStep} / 4
                </div>
              </div>
              <button
                type="button"
                className="text-sm text-[var(--color-text-muted)]"
                onClick={closeWizard}
              >
                关闭
              </button>
            </div>

            <div className="flex-1 overflow-auto px-8 py-6">
              <div className="grid gap-3 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-3 md:grid-cols-4">
                {[
                  draft.pluginId ? "插件类型" : "选择类型",
                  "基础信息",
                  "运行配置",
                  "预览确认",
                ].map((label, index) => {
                  const step = index + 1;
                  const active = step === wizardStep;
                  return (
                    <div
                      key={label}
                      className={`flex items-center justify-center gap-3 rounded-[var(--radius-md)] border px-3 py-3 text-sm ${
                        active
                          ? "border-[var(--color-action-primary)] bg-[var(--color-action-primary)] font-semibold text-white shadow-[var(--shadow-sm)]"
                          : "border-transparent bg-white text-[var(--color-text-muted)]"
                      }`}
                    >
                      <span
                        className={`flex h-7 w-7 items-center justify-center rounded-full border ${
                          active
                            ? "border-white/50 bg-white/10"
                            : "border-[var(--color-border-default)]"
                        }`}
                      >
                        {step}
                      </span>
                      <span>{label}</span>
                    </div>
                  );
                })}
              </div>

              <div className="mt-6">
                {wizardStep === 1 && (
                  <div>
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      {draft.pluginId ? "第一步：查看插件类型" : "第一步：选择来源与运行方式"}
                    </div>
                    <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                      {draft.pluginId ? (
                        "插件类型在创建时已确定，编辑时不可修改。"
                      ) : (
                        <>来源固定为 `custom`，`builtin` 保持只读。</>
                      )}
                    </div>
                    {draft.pluginId ? (
                      <div className="mt-4 rounded-[var(--radius-lg)] border border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.06)] p-4">
                        <div className="text-base font-semibold text-[var(--color-text-primary)]">
                          {draft.runtimeType ? runtimeLabel[draft.runtimeType] : "未设置"}
                        </div>
                        <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                          {draft.runtimeType === "api" &&
                            "向导式配置请求方式、响应模式和 query/header/body 参数。"}
                          {draft.runtimeType === "mcp" &&
                            "维护连接地址与同步状态，当前主要管理定义与手动同步。"}
                          {draft.runtimeType === "code" &&
                            "配置输入参数、语言和脚本内容。"}
                        </div>
                      </div>
                    ) : (
                      <div className="mt-4 grid gap-3 md:grid-cols-3">
                        {(["api", "mcp", "code"] as Array<Exclude<RuntimeType, "">>).map(
                          (runtime) => (
                            <button
                              key={runtime}
                              type="button"
                              onClick={() =>
                                setDraft((current) => ({
                                  ...current,
                                  runtimeType: runtime,
                                  tool:
                                    runtime === "code"
                                      ? applyCodeToolDefaults(current.tool)
                                      : current.tool,
                                }))
                              }
                              className={`rounded-[var(--radius-lg)] border p-4 text-left ${
                                draft.runtimeType === runtime
                                  ? "border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.06)]"
                                  : "border-[var(--color-border-default)]"
                              }`}
                            >
                              <div className="text-base font-semibold text-[var(--color-text-primary)]">
                                {runtimeLabel[runtime]}
                              </div>
                              <div className="mt-2 text-xs text-[var(--color-text-muted)]">
                                {runtime === "api" &&
                                  "向导式配置请求方式、响应模式和 query/header/body 参数。"}
                                {runtime === "mcp" &&
                                  "维护连接地址与同步状态，当前主要管理定义与手动同步。"}
                                {runtime === "code" &&
                                  "配置输入参数、语言和脚本内容，并固定使用非流式。"}
                              </div>
                            </button>
                          )
                        )}
                      </div>
                    )}
                  </div>
                )}

                {wizardStep === 2 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第二步：填写基础信息
                    </div>
                    <FormFieldRow label="插件名称" required>
                      <UiInput
                        placeholder="请输入插件名称"
                        value={draft.name}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, name: event.target.value }))
                        }
                      />
                    </FormFieldRow>
                    <FormFieldRow label="插件描述">
                      <UiInput
                        placeholder="请输入插件描述"
                        value={draft.description}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, description: event.target.value }))
                        }
                      />
                    </FormFieldRow>
                    <FormFieldRow label="头像">
                      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-3">
                        <div className="flex flex-col gap-3 lg:flex-row">
                          <div className="min-w-0 flex-1">
                            <UiInput
                              type="text"
                              inputMode="url"
                              autoComplete="off"
                              spellCheck={false}
                              placeholder="请输入头像 URL（可选）"
                              value={draft.icon}
                              onChange={(event) =>
                                setDraft((current) => ({ ...current, icon: event.target.value }))
                              }
                            />
                          </div>
                          <UiButton
                            variant="secondary"
                            onClick={openIconUpload}
                            loading={uploadingIcon}
                          >
                            上传头像
                          </UiButton>
                        </div>
                        <div className="mt-3 flex items-center gap-3 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border-default)] bg-white px-3 py-2 text-xs text-[var(--color-text-muted)]">
                          <PluginAvatar
                            src={draft.icon}
                            name={draft.name}
                            fallbackSrc={defaultPluginIconURL}
                            className="h-12 w-12 shrink-0"
                          />
                          <span className="truncate">
                            {draft.icon.trim()
                              ? "已回填头像地址，可继续手动调整"
                              : defaultPluginIconURL
                                ? "未填写时将使用后端默认头像，也可上传后回填 URL"
                                : "支持直接输入 URL，或上传图片后自动回填 URL"}
                          </span>
                        </div>
                        {/* eslint-disable-next-line no-restricted-syntax */}
                        <input
                          ref={iconUploadInputRef}
                          type="file"
                          accept="image/*"
                          className="hidden"
                          onChange={(event) => {
                            void handleIconFileChange(event);
                          }}
                        />
                      </div>
                    </FormFieldRow>
                    <label className="flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--color-border-default)] px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                      <input
                        type="checkbox"
                        checked={draft.enabled}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, enabled: event.target.checked }))
                        }
                      />
                      创建后立即启用
                    </label>
                  </div>
                )}

                {wizardStep === 3 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第三步：填写运行配置
                    </div>

                    {draft.runtimeType === "api" && (
                      <>
                        <div className="space-y-4">
                          <FormFieldRow label="工具名称" required>
                            <UiInput
                              placeholder="请输入工具名称"
                              value={draft.tool.toolName}
                              onChange={(event) =>
                                setDraft((current) => ({
                                  ...current,
                                  tool: { ...current.tool, toolName: event.target.value },
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="工具描述">
                            <UiInput
                              placeholder="请输入工具描述"
                              value={draft.tool.description}
                              onChange={(event) =>
                                setDraft((current) => ({
                                  ...current,
                                  tool: { ...current.tool, description: event.target.value },
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="请求方式" required>
                            <UiSelect
                              value={draft.tool.apiRequestType}
                              onChange={(event) =>
                                setDraft((current) => ({
                                  ...current,
                                  tool: {
                                    ...current.tool,
                                    apiRequestType: event.target.value as RequestType,
                                    bodyFields:
                                      event.target.value === "GET"
                                        ? []
                                        : current.tool.bodyFields,
                                  },
                                }))
                              }
                            >
                              <option value="GET">GET</option>
                              <option value="POST">POST</option>
                            </UiSelect>
                          </FormFieldRow>
                          <FormFieldRow label="响应模式" required>
                            <UiSelect
                              value={draft.tool.toolResponseMode}
                              onChange={(event) =>
                                setDraft((current) => ({
                                  ...current,
                                  tool: {
                                    ...current.tool,
                                    toolResponseMode: event.target.value as ResponseMode,
                                  },
                                }))
                              }
                            >
                              <option value="non_streaming">{responseModeLabel.non_streaming}</option>
                              <option value="streaming">{responseModeLabel.streaming}</option>
                            </UiSelect>
                          </FormFieldRow>
                          <FormFieldRow label="调用地址" required>
                            <UiInput
                              placeholder="请输入 RequestURL"
                              value={draft.tool.requestURL}
                              onChange={(event) =>
                                setDraft((current) => ({
                                  ...current,
                                  tool: { ...current.tool, requestURL: event.target.value },
                                }))
                              }
                            />
                          </FormFieldRow>
                          <FormFieldRow label="超时">
                            <div className="flex items-center gap-2">
                              <UiInput
                                type="number"
                                min={1}
                                step={1}
                                value={String(draft.tool.timeoutMS)}
                                onChange={(event) =>
                                  setDraft((current) => {
                                    const nextValue = Number(event.target.value);
                                    return {
                                      ...current,
                                      tool: {
                                        ...current.tool,
                                        timeoutMS:
                                          Number.isFinite(nextValue) && nextValue >= 1
                                            ? nextValue
                                            : 1,
                                      },
                                    };
                                  })
                                }
                              />
                              <span className="shrink-0 text-sm text-[var(--color-text-secondary)]">
                                毫秒
                              </span>
                            </div>
                          </FormFieldRow>
                        </div>
                        <div className="space-y-4">
                          <FieldListEditor
                            label="Query 字段"
                            fields={draft.tool.queryFields}
                            onChange={(nextFields) =>
                              setDraft((current) => ({
                                ...current,
                                tool: { ...current.tool, queryFields: nextFields },
                              }))
                            }
                          />
                          <FieldListEditor
                            label="Header 字段"
                            fields={draft.tool.headerFields}
                            onChange={(nextFields) =>
                              setDraft((current) => ({
                                ...current,
                                tool: { ...current.tool, headerFields: nextFields },
                              }))
                            }
                          />
                        </div>
                        {draft.tool.apiRequestType === "POST" ? (
                          <FieldListEditor
                            label="Body 字段"
                            fields={draft.tool.bodyFields}
                            onChange={(nextFields) =>
                              setDraft((current) => ({
                                ...current,
                                tool: { ...current.tool, bodyFields: nextFields },
                              }))
                            }
                          />
                        ) : null}
                      </>
                    )}

                    {draft.runtimeType === "mcp" && (
                      <div className="space-y-4">
                        <FormFieldRow label="MCP URL" required>
                          <UiInput
                            placeholder="请输入 MCP URL"
                            value={draft.mcpURL}
                            onChange={(event) =>
                              setDraft((current) => ({ ...current, mcpURL: event.target.value }))
                            }
                          />
                        </FormFieldRow>
                        <FormFieldRow label="MCP 协议">
                          <UiSelect
                            value={draft.mcpProtocol}
                            onChange={(event) =>
                              setDraft((current) => ({
                                ...current,
                                mcpProtocol: normalizeMCPProtocolValue(event.target.value),
                              }))
                            }
                          >
                            <option value="sse">{mcpProtocolLabel.sse}</option>
                            <option value="streamableHttp">
                              {mcpProtocolLabel.streamableHttp}
                            </option>
                          </UiSelect>
                        </FormFieldRow>
                      </div>
                    )}

                    {draft.runtimeType === "code" && (
                      <CodeToolEditor
                        tool={draft.tool}
                        onDebugRun={debugCodeTool}
                        onChange={(nextTool) =>
                          setDraft((current) => ({
                            ...current,
                            tool: nextTool,
                          }))
                        }
                      />
                    )}
                  </div>
                )}

                {wizardStep === 4 && (
                  <div className="space-y-4">
                    <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                      第四步：预览并确认
                    </div>
                    <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-3 py-3 text-xs text-[var(--color-text-muted)]">
                      将写入 plugin 定义，并生成对应的 tool 元数据；敏感信息仅保存引用，不经前端透传。
                    </div>
                    <pre className="overflow-auto rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgb(15,23,42)] p-4 text-xs text-[rgb(226,232,240)]">
                      {previewJSON}
                    </pre>
                  </div>
                )}

                {formError && (
                  <div className="mt-4 rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.06)] px-3 py-2 text-sm text-[var(--color-state-error)]">
                    {formError}
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between gap-3 border-t border-[var(--color-border-default)] px-8 py-5">
              <UiButton
                type="button"
                variant="secondary"
                onClick={() => {
                  setFormError("");
                  setWizardStep((current) => Math.max(1, current - 1));
                }}
                disabled={wizardStep === 1 || saving}
              >
                上一步
              </UiButton>
              <UiButton type="button" onClick={() => void goNext()} disabled={saving}>
                {wizardStep === 4 ? (saving ? "保存中..." : "确认提交") : "下一步"}
              </UiButton>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
