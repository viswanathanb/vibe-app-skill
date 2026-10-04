import { X } from "lucide-react";
import { useState, type FormEvent } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { errorMessage } from "@/lib/api";
import { useAccessList, useGrant, useRevoke } from "./api";

/**
 * Lists who has access to an object and lets managers grant/revoke relations.
 * Works for every type in the backend authz schema (generic /api/authz/:type/:id API).
 * Enter an email to add a user, or a subject like "team:3#member" to add a whole team.
 */
export function AccessPanel({ objectType, objectId }: { objectType: string; objectId: number | string }) {
  const { data, isLoading, error } = useAccessList(objectType, objectId);
  const grant = useGrant(objectType, objectId);
  const revoke = useRevoke(objectType, objectId);
  const [who, setWho] = useState("");
  const [relation, setRelation] = useState("");

  if (isLoading) return <p className="text-sm text-muted-foreground">Loading…</p>;
  if (error) return <p className="text-sm text-destructive">{errorMessage(error)}</p>;
  if (!data) return null;

  const selected = relation || data.relations[0] || "";

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const value = who.trim();
    if (!value) return;
    const input = value.includes("@") ? { relation: selected, email: value } : { relation: selected, subject: value };
    grant.mutate(input, {
      onSuccess: () => {
        setWho("");
        toast.success("Access granted");
      },
      onError: (err) => toast.error(errorMessage(err)),
    });
  };

  return (
    <div className="grid gap-4">
      <form onSubmit={onSubmit} className="flex flex-wrap gap-2">
        <Input
          className="min-w-48 flex-1"
          placeholder="email@example.com or team:3#member"
          value={who}
          onChange={(e) => setWho(e.target.value)}
        />
        <Select value={selected} onValueChange={setRelation}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {data.relations.map((r) => (
              <SelectItem key={r} value={r} className="capitalize">
                {r}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="submit" disabled={grant.isPending || !who.trim()}>
          Add
        </Button>
      </form>

      <ul className="divide-y rounded-md border">
        {data.tuples.length === 0 && <li className="p-3 text-sm text-muted-foreground">Nobody has access yet.</li>}
        {data.tuples.map((t) => (
          <li key={`${t.relation}|${t.subject}`} className="flex items-center gap-3 p-3 text-sm">
            <span className="flex-1 truncate">{t.label}</span>
            <Badge variant="secondary" className="capitalize">
              {t.relation}
            </Badge>
            <Button
              variant="ghost"
              size="icon"
              aria-label={`Remove ${t.label}`}
              disabled={revoke.isPending}
              onClick={() =>
                revoke.mutate({ relation: t.relation, subject: t.subject }, { onError: (err) => toast.error(errorMessage(err)) })
              }
            >
              <X />
            </Button>
          </li>
        ))}
      </ul>
    </div>
  );
}
