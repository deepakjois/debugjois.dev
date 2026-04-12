import { createFileRoute } from "@tanstack/react-router";
import { LoggerPage } from "../pages/LoggerPage";

export const Route = createFileRoute("/logger")({
  component: LoggerPage,
});
