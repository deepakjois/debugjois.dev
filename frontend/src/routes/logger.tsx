import { Link, createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth";

const API_URL = import.meta.env.VITE_SITE_BACKEND_URL;

export const Route = createFileRoute("/logger")({
  component: Logger,
});

export function Logger() {
  const { token, signOut } = useAuth();

  const { data, isFetching, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const res = await fetch(`${API_URL}/health`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return res.json() as Promise<{ status: string; email?: string }>;
    },
    enabled: false,
  });

  return (
    <div>
      <header style={{ marginBottom: "1rem" }}>
        <h1>Logger</h1>
        <p>Check the backend health endpoint for the signed-in user.</p>
        <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
          <Link to="/">Back to apps</Link>
          <button onClick={signOut}>Sign out</button>
        </div>
      </header>
      <button onClick={() => refetch()} disabled={isFetching}>
        {isFetching ? "Checking..." : "Check Health"}
      </button>
      {data && (
        <p>
          Backend status: {data.status}
          {data.email && ` (user: ${data.email})`}
        </p>
      )}
      {error && <p style={{ color: "red" }}>Error: {error.message}</p>}
    </div>
  );
}
