"use client";

import { useEffect, useRef, useState } from "react";

type MonacoEditorInstance = {
  getValue: () => string;
  setValue: (value: string) => void;
  getModel: () => unknown;
  onDidChangeModelContent: (listener: () => void) => { dispose: () => void };
  updateOptions: (options: Record<string, unknown>) => void;
  dispose: () => void;
};

type MonacoNamespace = {
  editor: {
    create: (element: HTMLElement, options: Record<string, unknown>) => MonacoEditorInstance;
    setModelLanguage: (model: unknown, languageId: string) => void;
  };
};

declare global {
  interface Window {
    MonacoEnvironment?: {
      getWorkerUrl?: (moduleId: string, label: string) => string;
    };
    monaco?: MonacoNamespace;
    require?: {
      config: (config: Record<string, unknown>) => void;
      (modules: string[], onLoad: () => void, onError?: (error: unknown) => void): void;
    };
  }
}

type UiMonacoEditorProps = {
  value: string;
  language: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  fontSize?: number;
  minHeight?: number;
  className?: string;
  placeholder?: string;
};

const MONACO_VERSION = "0.52.2";
const MONACO_BASE_URL = `https://cdn.jsdelivr.net/npm/monaco-editor@${MONACO_VERSION}/min/vs`;
const MONACO_LOADER_URL = `${MONACO_BASE_URL}/loader.js`;

let monacoLoaderPromise: Promise<MonacoNamespace> | null = null;

const buildWorkerURL = () =>
  `data:text/javascript;charset=utf-8,${encodeURIComponent(
    `self.MonacoEnvironment = { baseUrl: '${MONACO_BASE_URL}/' }; importScripts('${MONACO_BASE_URL}/base/worker/workerMain.js');`
  )}`;

const loadMonaco = async (): Promise<MonacoNamespace> => {
  if (typeof window === "undefined") {
    throw new Error("Monaco can only run in browser");
  }

  if (window.monaco?.editor) {
    return window.monaco;
  }

  if (!monacoLoaderPromise) {
    monacoLoaderPromise = new Promise<MonacoNamespace>((resolve, reject) => {
      const handleReady = () => {
        const amdRequire = window.require;
        if (typeof amdRequire !== "function") {
          reject(new Error("Monaco loader is unavailable"));
          return;
        }

        window.MonacoEnvironment = {
          getWorkerUrl: () => buildWorkerURL(),
        };

        amdRequire.config({
          paths: {
            vs: MONACO_BASE_URL,
          },
        });

        amdRequire(
          ["vs/editor/editor.main"],
          () => {
            if (!window.monaco?.editor) {
              reject(new Error("Monaco editor did not initialize"));
              return;
            }
            resolve(window.monaco);
          },
          (error: unknown) => reject(error)
        );
      };

      const existingScript = document.querySelector<HTMLScriptElement>(
        `script[data-monaco-loader="${MONACO_VERSION}"]`
      );
      if (existingScript) {
        if (typeof window.require === "function") {
          handleReady();
          return;
        }
        existingScript.addEventListener("load", handleReady, { once: true });
        existingScript.addEventListener(
          "error",
          () => reject(new Error("Monaco loader failed to load")),
          { once: true }
        );
        return;
      }

      const script = document.createElement("script");
      script.src = MONACO_LOADER_URL;
      script.async = true;
      script.dataset.monacoLoader = MONACO_VERSION;
      script.addEventListener("load", handleReady, { once: true });
      script.addEventListener(
        "error",
        () => reject(new Error("Monaco loader failed to load")),
        { once: true }
      );
      document.head.appendChild(script);
    });
  }

  return monacoLoaderPromise;
};

export function UiMonacoEditor({
  value,
  language,
  onChange,
  readOnly = false,
  fontSize = 14,
  minHeight = 320,
  className = "",
  placeholder,
}: UiMonacoEditorProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const editorRef = useRef<MonacoEditorInstance | null>(null);
  const internalSyncRef = useRef(false);
  const onChangeRef = useRef(onChange);
  const initialValueRef = useRef(value);
  const initialLanguageRef = useRef(language);
  const initialReadOnlyRef = useRef(readOnly);
  const initialFontSizeRef = useRef(fontSize);
  const [isReady, setIsReady] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    let disposed = false;
    let changeSubscription: { dispose: () => void } | null = null;

    const mountEditor = async () => {
      if (!containerRef.current) return;

      try {
        const monaco = await loadMonaco();
        if (disposed || !containerRef.current) return;

        const editor = monaco.editor.create(containerRef.current, {
          value: initialValueRef.current,
          language: initialLanguageRef.current,
          theme: "vs-dark",
          readOnly: initialReadOnlyRef.current,
          automaticLayout: true,
          minimap: { enabled: false },
          lineNumbers: "on",
          lineNumbersMinChars: 3,
          scrollBeyondLastLine: false,
          wordWrap: "on",
          wrappingIndent: "indent",
          tabSize: 2,
          insertSpaces: true,
          fontSize: initialFontSizeRef.current,
          fontLigatures: true,
          fontFamily: "JetBrains Mono, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
          padding: { top: 16, bottom: 16 },
          roundedSelection: false,
          smoothScrolling: true,
          renderLineHighlight: "gutter",
        });

        changeSubscription = editor.onDidChangeModelContent(() => {
          if (internalSyncRef.current) return;
          onChangeRef.current(editor.getValue());
        });

        editorRef.current = editor;
        setIsReady(true);
      } catch (error) {
        console.error("Failed to load Monaco editor", error);
        if (!disposed) {
          setLoadFailed(true);
        }
      }
    };

    void mountEditor();

    return () => {
      disposed = true;
      changeSubscription?.dispose();
      editorRef.current?.dispose();
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    if (editor.getValue() === value) return;
    internalSyncRef.current = true;
    editor.setValue(value);
    internalSyncRef.current = false;
  }, [value]);

  useEffect(() => {
    const editor = editorRef.current;
    const monaco = window.monaco;
    if (!editor || !monaco?.editor) return;
    const model = editor.getModel();
    if (!model) return;
    monaco.editor.setModelLanguage(model, language);
  }, [language]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    editor.updateOptions({ readOnly });
  }, [readOnly]);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) return;
    editor.updateOptions({ fontSize });
  }, [fontSize]);

  if (loadFailed) {
    return (
      <div
        className={`flex items-center justify-center rounded-[var(--radius-md)] border border-[rgba(248,113,113,0.24)] bg-[rgba(127,29,29,0.18)] px-4 py-6 text-sm text-[rgb(254,202,202)] ${className}`.trim()}
        style={{ minHeight }}
      >
        Monaco 编辑器加载失败，请检查网络或 CDN 配置后重试。
      </div>
    );
  }

  return (
    <div className={`relative overflow-hidden ${className}`.trim()} style={{ minHeight }}>
      {!isReady && (
        <div className="absolute inset-0 flex items-center justify-center bg-[rgba(15,23,42,0.42)] text-sm text-[rgba(226,232,240,0.82)]">
          正在加载 Monaco 编辑器...
        </div>
      )}
      <div
        ref={containerRef}
        className="h-full w-full"
        style={{ minHeight }}
        aria-label={placeholder || "code editor"}
      />
    </div>
  );
}
