import { randomUUID } from "node:crypto";

import { fetchPromptfooToken } from "./auth-token.mjs";
import { parseAguiSseText } from "./agui-sse-parser.mjs";
import { buildChatSsePayload } from "./request-payload.mjs";

function readStringSetting(value) {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeBaseUrl(baseUrl) {
  if (!baseUrl || typeof baseUrl !== "string") {
    throw new Error("OPENINTERN_BASE_URL is required");
  }
  return baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
}

function resolveSelectedToolIds(selectedToolIds) {
  if (!Array.isArray(selectedToolIds)) {
    return [];
  }
  return selectedToolIds;
}

// runOpenInternChat executes one promptfoo eval request against the openIntern chat SSE endpoint.
export async function runOpenInternChat({
  prompt,
  vars = {},
  env = process.env,
  fetchImpl = fetch
}) {
  const baseUrl = normalizeBaseUrl(env.OPENINTERN_BASE_URL);
  const identifier = readStringSetting(env.OPENINTERN_USERNAME);
  const password = readStringSetting(env.OPENINTERN_PASSWORD);
  const providerId = readStringSetting(vars.providerId) || readStringSetting(env.OPENINTERN_PROVIDER_ID);
  const modelId = readStringSetting(vars.modelId) || readStringSetting(env.OPENINTERN_MODEL_ID);
  const conversationMode =
    readStringSetting(vars.conversationMode) ||
    readStringSetting(env.OPENINTERN_CONVERSATION_MODE) ||
    "chat";
  const selectedAgentId =
    readStringSetting(vars.selectedAgentId) ||
    readStringSetting(env.OPENINTERN_SELECTED_AGENT_ID);
  const selectedToolIds = resolveSelectedToolIds(vars.selectedToolIds);
  const threadId = readStringSetting(vars.threadId) || randomUUID();

  const token = await fetchPromptfooToken({
    baseUrl,
    identifier,
    password,
    fetchImpl
  });

  const payload = buildChatSsePayload({
    threadId,
    input: prompt,
    providerId,
    modelId,
    conversationMode,
    selectedAgentId,
    selectedToolIds
  });

  const response = await fetchImpl(`${baseUrl}/v1/chat/sse`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
  const raw = await response.text();

  if (!response.ok) {
    throw new Error(
      `openIntern chat request failed: ${response.status} ${response.statusText}`
    );
  }

  return {
    output: parseAguiSseText(raw),
    raw,
    prompt,
    metadata: {
      threadId,
      conversationMode
    }
  };
}
