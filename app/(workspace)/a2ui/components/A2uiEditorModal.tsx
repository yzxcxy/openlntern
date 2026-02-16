import { Modal } from "./Modal";

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
          <button
            className="flex items-center gap-2 rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
            type="button"
            onClick={onClose}
          >
            <svg
              className="h-4 w-4 text-gray-500"
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
          </button>
          <button
            className="flex items-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-500 disabled:opacity-60"
            type="button"
            onClick={onSave}
            disabled={saving}
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
              <path d="M5 12l4 4L19 6" />
            </svg>
            {saving ? "保存中..." : "保存"}
          </button>
        </>
      }
    >
      <div className="grid gap-3 md:grid-cols-2">
        <label className="space-y-1 text-sm text-gray-600">
          <span>名称</span>
          <input
            className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
            value={values.name}
            onChange={(e) => onChange({ ...values, name: e.target.value })}
          />
        </label>
        <label className="space-y-1 text-sm text-gray-600">
          <span>描述</span>
          <input
            className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
            value={values.description}
            onChange={(e) =>
              onChange({ ...values, description: e.target.value })
            }
          />
        </label>
        <label className="space-y-1 text-sm text-gray-600 md:col-span-2">
          <span>UI JSON</span>
          <textarea
            className="min-h-[120px] w-full rounded-md border px-3 py-2 text-sm text-gray-900"
            value={values.ui_json}
            onChange={(e) => onChange({ ...values, ui_json: e.target.value })}
          />
        </label>
        <label className="space-y-1 text-sm text-gray-600 md:col-span-2">
          <span>数据 JSON</span>
          <textarea
            className="min-h-[120px] w-full rounded-md border px-3 py-2 text-sm text-gray-900"
            value={values.data_json}
            onChange={(e) => onChange({ ...values, data_json: e.target.value })}
          />
        </label>
      </div>
    </Modal>
  );
}
