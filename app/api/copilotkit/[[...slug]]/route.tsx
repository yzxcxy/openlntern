import {
  CopilotRuntime,
  createCopilotEndpoint,
} from "@copilotkit/runtime/v2";
import { handle } from "hono/vercel";
import { HttpAgent } from "@ag-ui/client";
import { ExperimentalEmptyAdapter } from "@copilotkit/runtime";


// 1. You can use any service adapter here for multi-agent support. We use
//    the empty adapter since we're only using one agent.
const serviceAdapter = new ExperimentalEmptyAdapter();

// 2. Create the CopilotRuntime instance and utilize the HttpAgent to setup the connection.
const runtime = new CopilotRuntime({
  agents: {
    default: new HttpAgent({
      url: "http://localhost:8080/v1/chat/sse",
    }),
  },
});


const app = createCopilotEndpoint({
  runtime,
  basePath: "/api/copilotkit",
});

export const GET = handle(app);
export const POST = handle(app);
