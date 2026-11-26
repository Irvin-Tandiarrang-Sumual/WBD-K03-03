/**
 * Pure table manipulation and column visibility helpers
 */

export function toggleColumnVisibility<T extends Record<string, boolean>>(
  current: T,
  key: string,
  minVisible = 1
): T {
  const currentlyVisibleCount = Object.values(current).filter(Boolean).length;
  const isTargetVisible = Boolean(current[key]);

  if (isTargetVisible && currentlyVisibleCount <= minVisible) {
    return current;
  }

  return { ...current, [key]: !isTargetVisible };
}
