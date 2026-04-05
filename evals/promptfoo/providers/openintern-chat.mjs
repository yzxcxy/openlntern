import { runOpenInternChat } from "../src/openintern-chat.mjs";

export default class OpenInternChatProvider {
  constructor(options = {}) {
    this.providerId = options.id || "openintern-chat";
    this.config = options.config || {};
  }

  id() {
    return this.providerId;
  }

  async callApi(prompt, context) {
    const vars = {
      ...this.config,
      ...(context?.vars || {})
    };
    return runOpenInternChat({ prompt, vars });
  }
}
