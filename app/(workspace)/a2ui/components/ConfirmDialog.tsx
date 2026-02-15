import { Modal } from "./Modal";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  description: string;
  confirmText?: string;
  cancelText?: string;
  confirming?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  open,
  title,
  description,
  confirmText = "确认",
  cancelText = "取消",
  confirming = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      title={title}
      onClose={onCancel}
      footer={
        <>
          <button
            className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
            type="button"
            onClick={onCancel}
          >
            {cancelText}
          </button>
          <button
            className="rounded-md bg-red-600 px-4 py-2 text-sm text-white hover:bg-red-500 disabled:opacity-60"
            type="button"
            onClick={onConfirm}
            disabled={confirming}
          >
            {confirming ? "处理中..." : confirmText}
          </button>
        </>
      }
    >
      <div className="text-sm text-gray-600">{description}</div>
    </Modal>
  );
}
