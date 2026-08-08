import type { ReactNode } from "react";

import { AdminGate } from "@/components/admin/admin-gate";
import { PageShell } from "@/components/trips/page-shell";

export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <PageShell back={{ href: "/", label: "Home" }}>
      <AdminGate>{children}</AdminGate>
    </PageShell>
  );
}
