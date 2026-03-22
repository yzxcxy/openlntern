"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { UiButton } from "../components/ui/UiButton";
import { UiInput } from "../components/ui/UiInput";

const REGISTER_BULLETS = [
  "在同一工作区管理对话、Agent、插件与模型服务。",
  "使用知识库、Skill 和工具完成任务。",
  "创建账户后即可进入完整工作区。",
];

export default function RegisterPage() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch("/api/backend/v1/auth/register", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, email, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.message || "注册失败");
      }

      router.push("/login");
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-shell">
      <div className="auth-ambient left-[8%] top-[12%] h-60 w-60 bg-[rgba(199,104,67,0.16)]" />
      <div className="auth-ambient bottom-[8%] right-[10%] h-64 w-64 bg-[rgba(31,95,114,0.16)]" />
      <div className="auth-grid items-stretch">
        <section className="auth-hero-card hidden lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-8">
            <div className="auth-kicker">
              <span className="h-2 w-2 rounded-full bg-[var(--color-action-secondary)]" />
              Create your workspace
            </div>
            <div className="max-w-3xl space-y-5">
              <p className="text-sm font-semibold uppercase tracking-[0.18em] text-[var(--color-text-muted)]">
                Get Started
              </p>
              <h1 className="max-w-2xl text-5xl leading-[0.92] text-[var(--color-text-primary)] xl:text-7xl">
                创建账户，开始使用 openIntern。
              </h1>
              <p className="max-w-xl text-lg leading-8 text-[var(--color-text-secondary)]">
                注册后即可进入聊天、Agent、插件、模型与知识库工作区。
              </p>
            </div>
            <div className="grid gap-3">
              {REGISTER_BULLETS.map((item) => (
                <div
                  key={item}
                  className="flex items-center gap-3 rounded-[22px] border border-[rgba(126,96,69,0.12)] bg-[rgba(255,252,247,0.72)] px-4 py-3 text-sm font-medium text-[var(--color-text-secondary)]"
                >
                  <span className="flex h-8 w-8 items-center justify-center rounded-full bg-[rgba(199,104,67,0.12)] text-[var(--color-action-primary)]">
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
                  </span>
                  {item}
                </div>
              ))}
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="auth-metric-card">
              <strong>01</strong>
              <span>一个账户，进入完整 AI 协作工作区。</span>
            </div>
            <div className="auth-metric-card">
              <strong>02</strong>
              <span>统一管理模型、插件、知识库与对话。</span>
            </div>
          </div>
        </section>

        <section className="auth-form-card">
          <div className="space-y-6">
            <div className="space-y-3">
              <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[var(--color-text-muted)]">
                Create Account
              </p>
              <div>
                <h2 className="text-4xl leading-none text-[var(--color-text-primary)]">
                  创建新的工作空间身份
                </h2>
                <p className="mt-3 text-sm leading-6 text-[var(--color-text-secondary)]">
                  建立账户后即可进入聊天、插件、模型与知识库协作区。
                </p>
              </div>
            </div>

            <div className="rounded-[24px] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.7)] p-4 text-sm text-[var(--color-text-secondary)]">
              <div className="flex items-center justify-between gap-4">
                <span className="font-semibold text-[var(--color-text-primary)]">已经有账户？</span>
                <Link
                  href="/login"
                  className="font-semibold text-[var(--color-action-primary)] transition hover:text-[var(--color-action-primary-hover)]"
                >
                  返回登录
                </Link>
              </div>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit}>
              {/* 注册表单保持简单直接，避免在首个转化步骤引入额外认知负担。 */}
              <div className="space-y-4">
                <div className="space-y-2">
                  <label htmlFor="username" className="text-sm font-semibold text-[var(--color-text-primary)]">
                    用户名
                  </label>
                  <UiInput
                    id="username"
                    name="username"
                    type="text"
                    required
                    placeholder="例如 openintern-team"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="email" className="text-sm font-semibold text-[var(--color-text-primary)]">
                    邮箱
                  </label>
                  <UiInput
                    id="email"
                    name="email"
                    type="email"
                    required
                    placeholder="name@company.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="password" className="text-sm font-semibold text-[var(--color-text-primary)]">
                    密码
                  </label>
                  <UiInput
                    id="password"
                    name="password"
                    type="password"
                    required
                    placeholder="设置密码"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </div>
              </div>

              {error && (
                <div className="rounded-[18px] border border-[rgba(179,64,51,0.16)] bg-[rgba(179,64,51,0.08)] px-4 py-3 text-sm text-[var(--color-state-error)]">
                  {error}
                </div>
              )}

              <UiButton type="submit" disabled={loading} className="w-full justify-center">
                <svg
                  className="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M16 7a4 4 0 1 0-8 0" />
                  <path d="M3 21v-1a6 6 0 0 1 6-6h2" />
                  <path d="M17 14v6" />
                  <path d="M14 17h6" />
                </svg>
                {loading ? "注册中..." : "创建账户"}
              </UiButton>
            </form>
          </div>
        </section>
      </div>
    </div>
  );
}
