import SiteNav from "@/components/SiteNav";
import DashboardClient from "@/components/DashboardClient";

export default function DashboardPage() {
  return (
    <div className="min-h-screen">
      <SiteNav />
      <main className="container-page py-10">
        <DashboardClient />
      </main>
    </div>
  );
}
