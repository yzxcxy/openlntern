"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

type UserInfo = {
  user_id?: string;
  username?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  role?: string;
  created_at?: string;
  updated_at?: string;
};

const API_BASE = "/api/backend";

export default function UserPage() {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(() => {
    if (typeof window === "undefined") return null;
    const storedUser = localStorage.getItem("user");
    if (!storedUser) return null;
    try {
      return JSON.parse(storedUser);
    } catch {
      return null;
    }
  });
  const [formValues, setFormValues] = useState(() => ({
    username: userInfo?.username ?? "",
    email: userInfo?.email ?? "",
    phone: userInfo?.phone ?? "",
  }));
  const [password, setPassword] = useState("");
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const router = useRouter();

  const applyUser = useCallback((user: UserInfo | null) => {
    setUserInfo(user);
    setFormValues({
      username: user?.username ?? "",
      email: user?.email ?? "",
      phone: user?.phone ?? "",
    });
  }, []);

  const getErrorMessage = (err: unknown, fallback: string) => {
    if (err instanceof Error && err.message) {
      return err.message;
    }
    return fallback;
  };

  const fetchUser = useCallback(
    async (token: string, userId: string, showLoading: boolean) => {
      if (showLoading) {
        setLoading(true);
      }
      setError("");
      try {
        const res = await fetch(`${API_BASE}/v1/users/${userId}`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        const data = await res.json();
        if (!res.ok || data.code !== 0) {
          throw new Error(data.message || "获取用户信息失败");
        }
        applyUser(data.data);
        localStorage.setItem("user", JSON.stringify(data.data));
        window.dispatchEvent(new Event("user-updated"));
      } catch (err) {
        setError(getErrorMessage(err, "获取用户信息失败"));
      } finally {
        if (showLoading) {
          setLoading(false);
        }
      }
    },
    [applyUser]
  );

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    const storedUser = localStorage.getItem("user");
    if (storedUser) {
      try {
        const parsed = JSON.parse(storedUser);
        applyUser(parsed);
        if (parsed?.user_id) {
          fetchUser(token, parsed.user_id, true);
          return;
        }
      } catch {
        applyUser(null);
      }
    }
    setLoading(false);
  }, [applyUser, fetchUser, router]);

  const handleAvatarClick = () => {
    setSuccess("");
    setError("");
    fileInputRef.current?.click();
  };

  const handleAvatarChange = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!userInfo?.user_id) {
      setError("无法获取用户ID");
      return;
    }
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    setError("");
    setSuccess("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch(
        `${API_BASE}/v1/users/${userInfo.user_id}/avatar`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
          },
          body: formData,
        }
      );
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "头像上传失败");
      }
      const nextUser = {
        ...userInfo,
        avatar: data.data?.url || userInfo.avatar,
      };
      applyUser(nextUser);
      localStorage.setItem("user", JSON.stringify(nextUser));
      window.dispatchEvent(new Event("user-updated"));
      setSuccess("头像更新成功");
    } catch (err) {
      setError(getErrorMessage(err, "头像上传失败"));
    }
  };

  const startEdit = () => {
    setEditing(true);
    setSuccess("");
    setError("");
    setPassword("");
  };

  const cancelEdit = () => {
    applyUser(userInfo);
    setEditing(false);
    setSuccess("");
    setError("");
    setPassword("");
  };

  const handleSave = async () => {
    if (!userInfo?.user_id) {
      setError("无法获取用户ID");
      return;
    }
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/login");
      return;
    }
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const updates: Record<string, string> = {};
      if (formValues.username !== (userInfo.username ?? "")) {
        updates.username = formValues.username;
      }
      if (formValues.email !== (userInfo.email ?? "")) {
        updates.email = formValues.email;
      }
      if (formValues.phone !== (userInfo.phone ?? "")) {
        updates.phone = formValues.phone;
      }
      if (password) {
        updates.password = password;
      }
      if (Object.keys(updates).length === 0) {
        setEditing(false);
        setSuccess("信息未变更");
        setPassword("");
        return;
      }
      const res = await fetch(`${API_BASE}/v1/users/${userInfo.user_id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(updates),
      });
      const data = await res.json();
      if (!res.ok || data.code !== 0) {
        throw new Error(data.message || "更新失败");
      }
      await fetchUser(token, userInfo.user_id, false);
      setEditing(false);
      setPassword("");
      setSuccess("信息更新成功");
    } catch (err) {
      setError(getErrorMessage(err, "更新失败"));
    } finally {
      setSaving(false);
    }
  };

  const displayName = userInfo?.username || userInfo?.email || "用户";
  const avatarLabel = displayName ? displayName.slice(0, 1) : "U";

  return (
    <div className="flex h-full w-full items-center justify-center px-4 py-12">
      <div className="w-full max-w-lg rounded-xl border bg-white p-6 shadow-sm">
        <div className="flex items-center gap-4">
          {userInfo?.avatar ? (
            <img
              src={userInfo.avatar}
              alt={displayName}
              className="h-14 w-14 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-gray-200 text-lg font-semibold text-gray-700">
              {avatarLabel}
            </div>
          )}
          <div className="flex-1">
            <div className="text-lg font-semibold text-gray-900">
              {displayName}
            </div>
            {userInfo?.email && (
              <div className="text-sm text-gray-500">{userInfo.email}</div>
            )}
            <div className="mt-2 flex items-center gap-2">
              <button
                onClick={handleAvatarClick}
                className="flex items-center gap-1 rounded-md border px-3 py-1 text-xs text-gray-600 hover:bg-gray-50"
                type="button"
              >
                <svg
                  className="h-3.5 w-3.5 text-gray-500"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M4 7h4l2-2h4l2 2h4v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7z" />
                  <circle cx="12" cy="13" r="3" />
                </svg>
                更换头像
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={handleAvatarChange}
              />
            </div>
          </div>
        </div>
        {loading ? (
          <div className="mt-6 text-sm text-gray-500">加载中...</div>
        ) : (
          <div className="mt-6 space-y-4">
            {editing ? (
              <div className="grid gap-4">
                <label className="space-y-1 text-sm text-gray-600">
                  <span className="flex items-center gap-2">
                    <svg
                      className="h-3.5 w-3.5 text-gray-400"
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
                    用户名
                  </span>
                  <input
                    className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
                    value={formValues.username}
                    onChange={(e) =>
                      setFormValues((prev) => ({
                        ...prev,
                        username: e.target.value,
                      }))
                    }
                  />
                </label>
                <label className="space-y-1 text-sm text-gray-600">
                  <span className="flex items-center gap-2">
                    <svg
                      className="h-3.5 w-3.5 text-gray-400"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M4 6h16v12H4z" />
                      <path d="M4 7l8 6 8-6" />
                    </svg>
                    邮箱
                  </span>
                  <input
                    className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
                    value={formValues.email}
                    onChange={(e) =>
                      setFormValues((prev) => ({
                        ...prev,
                        email: e.target.value,
                      }))
                    }
                  />
                </label>
                <label className="space-y-1 text-sm text-gray-600">
                  <span className="flex items-center gap-2">
                    <svg
                      className="h-3.5 w-3.5 text-gray-400"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <rect x="7" y="2" width="10" height="20" rx="2" />
                      <path d="M11 18h2" />
                    </svg>
                    手机号
                  </span>
                  <input
                    className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
                    value={formValues.phone}
                    onChange={(e) =>
                      setFormValues((prev) => ({
                        ...prev,
                        phone: e.target.value,
                      }))
                    }
                  />
                </label>
                <label className="space-y-1 text-sm text-gray-600">
                  <span className="flex items-center gap-2">
                    <svg
                      className="h-3.5 w-3.5 text-gray-400"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M7 10V7a5 5 0 0 1 10 0v3" />
                      <rect x="5" y="10" width="14" height="10" rx="2" />
                      <circle cx="12" cy="15" r="1" />
                    </svg>
                    密码
                  </span>
                  <input
                    type="password"
                    className="w-full rounded-md border px-3 py-2 text-sm text-gray-900"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="不修改可留空"
                  />
                </label>
              </div>
            ) : (
              <div className="grid gap-3 text-sm text-gray-600">
                <div className="rounded-lg border px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-gray-400">
                    <svg
                      className="h-3.5 w-3.5"
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
                    用户名
                  </div>
                  <div className="text-gray-900">
                    {userInfo?.username || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-gray-400">
                    <svg
                      className="h-3.5 w-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M4 6h16v12H4z" />
                      <path d="M4 7l8 6 8-6" />
                    </svg>
                    邮箱
                  </div>
                  <div className="text-gray-900">
                    {userInfo?.email || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-gray-400">
                    <svg
                      className="h-3.5 w-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <rect x="7" y="2" width="10" height="20" rx="2" />
                      <path d="M11 18h2" />
                    </svg>
                    手机号
                  </div>
                  <div className="text-gray-900">
                    {userInfo?.phone || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="flex items-center gap-2 text-xs text-gray-400">
                    <svg
                      className="h-3.5 w-3.5"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M12 2l3 6 6 .8-4.5 4.2 1 6-5.5-3-5.5 3 1-6L3 8.8 9 8l3-6z" />
                    </svg>
                    角色
                  </div>
                  <div className="text-gray-900">
                    {userInfo?.role || "-"}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
        {error && <div className="mt-4 text-sm text-red-600">{error}</div>}
        {success && (
          <div className="mt-4 text-sm text-emerald-600">{success}</div>
        )}
        <div className="mt-6 flex justify-end gap-2">
          {editing ? (
            <>
              <button
                onClick={cancelEdit}
                className="flex items-center gap-2 rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                type="button"
                disabled={saving}
              >
                <svg
                  className="h-4 w-4 text-gray-500"
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
                取消
              </button>
              <button
                onClick={handleSave}
                className="flex items-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-500 disabled:opacity-60"
                type="button"
                disabled={saving}
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
                  <path d="M5 12l4 4L19 6" />
                </svg>
                {saving ? "保存中..." : "保存"}
              </button>
            </>
          ) : (
            <button
              onClick={startEdit}
              className="flex items-center gap-2 rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
              type="button"
              disabled={loading}
            >
              <svg
                className="h-4 w-4 text-gray-500"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M12 20h9" />
                <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
              </svg>
              编辑信息
            </button>
          )}
          <button
            onClick={() => router.push("/")}
            className="flex items-center gap-2 rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            type="button"
          >
            <svg
              className="h-4 w-4 text-gray-500"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M3 11l9-7 9 7" />
              <path d="M5 10v10h14V10" />
              <path d="M9 20v-6h6v6" />
            </svg>
            返回首页
          </button>
        </div>
      </div>
    </div>
  );
}
