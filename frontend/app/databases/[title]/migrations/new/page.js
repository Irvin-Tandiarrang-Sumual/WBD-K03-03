import MigrationDiff from "@/components/feature/databases/migrations/MigrationDiff";

export const instant = false;

export default async function NewMigrationPage({ params, searchParams }) {
  const [{ title }, query] = await Promise.all([params, searchParams]);
  const databaseName = query?.dbName || decodeURIComponent(title);

  return <MigrationDiff databaseName={databaseName} />;
}
