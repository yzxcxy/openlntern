import { NextRequest, NextResponse } from "next/server";

const apiBaseUrl = process.env.API_BASE_URL ?? "";

const normalizeBaseUrl = (value: string) => {
  if (value.endsWith("/")) {
    return value.slice(0, -1);
  }
  return value;
};

const proxyRequest = async (
  request: NextRequest,
  context: { params?: { path?: string[] } }
) => {
  if (!apiBaseUrl) {
    return NextResponse.json(
      { message: "API_BASE_URL 未配置" },
      { status: 500 }
    );
  }
  const pathname = request.nextUrl.pathname;
  const fallbackPath = pathname.replace(/^\/api\/backend\/?/, "");
  const pathSegments =
    context.params?.path ??
    (fallbackPath ? fallbackPath.split("/") : []);
  const base = normalizeBaseUrl(apiBaseUrl);
  const targetUrl = new URL(`${base}/${pathSegments.join("/")}`);
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers(request.headers);
  headers.delete("host");
  headers.delete("content-length");

  const init: RequestInit = {
    method: request.method,
    headers,
  };

  if (request.method !== "GET" && request.method !== "HEAD") {
    init.body = await request.arrayBuffer();
  }

  const response = await fetch(targetUrl, init);
  const responseHeaders = new Headers(response.headers);

  return new NextResponse(response.body, {
    status: response.status,
    headers: responseHeaders,
  });
};

export const GET = proxyRequest;
export const POST = proxyRequest;
export const PUT = proxyRequest;
export const PATCH = proxyRequest;
export const DELETE = proxyRequest;
export const OPTIONS = proxyRequest;
