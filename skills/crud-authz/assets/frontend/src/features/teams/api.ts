import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, toQuery, type ListParams, type Paginated } from "@/lib/api";

export type Team = {
  id: number;
  name: string;
  description: string;
  createdById: number;
  createdAt: string;
  updatedAt: string;
};

export type TeamDetail = Team & { permissions: string[] };
export type TeamInput = { name: string; description: string };

export const teamKeys = {
  all: ["teams"] as const,
  list: (p: ListParams) => [...teamKeys.all, "list", p] as const,
  detail: (id: number) => [...teamKeys.all, "detail", id] as const,
};

export function useTeams(params: ListParams) {
  return useQuery({
    queryKey: teamKeys.list(params),
    queryFn: () => api.get<Paginated<Team>>(`/teams${toQuery(params)}`),
    placeholderData: keepPreviousData,
  });
}

export function useTeam(id: number) {
  return useQuery({
    queryKey: teamKeys.detail(id),
    queryFn: () => api.get<TeamDetail>(`/teams/${id}`),
  });
}

export function useCreateTeam() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: TeamInput) => api.post<Team>("/teams", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: teamKeys.all }),
  });
}

export function useUpdateTeam(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: Partial<TeamInput>) => api.patch<Team>(`/teams/${id}`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: teamKeys.all }),
  });
}

export function useDeleteTeam() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/teams/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: teamKeys.all }),
  });
}
