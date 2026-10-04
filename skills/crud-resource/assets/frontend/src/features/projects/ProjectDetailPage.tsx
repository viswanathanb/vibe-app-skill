import { Pencil, Trash2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { PageHeader } from "@/components/PageHeader";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ShareDialog } from "@/features/sharing/ShareDialog";
import { errorMessage } from "@/lib/api";
import { formatDate } from "@/lib/forms";
import { useDeleteProject, useProject, useUpdateProject } from "./api";
import { ProjectFormDialog } from "./ProjectFormDialog";

export function ProjectDetailPage() {
  const id = Number(useParams().id);
  const navigate = useNavigate();
  const { data: project, isLoading, error } = useProject(id);
  const update = useUpdateProject(id);
  const remove = useDeleteProject();

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>;
  if (error || !project) return <p className="text-destructive">{errorMessage(error)}</p>;

  // ReBAC: the backend returns what this user may do with this specific project.
  const can = (perm: string) => project.permissions.includes(perm);
  const archived = project.status === "archived";

  return (
    <>
      <PageHeader
        title={project.name}
        description={
          <Badge variant={archived ? "secondary" : "default"} className="capitalize">
            {project.status}
          </Badge>
        }
        actions={
          <>
            {can("manage") && <ShareDialog objectType="project" objectId={project.id} title={project.name} />}
            {can("edit") && (
              <>
                <Button
                  variant="outline"
                  disabled={update.isPending}
                  onClick={() =>
                    update.mutate(
                      { status: archived ? "active" : "archived" },
                      { onError: (err) => toast.error(errorMessage(err)) },
                    )
                  }
                >
                  {archived ? "Unarchive" : "Archive"}
                </Button>
                <ProjectFormDialog
                  project={project}
                  trigger={
                    <Button variant="outline">
                      <Pencil />
                      Edit
                    </Button>
                  }
                />
              </>
            )}
            {can("delete") && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button variant="destructive">
                    <Trash2 />
                    Delete
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Delete “{project.name}”?</AlertDialogTitle>
                    <AlertDialogDescription>This cannot be undone.</AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() =>
                        remove.mutate(project.id, {
                          onSuccess: () => {
                            toast.success("Project deleted");
                            navigate("/projects", { replace: true });
                          },
                          onError: (err) => toast.error(errorMessage(err)),
                        })
                      }
                    >
                      Delete
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </>
        }
      />
      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 text-sm">
          <p className="whitespace-pre-wrap">{project.description || <span className="text-muted-foreground">No description.</span>}</p>
          <dl className="grid grid-cols-[8rem_1fr] gap-y-1 text-muted-foreground">
            <dt>Created</dt>
            <dd>{formatDate(project.createdAt)}</dd>
            <dt>Updated</dt>
            <dd>{formatDate(project.updatedAt)}</dd>
          </dl>
        </CardContent>
      </Card>
    </>
  );
}
