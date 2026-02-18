"use client";

import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import {
  buildAuthHeaders,
  getUserIdFromToken,
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
  const router = useRouter();
  const pathname = usePathname();
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
  const fetchThreads = useCallback(async () => {
    const token = getValidToken();
    if (!token) return;
    const user = readUserFromStorage();
    const userId =
      typeof user?.user_id === "string" || typeof user?.user_id === "number"
        ? String(user.user_id)
        : getUserIdFromToken(token);
    setHistoryLoading(true);
    setHistoryError("");
    try {
      const params = new URLSearchParams();
      params.set("page", "1");
      params.set("page_size", "5");
    const res = await fetch(`/api/backend/v1/threads?${params.toString()}`, {
      headers: buildAuthHeaders(token, userId),
    });
      updateTokenFromResponse(res);
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "获取历史会话失败");
      }
      setHistoryItems(Array.isArray(data.data?.data) ? data.data.data : []);
    } catch (err) {
      if (err instanceof Error && err.message) {
        setHistoryError(err.message);
      } else {
        setHistoryError("获取历史会话失败");
      }
    } finally {
      setHistoryLoading(false);
    }
  }, [getValidToken, readUserFromStorage]);

  useEffect(() => {
    fetchThreads();
  }, [fetchThreads]);

  const handleLogout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    router.push("/login");
  };

  const handleUserManage = () => {
    router.push("/user");
  };

  const displayName =
    userInfo?.username || userInfo?.email || "未登录用户";
  const displayEmail = userInfo?.email || "";
  const avatarLabel = displayName ? displayName.slice(0, 1) : "U";
  const isA2ui = pathname === "/a2ui";
  const isChat = pathname === "/chat";
  const isSkill = pathname.startsWith("/skills");

  return (
    <main
      className="flex h-full w-screen overflow-hidden"
      style={{ minHeight: "100dvh" }}
    >
        <aside
          className={`flex h-full flex-col border-r bg-white transition-all ${
            isSidebarCollapsed ? "w-16" : "w-72"
          }`}
        >
          <div className="px-4 pt-4">
            <div
              className={`relative flex items-center ${
                isSidebarCollapsed
                  ? "flex-col-reverse justify-center gap-3"
                  : "justify-between"
              }`}
            >
              <img
                src="/openIntern_logo_concept_3_dialogue_flow.svg"
                alt="openIntern"
                className="h-8 w-8"
              />
              {!isSidebarCollapsed && (
                <button
                  onClick={() => setIsSidebarCollapsed(true)}
                  className="flex h-9 w-9 items-center justify-center rounded-md border text-gray-500 hover:bg-gray-50"
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
                  className="flex h-9 w-9 items-center justify-center rounded-md border text-gray-500 hover:bg-gray-50"
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
            <div className={`mt-4 ${isSidebarCollapsed ? "flex justify-center" : ""}`}>
              <button
                onClick={() =>
                  router.push(`/chat?threadId=${createThreadId()}&new=1`)
                }
                className={`flex items-center rounded-full border px-3 py-2 text-gray-700 hover:bg-gray-50 ${
                  isSidebarCollapsed ? "h-10 w-10 justify-center px-0" : "w-full justify-between"
                } ${isChat ? "bg-gray-50" : ""}`}
              >
                <div className="flex items-center gap-2">
                  <span className="flex h-7 w-7 items-center justify-center rounded-full border bg-white text-gray-600">
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
                  {!isSidebarCollapsed && (
                    <span className="text-sm font-semibold">新建对话</span>
                  )}
                </div>
              </button>
            </div>
          </div>
          {!isSidebarCollapsed && (
            <div className="flex-1 overflow-auto px-4 pb-4 pt-4">
              <div className="space-y-4">
                <div className="rounded-lg border bg-white p-3">
                  <div className="flex items-center gap-2 text-sm font-semibold text-gray-900">
                    <svg
                      className="h-4 w-4 text-gray-500"
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
                      className={`flex w-full items-center justify-between rounded-md border px-3 py-2 text-gray-700 ${
                        isA2ui ? "bg-gray-50" : ""
                      }`}
                    >
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-4 w-4 text-gray-500"
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
                      <span className="flex items-center gap-1 text-xs text-gray-400">
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
                      className={`flex w-full items-center justify-between rounded-md border px-3 py-2 text-gray-700 ${
                        isSkill ? "bg-gray-50" : ""
                      }`}
                    >
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-4 w-4 text-gray-500"
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
                      <span className="flex items-center gap-1 text-xs text-gray-400">
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
                <div className="rounded-lg border bg-white p-3">
                  <div className="flex items-center gap-2 text-sm font-semibold text-gray-900">
                    <svg
                      className="h-4 w-4 text-gray-500"
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
                  <div className="mt-3 space-y-3 text-sm text-gray-600">
                    {historyLoading && (
                      <div className="text-xs text-gray-400">加载中...</div>
                    )}
                    {historyError && (
                      <div className="text-xs text-red-500">{historyError}</div>
                    )}
                    {!historyLoading && !historyError && historyItems.length === 0 && (
                      <div className="text-xs text-gray-400">暂无历史会话</div>
                    )}
                    {historyItems.map((item) => (
                      <button
                        key={item.thread_id}
                        onClick={() =>
                          router.push(`/chat?threadId=${item.thread_id}`)
                        }
                        className="flex w-full items-center gap-2 text-left"
                      >
                        <span>{item.title || item.thread_id || "未命名会话"}</span>
                      </button>
                    ))}
                    <button className="flex items-center gap-2 text-left text-sm font-semibold text-gray-800">
                      查看全部
                      <svg
                        className="h-4 w-4 text-gray-500"
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
                  </div>
                </div>
              </div>
            </div>
          )}
          {!isSidebarCollapsed && (
            <div className="mx-4 mb-4 rounded-lg border px-3 py-2">
              <button
                onClick={handleUserManage}
                className="flex w-full items-center gap-3 text-left hover:bg-gray-50"
              >
                {userInfo?.avatar ? (
                  <img
                    src={userInfo.avatar}
                    alt={displayName}
                    className="h-9 w-9 rounded-full object-cover"
                  />
                ) : (
                  <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gray-200 text-sm font-semibold text-gray-700">
                    {avatarLabel}
                  </div>
                )}
                <div className="flex flex-col">
                  <span className="text-sm font-medium text-gray-900">
                    {displayName}
                  </span>
                  {displayEmail && (
                    <span className="text-xs text-gray-500">{displayEmail}</span>
                  )}
                </div>
              </button>
              <button
                onClick={handleLogout}
                className="mt-3 flex w-full items-center justify-center gap-2 rounded-md border px-3 py-2 text-sm text-red-600 hover:bg-gray-50 hover:text-red-700"
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
        </aside>
        <section className="relative flex h-full flex-1 flex-col overflow-hidden">
          <div className="flex-1 overflow-hidden bg-gray-50">
            {children}
          </div>
        </section>
    </main>
  );
}
