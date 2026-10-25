import MigrationDiff from "@/components/feature/databases/migrations/MigrationDiff";

export const instant = false;

interface NewMigrationPageProps {
  params: Promise<{ title: string }>;
  searchParams: Promise<{ dbName?: string }>;
}

export default async function NewMigrationPage({
  params,
  searchParams,
}: NewMigrationPageProps) {
  const [{ title }, query] = await Promise.all([params, searchParams]);
  const databaseName = query?.dbName || decodeURIComponent(title);

  return <MigrationDiff databaseId={databaseName} />;
}
