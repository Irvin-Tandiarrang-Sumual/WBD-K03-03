/**
 * Pure formatting and entity helpers
 */

export function formatMigrationVersion(version: number): string {
  if (!version || version <= 0) {
    return 'Base (#0)';
  }
  return `#${version}`;
}

export function cleanEntityId(id: string | number, prefix?: string): string | number {
  if (typeof id === 'number') return id;
  if (prefix) {
    return id.startsWith(prefix) ? id.slice(prefix.length) : id;
  }
  return id.replace(/^[a-zA-Z]+-/, '');
}

export function generateUserInitials(username: string): string {
  const parts = username.split(/[._-]/);
  const initials = parts
    .map((p) => p[0] || '')
    .join('')
    .slice(0, 2)
    .toUpperCase();
  return initials || username.slice(0, 2).toUpperCase() || 'US';
}
