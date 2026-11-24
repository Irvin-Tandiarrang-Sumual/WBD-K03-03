import type { DbPermission } from './database';

export interface GroupDbAccess {
  databaseId: string;
  permission: DbPermission;
}

export interface AccessGroup {
  id: string;
  name: string;
  description: string;
  leaderId: string;
  memberIds: string[];
  databaseAccess: GroupDbAccess[];
}
