import { Link, createFileRoute } from "@tanstack/react-router";
import { useAuth } from "../auth";

export const Route = createFileRoute("/podscriber")({
  component: Podscriber,
});

export function Podscriber() {
  const { signOut } = useAuth();

  return (
    <div>
      <h1>Podscriber</h1>
      <p>Placeholder for the next app.</p>
      <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
        <Link to="/">Back to apps</Link>
        <button onClick={signOut}>Sign out</button>
      </div>
    </div>
  );
}
