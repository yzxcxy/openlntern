"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function UserPage() {
  const [userInfo] = useState<{
    username?: string;
    email?: string;
    avatarUrl?: string;
  } | null>(() => {
    if (typeof window === "undefined") return null;
    const storedUser = localStorage.getItem("user");
    if (!storedUser) return null;
    try {
      return JSON.parse(storedUser);
    } catch {
      return null;
    }
  });
  const router = useRouter();

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
  }, [router]);

  const displayName =
    userInfo?.username || userInfo?.email || "未登录用户";
  const displayEmail = userInfo?.email || "";
  const avatarLabel = displayName ? displayName.slice(0, 1) : "U";

  return (
    <div className="flex h-full w-full items-center justify-center px-4 py-12">
      <div className="w-full max-w-lg rounded-xl border bg-white p-6 shadow-sm">
        <div className="flex items-center gap-4">
          {userInfo?.avatarUrl ? (
            <img
              src={userInfo.avatarUrl}
              alt={displayName}
              className="h-14 w-14 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-gray-200 text-lg font-semibold text-gray-700">
              {avatarLabel}
            </div>
          )}
          <div>
            <div className="text-lg font-semibold text-gray-900">
              {displayName}
            </div>
            {displayEmail && (
              <div className="text-sm text-gray-500">{displayEmail}</div>
            )}
          </div>
        </div>
        <div className="mt-6 space-y-2 text-sm text-gray-600">
          <div className="rounded-lg border px-3 py-2">
            这里是用户管理入口，可对接后端管理功能
          </div>
          <div className="rounded-lg border px-3 py-2">
            支持账号信息、权限与通知设置等模块
          </div>
        </div>
        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={() => router.push("/")}
            className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
          >
            返回首页
          </button>
        </div>
      </div>
    </div>
  );
}
