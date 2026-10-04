import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, toQuery, type ListParams, type Paginated } from "@/lib/api";

// Types mirror backend/internal/project/model.go (JSON is camelCase).
export type ProjectStatus = "active" | "archived";

export type Project = {
  id: number;
  name: string;
  description: string;
  status: ProjectStatus;
  createdById: number;
  createdAt: string;
  updatedAt: string;
};

/** GET /projects/:id also returns the caller's permissions (view/edit/delete/manage). */
export type ProjectDetail = Project & { permissions: string[] };

export type ProjectInput = { name: string; description: string };
export type ProjectUpdate = Partial<ProjectInput & { status: ProjectStatus }>;

export const projectKeys = {
  all: ["projects"] as const,
  list: (p: ListParams) => [...projectKeys.all, "list", p] as const,
  detail: (id: number) => [...projectKeys.all, "detail", id] as const,
};

export function useProjects(params: ListParams) {
  return useQuery({
    queryKey: projectKeys.list(params),
    queryFn: () => api.get<Paginated<Project>>(`/projects${toQuery(params)}`),
    placeholderData: keepPreviousData,
  });
}

export function useProject(id: number) {
  return useQuery({
    queryKey: projectKeys.detail(id),
    queryFn: () => api.get<ProjectDetail>(`/projects/${id}`),
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ProjectInput) => api.post<Project>("/projects", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.all }),
  });
}

export function useUpdateProject(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ProjectUpdate) => api.patch<Project>(`/projects/${id}`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.all }),
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/projects/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.all }),
  });
}
