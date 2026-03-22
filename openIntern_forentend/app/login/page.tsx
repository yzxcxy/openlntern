"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { UiButton } from "../components/ui/UiButton";
import { UiInput } from "../components/ui/UiInput";

const LOGIN_METRICS = [
  { value: "Agents", label: "管理智能体、技能和插件资产。" },
  { value: "Knowledge", label: "在会话中使用知识库与上下文。" },
  { value: "Models", label: "统一查看和切换模型服务配置。" },
];

export default function LoginPage() {
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch("/api/backend/v1/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ identifier, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.message || "登录失败");
      }

      if (data.data && data.data.token) {
        localStorage.setItem("token", data.data.token);
        localStorage.setItem("user", JSON.stringify(data.data.user));
        router.push("/");
      } else {
        throw new Error("Invalid response from server");
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-shell">
      <div className="auth-ambient left-[6%] top-[8%] h-56 w-56 bg-[rgba(209,157,86,0.2)]" />
      <div className="auth-ambient bottom-[6%] right-[8%] h-64 w-64 bg-[rgba(31,95,114,0.18)]" />
      <div className="auth-grid items-stretch">
        <section className="auth-hero-card hidden lg:flex lg:flex-col lg:justify-between">
          <div className="space-y-8">
            <div className="auth-kicker">
              <span className="h-2 w-2 rounded-full bg-[var(--color-action-primary)]" />
              openIntern workspace
            </div>
            <div className="max-w-3xl space-y-5">
              <p className="text-sm font-semibold uppercase tracking-[0.18em] text-[var(--color-text-muted)]">
                AI Collaboration
              </p>
              <h1 className="max-w-2xl text-5xl leading-[0.92] text-[var(--color-text-primary)] xl:text-7xl">
                登录后继续你的 AI 协作任务。
              </h1>
              <p className="max-w-xl text-lg leading-8 text-[var(--color-text-secondary)]">
                在一个工作区里处理聊天、Agent、模型、插件和知识库。
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <span className="auth-chip">聊天</span>
              <span className="auth-chip">Agent</span>
              <span className="auth-chip">知识库</span>
            </div>
          </div>

          <div className="grid gap-4 xl:grid-cols-3">
            {LOGIN_METRICS.map((metric) => (
              <div key={metric.value} className="auth-metric-card">
                <strong>{metric.value}</strong>
                <span>{metric.label}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="auth-form-card">
          <div className="space-y-6">
            <div className="space-y-3">
              <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[var(--color-text-muted)]">
                Welcome Back
              </p>
              <div>
                <h2 className="text-4xl leading-none text-[var(--color-text-primary)]">
                  登录 openIntern
                </h2>
                <p className="mt-3 text-sm leading-6 text-[var(--color-text-secondary)]">
                  继续处理会话、知识库和 Agent 任务。
                </p>
              </div>
            </div>

            <div className="rounded-[24px] border border-[var(--color-border-default)] bg-[rgba(255,252,247,0.7)] p-4 text-sm text-[var(--color-text-secondary)]">
              <div className="flex items-center justify-between gap-4">
                <span className="font-semibold text-[var(--color-text-primary)]">还没有账户？</span>
                <Link
                  href="/register"
                  className="font-semibold text-[var(--color-action-primary)] transition hover:text-[var(--color-action-primary-hover)]"
                >
                  创建账户
                </Link>
              </div>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit}>
              {/* 表单区维持最短路径，避免装饰抢夺登录动作。 */}
              <div className="space-y-4">
                <div className="space-y-2">
                  <label htmlFor="identifier" className="text-sm font-semibold text-[var(--color-text-primary)]">
                    用户名或邮箱
                  </label>
                  <UiInput
                    id="identifier"
                    name="identifier"
                    type="text"
                    required
                    placeholder="name@company.com"
                    value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)}
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
                    placeholder="输入密码"
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
                  <path d="M7 10V7a5 5 0 0 1 10 0v3" />
                  <rect x="5" y="10" width="14" height="10" rx="2" />
                  <circle cx="12" cy="15" r="1" />
                </svg>
                {loading ? "登录中..." : "进入工作台"}
              </UiButton>
            </form>
          </div>
        </section>
      </div>
    </div>
  );
}
