"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { UiButton } from "../components/ui/UiButton";
import { UiInput } from "../components/ui/UiInput";

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

      // 保存 token
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
    <div className="flex min-h-screen items-center justify-center bg-[var(--color-bg-page)] px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md space-y-8">
        <div className="rounded-[var(--radius-xl)] border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-6 shadow-[var(--shadow-md)]">
          <h2 className="mt-2 text-center text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
            登录您的账户
          </h2>
          <p className="mt-2 text-center text-sm text-[var(--color-text-secondary)]">
            或者{" "}
            <Link
              href="/register"
              className="font-medium text-[var(--color-action-primary)] hover:text-[var(--color-action-primary-hover)]"
            >
              注册新账户
            </Link>
          </p>
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-3">
              <div>
                <label htmlFor="identifier" className="sr-only">
                  用户名或邮箱
                </label>
                <UiInput
                  id="identifier"
                  name="identifier"
                  type="text"
                  required
                  placeholder="用户名或邮箱"
                  value={identifier}
                  onChange={(e) => setIdentifier(e.target.value)}
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
                  <path d="M7 10V7a5 5 0 0 1 10 0v3" />
                  <rect x="5" y="10" width="14" height="10" rx="2" />
                  <circle cx="12" cy="15" r="1" />
                </svg>
                {loading ? "登录中..." : "登录"}
              </UiButton>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
