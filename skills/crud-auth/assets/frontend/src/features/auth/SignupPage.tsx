import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { Link, Navigate, useNavigate } from "react-router";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { applyServerErrors } from "@/lib/forms";
import { useAuthConfig, useMe, useSignup } from "./api";

const schema = z.object({
  name: z.string().trim().max(200),
  email: z.email("Enter a valid email"),
  password: z.string().min(8, "At least 8 characters").max(72, "At most 72 characters"),
});
type FormValues = z.infer<typeof schema>;

export function SignupPage() {
  const { data: me } = useMe();
  const { data: config, isLoading } = useAuthConfig();
  const signup = useSignup();
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { name: "", email: "", password: "" } });

  if (me) return <Navigate to="/" replace />;
  if (!isLoading && !config?.signupEnabled) return <Navigate to="/login" replace />;

  const onSubmit = handleSubmit((values) =>
    signup.mutate(values, {
      onSuccess: () => navigate("/", { replace: true }),
      onError: (err) => applyServerErrors(err, setError),
    }),
  );

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Create an account</CardTitle>
          <CardDescription>It only takes a moment.</CardDescription>
        </CardHeader>
        <form onSubmit={onSubmit} noValidate>
          <CardContent className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Name</Label>
              <Input id="name" autoComplete="name" {...register("name")} />
              {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" autoComplete="email" {...register("email")} />
              {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" autoComplete="new-password" {...register("password")} />
              {errors.password && <p className="text-sm text-destructive">{errors.password.message}</p>}
            </div>
          </CardContent>
          <CardFooter className="mt-4 flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={signup.isPending}>
              {signup.isPending ? "Creating…" : "Create account"}
            </Button>
            <p className="text-sm text-muted-foreground">
              Already registered?{" "}
              <Link to="/login" className="underline">
                Sign in
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
