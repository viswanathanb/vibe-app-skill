import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, toQuery } from "@/lib/api";

export type AccessEntry = { relation: string; subject: string; label: string };
export type AccessList = { relations: string[]; tuples: AccessEntry[] };

/** Grant to a user by email, or to a subject such as "team:3#member". */
export type GrantInput = { relation: string; email?: string; subject?: string };

const base = (type: string, id: number | string) => `/authz/${type}/${id}`;

export const accessKeys = {
  list: (type: string, id: number | string) => ["authz", type, String(id), "tuples"] as const,
  permissions: (type: string, id: number | string) => ["authz", type, String(id), "permissions"] as const,
};

/** The caller's permissions on an object, e.g. ["view", "edit"] (ReBAC). */
export function usePermissions(type: string, id: number | string) {
  return useQuery({
    queryKey: accessKeys.permissions(type, id),
    queryFn: () => api.get<{ permissions: string[] }>(`${base(type, id)}/permissions`),
    select: (d) => d.permissions,
  });
}

export function useAccessList(type: string, id: number | string, enabled = true) {
  return useQuery({
    queryKey: accessKeys.list(type, id),
    queryFn: () => api.get<AccessList>(`${base(type, id)}/tuples`),
    enabled,
  });
}

export function useGrant(type: string, id: number | string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: GrantInput) => api.post<void>(`${base(type, id)}/tuples`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["authz", type, String(id)] }),
  });
}

export function useRevoke(type: string, id: number | string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (e: { relation: string; subject: string }) =>
      api.delete(`${base(type, id)}/tuples${toQuery({ relation: e.relation, subject: e.subject })}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["authz", type, String(id)] }),
  });
}
