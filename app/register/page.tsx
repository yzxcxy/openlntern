"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { UiButton } from "../components/ui/UiButton";
import { UiInput } from "../components/ui/UiInput";

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

      // 注册成功后跳转到登录页
      router.push("/login");
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--color-bg-page)] px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md space-y-8">
        <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-md)]">
          <h2 className="mt-2 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
            注册新账户
          </h2>
          <p className="mt-2 text-center text-sm text-[var(--color-text-secondary)]">
            已有账户？{" "}
            <Link
              href="/login"
              className="font-medium text-[var(--color-action-primary)] hover:text-[var(--color-action-primary-hover)]"
            >
              立即登录
            </Link>
          </p>
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-3">
              <div>
                <label htmlFor="username" className="sr-only">
                  用户名
                </label>
                <UiInput
                  id="username"
                  name="username"
                  type="text"
                  required
                  placeholder="用户名"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="email" className="sr-only">
                  邮箱
                </label>
                <UiInput
                  id="email"
                  name="email"
                  type="email"
                  required
                  placeholder="邮箱"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="password" className="sr-only">
                  密码
                </label>
                <UiInput
                  id="password"
                  name="password"
                  type="password"
                  required
                  placeholder="密码"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
            </div>

            {error && (
              <div className="text-center text-sm text-[var(--color-state-error)]">
                {error}
              </div>
            )}

            <div>
              <UiButton type="submit" disabled={loading} className="w-full">
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
                {loading ? "注册中..." : "注册"}
              </UiButton>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
