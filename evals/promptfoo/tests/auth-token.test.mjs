import test from "node:test";
import assert from "node:assert/strict";

import { fetchPromptfooToken } from "../src/auth-token.mjs";

test("fetchPromptfooToken returns the JWT from the openIntern login envelope", async () => {
  const token = await fetchPromptfooToken({
    baseUrl: "http://127.0.0.1:8080",
    identifier: "demo@example.com",
    password: "secret",
    fetchImpl: async (url, init) => {
      assert.equal(url, "http://127.0.0.1:8080/v1/auth/login");
      assert.equal(init.method, "POST");
      assert.match(init.headers["Content-Type"], /application\/json/);
      assert.equal(
        init.body,
        JSON.stringify({
          identifier: "demo@example.com",
          password: "secret"
        })
      );
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
  });

  assert.equal(token, "jwt-token-value");
});

test("fetchPromptfooToken throws when the login response does not contain a usable token", async () => {
  await assert.rejects(
    () =>
      fetchPromptfooToken({
        baseUrl: "http://127.0.0.1:8080",
        identifier: "demo@example.com",
        password: "secret",
        fetchImpl: async () => ({
          ok: true,
          status: 200,
          statusText: "OK",
          async json() {
            return {
              code: 0,
              data: {}
            };
          }
        })
      }),
    /token/i
  );
});
