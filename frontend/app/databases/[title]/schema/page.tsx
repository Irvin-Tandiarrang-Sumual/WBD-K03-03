import styles from "@/app/page.module.css";
import SchemaViewer from "@/components/feature/databases/schema/SchemaViewer";

export const instant = false;

interface DatabaseDetailProps {
  params: Promise<{ title: string }>;
}

export default async function DatabaseDetail({
  params,
}: DatabaseDetailProps) {
  const { title } = await params;
  const decodedTitle = decodeURIComponent(title);
  const databaseId = `${decodedTitle}`;

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <SchemaViewer databaseId={databaseId} />
      </main>
    </div>
  );
}
