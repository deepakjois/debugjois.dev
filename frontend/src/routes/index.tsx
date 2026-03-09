import { Link, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Index,
});

export function Index() {
  return (
    <div>
      <p>Select an app.</p>
      <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
        <Link to="/logger">Open Logger</Link>
        <Link to="/podscriber">Open Podscriber</Link>
      </div>
    </div>
  );
}
