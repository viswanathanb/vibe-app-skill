import { Navigate, Outlet, useLocation } from "react-router";
import { useMe } from "@/features/auth/api";

/** Route guard: renders child routes only for signed-in users. */
export function RequireAuth() {
  const { data: me, isLoading, isError } = useMe();
  const location = useLocation();

  if (isLoading) {
    return <div className="flex min-h-svh items-center justify-center text-muted-foreground">Loading…</div>;
  }
  if (isError) {
    return <div className="flex min-h-svh items-center justify-center text-destructive">Cannot reach the server.</div>;
  }
  if (!me) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  return <Outlet />;
}
