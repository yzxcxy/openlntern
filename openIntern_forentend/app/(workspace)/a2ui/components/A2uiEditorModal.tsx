import { Modal } from "./Modal";
import { UiButton } from "../../../components/ui/UiButton";
import { UiInput } from "../../../components/ui/UiInput";
import { UiTextarea } from "../../../components/ui/UiTextarea";

export type A2uiFormValues = {
  name: string;
  description: string;
  ui_json: string;
  data_json: string;
};

type A2uiEditorModalProps = {
  open: boolean;
  mode: "create" | "edit";
  values: A2uiFormValues;
  onChange: (values: A2uiFormValues) => void;
  onClose: () => void;
  onSave: () => void;
  saving: boolean;
};

export function A2uiEditorModal({
  open,
  mode,
  values,
  onChange,
  onClose,
  onSave,
  saving,
}: A2uiEditorModalProps) {
  return (
    <Modal
      open={open}
      title={mode === "create" ? "新增 A2UI" : "编辑 A2UI"}
      onClose={onClose}
      footer={
        <>
          <UiButton type="button" variant="secondary" onClick={onClose}>
            <svg
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M6 6l12 12" />
              <path d="M6 18L18 6" />
            </svg>
            取消
          </UiButton>
          <UiButton type="button" onClick={onSave} disabled={saving}>
            <svg
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 12l4 4L19 6" />
            </svg>
            {saving ? "保存中..." : "保存"}
          </UiButton>
        </>
      }
    >
      <div className="grid gap-3 md:grid-cols-2">
        <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
          <span>名称</span>
          <UiInput
            value={values.name}
            onChange={(e) => onChange({ ...values, name: e.target.value })}
          />
        </label>
        <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
          <span>描述</span>
          <UiInput
            value={values.description}
            onChange={(e) =>
              onChange({ ...values, description: e.target.value })
            }
          />
        </label>
        <label className="space-y-1 text-sm text-[var(--color-text-secondary)] md:col-span-2">
          <span>UI JSON</span>
          <UiTextarea
            className="min-h-[120px]"
            value={values.ui_json}
            onChange={(e) => onChange({ ...values, ui_json: e.target.value })}
          />
        </label>
        <label className="space-y-1 text-sm text-[var(--color-text-secondary)] md:col-span-2">
          <span>数据 JSON</span>
          <UiTextarea
            className="min-h-[120px]"
            value={values.data_json}
            onChange={(e) => onChange({ ...values, data_json: e.target.value })}
          />
        </label>
      </div>
    </Modal>
  );
}
