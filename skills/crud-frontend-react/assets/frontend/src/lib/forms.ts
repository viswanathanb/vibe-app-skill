import type { FieldValues, Path, UseFormSetError } from "react-hook-form";
import { toast } from "sonner";
import { ApiError, errorMessage } from "./api";

const messages: Record<string, string> = {
  required: "Required",
  email: "Enter a valid email",
};

/**
 * Shows server-side validation errors next to the matching form fields
 * (backend details look like {"name": "required", "title": "max=200"}), or a toast otherwise.
 */
export function applyServerErrors<T extends FieldValues>(err: unknown, setError: UseFormSetError<T>) {
  if (err instanceof ApiError && err.details) {
    for (const [field, rule] of Object.entries(err.details)) {
      const [tag, param] = rule.split("=");
      const message = messages[tag] ?? (tag === "max" ? `At most ${param} characters` : rule);
      setError(field as Path<T>, { type: "server", message });
    }
    return;
  }
  toast.error(errorMessage(err));
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}
