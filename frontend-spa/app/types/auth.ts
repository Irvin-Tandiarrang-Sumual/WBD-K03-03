export type AppRole = 'Admin' | 'Leader' | 'User';

export const APP_ROLES: AppRole[] = ['Admin', 'Leader', 'User'];

export interface PlatformUser {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: 'admin' | 'user';
  avatarInitials: string;
  avatarColor: string;
}

export const ROLE_PERSONA_MAP: Record<AppRole, string> = {
  Admin: 'usr-1',
  Leader: 'usr-2',
  User: 'usr-4',
};
