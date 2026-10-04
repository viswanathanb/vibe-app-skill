import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, api } from "@/lib/api";
import { meQueryKey } from "@/lib/query-client";

export type Role = "admin" | "member" | "viewer";

export type User = {
  id: number;
  email: string;
  name: string;
  role: Role;
  disabled: boolean;
  createdAt: string;
  updatedAt: string;
};

/** Current session: the user plus the RBAC actions their role grants ("*" = all). */
export type Me = { user: User; actions: string[] };

export type LoginInput = { email: string; password: string };
export type SignupInput = LoginInput & { name: string };

/** Returns the session, or null when signed out. */
export function useMe() {
  return useQuery({
    queryKey: meQueryKey,
    queryFn: async (): Promise<Me | null> => {
      try {
        return await api.get<Me>("/auth/me");
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null;
        throw err;
      }
    },
    staleTime: 5 * 60_000,
  });
}

/** RBAC helper for the UI: can("project:create"). The backend enforces the same rule. */
export function useCan() {
  const { data } = useMe();
  return (action: string) => !!data && (data.actions.includes("*") || data.actions.includes(action));
}

export function useAuthConfig() {
  return useQuery({
    queryKey: ["auth", "config"],
    queryFn: () => api.get<{ signupEnabled: boolean }>("/auth/config"),
    staleTime: Infinity,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: LoginInput) => api.post<Me>("/auth/login", input),
    onSuccess: (me) => qc.setQueryData(meQueryKey, me),
  });
}

export function useSignup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SignupInput) => api.post<Me>("/auth/signup", input),
    onSuccess: (me) => qc.setQueryData(meQueryKey, me),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<void>("/auth/logout"),
    onSuccess: () => {
      qc.clear();
      qc.setQueryData(meQueryKey, null);
    },
  });
}
