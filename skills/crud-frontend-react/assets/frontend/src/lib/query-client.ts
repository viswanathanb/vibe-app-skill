import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api";

/** Query key of the current session (see features/auth/api.ts). */
export const meQueryKey = ["auth", "me"] as const;

function onError(err: unknown) {
  // Session expired or revoked: clear the user so <RequireAuth> redirects to /login.
  if (err instanceof ApiError && err.status === 401) queryClient.setQueryData(meQueryKey, null);
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError }),
  mutationCache: new MutationCache({ onError }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: (count, err) => !(err instanceof ApiError && err.status < 500) && count < 2,
    },
  },
});
