import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, toQuery, type ListParams, type Paginated } from "@/lib/api";
import type { Role, User } from "@/features/auth/api";

export type CreateUserInput = { email: string; name: string; password: string; role: Role };
export type UpdateUserInput = Partial<{ name: string; role: Role; disabled: boolean }>;

const keys = {
  all: ["users"] as const,
  list: (p: ListParams) => [...keys.all, "list", p] as const,
};

export function useUsers(params: ListParams) {
  return useQuery({
    queryKey: keys.list(params),
    queryFn: () => api.get<Paginated<User>>(`/users${toQuery(params)}`),
    placeholderData: keepPreviousData,
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUserInput) => api.post<User>("/users", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useUpdateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateUserInput & { id: number }) => api.patch<User>(`/users/${id}`, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.all }),
  });
}
