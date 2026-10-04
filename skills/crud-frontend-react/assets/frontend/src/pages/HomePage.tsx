import { Link } from "react-router";
import { PageHeader } from "@/components/PageHeader";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useMe } from "@/features/auth/api";

// Dashboard landing page. Replace the cards with the app's main entry points.
export function HomePage() {
  const { data: me } = useMe();
  return (
    <>
      <PageHeader title={`Welcome${me?.user.name ? `, ${me.user.name}` : ""}`} description="Pick where to start." />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Link to="/teams">
          <Card className="transition-colors hover:bg-muted/50">
            <CardHeader>
              <CardTitle>Teams</CardTitle>
              <CardDescription>Group people and share resources with a whole team.</CardDescription>
            </CardHeader>
          </Card>
        </Link>
        {/* crud:home (add resource cards above this line) */}
      </div>
    </>
  );
}
