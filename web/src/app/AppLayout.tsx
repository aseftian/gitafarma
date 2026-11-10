import { NavLink, Outlet } from "react-router-dom";
import { Pill, Search, ShieldCheck } from "lucide-react";

export function AppLayout() {
  const navClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors ${
      isActive ? "bg-primary/10 text-primary" : "text-muted-foreground hover:text-foreground"
    }`;

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-10 border-b border-border bg-card/95 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-3xl items-center justify-between px-4">
          <NavLink to="/" className="flex items-center gap-2 font-semibold text-primary">
            <Pill className="h-5 w-5" />
            <span>Apotek Gita Farma</span>
          </NavLink>
          <nav className="flex items-center gap-1">
            <NavLink to="/" end className={navClass}>
              <Search className="h-4 w-4" />
              <span>Cek Harga</span>
            </NavLink>
            <NavLink to="/admin" className={navClass}>
              <ShieldCheck className="h-4 w-4" />
              <span>Admin</span>
            </NavLink>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
