import { Modal } from "./Modal";
import { UiButton } from "../../../components/ui/UiButton";

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
          <UiButton type="button" variant="secondary" onClick={onCancel}>
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
            {cancelText}
          </UiButton>
          <UiButton type="button" variant="danger" onClick={onConfirm} disabled={confirming}>
            <svg
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M6 18L18 6" />
              <path d="M6 6l12 12" />
            </svg>
            {confirming ? "处理中..." : confirmText}
          </UiButton>
        </>
      }
    >
      <div className="rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.12)] bg-[rgba(248,250,252,0.9)] px-3 py-3 text-sm text-[var(--color-text-secondary)]">
        {description}
      </div>
    </Modal>
  );
}
