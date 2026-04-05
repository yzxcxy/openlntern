function normalizeConversationMode(conversationMode) {
  return conversationMode === "agent" ? "agent" : "chat";
}

function normalizeSelectedToolIds(selectedToolIds) {
  if (!Array.isArray(selectedToolIds)) {
    return [];
  }

  return selectedToolIds
    .filter((item) => typeof item === "string")
    .map((item) => item.trim())
    .filter(Boolean);
}

// buildChatSsePayload constructs the JSON body expected by POST /v1/chat/sse.
export function buildChatSsePayload({
  threadId,
  input,
  providerId,
  modelId,
  conversationMode = "chat",
  selectedAgentId = "",
  selectedToolIds = []
}) {
  if (!threadId || typeof threadId !== "string") {
    throw new Error("threadId is required");
  }
  if (!input || typeof input !== "string") {
    throw new Error("input is required");
  }
  if (!providerId || typeof providerId !== "string") {
    throw new Error("providerId is required");
  }
  if (!modelId || typeof modelId !== "string") {
    throw new Error("modelId is required");
  }

  const normalizedMode = normalizeConversationMode(conversationMode);
  const normalizedToolIds = normalizeSelectedToolIds(selectedToolIds);
  const agentConfig = {
    conversation: {
      mode: normalizedMode
    },
    model: {
      providerId: providerId.trim(),
      modelId: modelId.trim()
    },
    plugins: {
      mode: "select",
      selectedToolIds: normalizedToolIds
    }
  };

  if (normalizedMode === "agent") {
    agentConfig.features = {
      selectedAgentId: String(selectedAgentId || "").trim()
    };
  }

  return {
    threadId: threadId.trim(),
    messages: [
      {
        id: `${threadId.trim()}-user`,
        role: "user",
        content: input
      }
    ],
    forwardedProps: {
      agentConfig
    }
  };
}
