"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { UiButton } from "../../components/ui/UiButton";
import { UiInput } from "../../components/ui/UiInput";
import { OPENINTERN_DEFAULT_AVATAR_URL } from "../../shared/avatar";
import { readStoredUser, readValidToken, requestBackend } from "../auth";

type UserInfo = {
  user_id?: string;
  username?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  created_at?: string;
  updated_at?: string;
};

const formatDateLabel = (value?: string) => {
  if (!value) return "未记录";
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(parsed));
};

export default function UserPage() {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(() =>
    readStoredUser<UserInfo>()
  );
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
  const getValidToken = useCallback(() => readValidToken(router), [router]);

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
    async (showLoading: boolean) => {
      if (showLoading) {
        setLoading(true);
      }
      setError("");
      try {
        const data = await requestBackend<UserInfo>("/v1/users/me", {
          fallbackMessage: "获取用户信息失败",
          router,
        });
        const nextUser = data.data ?? null;
        applyUser(nextUser);
        localStorage.setItem("user", JSON.stringify(nextUser));
        window.dispatchEvent(new Event("user-updated"));
      } catch (err) {
        setError(getErrorMessage(err, "获取用户信息失败"));
      } finally {
        if (showLoading) {
          setLoading(false);
        }
      }
    },
    [applyUser, router]
  );

  useEffect(() => {
    const token = getValidToken();
    if (!token) return;
    const storedUser = localStorage.getItem("user");
    if (storedUser) {
      try {
        const parsed = JSON.parse(storedUser);
        applyUser(parsed);
        if (parsed?.user_id) {
          fetchUser(true);
          return;
        }
      } catch {
        applyUser(null);
      }
    }
    setLoading(false);
  }, [applyUser, fetchUser, getValidToken]);

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
    if (!getValidToken()) return;
    setError("");
    setSuccess("");
    try {
      const formData = new FormData();
      formData.append("file", file);
      const data = await requestBackend<{ url?: string }>("/v1/users/me/avatar", {
        method: "POST",
        body: formData,
        fallbackMessage: "头像上传失败",
        router,
      });
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
    if (!getValidToken()) return;
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
      await requestBackend("/v1/users/me", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updates),
        fallbackMessage: "更新失败",
        router,
      });
      await fetchUser(false);
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
  const profileCompletion = Math.round(
    (
      [userInfo?.username, userInfo?.email, userInfo?.phone, userInfo?.avatar].filter(Boolean)
        .length /
      4
    ) * 100
  );

  return (
    <div className="h-full w-full overflow-y-auto px-4 py-8">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <section className="grid gap-6 lg:grid-cols-[minmax(0,1.45fr)_minmax(0,0.95fr)]">
          <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-sm)]">
            <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
              <img
                src={userInfo?.avatar || OPENINTERN_DEFAULT_AVATAR_URL}
                alt={displayName}
                className="h-16 w-16 rounded-full object-cover"
              />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">
                    {displayName}
                  </h1>
                </div>
                <div className="mt-1 text-sm text-[var(--color-text-muted)]">
                  {userInfo?.email || "建议补充邮箱，方便通知与找回"}
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  <UiButton
                    onClick={handleAvatarClick}
                    variant="secondary"
                    size="sm"
                    type="button"
                  >
                    <svg
                      className="h-3.5 w-3.5"
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
                  </UiButton>
                  <UiButton
                    onClick={() => router.push("/")}
                    variant="ghost"
                    size="sm"
                    type="button"
                  >
                    <svg
                      className="h-3.5 w-3.5"
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
                  </UiButton>
                </div>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={handleAvatarChange}
                />
              </div>
              <div className="grid gap-3 sm:w-44">
                <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-4 py-3">
                  <div className="text-xs text-[var(--color-text-muted)]">资料完整度</div>
                  <div className="mt-1 text-2xl font-semibold text-[var(--color-text-primary)]">
                    {profileCompletion}%
                  </div>
                </div>
                <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-bg-page)] px-4 py-3">
                  <div className="text-xs text-[var(--color-text-muted)]">最近更新</div>
                  <div className="mt-1 text-sm font-medium text-[var(--color-text-primary)]">
                    {formatDateLabel(userInfo?.updated_at)}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div
            id="settings"
            className="scroll-mt-6 rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-sm)]"
          >
            <div className="text-base font-semibold text-[var(--color-text-primary)]">
              用户设置（预留）
            </div>
            <div className="mt-2 text-sm text-[var(--color-text-muted)]">
              先把账户偏好的结构留出来，后续可以直接扩展成独立设置页，而不需要再调整导航层级。
            </div>
            <div className="mt-5 space-y-3">
              <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm font-medium text-[var(--color-text-primary)]">
                    通知偏好
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)]">规划中</span>
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  邮件提醒、站内消息、更新频率
                </div>
              </div>
              <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm font-medium text-[var(--color-text-primary)]">
                    界面偏好
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)]">规划中</span>
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  默认首页、展示密度、操作习惯
                </div>
              </div>
              <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm font-medium text-[var(--color-text-primary)]">
                    隐私与安全
                  </span>
                  <span className="text-xs text-[var(--color-text-muted)]">规划中</span>
                </div>
                <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                  登录保护、设备记录、会话管理
                </div>
              </div>
            </div>
            <div className="mt-4 rounded-[var(--radius-lg)] bg-[rgba(37,99,255,0.06)] px-4 py-3 text-xs text-[var(--color-text-secondary)]">
              当前版本先完成信息管理的结构升级，设置项的持久化会在下一阶段接入。
            </div>
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-[minmax(0,1.25fr)_minmax(0,0.75fr)]">
          <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-sm)]">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-base font-semibold text-[var(--color-text-primary)]">
                  基础信息
                </div>
                <div className="mt-1 text-sm text-[var(--color-text-muted)]">
                  集中维护账号资料，密码修改仍沿用本页编辑流。
                </div>
              </div>
              <span className="rounded-full bg-[var(--color-bg-page)] px-2.5 py-1 text-xs text-[var(--color-text-muted)]">
                {editing ? "编辑中" : "只读"}
              </span>
            </div>

            {loading ? (
              <div className="mt-6 text-sm text-[var(--color-text-muted)]">加载中...</div>
            ) : (
              <div className="mt-6 space-y-4">
                {editing ? (
                  <div className="grid gap-4">
                    <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-3.5 w-3.5 text-[var(--color-text-muted)]"
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
                      <UiInput
                        value={formValues.username}
                        onChange={(e) =>
                          setFormValues((prev) => ({
                            ...prev,
                            username: e.target.value,
                          }))
                        }
                      />
                    </label>
                    <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-3.5 w-3.5 text-[var(--color-text-muted)]"
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
                      <UiInput
                        value={formValues.email}
                        onChange={(e) =>
                          setFormValues((prev) => ({
                            ...prev,
                            email: e.target.value,
                          }))
                        }
                      />
                    </label>
                    <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-3.5 w-3.5 text-[var(--color-text-muted)]"
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
                      <UiInput
                        value={formValues.phone}
                        onChange={(e) =>
                          setFormValues((prev) => ({
                            ...prev,
                            phone: e.target.value,
                          }))
                        }
                      />
                    </label>
                    <label className="space-y-1 text-sm text-[var(--color-text-secondary)]">
                      <span className="flex items-center gap-2">
                        <svg
                          className="h-3.5 w-3.5 text-[var(--color-text-muted)]"
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
                      <UiInput
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="不修改可留空"
                      />
                    </label>
                  </div>
                ) : (
                  <div className="grid gap-3 text-sm text-[var(--color-text-secondary)]">
                    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
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
                      <div className="mt-1 text-[var(--color-text-primary)]">
                        {userInfo?.username || "-"}
                      </div>
                    </div>
                    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
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
                      <div className="mt-1 text-[var(--color-text-primary)]">
                        {userInfo?.email || "-"}
                      </div>
                    </div>
                    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
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
                      <div className="mt-1 text-[var(--color-text-primary)]">
                        {userInfo?.phone || "-"}
                      </div>
                    </div>
                    <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] px-4 py-3">
                      <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
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
                        账号类型
                      </div>
                      <div className="mt-1 text-[var(--color-text-primary)]">
                        当前账号
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {error && (
              <div className="mt-4 text-sm text-[var(--color-state-error)]">{error}</div>
            )}
            {success && (
              <div className="mt-4 text-sm text-[var(--color-state-success)]">
                {success}
              </div>
            )}

            <div className="mt-6 flex flex-wrap justify-end gap-2">
              {editing ? (
                <>
                  <UiButton
                    onClick={cancelEdit}
                    variant="secondary"
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
                      <path d="M6 6l12 12" />
                      <path d="M6 18L18 6" />
                    </svg>
                    取消
                  </UiButton>
                  <UiButton onClick={handleSave} type="button" disabled={saving}>
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
                  </UiButton>
                </>
              ) : (
                <UiButton
                  onClick={startEdit}
                  variant="secondary"
                  type="button"
                  disabled={loading}
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
                    <path d="M12 20h9" />
                    <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
                  </svg>
                  编辑信息
                </UiButton>
              )}
            </div>
          </div>

          <div className="space-y-6">
            <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-sm)]">
              <div className="text-base font-semibold text-[var(--color-text-primary)]">
                账户概览
              </div>
              <div className="mt-4 space-y-3 text-sm text-[var(--color-text-secondary)]">
                <div className="flex items-start justify-between gap-3">
                  <span className="text-[var(--color-text-muted)]">用户 ID</span>
                  <span className="max-w-[12rem] break-all text-right text-[var(--color-text-primary)]">
                    {userInfo?.user_id || "-"}
                  </span>
                </div>
                <div className="flex items-start justify-between gap-3">
                  <span className="text-[var(--color-text-muted)]">账号类型</span>
                  <span className="text-right text-[var(--color-text-primary)]">
                    当前账号
                  </span>
                </div>
                <div className="flex items-start justify-between gap-3">
                  <span className="text-[var(--color-text-muted)]">创建时间</span>
                  <span className="text-right text-[var(--color-text-primary)]">
                    {formatDateLabel(userInfo?.created_at)}
                  </span>
                </div>
                <div className="flex items-start justify-between gap-3">
                  <span className="text-[var(--color-text-muted)]">更新时间</span>
                  <span className="text-right text-[var(--color-text-primary)]">
                    {formatDateLabel(userInfo?.updated_at)}
                  </span>
                </div>
              </div>
            </div>

            <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-sm)]">
              <div className="text-base font-semibold text-[var(--color-text-primary)]">
                安全与支持
              </div>
              <div className="mt-2 text-sm text-[var(--color-text-muted)]">
                密码修改已并入“编辑信息”流程，后续会在设置区补充登录设备、二次验证和会话回收等能力。
              </div>
              <div className="mt-4 rounded-[var(--radius-lg)] bg-[var(--color-bg-page)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
                如果只是补充基础资料，直接使用左侧编辑入口；如果要做偏好配置，请使用右上角账户菜单进入“用户设置”预留区。
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
