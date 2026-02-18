import {
  CopilotRuntime,
  createCopilotEndpointSingleRoute,
} from "@copilotkit/runtime/v2";
import { handle } from "hono/vercel";
import { HttpAgent } from "@ag-ui/client";
import { ExperimentalEmptyAdapter } from "@copilotkit/runtime";

const apiBaseUrl = process.env.API_BASE_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

const createRuntime = () =>
  new CopilotRuntime({
    agents: {
      default: new HttpAgent({
        url: `${apiBaseUrl}/v1/chat/sse`,
      }),
    },
  });

const createApp = () =>
  createCopilotEndpointSingleRoute({
    runtime: createRuntime(),
    basePath: "/api/copilotkit",
  });

const handler = (request: Request) => {
  const app = createApp();
  return handle(app)(request);
};

export const GET = handler;
export const POST = handler;
