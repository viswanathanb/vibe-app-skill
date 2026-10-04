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
import { useCreateProject, useUpdateProject, type Project } from "./api";

// Keep these rules in sync with the `binding` tags in backend/internal/project/model.go.
const schema = z.object({
  name: z.string().trim().min(1, "Required").max(200, "At most 200 characters"),
  description: z.string().trim().max(2000, "At most 2000 characters"),
});
type FormValues = z.infer<typeof schema>;

/** Create (no `project`) or edit (with `project`) dialog. */
export function ProjectFormDialog({ project, trigger }: { project?: Project; trigger: ReactNode }) {
  const [open, setOpen] = useState(false);
  const create = useCreateProject();
  const update = useUpdateProject(project?.id ?? 0);
  const pending = create.isPending || update.isPending;

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: { name: project?.name ?? "", description: project?.description ?? "" },
  });

  const onSubmit = handleSubmit((values) => {
    const callbacks = {
      onSuccess: () => {
        toast.success(project ? "Project updated" : "Project created");
        reset();
        setOpen(false);
      },
      onError: (err: unknown) => applyServerErrors(err, setError),
    };
    if (project) update.mutate(values, callbacks);
    else create.mutate(values, callbacks);
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{project ? "Edit project" : "New project"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} noValidate className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="project-name">Name</Label>
            <Input id="project-name" autoFocus {...register("name")} />
            {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="project-description">Description</Label>
            <Textarea id="project-description" rows={4} {...register("description")} />
            {errors.description && <p className="text-sm text-destructive">{errors.description.message}</p>}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={pending}>
              {project ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
