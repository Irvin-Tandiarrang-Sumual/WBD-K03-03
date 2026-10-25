import DatabaseDetailView from "@/components/feature/databases/DatabaseDetailView";

export const instant = false;

interface DatabasePageProps {
  params: Promise<{ title: string }>;
}

export default async function DatabasePage({ params }: DatabasePageProps) {
  const { title } = await params;
  const decodedTitle = decodeURIComponent(title);

  return <DatabaseDetailView databaseId={decodedTitle} />;
}
