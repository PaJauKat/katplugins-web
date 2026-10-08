import LoginCard from "@/components/LoginCard";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string; error?: string }>;
}) {
  const params = await searchParams;
  return <LoginCard next={params.next ?? "/dashboard"} error={params.error} />;
}
