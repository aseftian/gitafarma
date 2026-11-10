import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "./features/auth/AuthContext";
import { RequireAdmin } from "./features/auth/components/RequireAdmin";
import { SearchPage } from "./features/search/SearchPage";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("SearchPage", () => {
  it("loads recent updates and performs debounced medicine search", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/api/v1/medicines/recent")) {
        return new Response(
          JSON.stringify({
            recent_days: 7,
            total: 1,
            items: [
              {
                id: "1",
                name: "Amoxicillin 500mg",
                barcode: "899111",
                sku: "GF-01",
                price: 12000,
                uom: "Strip",
                alt_uom: "Kapsul",
                updated_at: "2026-11-10T10:00:00Z",
              },
            ],
          }),
          { status: 200 },
        );
      }
      if (url.includes("/api/v1/medicines?q=parace")) {
        return new Response(
          JSON.stringify({
            total: 1,
            items: [
              {
                id: "2",
                name: "Paracetamol 500mg Tablet",
                barcode: "899222",
                sku: "GF-02",
                price: 5500,
                uom: "Strip",
                alt_uom: "Tablet",
                updated_at: "2026-11-10T10:00:00Z",
              },
            ],
          }),
          { status: 200 },
        );
      }
      return new Response("{}", { status: 404 });
    });

    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <SearchPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Amoxicillin 500mg")).toBeInTheDocument();
    expect(screen.getByText(/7 Hari Terakhir/i)).toBeInTheDocument();

    const input = screen.getByLabelText("Cari obat");
    fireEvent.change(input, { target: { value: "parace" } });

    await waitFor(
      () => {
        expect(screen.getByText("Paracetamol 500mg Tablet")).toBeInTheDocument();
      },
      { timeout: 1500 },
    );
  });
});

describe("RequireAdmin", () => {
  it("shows forbidden message when Google email is not in admins sheet", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("/api/v1/auth/me")) {
          return new Response(
            JSON.stringify({ error: "email tidak terdaftar atau tidak aktif sebagai admin" }),
            { status: 403 },
          );
        }
        return new Response("{}", { status: 404 });
      }),
    );

    render(
      <AuthProvider>
        <RequireAdmin>
          <div>Admin Secret</div>
        </RequireAdmin>
      </AuthProvider>,
    );

    expect(await screen.findByText(/Akses Ditolak/i)).toBeInTheDocument();
    expect(screen.queryByText("Admin Secret")).not.toBeInTheDocument();
  });

  it("shows Google SSO login screen when unauthenticated", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ error: "autentikasi diperlukan" }), { status: 401 })),
    );

    render(
      <AuthProvider>
        <RequireAdmin>
          <div>Admin Secret</div>
        </RequireAdmin>
      </AuthProvider>,
    );

    expect(await screen.findByText(/Login Pengelola Apotek/i)).toBeInTheDocument();
  });
});
