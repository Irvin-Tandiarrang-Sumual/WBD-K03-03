import type { ScenicSceneId } from '../types';

export function getScenicSceneForView(
  isAuthenticated: boolean,
  currentPath: string
): ScenicSceneId {
  if (!isAuthenticated || currentPath === '/login') {
    return 'valley_trestle';
  }

  const isDatabaseView =
    currentPath === '/databases' ||
    (currentPath.startsWith('/databases/') && !currentPath.includes('/drafts')) ||
    currentPath === '/admin/databases';

  if (isDatabaseView) {
    return 'valley_trestle';
  }

  return 'pine_forest';
}
