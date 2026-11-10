import { lazy, Suspense } from "react";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { AuthProvider } from "@/features/auth/AuthContext";
import { RequireAdmin } from "@/features/auth/components/RequireAdmin";
import { SearchPage } from "@/features/search/SearchPage";
import { AppLayout } from "./AppLayout";

const AdminDashboardPage = lazy(() =>
  import("@/features/admin-medicines/AdminDashboardPage").then((m) => ({
    default: m.AdminDashboardPage,
  })),
);

const router = createBrowserRouter([
  {
    path: "/",
    element: <AppLayout />,
    children: [
      { index: true, element: <SearchPage /> },
      {
        path: "admin",
        element: (
          <RequireAdmin>
            <Suspense
              fallback={
                <div className="p-6 text-center text-sm text-muted-foreground">
                  Memuat panel pengelola...
                </div>
              }
            >
              <AdminDashboardPage />
            </Suspense>
          </RequireAdmin>
        ),
      },
    ],
  },
]);

export function App() {
  return (
    <AuthProvider>
      <RouterProvider router={router} />
    </AuthProvider>
  );
}
