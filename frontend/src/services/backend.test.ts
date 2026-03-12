import { afterEach, describe, expect, it, vi } from "vitest";
import { BackendError, getDailyNote, saveDailyNote, validateSession } from "./backend";

const fetchMock = vi.fn<typeof fetch>();

describe("backend service", () => {
  afterEach(() => {
    fetchMock.mockReset();
    vi.unstubAllGlobals();
  });

  it("validates the current session", async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ message: "ok" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await validateSession("token-123");

    expect(fetchMock).toHaveBeenCalledWith("http://localhost:3000/", {
      body: undefined,
      headers: { Authorization: "Bearer token-123" },
      method: "GET",
      signal: undefined,
    });
  });

  it("maps 403 session validation failures to forbidden errors", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 403 }));
    vi.stubGlobal("fetch", fetchMock);
    const expectedError: Partial<BackendError> = {
      kind: "forbidden",
      status: 403,
    };

    await expect(validateSession("token-123")).rejects.toMatchObject(expectedError);
  });

  it("loads and decodes the daily note", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          contents: btoa("# Daily Note\n\nTest content."),
          title: "2026-03-12.md",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getDailyNote("token-123")).resolves.toEqual({
      contents: "# Daily Note\n\nTest content.",
      title: "2026-03-12.md",
    });
  });

  it("saves the daily note with encoded contents", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          contents: btoa("Updated note\n"),
          title: "2026-03-12.md",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      saveDailyNote("token-123", {
        contents: "Updated note\n",
        title: "2026-03-12.md",
      }),
    ).resolves.toEqual({
      contents: "Updated note\n",
      title: "2026-03-12.md",
    });

    expect(fetchMock).toHaveBeenCalledWith("http://localhost:3000/daily", {
      body: JSON.stringify({
        contents: btoa("Updated note\n"),
        title: "2026-03-12.md",
      }),
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer token-123",
      },
      method: "POST",
      signal: undefined,
    });
  });

  it("maps network failures to backend errors", async () => {
    fetchMock.mockRejectedValue(new TypeError("Failed to fetch"));
    vi.stubGlobal("fetch", fetchMock);
    const expectedError: Partial<BackendError> = {
      kind: "network",
      message: "Could not reach the backend.",
      status: null,
    };

    await expect(getDailyNote("token-123")).rejects.toMatchObject(expectedError);
  });
});
