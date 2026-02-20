"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
import { type ComponentPropsWithoutRef, useMemo } from "react";

type Variant = "user" | "assistant";

type CodeProps = ComponentPropsWithoutRef<"code"> & {
  inline?: boolean;
};

type MarkdownMessageProps = {
  content: string;
  variant?: Variant;
  className?: string;
};

const createMarkdownComponents = (variant: Variant) => {
  const isUser = variant === "user";
  const textPrimary = isUser ? "text-white" : "text-gray-900";
  const textSecondary = isUser ? "text-gray-200" : "text-gray-700";
  const borderColor = isUser ? "border-gray-700" : "border-gray-200";
  const linkColor = isUser ? "text-blue-300" : "text-blue-600";
  const inlineCode = isUser
    ? "bg-gray-800 text-gray-100"
    : "bg-gray-200 text-gray-800";

  return {
    h1: (props: ComponentPropsWithoutRef<"h1">) => (
      <h1 className={`mb-4 text-2xl font-semibold ${textPrimary}`} {...props} />
    ),
    h2: (props: ComponentPropsWithoutRef<"h2">) => (
      <h2 className={`mb-3 mt-6 text-xl font-semibold ${textPrimary}`} {...props} />
    ),
    h3: (props: ComponentPropsWithoutRef<"h3">) => (
      <h3 className={`mb-2 mt-5 text-lg font-semibold ${textPrimary}`} {...props} />
    ),
    p: (props: ComponentPropsWithoutRef<"p">) => (
      <p className={`my-3 text-sm ${textSecondary}`} {...props} />
    ),
    ul: (props: ComponentPropsWithoutRef<"ul">) => (
      <ul
        className={`my-3 list-disc space-y-1 pl-6 text-sm ${textSecondary}`}
        {...props}
      />
    ),
    ol: (props: ComponentPropsWithoutRef<"ol">) => (
      <ol
        className={`my-3 list-decimal space-y-1 pl-6 text-sm ${textSecondary}`}
        {...props}
      />
    ),
    li: (props: ComponentPropsWithoutRef<"li">) => (
      <li className={`text-sm ${textSecondary}`} {...props} />
    ),
    a: (props: ComponentPropsWithoutRef<"a">) => (
      <a
        className={`text-sm underline ${linkColor}`}
        target="_blank"
        rel="noreferrer"
        {...props}
      />
    ),
    code: ({ className, inline, ...props }: CodeProps) => {
      const isInline = inline ?? !className?.includes("language-");
      if (!isInline) {
        return <code className={className} {...props} />;
      }
      return (
        <code
          className={`rounded px-1 py-0.5 text-xs ${inlineCode}`}
          {...props}
        />
      );
    },
    pre: (props: ComponentPropsWithoutRef<"pre">) => (
      <pre
        className="my-3 overflow-auto rounded-lg bg-gray-900 p-4 text-xs text-gray-100"
        {...props}
      />
    ),
    table: ({
      className,
      ...props
    }: ComponentPropsWithoutRef<"table">) => (
      <div className="my-4 w-full overflow-x-auto">
        <table
          className={`w-full border-collapse text-sm ${textSecondary} ${className ?? ""}`}
          {...props}
        />
      </div>
    ),
    thead: (props: ComponentPropsWithoutRef<"thead">) => (
      <thead className={isUser ? "bg-gray-800" : "bg-gray-100"} {...props} />
    ),
    tbody: (props: ComponentPropsWithoutRef<"tbody">) => (
      <tbody className={`divide-y ${borderColor}`} {...props} />
    ),
    tr: (props: ComponentPropsWithoutRef<"tr">) => (
      <tr className={isUser ? "hover:bg-gray-800" : "hover:bg-gray-50"} {...props} />
    ),
    th: (props: ComponentPropsWithoutRef<"th">) => (
      <th
        className={`border px-3 py-2 text-left text-xs font-semibold ${borderColor} ${textSecondary}`}
        {...props}
      />
    ),
    td: (props: ComponentPropsWithoutRef<"td">) => (
      <td className={`border px-3 py-2 text-sm ${borderColor}`} {...props} />
    ),
    blockquote: (props: ComponentPropsWithoutRef<"blockquote">) => (
      <blockquote
        className={`my-3 border-l-4 pl-4 text-sm ${borderColor} ${textSecondary}`}
        {...props}
      />
    ),
    hr: (props: ComponentPropsWithoutRef<"hr">) => (
      <hr className={`my-6 ${borderColor}`} {...props} />
    ),
  };
};

export function MarkdownMessage({
  content,
  variant = "assistant",
  className,
}: MarkdownMessageProps) {
  const components = useMemo(() => createMarkdownComponents(variant), [variant]);

  return (
    <div className={className}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex, rehypeHighlight]}
        components={components}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
