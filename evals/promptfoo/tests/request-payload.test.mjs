import test from "node:test";
import assert from "node:assert/strict";

import { buildChatSsePayload } from "../src/request-payload.mjs";

test("buildChatSsePayload creates a chat-mode payload for /v1/chat/sse", () => {
  const payload = buildChatSsePayload({
    threadId: "thread-1",
    input: "介绍一下 openIntern",
    providerId: "provider-1",
    modelId: "model-1"
  });

  assert.deepEqual(payload, {
    threadId: "thread-1",
    messages: [
      {
        id: "thread-1-user",
        role: "user",
        content: "介绍一下 openIntern"
      }
    ],
    forwardedProps: {
      agentConfig: {
        conversation: {
          mode: "chat"
        },
        model: {
          providerId: "provider-1",
          modelId: "model-1"
        },
        plugins: {
          mode: "select",
          selectedToolIds: []
        }
      }
    }
  });
});

test("buildChatSsePayload applies agent-mode overrides when evaluating an enabled agent", () => {
  const payload = buildChatSsePayload({
    threadId: "thread-2",
    input: "帮我总结最近的项目变更",
    providerId: "provider-2",
    modelId: "model-2",
    conversationMode: "agent",
    selectedAgentId: "agent-123",
    selectedToolIds: ["tool-1", "tool-2"]
  });

  assert.deepEqual(payload.forwardedProps.agentConfig, {
    conversation: {
      mode: "agent"
    },
    model: {
      providerId: "provider-2",
      modelId: "model-2"
    },
    plugins: {
      mode: "select",
      selectedToolIds: ["tool-1", "tool-2"]
    },
    features: {
      selectedAgentId: "agent-123"
    }
  });
});
