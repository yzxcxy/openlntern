import { CopilotRuntime, createCopilotEndpoint } from "@copilotkit/runtime/v2";
import { handle } from "hono/vercel";
import { HttpAgent } from "@ag-ui/client";

const apiBaseUrl = process.env.API_BASE_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

const runtime = new CopilotRuntime({
  agents: {
    default: new HttpAgent({
      url: `${apiBaseUrl}/v1/chat/sse`,
    }),
  },
});

const app = createCopilotEndpoint({
  runtime,
  basePath: "/api/copilotkit",
});

const handler = (request: Request) => handle(app)(request);

export const GET = handler;
export const POST = handler;
