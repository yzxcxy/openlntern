import test from "node:test";
import assert from "node:assert/strict";

import { parseAguiSseText } from "../src/agui-sse-parser.mjs";

test("parseAguiSseText reconstructs assistant text from AG-UI SSE deltas", () => {
  const raw = [
    'data: {"type":"RUN_STARTED","threadId":"thread-1","runId":"run-1"}',
    'data: {"type":"TEXT_MESSAGE_START","messageId":"msg-1"}',
    'data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","delta":"你好"}',
    'data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","delta":"，世界"}',
    'data: {"type":"TEXT_MESSAGE_END","messageId":"msg-1"}',
    'data: {"type":"RUN_FINISHED","threadId":"thread-1","runId":"run-1"}'
  ].join("\n");

  assert.equal(parseAguiSseText(raw), "你好，世界");
});

test("parseAguiSseText ignores non-text events and malformed payload lines", () => {
  const raw = [
    "event: message",
    'data: {"type":"REASONING_MESSAGE_CONTENT","messageId":"reason-1","delta":"thinking"}',
    "data: not-json",
    'data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-2","delta":"final"}'
  ].join("\n");

  assert.equal(parseAguiSseText(raw), "final");
});
