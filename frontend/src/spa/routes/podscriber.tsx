import { createFileRoute } from "@tanstack/react-router";
import { PodscriberPage } from "../pages/PodscriberPage";

export const Route = createFileRoute("/podscriber")({
  component: PodscriberPage,
});
