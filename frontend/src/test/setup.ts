import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeAll, afterAll } from "vitest";
import { server } from "./mocks/server";

// React 19 checks globalThis.IS_REACT_ACT_ENVIRONMENT, but @testing-library/react
// sets self.IS_REACT_ACT_ENVIRONMENT. In jsdom these are different objects, causing
// act() warnings on every test. This bridges them.
(globalThis as Record<string, unknown>).IS_REACT_ACT_ENVIRONMENT = true;

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  cleanup();
});
afterAll(() => server.close());
