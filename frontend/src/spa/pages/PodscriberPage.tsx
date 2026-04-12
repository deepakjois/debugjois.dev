import { useEffect, useId, useState, type FormEvent } from "react";
import { Link } from "@tanstack/react-router";
import { useAuth } from "../auth";
import {
  BackendError,
  podcastTranscribe,
  type PodcastTranscribeResponse,
} from "../services/backend";
import "../routes/podscriber.css";

type SubmitState = "idle" | "submitting" | "success" | "error";

export function PodscriberPage() {
  const { token, signOut } = useAuth();
  const payloadId = useId();
  const [payload, setPayload] = useState("");
  const [submitState, setSubmitState] = useState<SubmitState>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [response, setResponse] = useState<PodcastTranscribeResponse | null>(null);

  useEffect(() => {
    if (submitState === "error") {
      setResponse(null);
    }
  }, [submitState]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (submitState === "submitting" || submitState === "success") {
      return;
    }

    const trimmedPayload = payload.trim();
    if (!trimmedPayload) {
      setSubmitState("error");
      setErrorMessage("Paste the Podcast Addict payload before submitting.");
      return;
    }

    setSubmitState("submitting");
    setErrorMessage(null);

    try {
      const result = await podcastTranscribe(token, trimmedPayload);
      setResponse(result);
      setSubmitState("success");
    } catch (error) {
      if (error instanceof BackendError) {
        if (error.kind === "unauthenticated") {
          signOut();
          return;
        }

        if (error.kind === "forbidden") {
          setSubmitState("error");
          setErrorMessage("Unauthorized access. Sign in with an approved account.");
          return;
        }

        if (error.kind === "http") {
          setSubmitState("error");
          setErrorMessage(
            error.responseMessage ?? `The backend rejected the request (HTTP ${error.status}).`,
          );
          return;
        }

        if (error.kind === "network") {
          setSubmitState("error");
          setErrorMessage(error.message);
          return;
        }
      }

      setSubmitState("error");
      setErrorMessage("Could not submit the Podcast Addict payload.");
    }
  }

  const isSubmitting = submitState === "submitting";
  const isSuccess = submitState === "success";
  const isDisabled = isSubmitting || isSuccess;
  const submitLabel = isSubmitting
    ? "Submitting..."
    : isSuccess
      ? "Submitted"
      : "Start Transcription";

  return (
    <div className="podscriber-page">
      <div className="podscriber-shell">
        <div className="podscriber-hero">
          <div className="podscriber-hero-copy">
            <p className="podscriber-eyebrow">Podscriber</p>
            <h1 className="podscriber-title">
              Turn a shared podcast episode into a transcription job.
            </h1>
            <p className="podscriber-description">
              Paste the full Podcast Addict share payload below. Once the backend accepts it, the
              parsed response appears here and the payload is locked in for reference.
            </p>
          </div>
          <div className="podscriber-nav">
            <Link className="podscriber-nav-link" to="/">
              Back to apps
            </Link>
            <button className="podscriber-signout" onClick={signOut} type="button">
              Sign out
            </button>
          </div>
        </div>

        <form className="podscriber-form-card" onSubmit={(event) => void handleSubmit(event)}>
          <label className="podscriber-field-label" htmlFor={payloadId}>
            PodcastAddict Payload
          </label>
          <textarea
            className="podscriber-textarea"
            disabled={isDisabled}
            id={payloadId}
            onChange={(event) => setPayload(event.target.value)}
            placeholder="Paste the Podcast Addict share text here..."
            rows={12}
            spellCheck={false}
            value={payload}
          />
          <div className="podscriber-actions">
            <button
              className={`podscriber-submit-button is-${submitState}`}
              disabled={isDisabled}
              type="submit"
            >
              {submitLabel}
            </button>
          </div>
        </form>

        {errorMessage ? (
          <section aria-live="polite" className="podscriber-error-card">
            <p className="podscriber-card-label">Request Error</p>
            <p className="podscriber-error-message">{errorMessage}</p>
          </section>
        ) : null}

        {response ? (
          <section aria-live="polite" className="podscriber-response-card">
            <div className="podscriber-response-header">
              <div>
                <p className="podscriber-card-label">Accepted</p>
                <h2 className="podscriber-response-title">Transcription request queued</h2>
              </div>
              <p className="podscriber-response-meta">{response.transcription_lambda_id}</p>
            </div>
            <pre className="podscriber-response-pre">
              <code>{JSON.stringify(response, null, 2)}</code>
            </pre>
          </section>
        ) : null}
      </div>
    </div>
  );
}
