import { ReactNode } from "react";

type ModalProps = {
  open: boolean;
  title?: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
};

export function Modal({ open, title, onClose, children, footer }: ModalProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/30"
        onClick={onClose}
        aria-label="关闭弹窗"
      />
      <div
        role="dialog"
        aria-modal="true"
        className="relative z-10 w-full max-w-3xl rounded-lg bg-white p-4 shadow-lg"
      >
        <div className="flex items-center justify-between gap-3">
          <div className="text-sm font-semibold text-gray-900">{title}</div>
          <button
            type="button"
            className="rounded-md px-2 py-1 text-xs text-gray-500 hover:bg-gray-100"
            onClick={onClose}
          >
            关闭
          </button>
        </div>
        <div className="mt-3">{children}</div>
        {footer && <div className="mt-4 flex justify-end gap-2">{footer}</div>}
      </div>
    </div>
  );
}
