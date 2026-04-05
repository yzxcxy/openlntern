function normalizeBaseUrl(baseUrl) {
  if (!baseUrl || typeof baseUrl !== "string") {
    throw new Error("OPENINTERN_BASE_URL is required");
  }
  return baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
}

// fetchPromptfooToken logs into openIntern and returns the JWT used for eval requests.
export async function fetchPromptfooToken({
  baseUrl,
  identifier,
  password,
  fetchImpl = fetch
}) {
  if (!identifier || typeof identifier !== "string") {
    throw new Error("OPENINTERN_USERNAME is required");
  }
  if (!password || typeof password !== "string") {
    throw new Error("OPENINTERN_PASSWORD is required");
  }

  const loginUrl = `${normalizeBaseUrl(baseUrl)}/v1/auth/login`;
  const response = await fetchImpl(loginUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      identifier,
      password
    })
  });

  const payload = await response.json();
  const token = payload?.data?.token;

  if (!response.ok || payload?.code !== 0 || typeof token !== "string" || token.trim() === "") {
    const message = payload?.message || `login failed: ${response.status} ${response.statusText}`;
    throw new Error(`openIntern login did not return a usable token: ${message}`);
  }

  return token;
}
