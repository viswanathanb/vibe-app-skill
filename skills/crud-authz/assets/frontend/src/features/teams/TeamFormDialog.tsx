import { zodResolver } from "@hookform/resolvers/zod";
import { useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { applyServerErrors } from "@/lib/forms";
import { useCreateTeam, useUpdateTeam, type Team } from "./api";

const schema = z.object({
  name: z.string().trim().min(1, "Required").max(200, "At most 200 characters"),
  description: z.string().trim().max(2000, "At most 2000 characters"),
});
type FormValues = z.infer<typeof schema>;

export function TeamFormDialog({ team, trigger }: { team?: Team; trigger: ReactNode }) {
  const [open, setOpen] = useState(false);
  const create = useCreateTeam();
  const update = useUpdateTeam(team?.id ?? 0);

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: { name: team?.name ?? "", description: team?.description ?? "" },
  });

  const onSubmit = handleSubmit((values) => {
    const callbacks = {
      onSuccess: () => {
        toast.success(team ? "Team updated" : "Team created");
        reset();
        setOpen(false);
      },
      onError: (err: unknown) => applyServerErrors(err, setError),
    };
    if (team) update.mutate(values, callbacks);
    else create.mutate(values, callbacks);
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{team ? "Edit team" : "New team"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} noValidate className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="team-name">Name</Label>
            <Input id="team-name" autoFocus {...register("name")} />
            {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="team-description">Description</Label>
            <Textarea id="team-description" rows={3} {...register("description")} />
            {errors.description && <p className="text-sm text-destructive">{errors.description.message}</p>}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={create.isPending || update.isPending}>
              {team ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
