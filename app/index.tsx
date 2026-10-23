import "./styles/tailwind.css";

import { config as fontAwesome } from "@fortawesome/fontawesome-svg-core";
import { ErrorBoundary, Provider as RollbarProvider } from "@rollbar/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createRoot } from "react-dom/client";
import Modal from "react-modal";

import { Config } from "../config";
import { App } from "./components/pages/App";
import { LocalStorageContextProvider } from "./hooks/contexts/use-local-storage-context";
import { SessionProvider } from "./hooks/contexts/use-session";
import { queryClient } from "./utils/query-client";
import { Rollbar } from "./utils/rollbar";

// Font Awesome's stylesheet is in styles/tailwind.css, so the site's styles can
// still override it from inside a CSS layer. Left to add it itself, it would
// sit outside every layer and win over them.
fontAwesome.autoAddCss = false;

Modal.setAppElement("#root");

function run() {
  const app = (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <LocalStorageContextProvider>
          <App />
        </LocalStorageContextProvider>
      </SessionProvider>
    </QueryClientProvider>
  );
  // A local installation has no reporting token. Rollbar's provider rejects
  // that unconfigured instance, so mount it only when reporting is configured.
  createRoot(document.getElementById("root")!).render(
    Config.ROLLBAR_TOKEN ? (
      <RollbarProvider instance={Rollbar}>
        <ErrorBoundary>{app}</ErrorBoundary>
      </RollbarProvider>
    ) : (
      app
    ),
  );
}
if (document.getElementById("root")) {
  run();
} else {
  window.addEventListener("DOMContentLoaded", run);
}
