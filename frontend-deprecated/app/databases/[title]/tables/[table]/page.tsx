import { Suspense } from "react";
import TableDataViewer from "@/components/feature/databases/tables/TableDataViewer";

export const instant = false;

export default function TablePage() {
  return (
    <Suspense fallback={null}>
      <TableDataViewer />
    </Suspense>
  );
}
