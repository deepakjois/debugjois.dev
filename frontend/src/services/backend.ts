export interface DailyNote {
  title: string;
  contents: string;
}

export type BackendErrorKind = "unauthenticated" | "forbidden" | "http" | "network";

export class BackendError extends Error {
  kind: BackendErrorKind;
  status: number | null;
  responseMessage: string | null;

  constructor(
    kind: BackendErrorKind,
    message: string,
    status: number | null = null,
    responseMessage: string | null = null,
  ) {
    super(message);
    this.name = "BackendError";
    this.kind = kind;
    this.status = status;
    this.responseMessage = responseMessage;
  }
}

const API_URL = import.meta.env.VITE_SITE_BACKEND_URL;

interface RequestOptions {
  body?: BodyInit;
  headers?: HeadersInit;
  method?: string;
  signal?: AbortSignal;
  token: string;
}

function authHeaders(token: string): HeadersInit {
  if (import.meta.env.VITE_AUTH_BYPASS === "true") {
    return {};
  }

  return { Authorization: `Bearer ${token}` };
}

function encodeBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";

  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return btoa(binary);
}

function decodeBase64(value: string): string {
  if (!value) {
    return "";
  }

  const binary = atob(value);
  const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

async function getErrorResponseMessage(response: Response): Promise<string | null> {
  try {
    const body = (await response.json()) as { error?: unknown };
    return typeof body.error === "string" && body.error.trim() ? body.error : null;
  } catch {
    return null;
  }
}

async function request(path: string, options: RequestOptions): Promise<Response> {
  const { body, headers: customHeaders, method = "GET", signal, token } = options;
  const headers = new Headers(authHeaders(token));

  if (customHeaders) {
    new Headers(customHeaders).forEach((value, key) => headers.set(key, value));
  }

  if (body && !(body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(`${API_URL}${path}`, {
      body,
      headers,
      method,
      signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw error;
    }

    throw new BackendError("network", "Could not reach the backend.");
  }

  if (response.ok) {
    return response;
  }

  const responseMessage = await getErrorResponseMessage(response);

  if (response.status === 401) {
    throw new BackendError("unauthenticated", "Unauthorized", response.status, responseMessage);
  }

  if (response.status === 403) {
    throw new BackendError("forbidden", "Forbidden", response.status, responseMessage);
  }

  throw new BackendError("http", `HTTP ${response.status}`, response.status, responseMessage);
}

export async function validateSession(token: string, signal?: AbortSignal): Promise<void> {
  await request("/", { signal, token });
}

export async function getDailyNote(token: string, signal?: AbortSignal): Promise<DailyNote> {
  const response = await request("/daily", { signal, token });
  const body = (await response.json()) as DailyNote;

  return {
    contents: decodeBase64(body.contents),
    title: body.title,
  };
}

export interface LinkPreview {
  title: string;
  description: string;
}

export interface PodcastTranscribeResponse {
  podcast: {
    source: string;
    podcast: {
      name: string;
      page_url: string;
      artwork_url: string;
    };
    episode: {
      title: string;
      page_url: string;
      mp3_url: string;
      publication_date: string;
      author: string;
      description_html: string;
    };
  };
  transcription_lambda_id: string;
}

export async function getLinkPreview(
  token: string,
  url: string,
  signal?: AbortSignal,
): Promise<LinkPreview> {
  const response = await request(`/linkpreview?q=${encodeURIComponent(url)}`, { signal, token });
  return (await response.json()) as LinkPreview;
}

export async function podcastTranscribe(
  token: string,
  text: string,
  signal?: AbortSignal,
): Promise<PodcastTranscribeResponse> {
  const response = await request("/podcast-transcribe", {
    body: new URLSearchParams({ text }),
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    method: "POST",
    signal,
    token,
  });

  return (await response.json()) as PodcastTranscribeResponse;
}

export async function saveDailyNote(
  token: string,
  note: DailyNote,
  signal?: AbortSignal,
): Promise<DailyNote> {
  const response = await request("/daily", {
    body: JSON.stringify({
      contents: encodeBase64(note.contents),
      title: note.title,
    }),
    method: "POST",
    signal,
    token,
  });
  const body = (await response.json()) as DailyNote;

  return {
    contents: decodeBase64(body.contents),
    title: body.title,
  };
}
