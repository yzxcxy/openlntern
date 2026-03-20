"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { UiMonacoEditor } from "../../components/ui/UiMonacoEditor";
import { UiSelect } from "../../components/ui/UiSelect";
import { UiTextarea } from "../../components/ui/UiTextarea";
import {
  createField,
  flattenFields,
  getCodeTemplate,
  getDefaultCodeBodyFields,
  getEffectiveCodeBodyFields,
  isRecord,
  sanitizeOutputFields,
  type FieldType,
  type PluginField,
  type ToolDraft,
} from "./plugin-editor";

// 插件页的编辑器组件统一放在这里，页面主文件只保留业务状态和请求流程。

export function PluginAvatar({
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
        <img
          src={imageURL}
          alt={name || "plugin"}
          className="h-full w-full object-cover"
        />
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

export function FieldListEditor({
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
  const commitChange = (next: PluginField[]) =>
    onChange(isOutput ? sanitizeOutputFields(next) : next);

  return (
    <div className="overflow-hidden rounded-[14px] border border-[var(--color-border-default)] bg-white">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border-default)] px-3 py-2.5">
        <div className="text-sm font-semibold text-[var(--color-text-primary)]">
          {label}
        </div>
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
            <div className="px-3 py-4 text-sm text-[var(--color-text-muted)]">
              暂无字段
            </div>
          ) : (
            <div className="divide-y divide-[var(--color-border-default)]">
              <FieldRowsEditor
                fields={displayFields}
                depth={0}
                onChange={commitChange}
                mode={mode}
              />
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
            onChange(
              fields.map((item, itemIndex) =>
                itemIndex === index ? nextField : item
              )
            )
          }
          onRemove={() =>
            onChange(fields.filter((_, itemIndex) => itemIndex !== index))
          }
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
              onChange={(event) =>
                onChange({ ...field, name: event.target.value })
              }
            />
          )}
        </div>
        <UiInput
          placeholder="参数描述"
          value={field.description}
          onChange={(event) =>
            onChange({ ...field, description: event.target.value })
          }
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
              onChange={(event) =>
                onChange({ ...field, required: event.target.checked })
              }
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
            onChange={(event) =>
              onChange({ ...field, defaultValue: event.target.value })
            }
          />
        )}
        {!isOutput &&
          (showEnumEditor ? (
            <UiInput
              placeholder="逗号分隔，可选"
              value={field.enumText}
              onChange={(event) =>
                onChange({ ...field, enumText: event.target.value })
              }
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
          onChange={(nextChildren) =>
            onChange({ ...field, children: nextChildren })
          }
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

export function DetailSectionTitle({ title }: { title: string }) {
  return (
    <div className="flex items-center gap-3">
      <span className="h-6 w-1 rounded-full bg-[var(--color-action-primary)]" />
      <div className="text-base font-semibold text-[var(--color-text-primary)]">
        {title}
      </div>
    </div>
  );
}

export function FormFieldRow({
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

export function FieldTable({
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
          <div
            key={row.id}
            className={`${gridClassName} px-4 py-3 text-sm text-[var(--color-text-secondary)]`}
          >
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

export function ToolDraftSwitcher({
  tools,
  activeIndex,
  onSelect,
  onAdd,
  onRemove,
}: {
  tools: ToolDraft[];
  activeIndex: number;
  onSelect: (index: number) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
}) {
  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="text-sm font-semibold text-[var(--color-text-primary)]">
          工具配置（{tools.length}）
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <UiButton type="button" variant="secondary" size="sm" onClick={onAdd}>
            添加工具
          </UiButton>
          <UiButton
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => onRemove(activeIndex)}
            disabled={tools.length <= 1}
          >
            删除当前工具
          </UiButton>
        </div>
      </div>
      <div className="mt-3 flex min-w-full gap-2 overflow-x-auto pb-1">
        {tools.map((tool, index) => {
          const active = index === activeIndex;
          return (
            <button
              key={`${tool.toolId || "draft"}-${index}`}
              type="button"
              onClick={() => onSelect(index)}
              className={`shrink-0 whitespace-nowrap rounded-[var(--radius-md)] border px-3 py-1.5 text-sm transition ${
                active
                  ? "border-[var(--color-action-primary)] bg-[rgba(37,99,255,0.08)] font-semibold text-[var(--color-action-primary)]"
                  : "border-[var(--color-border-default)] bg-white text-[var(--color-text-secondary)]"
              }`}
            >
              {tool.toolName.trim() || `工具 ${index + 1}`}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export function CodeToolEditor({
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
                  codeLanguage:
                    event.target.value.trim() === "python" ? "python" : "javascript",
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
              <UiButton
                type="button"
                size="sm"
                onClick={() => void runLocalCheck()}
                loading={debugging}
              >
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
