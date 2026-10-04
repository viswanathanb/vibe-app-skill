import { Share2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { AccessPanel } from "./AccessPanel";

/** "Share" button + dialog. Render it only when the user has the `manage` permission. */
export function ShareDialog({ objectType, objectId, title }: { objectType: string; objectId: number; title: string }) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Share2 />
          Share
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Share “{title}”</DialogTitle>
          <DialogDescription>Grant people or whole teams access.</DialogDescription>
        </DialogHeader>
        <AccessPanel objectType={objectType} objectId={objectId} />
      </DialogContent>
    </Dialog>
  );
}
