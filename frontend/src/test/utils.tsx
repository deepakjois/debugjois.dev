import React from "react";
import { render, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  Outlet,
  RouterProvider,
} from "@tanstack/react-router";

// ─── QueryClient factory ──────────────────────────────────────────────────────
//
// retry: false    — no 3x retry + backoff on failed queries (would time out tests)
// gcTime: 0       — evict cache entries immediately on unmount (no cross-test bleed)
// staleTime: 0    — always refetch; never serve stale data between tests

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

// ─── Test router ──────────────────────────────────────────────────────────────
//
// Builds a minimal route tree without importing routeTree.gen.ts.
// The generated file depends on the @tanstack/router-plugin Vite plugin, which
// is excluded from vitest.config.ts. Tests always create routes manually.
//
// defaultPendingMinMs: 0 removes the router's built-in 500 ms flicker-prevention
// delay so pending states are observable immediately in tests.

export interface CreateTestRouterOptions {
  /** Starting URL. Default '/' */
  initialEntry?: string;
  /** Component rendered at pathPattern. Defaults to a sentinel <div>. */
  routeComponent?: React.ComponentType;
  /** Path pattern for routeComponent. Default '/' */
  pathPattern?: string;
  /**
   * Root component.
   * - Pass RootComponent from __root.tsx to test the auth gate.
   * - Pass the result of makePreAuthenticatedRoot() to bypass it.
   */
  rootComponent: React.ComponentType;
  /** Provide a pre-created QueryClient to inspect cache state after render. */
  queryClient?: QueryClient;
}

export function createTestRouter(options: CreateTestRouterOptions) {
  const {
    initialEntry = "/",
    routeComponent,
    pathPattern = "/",
    rootComponent,
    queryClient = createTestQueryClient(),
  } = options;

  const rootRoute = createRootRoute({ component: rootComponent });

  const Sentinel = () =>
    React.createElement("div", { "data-testid": "route-content" }, "Route rendered");

  const testRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: pathPattern,
    component: routeComponent ?? Sentinel,
  });

  const router = createRouter({
    routeTree: rootRoute.addChildren([testRoute]),
    history: createMemoryHistory({ initialEntries: [initialEntry] }),
    defaultPendingMinMs: 0,
  });

  return { router, queryClient };
}

// ─── Render helper ────────────────────────────────────────────────────────────
//
// Wraps the router in a QueryClientProvider and waits for the initial navigation
// to complete (router.load()) before returning. Without this await, assertions
// can run before route components have mounted.

export async function renderWithRouter(options: CreateTestRouterOptions) {
  const { router, queryClient } = createTestRouter(options);

  const result = render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );

  // Wrap in act() to flush TanStack Router's internal Transitioner state updates,
  // which otherwise produce "not wrapped in act()" warnings in every test.
  await act(() => router.load());

  return { router, queryClient, ...result };
}

// ─── Pre-authenticated root component factory ─────────────────────────────────
//
// Returns a root component that unconditionally provides a real AuthContext value
// with a fake token. Use this when testing inner routes that call useAuth() but
// the test does not care about the login flow.
//
// Requires AuthContext to be exported from __root.tsx so the same context object
// reference is used here and inside the route components under test.

export function makePreAuthenticatedRoot(AuthContext: React.Context<{ token: string } | null>) {
  return function PreAuthenticatedRoot() {
    return (
      <AuthContext.Provider value={{ token: "fake-test-token" }}>
        <div data-testid="pre-auth-root">
          <Outlet />
        </div>
      </AuthContext.Provider>
    );
  };
}
