import { http, HttpResponse } from "msw";

// Matches VITE_SITE_BACKEND_URL defined in vitest.config.ts
const BASE_URL = "http://localhost:3000";

function authCheck(request: Request) {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) return HttpResponse.json({ error: "unauthorized" }, { status: 401 });
  if (authHeader === "Bearer wrong-user-token")
    return HttpResponse.json({ error: "forbidden" }, { status: 403 });
  return null;
}

export let latestDailyNote = {
  title: "2026-03-12.md",
  contents: "# Daily Note\n\nTest content.",
};

export let latestPodcastTranscribeText = "";

export function resetMockDailyNote() {
  latestDailyNote = {
    title: "2026-03-12.md",
    contents: "# Daily Note\n\nTest content.",
  };
}

export function resetMockPodcastTranscribe() {
  latestPodcastTranscribeText = "";
}

export const handlers = [
  http.get(`${BASE_URL}/`, ({ request }) => {
    return authCheck(request) ?? HttpResponse.json({ status: "ok", email: "test@example.com" });
  }),

  http.get(`${BASE_URL}/daily`, ({ request }) => {
    return (
      authCheck(request) ??
      HttpResponse.json({
        title: latestDailyNote.title,
        contents: btoa(latestDailyNote.contents),
      })
    );
  }),

  http.post(`${BASE_URL}/daily`, async ({ request }) => {
    const authResponse = authCheck(request);
    if (authResponse) {
      return authResponse;
    }

    const body = (await request.json()) as { title: string; contents: string };
    latestDailyNote = {
      title: body.title,
      contents: atob(body.contents),
    };

    return HttpResponse.json({
      title: latestDailyNote.title,
      contents: btoa(latestDailyNote.contents),
    });
  }),

  http.post(`${BASE_URL}/podcast-transcribe`, async ({ request }) => {
    const authResponse = authCheck(request);
    if (authResponse) {
      return authResponse;
    }

    const body = await request.formData();
    latestPodcastTranscribeText = String(body.get("text") ?? "");

    if (!latestPodcastTranscribeText.trim()) {
      return HttpResponse.json({ error: "text parameter is required" }, { status: 400 });
    }

    return HttpResponse.json({
      podcast: {
        source: "podcastaddict",
        podcast: {
          name: "Debug Jams",
          page_url: "https://example.com/podcast",
          artwork_url: "https://example.com/artwork.jpg",
        },
        episode: {
          title: "The Payload Episode",
          page_url: "https://example.com/episode",
          mp3_url: "https://example.com/episode.mp3",
          publication_date: "2026-04-09T00:00:00Z",
          author: "Debug Jams",
          description_html: "<p>Episode description</p>",
        },
      },
      transcription_lambda_id: "local-test-transcription",
    });
  }),
];
