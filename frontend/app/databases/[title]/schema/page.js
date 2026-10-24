import styles from "@/app/page.module.css";
import SchemaViewer from "@/components/feature/databases/schema/SchemaViewer";

export const instant = false;

export default async function DatabaseDetail({ params }) {
  const { title } = await params;
  const decodedTitle = decodeURIComponent(title);
  const databaseName = `${decodedTitle}`;

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <SchemaViewer dbName={databaseName} />
      </main>
    </div>
  );
}
