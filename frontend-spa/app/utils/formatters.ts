/**
 * Pure formatting helpers
 */

export function formatMigrationVersion(version: number): string {
  if (!version || version <= 0) {
    return 'Base (#0)';
  }
  return `#${version}`;
}
