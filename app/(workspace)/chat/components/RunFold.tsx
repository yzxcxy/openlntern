"use client";

import { useState } from "react";

type RunFoldProps = {
  toolItems?: string[];
  reasoningItems?: string[];
};

export function RunFold({ toolItems, reasoningItems }: RunFoldProps) {
  const tools = toolItems?.filter(Boolean) ?? [];
  const reasoning = reasoningItems?.filter(Boolean) ?? [];
  const hasContent = tools.length > 0 || reasoning.length > 0;
  const [open, setOpen] = useState(false);

  if (!hasContent) return null;

  return (
    <div className="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="flex w-full items-center justify-between"
      >
        <span>工具/推理</span>
        <span>{open ? "收起" : "展开"}</span>
      </button>
      {open && (
        <div className="mt-2 space-y-3">
          {reasoning.length > 0 && (
            <div className="space-y-1">
              <div className="text-[11px] text-gray-500">推理</div>
              <div className="whitespace-pre-wrap rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs text-gray-700">
                {reasoning.join("\n")}
              </div>
            </div>
          )}
          {tools.length > 0 && (
            <div className="space-y-1">
              <div className="text-[11px] text-gray-500">工具</div>
              <div className="whitespace-pre-wrap rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs text-gray-700">
                {tools.join("\n\n")}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
