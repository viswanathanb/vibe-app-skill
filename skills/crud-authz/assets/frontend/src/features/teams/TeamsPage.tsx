import { Plus } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { PageHeader } from "@/components/PageHeader";
import { Pagination } from "@/components/Pagination";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useCan } from "@/features/auth/api";
import { errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/forms";
import { useTeams } from "./api";
import { TeamFormDialog } from "./TeamFormDialog";

const LIMIT = 20;

export function TeamsPage() {
  const can = useCan();
  const [q, setQ] = useState("");
  const [offset, setOffset] = useState(0);
  const { data, isLoading, error } = useTeams({ q, limit: LIMIT, offset });

  return (
    <>
      <PageHeader
        title="Teams"
        description="Share resources with a whole team at once."
        actions={
          can("team:create") && (
            <TeamFormDialog
              trigger={
                <Button>
                  <Plus />
                  New team
                </Button>
              }
            />
          )
        }
      />
      <Input
        className="mb-4 max-w-sm"
        placeholder="Search…"
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
            <TableHead>Name</TableHead>
            <TableHead>Description</TableHead>
            <TableHead>Created</TableHead>
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
                No teams yet.
              </TableCell>
            </TableRow>
          )}
          {data?.items.map((t) => (
            <TableRow key={t.id}>
              <TableCell className="font-medium">
                <Link to={`/teams/${t.id}`} className="hover:underline">
                  {t.name}
                </Link>
              </TableCell>
              <TableCell className="max-w-md truncate text-muted-foreground">{t.description}</TableCell>
              <TableCell className="text-muted-foreground">{formatDate(t.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {data && <Pagination total={data.total} limit={LIMIT} offset={offset} onChange={setOffset} />}
    </>
  );
}
