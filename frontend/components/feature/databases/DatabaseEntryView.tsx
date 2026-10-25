"use client";

import DatabaseDetailView from "./DatabaseDetailView";

export default function DatabaseEntryView({
  databaseId,
}: {
  databaseId: string;
}) {
  return <DatabaseDetailView databaseId={databaseId} />;
}
