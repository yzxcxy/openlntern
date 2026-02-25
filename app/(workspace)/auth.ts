export type StoredUser = {
  user_id?: string | number;
  username?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  role?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
};

type RouterLike = {
  push: (href: string) => void;
};

export const parseTokenPayload = (token: string) => {
  if (!token) return null;
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
    return JSON.parse(atob(padded));
  } catch {
    return null;
  }
};

export const isTokenExpired = (token: string) => {
  const payload = parseTokenPayload(token);
  const exp = typeof payload?.exp === "number" ? payload.exp : Number(payload?.exp);
  if (!exp) return true;
  return Math.floor(Date.now() / 1000) >= exp;
};

export const getUserIdFromToken = (token: string) => {
  const decoded = parseTokenPayload(token);
  const userId = decoded?.user_id ?? decoded?.userId ?? decoded?.sub;
  if (typeof userId === "string" || typeof userId === "number") {
    return String(userId);
  }
  return "";
};

export const readStoredUser = <T = StoredUser>(): T | null => {
  if (typeof window === "undefined") return null;
  const storedUser = localStorage.getItem("user");
  if (!storedUser) return null;
  try {
    return JSON.parse(storedUser) as T;
  } catch {
    return null;
  }
};

export const readValidToken = (router?: RouterLike) => {
  if (typeof window === "undefined") return "";
  const token = localStorage.getItem("token");
  if (!token) return "";
  if (isTokenExpired(token)) {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    router?.push("/login");
    return "";
  }
  return token;
};

export const updateTokenFromResponse = (res: Response) => {
  const refreshedToken = res.headers.get("X-Access-Token");
  if (refreshedToken) {
    localStorage.setItem("token", refreshedToken);
  }
  const expiresAt = res.headers.get("X-Token-Expires");
  if (expiresAt) {
    localStorage.setItem("token_expires", expiresAt);
  }
};

export const buildAuthHeaders = (token: string, userId?: string) => {
  const headers: Record<string, string> = {};
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (userId) {
    headers["X-User-ID"] = userId;
  }
  return headers;
};
