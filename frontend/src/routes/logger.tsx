import { useMemo, useState } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { markdown } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorView } from "@codemirror/view";
import { tags } from "@lezer/highlight";
import { createFileRoute } from "@tanstack/react-router";
import "./logger.css";

const SAMPLE_MARKDOWN = `# Meeting Notes

Welcome to the source editor.

## Formatting

This paragraph includes **bold text**, *italic text*, and ==highlighted ideas==.

### Lists

- Capture the rough draft
- Refine the structure
- Keep the final version readable

1. Open the note
2. Edit in source mode
3. Save when ready

### Tasks

- [x] Sketch the layout
- [ ] Wire the backend
- [ ] Add custom commands

### Quote

> Write notes like you are thinking out loud, then shape them later.

### Link

Review the [Markdown Guide](https://www.markdownguide.org/basic-syntax/) for syntax details.

### Code

~~~ts
export function summarize(note: string) {
  return note.trim();
}
~~~

### Table

| Section | Purpose |
| --- | --- |
| Draft | Quick capture |
| Review | Clean up wording |
| Publish | Share the final version |
`;

const loggerHighlightStyle = HighlightStyle.define([
  { tag: tags.heading1, color: "#e0b36f", fontWeight: "700", fontSize: "1.46em" },
  { tag: tags.heading2, color: "#ddb06b", fontWeight: "700", fontSize: "1.24em" },
  { tag: tags.heading3, color: "#d8ab68", fontWeight: "600", fontSize: "1.1em" },
  { tag: [tags.heading4, tags.heading5, tags.heading6], color: "#cf9f5f", fontWeight: "600" },
  { tag: tags.strong, color: "#f4d3c2", fontWeight: "700" },
  { tag: tags.emphasis, color: "#d6cbff", fontStyle: "italic" },
  { tag: tags.link, color: "#91b4ff", textDecoration: "underline" },
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
    <svg
      aria-hidden="true"
      className="logger-editor-logo"
      viewBox="0 0 24 24"
      fill="none"
    >
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
      <path d="M10.5 7.25h1.25c1.1 0 2 .9 2 2v.5c0 1.1-.9 2-2 2H6.75" className="logger-editor-toggle-stroke" />
      <path d="m8.5 10.25-1.75 1.5 1.75 1.5" className="logger-editor-toggle-stroke" />
    </svg>
  );
}

export const Route = createFileRoute("/logger")({
  component: Logger,
});

export function Logger() {
  const [value, setValue] = useState(SAMPLE_MARKDOWN);
  const [wrapEnabled, setWrapEnabled] = useState(true);

  const extensions = useMemo(() => {
    const baseExtensions = [markdown(), syntaxHighlighting(loggerHighlightStyle)];

    if (wrapEnabled) {
      return [...baseExtensions, EditorView.lineWrapping];
    }

    return baseExtensions;
  }, [wrapEnabled]);

  return (
    <div className="logger-editor-page">
      <div className="logger-editor-shell">
        <div className="logger-editor-toolbar">
          <div className="logger-editor-title-group">
            <NoteLogo />
            <h1 className="logger-editor-title">Welcome.md</h1>
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
            theme={oneDark}
            value={value}
          />
        </div>
      </div>
    </div>
  );
}
