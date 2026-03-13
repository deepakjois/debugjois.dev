export interface DailyNote {
  title: string;
  contents: string;
}

export type BackendErrorKind = "unauthenticated" | "forbidden" | "http" | "network";

export class BackendError extends Error {
  kind: BackendErrorKind;
  status: number | null;

  constructor(kind: BackendErrorKind, message: string, status: number | null = null) {
    super(message);
    this.name = "BackendError";
    this.kind = kind;
    this.status = status;
  }
}

const API_URL = import.meta.env.VITE_SITE_BACKEND_URL;

interface RequestOptions {
  body?: BodyInit;
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

async function request(path: string, options: RequestOptions): Promise<Response> {
  const { body, method = "GET", signal, token } = options;
  const headers: HeadersInit = body
    ? { "Content-Type": "application/json", ...authHeaders(token) }
    : authHeaders(token);

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

  if (response.status === 401) {
    throw new BackendError("unauthenticated", "Unauthorized", response.status);
  }

  if (response.status === 403) {
    throw new BackendError("forbidden", "Forbidden", response.status);
  }

  throw new BackendError("http", `HTTP ${response.status}`, response.status);
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

export async function getLinkPreview(
  token: string,
  url: string,
  signal?: AbortSignal,
): Promise<LinkPreview> {
  const response = await request(`/linkpreview?q=${encodeURIComponent(url)}`, { signal, token });
  return (await response.json()) as LinkPreview;
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
