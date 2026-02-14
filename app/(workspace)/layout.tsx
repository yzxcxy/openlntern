"use client";

import { createA2UIMessageRenderer } from "@copilotkit/a2ui-renderer";
import { CopilotKitProvider } from "@copilotkit/react-core/v2";
import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { theme } from "../theme";

const A2UIMessageRenderer = createA2UIMessageRenderer({ theme });
const activityRenderers = [A2UIMessageRenderer];

type UserInfo = {
  username?: string;
  email?: string;
  avatar?: string;
} | null;

export default function WorkspaceLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [userInfo, setUserInfo] = useState<UserInfo>(null);
  const router = useRouter();
  const pathname = usePathname();
  const readUserFromStorage = useCallback((): UserInfo => {
    if (typeof window === "undefined") return null;
    const storedUser = localStorage.getItem("user");
    if (!storedUser) return null;
    try {
      return JSON.parse(storedUser);
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    const token = localStorage.getItem("token");
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

  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      showDevConsole="auto"
      renderActivityMessages={activityRenderers}
    >
      <main
        className="flex h-full w-screen overflow-hidden"
        style={{ minHeight: "100dvh" }}
      >
        <aside className="flex h-full w-72 flex-col border-r bg-white">
          <div className="flex-1 overflow-auto px-4 pb-4 pt-4">
            <div className="space-y-4">
              <div className="rounded-lg border bg-white p-3">
                <div className="text-sm font-semibold text-gray-900">
                  快捷入口
                </div>
                <div className="mt-3 space-y-2 text-sm">
                  <button
                    onClick={() => router.push("/a2ui")}
                    className={`flex w-full items-center justify-between rounded-md border px-3 py-2 text-gray-700 ${
                      isA2ui ? "bg-gray-50" : ""
                    }`}
                  >
                    <span>A2UI 管理</span>
                    <span className="text-xs text-gray-400">查看</span>
                  </button>
                  <button
                    onClick={() => router.push("/chat")}
                    className={`flex w-full items-center justify-between rounded-md border px-3 py-2 text-gray-700 ${
                      isChat ? "bg-gray-50" : ""
                    }`}
                  >
                    <span>对话</span>
                    <span className="text-xs text-gray-400">进入</span>
                  </button>
                </div>
              </div>
              <div className="rounded-lg border bg-white p-3">
                <div className="text-sm font-semibold text-gray-900">
                  历史会话
                </div>
                <div className="mt-3 space-y-3 text-sm text-gray-600">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-400">置顶</span>
                    <span>7-9月HRM系统首页与...</span>
                  </div>
                  <div>GORM迁移与MySQL文本</div>
                  <div>MySQL 认证错误</div>
                  <div>Python GIL逐步移除时间表</div>
                  <div>技能存储位置</div>
                  <button className="text-left text-sm font-semibold text-gray-800">
                    查看全部
                  </button>
                </div>
              </div>
            </div>
          </div>
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
              className="mt-3 w-full rounded-md border px-3 py-2 text-sm text-red-600 hover:bg-gray-50 hover:text-red-700"
            >
              退出登录
            </button>
          </div>
        </aside>
        <section className="relative flex h-full flex-1 flex-col overflow-hidden">
          <div className="flex-1 overflow-hidden bg-gray-50">
            {children}
          </div>
        </section>
      </main>
    </CopilotKitProvider>
  );
}
