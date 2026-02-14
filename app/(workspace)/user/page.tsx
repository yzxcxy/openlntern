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

const API_BASE = "http://localhost:8080";

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
                className="rounded-md border px-3 py-1 text-xs text-gray-600 hover:bg-gray-50"
                type="button"
              >
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
                  <span>用户名</span>
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
                  <span>邮箱</span>
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
                  <span>手机号</span>
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
                  <span>密码</span>
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
                  <div className="text-xs text-gray-400">用户名</div>
                  <div className="text-gray-900">
                    {userInfo?.username || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="text-xs text-gray-400">邮箱</div>
                  <div className="text-gray-900">
                    {userInfo?.email || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="text-xs text-gray-400">手机号</div>
                  <div className="text-gray-900">
                    {userInfo?.phone || "-"}
                  </div>
                </div>
                <div className="rounded-lg border px-3 py-2">
                  <div className="text-xs text-gray-400">角色</div>
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
                className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                type="button"
                disabled={saving}
              >
                取消
              </button>
              <button
                onClick={handleSave}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-500 disabled:opacity-60"
                type="button"
                disabled={saving}
              >
                {saving ? "保存中..." : "保存"}
              </button>
            </>
          ) : (
            <button
              onClick={startEdit}
              className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
              type="button"
              disabled={loading}
            >
              编辑信息
            </button>
          )}
          <button
            onClick={() => router.push("/")}
            className="rounded-md border px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            type="button"
          >
            返回首页
          </button>
        </div>
      </div>
    </div>
  );
}
