import { ReactNode } from "react";
import { UiButton } from "../../../components/ui/UiButton";

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
    <div className="fixed inset-0 z-50 flex items-center justify-center px-4 py-6">
      <button
        type="button"
        className="motion-safe-fade-in absolute inset-0 bg-[rgba(15,23,42,0.38)] backdrop-blur-[2px]"
        onClick={onClose}
        aria-label="关闭弹窗"
      />
      <div
        role="dialog"
        aria-modal="true"
        className="motion-safe-slide-up relative z-10 w-full max-w-3xl rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.96)] p-4 shadow-[var(--shadow-lg)] backdrop-blur-sm"
      >
        <div className="flex items-center justify-between gap-3 border-b border-[var(--color-border-default)] pb-3">
          <div className="text-sm font-semibold text-[var(--color-text-primary)]">{title}</div>
          <UiButton type="button" variant="ghost" size="sm" onClick={onClose}>
            <svg
              className="h-3.5 w-3.5"
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
            关闭
          </UiButton>
        </div>
        <div className="mt-4">{children}</div>
        {footer && <div className="mt-5 flex justify-end gap-2">{footer}</div>}
      </div>
    </div>
  );
}
