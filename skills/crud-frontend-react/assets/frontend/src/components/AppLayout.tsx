import { LogOut, User as UserIcon } from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useCan, useLogout, useMe } from "@/features/auth/api";
import { cn } from "@/lib/utils";

const APP_NAME = "App";

type NavItem = { to: string; label: string; action?: string };

// Top navigation. `action` hides the link unless the user's role grants it (RBAC).
const navItems: NavItem[] = [
  { to: "/teams", label: "Teams" },
  // crud:nav (add resource links above this line)
  { to: "/admin/users", label: "Users", action: "users:manage" },
];

export function AppLayout() {
  const { data: me } = useMe();
  const can = useCan();
  const logout = useLogout();
  const navigate = useNavigate();

  return (
    <div className="min-h-svh bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4">
          <NavLink to="/" className="font-semibold">
            {APP_NAME}
          </NavLink>
          <nav className="flex items-center gap-4 text-sm">
            {navItems
              .filter((item) => !item.action || can(item.action))
              .map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    cn("text-muted-foreground hover:text-foreground", isActive && "font-medium text-foreground")
                  }
                >
                  {item.label}
                </NavLink>
              ))}
          </nav>
          <div className="ml-auto">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm">
                  <UserIcon />
                  {me?.user.name || me?.user.email}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>
                  <div className="font-medium">{me?.user.email}</div>
                  <div className="text-xs text-muted-foreground capitalize">{me?.user.role}</div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onSelect={() => logout.mutate(undefined, { onSuccess: () => navigate("/login", { replace: true }) })}
                >
                  <LogOut />
                  Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
