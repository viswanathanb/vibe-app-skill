import { Plus } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { PageHeader } from "@/components/PageHeader";
import { Pagination } from "@/components/Pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useCan } from "@/features/auth/api";
import { errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/forms";
import { useProjects } from "./api";
import { ProjectFormDialog } from "./ProjectFormDialog";

const LIMIT = 20;
const ALL = "all";

export function ProjectsPage() {
  const can = useCan();
  const [q, setQ] = useState("");
  const [status, setStatus] = useState(ALL);
  const [offset, setOffset] = useState(0);
  const { data, isLoading, error } = useProjects({
    q,
    status: status === ALL ? undefined : status,
    sort: "-updatedAt",
    limit: LIMIT,
    offset,
  });

  return (
    <>
      <PageHeader
        title="Projects"
        description="Projects you own or that were shared with you."
        actions={
          can("project:create") && (
            <ProjectFormDialog
              trigger={
                <Button>
                  <Plus />
                  New project
                </Button>
              }
            />
          )
        }
      />

      <div className="mb-4 flex flex-wrap gap-2">
        <Input
          className="max-w-sm"
          placeholder="Search…"
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setOffset(0);
          }}
        />
        <Select
          value={status}
          onValueChange={(v) => {
            setStatus(v);
            setOffset(0);
          }}
        >
          <SelectTrigger className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="archived">Archived</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {error && <p className="text-sm text-destructive">{errorMessage(error)}</p>}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Updated</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && (
            <TableRow>
              <TableCell colSpan={3} className="text-muted-foreground">
                Loading…
              </TableCell>
            </TableRow>
          )}
          {data?.items.length === 0 && (
            <TableRow>
              <TableCell colSpan={3} className="text-muted-foreground">
                No projects yet.
              </TableCell>
            </TableRow>
          )}
          {data?.items.map((p) => (
            <TableRow key={p.id}>
              <TableCell className="font-medium">
                <Link to={`/projects/${p.id}`} className="hover:underline">
                  {p.name}
                </Link>
              </TableCell>
              <TableCell>
                <Badge variant={p.status === "active" ? "default" : "secondary"} className="capitalize">
                  {p.status}
                </Badge>
              </TableCell>
              <TableCell className="text-muted-foreground">{formatDate(p.updatedAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {data && <Pagination total={data.total} limit={LIMIT} offset={offset} onChange={setOffset} />}
    </>
  );
}
