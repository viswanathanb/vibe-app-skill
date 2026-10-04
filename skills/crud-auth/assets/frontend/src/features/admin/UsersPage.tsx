import { useState } from "react";
import { Navigate } from "react-router";
import { toast } from "sonner";
import { PageHeader } from "@/components/PageHeader";
import { Pagination } from "@/components/Pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useCan, useMe, type Role } from "@/features/auth/api";
import { errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/forms";
import { useUpdateUser, useUsers } from "./api";
import { CreateUserDialog } from "./CreateUserDialog";

const LIMIT = 20;

export function UsersPage() {
  const can = useCan();
  const { data: me } = useMe();
  const [q, setQ] = useState("");
  const [offset, setOffset] = useState(0);
  const { data, isLoading, error } = useUsers({ q, limit: LIMIT, offset });
  const update = useUpdateUser();

  if (!can("users:manage")) return <Navigate to="/" replace />;

  const save = (id: number, input: { role?: Role; disabled?: boolean }) =>
    update.mutate({ id, ...input }, { onError: (err) => toast.error(errorMessage(err)) });

  return (
    <>
      <PageHeader title="Users" description="Global roles (RBAC). Per-item access is managed with Share." actions={<CreateUserDialog />} />
      <Input
        className="mb-4 max-w-sm"
        placeholder="Search email or name…"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setOffset(0);
        }}
      />
      {error && <p className="text-sm text-destructive">{errorMessage(error)}</p>}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Created</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && (
            <TableRow>
              <TableCell colSpan={6} className="text-muted-foreground">
                Loading…
              </TableCell>
            </TableRow>
          )}
          {data?.items.map((u) => (
            <TableRow key={u.id}>
              <TableCell className="font-medium">{u.email}</TableCell>
              <TableCell>{u.name}</TableCell>
              <TableCell>
                <Select value={u.role} onValueChange={(role) => save(u.id, { role: role as Role })}>
                  <SelectTrigger className="w-32" size="sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="admin">Admin</SelectItem>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="viewer">Viewer</SelectItem>
                  </SelectContent>
                </Select>
              </TableCell>
              <TableCell>
                {u.disabled ? <Badge variant="destructive">Disabled</Badge> : <Badge variant="secondary">Active</Badge>}
              </TableCell>
              <TableCell className="text-muted-foreground">{formatDate(u.createdAt)}</TableCell>
              <TableCell className="text-right">
                {u.id !== me?.user.id && (
                  <Button variant="ghost" size="sm" onClick={() => save(u.id, { disabled: !u.disabled })}>
                    {u.disabled ? "Enable" : "Disable"}
                  </Button>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {data && <Pagination total={data.total} limit={LIMIT} offset={offset} onChange={setOffset} />}
    </>
  );
}
