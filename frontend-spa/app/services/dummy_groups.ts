import type { AccessGroup, GroupDbAccess } from '~/types/group';
import type { DbPermission } from '~/types/database';

export const INITIAL_GROUPS: AccessGroup[] = [
  {
    id: 'grp-payments',
    name: 'Payments Reliability',
    description: 'Settlement ledgers and billing services',
    leaderId: 'usr-2',
    memberIds: ['usr-2', 'usr-4', 'usr-6'],
    databaseAccess: [
      { databaseId: 'db-payments', permission: 'view_edit' },
      { databaseId: 'db-ledger', permission: 'view_edit' },
      { databaseId: 'db-analytics', permission: 'view_only' },
    ],
  },
  {
    id: 'grp-core',
    name: 'Core Platform',
    description: 'Identity schemas and core transactions',
    leaderId: 'usr-3',
    memberIds: ['usr-3', 'usr-4', 'usr-1', 'usr-2'],
    databaseAccess: [
      { databaseId: 'db-auth', permission: 'view_edit' },
      { databaseId: 'db-ledger', permission: 'view_edit' },
      { databaseId: 'db-payments', permission: 'view_only' },
    ],
  },
  {
    id: 'grp-data',
    name: 'Data & Analytics',
    description: 'BI warehouse tables and rollups',
    leaderId: 'usr-5',
    memberIds: ['usr-5', 'usr-6'],
    databaseAccess: [
      { databaseId: 'db-analytics', permission: 'view_edit' },
      { databaseId: 'db-ledger', permission: 'view_only' },
    ],
  },
];

let memoryGroups: AccessGroup[] = [...INITIAL_GROUPS];

export const fetchGroups = async (): Promise<AccessGroup[]> => {
  await new Promise((resolve) => setTimeout(resolve, 60));
  return [...memoryGroups];
};

export const fetchGroupById = async (id: string): Promise<AccessGroup | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 40));
  return memoryGroups.find((g) => g.id === id);
};

export const updateGroupDatabaseAccess = async (
  groupId: string,
  databaseId: string,
  newPermission: DbPermission
): Promise<AccessGroup> => {
  await new Promise((resolve) => setTimeout(resolve, 80));
  const group = memoryGroups.find((g) => g.id === groupId);
  if (!group) throw new Error('Group not found');

  const existingAccessIndex = group.databaseAccess.findIndex((a) => a.databaseId === databaseId);
  let updatedAccess: GroupDbAccess[];

  if (existingAccessIndex >= 0) {
    updatedAccess = group.databaseAccess.map((a) =>
      a.databaseId === databaseId ? { ...a, permission: newPermission } : a
    );
  } else {
    updatedAccess = [...group.databaseAccess, { databaseId, permission: newPermission }];
  }

  const updatedGroup: AccessGroup = {
    ...group,
    databaseAccess: updatedAccess,
  };

  memoryGroups = memoryGroups.map((g) => (g.id === groupId ? updatedGroup : g));
  return updatedGroup;
};

export const promoteAndRemoveLeader = async (
  groupId: string,
  newLeaderId: string
): Promise<AccessGroup> => {
  await new Promise((resolve) => setTimeout(resolve, 80));
  const group = memoryGroups.find((g) => g.id === groupId);
  if (!group) throw new Error('Group not found');

  const currentLeaderId = group.leaderId;
  const newMemberIds = group.memberIds.filter((id) => id !== currentLeaderId);
  if (!newMemberIds.includes(newLeaderId)) {
    newMemberIds.push(newLeaderId);
  }

  const updatedGroup: AccessGroup = {
    ...group,
    leaderId: newLeaderId,
    memberIds: newMemberIds,
  };

  memoryGroups = memoryGroups.map((g) => (g.id === groupId ? updatedGroup : g));
  return updatedGroup;
};

export const removeGroupMember = async (groupId: string, memberId: string): Promise<AccessGroup> => {
  await new Promise((resolve) => setTimeout(resolve, 60));
  const group = memoryGroups.find((g) => g.id === groupId);
  if (!group) throw new Error('Group not found');
  if (group.leaderId === memberId) {
    throw new Error('Cannot remove group leader directly. A replacement must be appointed.');
  }

  const updatedGroup: AccessGroup = {
    ...group,
    memberIds: group.memberIds.filter((id) => id !== memberId),
  };

  memoryGroups = memoryGroups.map((g) => (g.id === groupId ? updatedGroup : g));
  return updatedGroup;
};
