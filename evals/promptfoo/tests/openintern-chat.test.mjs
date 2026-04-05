import test from "node:test";
import assert from "node:assert/strict";

import { runOpenInternChat } from "../src/openintern-chat.mjs";

test("runOpenInternChat logs in, calls /v1/chat/sse, and returns parsed assistant text", async () => {
  const requests = [];
  const response = await runOpenInternChat({
    prompt: "介绍一下 openIntern",
    vars: {
      threadId: "thread-eval-1",
      providerId: "provider-1",
      modelId: "model-1"
    },
    env: {
      OPENINTERN_BASE_URL: "http://127.0.0.1:8080",
      OPENINTERN_USERNAME: "demo@example.com",
      OPENINTERN_PASSWORD: "secret"
    },
    fetchImpl: async (url, init = {}) => {
      requests.push({ url, init });

      if (url.endsWith("/v1/auth/login")) {
        return {
          ok: true,
          status: 200,
          statusText: "OK",
          async json() {
            return {
              code: 0,
              data: {
                token: "jwt-token-value"
              }
            };
          }
        };
      }

      if (url.endsWith("/v1/chat/sse")) {
        assert.equal(init.method, "POST");
        assert.equal(init.headers.Authorization, "Bearer jwt-token-value");
        assert.match(init.headers["Content-Type"], /application\/json/);

        const body = JSON.parse(init.body);
        assert.equal(body.threadId, "thread-eval-1");
        assert.equal(body.forwardedProps.agentConfig.model.providerId, "provider-1");
        assert.equal(body.forwardedProps.agentConfig.model.modelId, "model-1");

        return {
          ok: true,
          status: 200,
          statusText: "OK",
          async text() {
            return [
              'data: {"type":"TEXT_MESSAGE_START","messageId":"msg-1"}',
              'data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","delta":"openIntern"}',
              'data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","delta":" 已启动"}',
              'data: {"type":"TEXT_MESSAGE_END","messageId":"msg-1"}'
            ].join("\n");
          }
        };
      }

      throw new Error(`unexpected request: ${url}`);
    }
  });

  assert.equal(requests.length, 2);
  assert.equal(response.output, "openIntern 已启动");
  assert.equal(response.metadata.threadId, "thread-eval-1");
});
