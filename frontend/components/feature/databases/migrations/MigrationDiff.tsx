"use client";

import ObsidianMigrationDiff from "./ObsidianMigrationDiff";

interface MigrationDiffProps {
  databaseId: string;
  databaseName?: string;
}

export default function MigrationDiff(props: MigrationDiffProps) {
  return <ObsidianMigrationDiff {...props} />;
}
