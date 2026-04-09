import { afterEach, describe, expect, it, vi } from "vitest";
import {
  BackendError,
  getDailyNote,
  getLinkPreview,
  podcastTranscribe,
  saveDailyNote,
  validateSession,
} from "./backend";

const fetchMock = vi.fn<typeof fetch>();

function getRequestInit(): RequestInit {
  return fetchMock.mock.calls[0]?.[1] as RequestInit;
}

describe("backend service", () => {
  afterEach(() => {
    fetchMock.mockReset();
    vi.unstubAllGlobals();
  });

  it("validates the current session", async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ message: "ok" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await validateSession("token-123");

    expect(fetchMock).toHaveBeenCalledWith("http://localhost:3000/", expect.any(Object));
    expect(getRequestInit()).toMatchObject({
      body: undefined,
      method: "GET",
      signal: undefined,
    });
    expect(new Headers(getRequestInit().headers).get("Authorization")).toBe("Bearer token-123");
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

    expect(fetchMock).toHaveBeenCalledWith("http://localhost:3000/daily", expect.any(Object));
    expect(getRequestInit()).toMatchObject({
      body: JSON.stringify({
        contents: btoa("Updated note\n"),
        title: "2026-03-12.md",
      }),
      method: "POST",
      signal: undefined,
    });
    expect(new Headers(getRequestInit().headers).get("Content-Type")).toBe("application/json");
    expect(new Headers(getRequestInit().headers).get("Authorization")).toBe("Bearer token-123");
  });

  it("fetches a link preview", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ title: "Example Title", description: "Example description" }), {
        status: 200,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getLinkPreview("token-123", "https://example.com")).resolves.toEqual({
      title: "Example Title",
      description: "Example description",
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/linkpreview?q=https%3A%2F%2Fexample.com",
      expect.any(Object),
    );
    expect(getRequestInit()).toMatchObject({
      body: undefined,
      method: "GET",
      signal: undefined,
    });
    expect(new Headers(getRequestInit().headers).get("Authorization")).toBe("Bearer token-123");
  });

  it("posts podcast payloads as form data", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          podcast: {
            source: "podcastaddict",
            podcast: {
              name: "Debug Jams",
              page_url: "https://example.com/podcast",
              artwork_url: "https://example.com/artwork.jpg",
            },
            episode: {
              title: "Episode title",
              page_url: "https://example.com/episode",
              mp3_url: "https://example.com/episode.mp3",
              publication_date: "2026-04-09T00:00:00Z",
              author: "Debug Jams",
              description_html: "<p>Episode description</p>",
            },
          },
          transcription_lambda_id: "local-123",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      podcastTranscribe("token-123", "Shared from Podcast Addict https://example.com/episode"),
    ).resolves.toMatchObject({
      transcription_lambda_id: "local-123",
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:3000/podcast-transcribe",
      expect.any(Object),
    );
    expect(getRequestInit()).toMatchObject({
      body: new URLSearchParams({
        text: "Shared from Podcast Addict https://example.com/episode",
      }),
      method: "POST",
      signal: undefined,
    });
    expect(new Headers(getRequestInit().headers).get("Content-Type")).toBe(
      "application/x-www-form-urlencoded",
    );
    expect(new Headers(getRequestInit().headers).get("Authorization")).toBe("Bearer token-123");
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
