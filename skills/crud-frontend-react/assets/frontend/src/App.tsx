import { createBrowserRouter, RouterProvider } from "react-router";
import { AppLayout } from "@/components/AppLayout";
import { RequireAuth } from "@/components/RequireAuth";
import { UsersPage } from "@/features/admin/UsersPage";
import { LoginPage } from "@/features/auth/LoginPage";
import { SignupPage } from "@/features/auth/SignupPage";
import { TeamDetailPage } from "@/features/teams/TeamDetailPage";
import { TeamsPage } from "@/features/teams/TeamsPage";
import { HomePage } from "@/pages/HomePage";
import { NotFoundPage } from "@/pages/NotFoundPage";

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/signup", element: <SignupPage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { index: true, element: <HomePage /> },
          { path: "teams", element: <TeamsPage /> },
          { path: "teams/:id", element: <TeamDetailPage /> },
          // crud:routes (add resource routes above this line)
          { path: "admin/users", element: <UsersPage /> },
        ],
      },
    ],
  },
  { path: "*", element: <NotFoundPage /> },
]);

export function App() {
  return <RouterProvider router={router} />;
}
