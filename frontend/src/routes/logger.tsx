import { useEffect, useMemo, useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { markdown } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags } from "@lezer/highlight";
import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "../auth";
import "./logger.css";

const API_URL = import.meta.env.VITE_SITE_BACKEND_URL;
type LoadState = "checking" | "ready" | "unauthenticated" | "forbidden" | "error";
type DailyNote = { title: string; contents: string };

const loggerHighlightStyle = HighlightStyle.define([
  { tag: tags.heading1, color: "#e0b36f", fontWeight: "700", fontSize: "1.46em" },
  { tag: tags.heading2, color: "#ddb06b", fontWeight: "700", fontSize: "1.24em" },
  { tag: tags.heading3, color: "#d8ab68", fontWeight: "600", fontSize: "1.1em" },
  { tag: [tags.heading4, tags.heading5, tags.heading6], color: "#cf9f5f", fontWeight: "600" },
  { tag: tags.strong, color: "#f4d3c2", fontWeight: "700" },
  { tag: tags.emphasis, color: "#d6cbff", fontStyle: "italic" },
  { tag: tags.link, color: "#91b4ff", textDecoration: "underline" },
  { tag: tags.labelName, color: "#91b4ff", textDecoration: "underline" },
  { tag: tags.url, color: "#84aefc" },
  { tag: tags.monospace, color: "#e8a7ab", backgroundColor: "rgba(255, 255, 255, 0.045)" },
  { tag: tags.quote, color: "#95d0d8", fontStyle: "italic" },
  { tag: tags.list, color: "#efaaa3" },
  { tag: tags.processingInstruction, color: "#c7a7f9" },
  { tag: tags.contentSeparator, color: "#4c4767" },
  { tag: tags.meta, color: "#6d678f" },
  { tag: tags.string, color: "#f0bf81" },
  { tag: tags.keyword, color: "#c7a7f9" },
  { tag: tags.punctuation, color: "#918cb0" },
]);

function NoteLogo() {
  return (
    <svg aria-hidden="true" className="logger-editor-logo" viewBox="0 0 24 24" fill="none">
      <path
        d="M12 2.75 4 7.25v9.5l8 4.5 8-4.5v-9.5l-8-4.5Z"
        className="logger-editor-logo-outline"
      />
      <path d="M12 2.75v18.5" className="logger-editor-logo-outline" />
      <path d="m4 7.25 8 4.5 8-4.5" className="logger-editor-logo-outline" />
    </svg>
  );
}

function WrapIcon() {
  return (
    <svg aria-hidden="true" className="logger-editor-toggle-icon" viewBox="0 0 16 16" fill="none">
      <path d="M2.25 3.5h11.5" className="logger-editor-toggle-stroke" />
      <path d="M2.25 7.25h8.25" className="logger-editor-toggle-stroke" />
      <path
        d="M10.5 7.25h1.25c1.1 0 2 .9 2 2v.5c0 1.1-.9 2-2 2H6.75"
        className="logger-editor-toggle-stroke"
      />
      <path d="m8.5 10.25-1.75 1.5 1.75 1.5" className="logger-editor-toggle-stroke" />
    </svg>
  );
}

type StatusScreenProps = {
  eyebrow: string;
  title: string;
  message?: string | null;
  tone?: "default" | "error";
};

function StatusScreen({ eyebrow, title, message, tone = "default" }: StatusScreenProps) {
  return (
    <div className="logger-editor-status">
      <div className="logger-editor-status-card">
        <p className="logger-editor-status-label">{eyebrow}</p>
        <h1 className="logger-editor-status-title">{title}</h1>
        {message ? (
          <p className={`logger-editor-status-message${tone === "error" ? " is-error" : ""}`}>
            {message}
          </p>
        ) : null}
      </div>
    </div>
  );
}

export const Route = createFileRoute("/logger")({
  component: Logger,
});

export function Logger() {
  const { token, signOut } = useAuth();
  const [value, setValue] = useState("");
  const [title, setTitle] = useState("");
  const [wrapEnabled, setWrapEnabled] = useState(true);
  const [loadState, setLoadState] = useState<LoadState>("checking");
  const [loadMessage, setLoadMessage] = useState<string | null>(null);

  const extensions = useMemo(() => {
    const baseExtensions = [markdown(), syntaxHighlighting(loggerHighlightStyle)];

    if (wrapEnabled) {
      return [...baseExtensions, EditorView.lineWrapping];
    }

    return baseExtensions;
  }, [wrapEnabled]);

  useEffect(() => {
    const controller = new AbortController();

    async function loadInitialData() {
      setLoadState("checking");
      setLoadMessage(null);

      const headers: HeadersInit =
        import.meta.env.VITE_AUTH_BYPASS === "true" ? {} : { Authorization: `Bearer ${token}` };

      try {
        const res = await fetch(`${API_URL}/daily`, {
          signal: controller.signal,
          headers,
        });

        if (res.ok) {
          const body: DailyNote = await res.json();
          setTitle(body.title);
          setValue(
            body.contents
              ? new TextDecoder().decode(Uint8Array.from(atob(body.contents), (c) => c.charCodeAt(0)))
              : "",
          );
          setLoadState("ready");
          return;
        }

        if (res.status === 401) {
          setLoadState("unauthenticated");
          return;
        }

        if (res.status === 403) {
          setLoadState("forbidden");
          setLoadMessage("Unauthorized access. Sign in with an approved account.");
          return;
        }

        setLoadState("error");
        setLoadMessage(`Could not load editor data (HTTP ${res.status}).`);
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setLoadState("error");
        setLoadMessage("Could not reach the backend.");
      }
    }

    void loadInitialData();

    return () => controller.abort();
  }, [token]);

  useEffect(() => {
    if (loadState === "unauthenticated") {
      signOut();
    }
  }, [loadState, signOut]);

  if (loadState === "checking") {
    return <StatusScreen eyebrow="Logger" title="Loading editor..." />;
  }

  if (loadState === "unauthenticated") {
    return (
      <StatusScreen
        eyebrow="Logger"
        title="Sign in to continue."
        message="Use an approved Google account to open the editor."
      />
    );
  }

  if (loadState === "forbidden" || loadState === "error") {
    return (
      <StatusScreen
        eyebrow="Logger"
        title="Could not open the editor."
        message={loadMessage}
        tone="error"
      />
    );
  }

  return (
    <div className="logger-editor-page">
      <div className="logger-editor-shell">
        <div className="logger-editor-toolbar">
          <div className="logger-editor-title-group">
            <NoteLogo />
            <h1 className="logger-editor-title">{title}</h1>
          </div>
          <label className="logger-editor-toggle" title="Toggle word wrap">
            <input
              aria-label="Wrap"
              checked={wrapEnabled}
              onChange={(event) => setWrapEnabled(event.target.checked)}
              type="checkbox"
            />
            <WrapIcon />
          </label>
        </div>

        <div className="logger-editor-pane">
          <CodeMirror
            basicSetup={{
              foldGutter: false,
              highlightActiveLineGutter: false,
              lineNumbers: false,
            }}
            className="logger-editor-codemirror"
            extensions={extensions}
            height="100%"
            onChange={setValue}
            value={value}
          />
        </div>
      </div>
    </div>
  );
}
