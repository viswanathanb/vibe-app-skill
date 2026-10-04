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
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AccessPanel } from "@/features/sharing/AccessPanel";
import { errorMessage } from "@/lib/api";
import { useDeleteTeam, useTeam } from "./api";
import { TeamFormDialog } from "./TeamFormDialog";

export function TeamDetailPage() {
  const id = Number(useParams().id);
  const navigate = useNavigate();
  const { data: team, isLoading, error } = useTeam(id);
  const remove = useDeleteTeam();

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>;
  if (error || !team) return <p className="text-destructive">{errorMessage(error)}</p>;

  const can = (perm: string) => team.permissions.includes(perm);

  return (
    <>
      <PageHeader
        title={team.name}
        description={team.description}
        actions={
          <>
            {can("edit") && (
              <TeamFormDialog
                team={team}
                trigger={
                  <Button variant="outline">
                    <Pencil />
                    Edit
                  </Button>
                }
              />
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
                    <AlertDialogTitle>Delete “{team.name}”?</AlertDialogTitle>
                    <AlertDialogDescription>Everything shared with this team loses that access.</AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() =>
                        remove.mutate(team.id, {
                          onSuccess: () => navigate("/teams", { replace: true }),
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
          <CardTitle>Members</CardTitle>
          <CardDescription>
            Share any resource with this team using the subject <code>team:{team.id}#member</code>.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {can("manage") ? (
            <AccessPanel objectType="team" objectId={team.id} />
          ) : (
            <p className="text-sm text-muted-foreground">Only team admins can see and change membership.</p>
          )}
        </CardContent>
      </Card>
    </>
  );
}
