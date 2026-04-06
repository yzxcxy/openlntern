const BACKEND_API_BASE = "/api/backend";

// resolveBackendAssetUrl keeps browser asset requests on the existing backend proxy origin.
export const resolveBackendAssetUrl = (value?: string) => {
  const trimmed = (value ?? "").trim();
  if (!trimmed) return "";
  if (
    /^https?:\/\//i.test(trimmed) ||
    trimmed.startsWith("data:") ||
    trimmed.startsWith("blob:")
  ) {
    return trimmed;
  }
  if (trimmed.startsWith(`${BACKEND_API_BASE}/`)) {
    return trimmed;
  }
  if (trimmed.startsWith("/")) {
    return `${BACKEND_API_BASE}${trimmed}`;
  }
  return trimmed;
};
