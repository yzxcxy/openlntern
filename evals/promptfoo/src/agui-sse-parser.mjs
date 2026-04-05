const DATA_PREFIX = "data:";

// parseAguiSseText rebuilds assistant text from AG-UI TEXT_MESSAGE_CONTENT events.
export function parseAguiSseText(rawText) {
  if (!rawText) {
    return "";
  }

  let output = "";
  const lines = String(rawText).split(/\r?\n/u);

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line.startsWith(DATA_PREFIX)) {
      continue;
    }

    const payload = line.slice(DATA_PREFIX.length).trim();
    if (!payload || payload === "[DONE]") {
      continue;
    }

    try {
      const event = JSON.parse(payload);
      if (
        event &&
        event.type === "TEXT_MESSAGE_CONTENT" &&
        typeof event.delta === "string"
      ) {
        output += event.delta;
      }
    } catch {
      continue;
    }
  }

  return output.trim();
}
