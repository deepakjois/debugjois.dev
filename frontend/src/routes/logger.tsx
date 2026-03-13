import { useEffect, useMemo, useRef, useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { markdown } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { type Extension } from "@codemirror/state";
import { tags } from "@lezer/highlight";
import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "../auth";
import { BackendError, getDailyNote, getLinkPreview, saveDailyNote } from "../services/backend";
import "./logger.css";

function isUrl(text: string): boolean {
  return /^https?:\/\/\S+$/.test(text.trim());
}

function buildPasteExtension(token: string): Extension {
  return EditorView.domEventHandlers({
    paste(event: ClipboardEvent, view: EditorView) {
      const text = event.clipboardData?.getData("text/plain")?.trim() ?? "";
      if (!isUrl(text)) return false;

      event.preventDefault();

      const { from, to } = view.state.selection.main;
      const selectedText = view.state.sliceDoc(from, to);

      if (selectedText) {
        view.dispatch({
          changes: { from, to, insert: `[${selectedText}](${text})` },
          selection: { anchor: from + selectedText.length + text.length + 4 },
        });
        return true;
      }

      const placeholder = `[Fetching title...](${text})`;
      view.dispatch({
        changes: { from, to, insert: placeholder },
        selection: { anchor: from + placeholder.length },
      });

      void getLinkPreview(token, text)
        .then((preview) => {
          const title = preview.title.trim() || text;
          const link = `[${title}](${text})`;
          const content = view.state.doc.toString();
          const idx = content.indexOf(placeholder);
          if (idx !== -1) {
            view.dispatch({ changes: { from: idx, to: idx + placeholder.length, insert: link } });
          }
        })
        .catch(() => {
          const content = view.state.doc.toString();
          const idx = content.indexOf(placeholder);
          if (idx !== -1) {
            const fallback = `[${text}](${text})`;
            view.dispatch({
              changes: { from: idx, to: idx + placeholder.length, insert: fallback },
            });
          }
        });

      return true;
    },
  });
}

type LoadState = "checking" | "ready" | "unauthenticated" | "forbidden" | "error";
type SaveState = "idle" | "saving" | "saved" | "error";

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

function SaveIcon() {
  return (
    <svg aria-hidden="true" className="logger-editor-save-icon" viewBox="0 0 16 16" fill="none">
      <path d="M3 2.75h8.75l1.5 1.5v9H3z" className="logger-editor-save-stroke" />
      <path d="M5 2.75v3.5h5V2.75" className="logger-editor-save-stroke" />
      <path d="M5.25 10.25h5.5" className="logger-editor-save-stroke" />
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
  const [savedValue, setSavedValue] = useState("");
  const [title, setTitle] = useState("");
  const [loadState, setLoadState] = useState<LoadState>("checking");
  const [loadMessage, setLoadMessage] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const saveResetTimeoutRef = useRef<number | null>(null);

  const isDirty = loadState === "ready" && value !== savedValue;

  const extensions = useMemo(
    () => [
      markdown(),
      syntaxHighlighting(loggerHighlightStyle),
      EditorView.lineWrapping,
      EditorView.contentAttributes.of({
        spellcheck: "true",
        autocorrect: "on",
        autocapitalize: "off",
        lang: "en",
      }),
      buildPasteExtension(token),
    ],
    [token],
  );

  useEffect(() => {
    const controller = new AbortController();

    async function loadInitialData() {
      setLoadState("checking");
      setLoadMessage(null);

      try {
        const note = await getDailyNote(token, controller.signal);
        setTitle(note.title);
        setValue(note.contents);
        setSavedValue(note.contents);
        setSaveState("idle");
        setSaveMessage(null);
        setLoadState("ready");
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;

        if (error instanceof BackendError) {
          if (error.kind === "unauthenticated") {
            setLoadState("unauthenticated");
            return;
          }

          if (error.kind === "forbidden") {
            setLoadState("forbidden");
            setLoadMessage("Unauthorized access. Sign in with an approved account.");
            return;
          }

          if (error.kind === "http" && error.status !== null) {
            setLoadState("error");
            setLoadMessage(`Could not load editor data (HTTP ${error.status}).`);
            return;
          }

          if (error.kind === "network") {
            setLoadState("error");
            setLoadMessage(error.message);
            return;
          }
        }

        setLoadState("error");
        setLoadMessage("Could not reach the backend.");
      }
    }

    void loadInitialData();

    return () => controller.abort();
  }, [token]);

  useEffect(() => {
    return () => {
      if (saveResetTimeoutRef.current !== null) {
        window.clearTimeout(saveResetTimeoutRef.current);
      }
    };
  }, []);

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

  async function handleSave() {
    if (!isDirty || saveState === "saving") {
      return;
    }

    if (saveResetTimeoutRef.current !== null) {
      window.clearTimeout(saveResetTimeoutRef.current);
      saveResetTimeoutRef.current = null;
    }

    setSaveState("saving");
    setSaveMessage(null);

    try {
      const savedNote = await saveDailyNote(token, { contents: value, title });
      setTitle(savedNote.title);
      setValue(savedNote.contents);
      setSavedValue(savedNote.contents);
      setSaveState("saved");
      saveResetTimeoutRef.current = window.setTimeout(() => {
        setSaveState("idle");
        saveResetTimeoutRef.current = null;
      }, 1800);
    } catch (error) {
      if (error instanceof BackendError) {
        if (error.kind === "unauthenticated") {
          setLoadState("unauthenticated");
          return;
        }

        if (error.kind === "forbidden") {
          setSaveState("error");
          setSaveMessage("Unauthorized access. Sign in with an approved account.");
          return;
        }

        if (error.kind === "http" && error.status !== null) {
          setSaveState("error");
          setSaveMessage(`Could not save note (HTTP ${error.status}).`);
          return;
        }

        if (error.kind === "network") {
          setSaveState("error");
          setSaveMessage(error.message);
          return;
        }
      }

      setSaveState("error");
      setSaveMessage("Could not save note.");
    }
  }

  const saveButtonLabel =
    saveState === "saving" ? "Saving..." : saveState === "saved" ? "Saved" : "Save";

  return (
    <div className="logger-editor-page">
      <div className="logger-editor-shell">
        <div className="logger-editor-toolbar">
          <div className="logger-editor-title-group">
            <NoteLogo />
            <h1 className="logger-editor-title">{title}</h1>
          </div>
          <div className="logger-editor-toolbar-actions">
            {saveMessage ? <p className="logger-editor-save-message">{saveMessage}</p> : null}
            <button
              className={`logger-editor-save-button is-${saveState}`}
              disabled={!isDirty || saveState === "saving"}
              onClick={() => void handleSave()}
              type="button"
            >
              <SaveIcon />
              <span>{saveButtonLabel}</span>
            </button>
          </div>
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
