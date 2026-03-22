"use client";

import { UiButton } from "../../components/ui/UiButton";
import { UiModal } from "../../components/ui/UiModal";
import { UiTextarea } from "../../components/ui/UiTextarea";

type ChatComposerExpandModalProps = {
  open: boolean;
  value: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onApply: () => void;
};

// 大段文本在弹窗里编辑，避免主输入区被内容高度持续撑开。
export function ChatComposerExpandModal({
  open,
  value,
  onChange,
  onClose,
  onApply,
}: ChatComposerExpandModalProps) {
  return (
    <UiModal
      open={open}
      title="展开编辑"
      onClose={onClose}
      footer={
        <>
          <UiButton variant="ghost" onClick={onClose}>
            取消
          </UiButton>
          <UiButton onClick={onApply}>保存到输入框</UiButton>
        </>
      }
    >
      <div>
        <UiTextarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="输入消息；@ 选择知识库，# 选择 Skill"
          className="min-h-[460px] max-h-[72vh] resize-none overflow-y-auto bg-[rgba(248,250,252,0.92)] leading-6"
        />
      </div>
    </UiModal>
  );
}
