import DatabaseEntryView from "@/components/feature/databases/DatabaseEntryView";

export const instant = false;

interface DatabasePageProps {
  params: Promise<{ title: string }>;
}

export default async function DatabasePage({ params }: DatabasePageProps) {
  const { title } = await params;
  const decodedTitle = decodeURIComponent(title);

  return <DatabaseEntryView databaseId={decodedTitle} />;
}
