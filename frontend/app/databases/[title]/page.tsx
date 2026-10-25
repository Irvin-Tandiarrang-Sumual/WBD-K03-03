import { redirect } from "next/navigation";

export const instant = false;

interface DatabasePageProps {
  params: Promise<{ title: string }>;
}

export default async function DatabasePage({ params }: DatabasePageProps) {
  const { title } = await params;
  redirect(`/databases/${encodeURIComponent(title)}/schema`);
}
