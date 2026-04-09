import { Link } from "@tanstack/react-router";

export function IndexPage() {
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
