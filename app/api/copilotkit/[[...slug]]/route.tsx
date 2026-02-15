import {
  CopilotRuntime,
  createCopilotEndpoint,
} from "@copilotkit/runtime/v2";
import { handle } from "hono/vercel";
import { HttpAgent } from "@ag-ui/client";
import { ExperimentalEmptyAdapter } from "@copilotkit/runtime";


const serviceAdapter = new ExperimentalEmptyAdapter();

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

export const GET = handle(app);
export const POST = handle(app);
