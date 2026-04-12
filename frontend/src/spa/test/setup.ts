import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeAll, afterAll } from "vitest";
import { resetMockDailyNote, resetMockPodcastTranscribe } from "./mocks/handlers";
import { server } from "./mocks/server";

// React 19 checks globalThis.IS_REACT_ACT_ENVIRONMENT, but @testing-library/react
// sets self.IS_REACT_ACT_ENVIRONMENT. In jsdom these are different objects, causing
// act() warnings on every test. This bridges them.
(globalThis as Record<string, unknown>).IS_REACT_ACT_ENVIRONMENT = true;

if (!Range.prototype.getClientRects) {
  Range.prototype.getClientRects = function getClientRects() {
    return {
      item: () => null,
      length: 0,
      [Symbol.iterator]: function* iterator() {},
    } as DOMRectList;
  };
}

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  resetMockDailyNote();
  resetMockPodcastTranscribe();
  cleanup();
});
afterAll(() => server.close());
