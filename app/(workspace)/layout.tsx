"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { PageContainer } from "../components/layout/PageContainer";
import { AppShell } from "../components/layout/AppShell";
import { Sidebar } from "../components/layout/Sidebar";
import { UiButton } from "../components/ui/UiButton";
import { UiInput } from "../components/ui/UiInput";
import { ConfirmDialog } from "./a2ui/components/ConfirmDialog";
import { Modal } from "./a2ui/components/Modal";
import {
  buildAuthHeaders,
  readStoredUser,
  readValidToken,
  updateTokenFromResponse,
} from "./auth";

type UserInfo = {
  username?: string;
  email?: string;
  avatar?: string;
  user_id?: string | number;
} | null;

type ThreadItem = {
  thread_id?: string;
  title?: string;
  updated_at?: string;
  created_at?: string;
};

const createThreadId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
};

const HISTORY_PAGE_SIZE = 10;

const joinClasses = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

const sidebarIconButtonClass =
  "motion-safe-highlight flex h-9 w-9 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-secondary)]";

const sidebarPanelClass =
  "motion-safe-lift rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.92)] p-3 shadow-[var(--shadow-sm)] backdrop-blur-sm";

const quickEntryClass = (active: boolean) =>
  joinClasses(
    "motion-safe-highlight flex w-full items-center justify-between rounded-[var(--radius-md)] border border-[var(--color-border-default)] px-3 py-2 text-[var(--color-text-secondary)]",
    "hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]",
    active &&
      "border-[rgba(37,99,255,0.2)] bg-[linear-gradient(90deg,rgba(37,99,255,0.14),rgba(37,99,255,0.04))] text-[var(--color-text-primary)]"
  );

const historyEntryClass = (active: boolean) =>
  joinClasses(
    "motion-safe-highlight flex w-full items-center gap-3 rounded-[var(--radius-md)] border px-2.5 py-2 text-left",
    active
      ? "border-[rgba(37,99,255,0.18)] bg-[linear-gradient(90deg,rgba(37,99,255,0.12),rgba(37,99,255,0.03))] text-[var(--color-text-primary)]"
      : "border-transparent text-[var(--color-text-secondary)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]"
  );

const formatThreadTime = (value?: string) => {
  if (!value) return "";
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return "";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(parsed));
};

export default function WorkspaceLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [userInfo, setUserInfo] = useState<UserInfo>(null);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [historyItems, setHistoryItems] = useState<ThreadItem[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState("");
  const [historyPage, setHistoryPage] = useState(1);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    item: ThreadItem;
  } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ThreadItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [renameTarget, setRenameTarget] = useState<ThreadItem | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [renaming, setRenaming] = useState(false);
  const [renameError, setRenameError] = useState("");
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const readUserFromStorage = useCallback((): UserInfo => readStoredUser(), []);

  useEffect(() => {
    const token = readValidToken(router);
    if (!token) {
      router.push("/login");
    }
  }, [router]);

  useEffect(() => {
    setUserInfo(readUserFromStorage());
    const handleUserUpdated = () => {
      setUserInfo(readUserFromStorage());
    };
    const handleStorage = (event: StorageEvent) => {
      if (event.key === "user") {
        handleUserUpdated();
      }
    };
    window.addEventListener("user-updated", handleUserUpdated);
    window.addEventListener("storage", handleStorage);
    return () => {
      window.removeEventListener("user-updated", handleUserUpdated);
      window.removeEventListener("storage", handleStorage);
    };
  }, [readUserFromStorage]);

  const getValidToken = useCallback(() => readValidToken(router), [router]);
  const fetchThreads = useCallback(
    async (options?: { page?: number; append?: boolean }) => {
      const token = getValidToken();
      if (!token) return;
      const nextPage = options?.page ?? 1;
      const shouldAppend = options?.append ?? false;
      setHistoryLoading(true);
      setHistoryError("");
      try {
        const params = new URLSearchParams();
        params.set("page", String(nextPage));
        params.set("page_size", String(HISTORY_PAGE_SIZE));
        const res = await fetch(`/api/backend/v1/threads?${params.toString()}`, {
          headers: buildAuthHeaders(token),
        });
        updateTokenFromResponse(res);
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "获取历史会话失败");
        }
        const items = Array.isArray(data.data?.data) ? data.data.data : [];
        const total = typeof data.data?.total === "number" ? data.data.total : 0;
        setHistoryItems((prev) => (shouldAppend ? [...prev, ...items] : items));
        setHistoryPage(nextPage);
        setHistoryTotal(total);
      } catch (err) {
        if (err instanceof Error && err.message) {
          setHistoryError(err.message);
        } else {
          setHistoryError("获取历史会话失败");
        }
      } finally {
        setHistoryLoading(false);
      }
    },
    [getValidToken]
  );

  useEffect(() => {
    fetchThreads({ page: 1, append: false });
  }, [fetchThreads]);

  useEffect(() => {
    const handleThreadsRefresh = () => {
      fetchThreads({ page: 1, append: false });
    };
    window.addEventListener("threads-refresh", handleThreadsRefresh);
    return () => {
      window.removeEventListener("threads-refresh", handleThreadsRefresh);
    };
  }, [fetchThreads]);

  useEffect(() => {
    if (!contextMenu) return;
    const handleClose = () => setContextMenu(null);
    window.addEventListener("click", handleClose);
    window.addEventListener("contextmenu", handleClose);
    window.addEventListener("scroll", handleClose, true);
    return () => {
      window.removeEventListener("click", handleClose);
      window.removeEventListener("contextmenu", handleClose);
      window.removeEventListener("scroll", handleClose, true);
    };
  }, [contextMenu]);

  useEffect(() => {
    if (!isUserMenuOpen) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (userMenuRef.current?.contains(event.target as Node)) {
        return;
      }
      setIsUserMenuOpen(false);
    };
    window.addEventListener("mousedown", handlePointerDown);
    return () => {
      window.removeEventListener("mousedown", handlePointerDown);
    };
  }, [isUserMenuOpen]);

  const handleLogout = () => {
    setIsUserMenuOpen(false);
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    router.push("/login");
  };

  const handleUserManage = () => {
    setIsUserMenuOpen(false);
    router.push("/user");
  };

  const handleUserSettings = () => {
    setIsUserMenuOpen(false);
    router.push("/user#settings");
  };

  const openContextMenu = (event: React.MouseEvent, item: ThreadItem) => {
    event.preventDefault();
    event.stopPropagation();
    const menuWidth = 160;
    const menuHeight = 92;
    const padding = 12;
    const maxX = window.innerWidth - menuWidth - padding;
    const maxY = window.innerHeight - menuHeight - padding;
    const x = Math.min(event.clientX, Math.max(padding, maxX));
    const y = Math.min(event.clientY, Math.max(padding, maxY));
    setContextMenu({ x, y, item });
  };

  const openDelete = (item: ThreadItem) => {
    setContextMenu(null);
    setDeleteTarget(item);
  };

  const closeDelete = () => {
    setDeleteTarget(null);
    setDeleting(false);
  };

  const handleDelete = async () => {
    if (!deleteTarget?.thread_id) return;
    const token = getValidToken();
    if (!token) return;
    setDeleting(true);
    setHistoryError("");
    try {
      const res = await fetch(`/api/backend/v1/threads/${deleteTarget.thread_id}`, {
        method: "DELETE",
        headers: buildAuthHeaders(token),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "删除会话失败");
      }
      setHistoryItems((items) =>
        items.filter((item) => item.thread_id !== deleteTarget.thread_id)
      );
      setHistoryTotal((prev) => (prev > 0 ? prev - 1 : 0));
      closeDelete();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setHistoryError(err.message);
      } else {
        setHistoryError("删除会话失败");
      }
    } finally {
      setDeleting(false);
    }
  };

  const openRename = (item: ThreadItem) => {
    setContextMenu(null);
    setRenameTarget(item);
    setRenameValue(item.title ?? "");
    setRenameError("");
  };

  const closeRename = () => {
    setRenameTarget(null);
    setRenameValue("");
    setRenameError("");
    setRenaming(false);
  };

  const handleRename = async () => {
    if (!renameTarget?.thread_id) return;
    const title = renameValue.trim();
    if (!title) {
      setRenameError("请输入会话名称");
      return;
    }
    const token = getValidToken();
    if (!token) return;
    setRenaming(true);
    setRenameError("");
    try {
      const res = await fetch(`/api/backend/v1/threads/${renameTarget.thread_id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...buildAuthHeaders(token),
        },
        body: JSON.stringify({ title }),
      });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "重命名失败");
      }
      setHistoryItems((items) =>
        items.map((item) =>
          item.thread_id === renameTarget.thread_id ? { ...item, title } : item
        )
      );
      closeRename();
    } catch (err) {
      if (err instanceof Error && err.message) {
        setRenameError(err.message);
      } else {
        setRenameError("重命名失败");
      }
    } finally {
      setRenaming(false);
    }
  };

  const displayName = userInfo?.username || userInfo?.email || "未登录用户";
  const displayEmail = userInfo?.email || "";
  const avatarLabel = displayName ? displayName.slice(0, 1) : "U";
  const isA2ui = pathname === "/a2ui";
  const isChat = pathname === "/chat";
  const isSkill = pathname.startsWith("/skills");
  const isKB = pathname === "/kb";
  const hasMoreHistory = historyItems.length < historyTotal;
  const activeThreadId = isChat ? searchParams.get("threadId") ?? "" : "";
  const historyInitialLoading = historyLoading && historyItems.length === 0;

  const handleLoadMoreHistory = useCallback(() => {
    if (historyLoading || !hasMoreHistory) return;
    fetchThreads({ page: historyPage + 1, append: true });
  }, [fetchThreads, hasMoreHistory, historyLoading, historyPage]);

  return (
    <>
      <AppShell
        sidebar={
          <Sidebar collapsed={isSidebarCollapsed}>
            <div className="px-4 pt-4">
              <div
                className={joinClasses(
                  "motion-safe-fade-in relative flex items-center",
                  isSidebarCollapsed
                    ? "flex-col-reverse justify-center gap-3"
                    : "justify-between"
                )}
              >
                <img
                  src="/openIntern_logo_concept_3_dialogue_flow.svg"
                  alt="openIntern"
                  className="h-8 w-8"
                />
                {!isSidebarCollapsed && (
                  <button
                    onClick={() => setIsSidebarCollapsed(true)}
                    className={sidebarIconButtonClass}
                  >
                    <svg
                      viewBox="0 0 24 24"
                      className="h-4 w-4"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <rect x="3" y="4" width="7" height="16" rx="2" />
                      <path d="M14 8l4 4-4 4" />
                    </svg>
                  </button>
                )}
                {isSidebarCollapsed && (
                  <button
                    onClick={() => setIsSidebarCollapsed(false)}
                    className={sidebarIconButtonClass}
                  >
                    <svg
                      viewBox="0 0 24 24"
                      className="h-4 w-4"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <rect x="14" y="4" width="7" height="16" rx="2" />
                      <path d="M10 8l-4 4 4 4" />
                    </svg>
                  </button>
                )}
              </div>
              <div className={joinClasses("mt-4", isSidebarCollapsed && "flex justify-center")}>
                <button
                  onClick={() => router.push(`/chat?threadId=${createThreadId()}&new=1`)}
                  className={joinClasses(
                    "motion-safe-highlight flex items-center rounded-full border border-[var(--color-border-default)] px-3 py-2",
                    "text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]",
                    isSidebarCollapsed
                      ? "h-10 w-10 justify-center px-0"
                      : "w-full justify-between",
                    isChat &&
                      "border-[rgba(37,99,255,0.18)] bg-[linear-gradient(90deg,rgba(37,99,255,0.14),rgba(37,99,255,0.04))] text-[var(--color-text-primary)]"
                  )}
                >
                  <div className="flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-full border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] text-[var(--color-text-muted)]">
                      <svg
                        viewBox="0 0 24 24"
                        className="h-4 w-4"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <circle cx="12" cy="12" r="9" />
                        <path d="M12 8v8M8 12h8" />
                      </svg>
                    </span>
                    {!isSidebarCollapsed && <span className="text-sm font-semibold">新建对话</span>}
                  </div>
                </button>
              </div>
            </div>

            {!isSidebarCollapsed && (
              <div className="motion-safe-fade-in flex-1 overflow-auto px-4 pb-4 pt-4">
                <div className="space-y-4">
                  <div className={sidebarPanelClass}>
                    <div className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text-primary)]">
                      <svg
                        className="h-4 w-4 text-[var(--color-text-muted)]"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.8"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <rect x="4" y="4" width="7" height="7" rx="1.5" />
                        <rect x="13" y="4" width="7" height="7" rx="1.5" />
                        <rect x="4" y="13" width="7" height="7" rx="1.5" />
                        <rect x="13" y="13" width="7" height="7" rx="1.5" />
                      </svg>
                      快捷入口
                    </div>
                    <div className="mt-3 space-y-2 text-sm">
                      <button
                        onClick={() => router.push("/a2ui")}
                        className={quickEntryClass(isA2ui)}
                      >
                        <span className="flex items-center gap-2">
                          <svg
                            className="h-4 w-4 text-[var(--color-text-muted)]"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <rect x="3" y="5" width="18" height="14" rx="2" />
                            <path d="M7 9h10M7 13h6" />
                          </svg>
                          A2UI 管理
                        </span>
                        <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)]">
                          查看
                          <svg
                            className="h-3.5 w-3.5"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M9 5l6 7-6 7" />
                          </svg>
                        </span>
                      </button>

                      <button
                        onClick={() => router.push("/skills")}
                        className={quickEntryClass(isSkill)}
                      >
                        <span className="flex items-center gap-2">
                          <svg
                            className="h-4 w-4 text-[var(--color-text-muted)]"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M12 3l2.5 5 5.5.8-4 3.9.9 5.5-4.9-2.7-4.9 2.7.9-5.5-4-3.9 5.5-.8L12 3z" />
                          </svg>
                          Skill 市场
                        </span>
                        <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)]">
                          查看
                          <svg
                            className="h-3.5 w-3.5"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M9 5l6 7-6 7" />
                          </svg>
                        </span>
                      </button>

                      <button
                        onClick={() => router.push("/kb")}
                        className={quickEntryClass(isKB)}
                      >
                        <span className="flex items-center gap-2">
                          <svg
                            className="h-4 w-4 text-[var(--color-text-muted)]"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M4 4h7l2 2h7v12a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z" />
                          </svg>
                          知识库管理
                        </span>
                        <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)]">
                          查看
                          <svg
                            className="h-3.5 w-3.5"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M9 5l6 7-6 7" />
                          </svg>
                        </span>
                      </button>
                    </div>
                  </div>

                  <div className={sidebarPanelClass}>
                    <div className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text-primary)]">
                      <svg
                        className="h-4 w-4 text-[var(--color-text-muted)]"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="1.8"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <circle cx="12" cy="12" r="8" />
                        <path d="M12 8v4l3 2" />
                      </svg>
                      历史会话
                    </div>
                    <div className="mt-3 space-y-3 text-sm text-[var(--color-text-secondary)]">
                      {historyInitialLoading && (
                        <div className="space-y-2">
                          <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.12)]" />
                          <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.08)]" />
                          <div className="h-10 animate-pulse rounded-[var(--radius-md)] bg-[rgba(148,163,184,0.06)]" />
                        </div>
                      )}
                      {historyError && (
                        <div className="rounded-[var(--radius-md)] border border-[rgba(220,38,38,0.14)] bg-[rgba(220,38,38,0.08)] px-3 py-2 text-xs text-[var(--color-state-error)]">
                          {historyError}
                        </div>
                      )}
                      {!historyInitialLoading && !historyError && historyItems.length === 0 && (
                        <div className="text-xs text-[var(--color-text-muted)]">暂无历史会话</div>
                      )}
                      {historyItems.map((item) => (
                        <button
                          key={item.thread_id}
                          onClick={() => router.push(`/chat?threadId=${item.thread_id}`)}
                          onContextMenu={(event) => openContextMenu(event, item)}
                          className={historyEntryClass(item.thread_id === activeThreadId)}
                        >
                          <span
                            className={joinClasses(
                              "h-8 w-1 shrink-0 rounded-full",
                              item.thread_id === activeThreadId
                                ? "bg-[var(--color-action-primary)]"
                                : "bg-[rgba(148,163,184,0.18)]"
                            )}
                          />
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm font-medium">
                              {item.title || item.thread_id || "未命名会话"}
                            </span>
                            <span className="block truncate text-xs text-[var(--color-text-muted)]">
                              {formatThreadTime(item.updated_at || item.created_at) || "最近更新"}
                            </span>
                          </span>
                        </button>
                      ))}
                      {hasMoreHistory && (
                        <button
                          type="button"
                          onClick={handleLoadMoreHistory}
                          disabled={historyLoading}
                          className={joinClasses(
                            "motion-safe-highlight flex items-center gap-2 text-left text-sm font-semibold",
                            historyLoading
                              ? "text-[var(--color-text-muted)]"
                              : "text-[var(--color-text-primary)] hover:text-[var(--color-action-primary)]"
                          )}
                        >
                          {historyLoading ? "加载中..." : "查看更多"}
                          <svg
                            className="h-4 w-4"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="1.8"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="M9 5l6 7-6 7" />
                          </svg>
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {!isSidebarCollapsed && (
              <div
                ref={userMenuRef}
                className="motion-safe-fade-in relative mx-4 mb-4 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[rgba(255,255,255,0.92)] px-3 py-2 shadow-[var(--shadow-sm)] backdrop-blur-sm"
              >
                {isUserMenuOpen && (
                  <div className="motion-safe-slide-up absolute inset-x-0 bottom-full z-20 mb-2 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] p-2 shadow-[var(--shadow-lg)]">
                    <button
                      type="button"
                      onClick={handleUserManage}
                      className="motion-safe-highlight flex w-full items-center gap-2 rounded-[var(--radius-md)] px-3 py-2 text-left text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]"
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
                        <circle cx="12" cy="8" r="4" />
                        <path d="M4 20a8 8 0 0 1 16 0" />
                      </svg>
                      个人资料
                    </button>
                    <button
                      type="button"
                      onClick={handleUserSettings}
                      className="motion-safe-highlight mt-1 flex w-full items-center gap-2 rounded-[var(--radius-md)] px-3 py-2 text-left text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]"
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
                        <circle cx="12" cy="12" r="3" />
                        <path d="M19.4 15a1 1 0 0 0 .2 1.1l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1 1 0 0 0-1.1-.2 1 1 0 0 0-.6.9V20a2 2 0 1 1-4 0v-.2a1 1 0 0 0-.7-.9 1 1 0 0 0-1.1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1 1 0 0 0 .2-1.1 1 1 0 0 0-.9-.6H4a2 2 0 1 1 0-4h.2a1 1 0 0 0 .9-.7 1 1 0 0 0-.2-1.1l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1 1 0 0 0 1.1.2 1 1 0 0 0 .6-.9V4a2 2 0 1 1 4 0v.2a1 1 0 0 0 .7.9 1 1 0 0 0 1.1-.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1 1 0 0 0-.2 1.1 1 1 0 0 0 .9.6H20a2 2 0 1 1 0 4h-.2a1 1 0 0 0-.9.7Z" />
                      </svg>
                      用户设置
                    </button>
                    <div className="mx-2 my-2 h-px bg-[var(--color-border-default)]" />
                    <button
                      type="button"
                      onClick={handleLogout}
                      className="motion-safe-highlight flex w-full items-center gap-2 rounded-[var(--radius-md)] px-3 py-2 text-left text-sm text-[rgba(185,28,28,0.88)] hover:bg-[rgba(220,38,38,0.08)]"
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
                        <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                        <path d="M16 17l5-5-5-5" />
                        <path d="M21 12H9" />
                      </svg>
                      退出登录
                    </button>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleUserManage}
                    className="motion-safe-highlight flex min-w-0 flex-1 items-center gap-3 rounded-[var(--radius-md)] px-1 py-1 text-left hover:bg-[var(--color-bg-page)]"
                  >
                    {userInfo?.avatar ? (
                      <img
                        src={userInfo.avatar}
                        alt={displayName}
                        className="h-9 w-9 rounded-full object-cover"
                      />
                    ) : (
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[var(--color-bg-page)] text-sm font-semibold text-[var(--color-text-secondary)]">
                        {avatarLabel}
                      </div>
                    )}
                    <div className="flex min-w-0 flex-1 flex-col">
                      <span className="truncate text-sm font-medium text-[var(--color-text-primary)]">
                        {displayName}
                      </span>
                      {displayEmail ? (
                        <span className="truncate text-xs text-[var(--color-text-muted)]">
                          {displayEmail}
                        </span>
                      ) : (
                        <span className="text-xs text-[var(--color-text-muted)]">
                          账户中心
                        </span>
                      )}
                    </div>
                  </button>
                  <button
                    type="button"
                    aria-label="打开账户菜单"
                    aria-expanded={isUserMenuOpen}
                    onClick={() => setIsUserMenuOpen((open) => !open)}
                    className="motion-safe-highlight flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:bg-[var(--color-bg-page)] hover:text-[var(--color-text-primary)]"
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
                      <circle cx="12" cy="5" r="1.5" />
                      <circle cx="12" cy="12" r="1.5" />
                      <circle cx="12" cy="19" r="1.5" />
                    </svg>
                  </button>
                </div>
              </div>
            )}
          </Sidebar>
        }
      >
        <PageContainer className="h-full max-w-none px-0 py-0">
          {children}
        </PageContainer>
      </AppShell>

      {contextMenu && (
        <div
          className="motion-safe-slide-up fixed z-50 w-40 rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] py-1 text-sm shadow-[var(--shadow-lg)]"
          style={{ top: contextMenu.y, left: contextMenu.x }}
          onClick={(event) => event.stopPropagation()}
        >
          <button
            type="button"
            onClick={() => openRename(contextMenu.item)}
            className="motion-safe-highlight flex w-full items-center gap-2 px-3 py-2 text-left text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-page)]"
          >
            重命名
          </button>
          <button
            type="button"
            onClick={() => openDelete(contextMenu.item)}
            className="motion-safe-highlight flex w-full items-center gap-2 px-3 py-2 text-left text-[var(--color-state-error)] hover:bg-[var(--color-bg-page)]"
          >
            删除
          </button>
        </div>
      )}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="确认删除"
        description={`确定要删除会话“${
          deleteTarget?.title || deleteTarget?.thread_id || "未命名会话"
        }”吗？此操作不可撤销。`}
        confirmText="删除"
        cancelText="取消"
        confirming={deleting}
        onConfirm={handleDelete}
        onCancel={closeDelete}
      />

      <Modal
        open={Boolean(renameTarget)}
        title="重命名会话"
        onClose={closeRename}
        footer={
          <>
            <UiButton type="button" variant="secondary" onClick={closeRename}>
              取消
            </UiButton>
            <UiButton
              type="button"
              onClick={handleRename}
              disabled={renaming}
              className="min-w-24"
            >
              {renaming ? "保存中..." : "保存"}
            </UiButton>
          </>
        }
      >
        <div className="space-y-2">
          <UiInput
            value={renameValue}
            onChange={(event) => setRenameValue(event.target.value)}
            placeholder="请输入会话名称"
          />
          {renameError && <div className="text-xs text-[var(--color-state-error)]">{renameError}</div>}
        </div>
      </Modal>
    </>
  );
}
