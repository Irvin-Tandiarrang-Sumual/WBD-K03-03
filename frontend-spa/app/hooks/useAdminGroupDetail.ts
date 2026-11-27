import { useState } from 'react';
import { useAuth } from '../context/auth-context';
import { groupService } from '../services/group_service';
import type { AccessGroup, DbPermission, TargetDatabase } from '../types';
import { useAsyncAction } from './useAsyncAction';

interface PendingAccessChange {
  groupId: string;
  dbId: string;
  dbName: string;
  currentPermission: DbPermission;
  nextPermission: DbPermission;
}

export function useAdminGroupDetail(initialGroup: AccessGroup, databases: TargetDatabase[]) {
  const { showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [group, setGroup] = useState<AccessGroup>(initialGroup);
  const [leaderReplaceOpen, setLeaderReplaceOpen] = useState(false);
  const [pendingAccessChange, setPendingAccessChange] = useState<PendingAccessChange | null>(null);
  const [pendingRevokeDb, setPendingRevokeDb] = useState<{ dbId: string; dbName: string } | null>(null);

  const handleAddMember = async (userId: string) => {
    await runAsync(
      async () => {
        await groupService.addUserToGroup(group.id, userId);
        return {
          ...group,
          memberIds: Array.from(new Set([...group.memberIds, userId])),
        };
      },
      {
        successMessage: 'Member added to group',
        successTone: 'pine',
        errorMessage: 'Failed to add member to group',
        onSuccess: (updated) => setGroup(updated),
      }
    );
  };

  const handleSetLeader = async (userId: string) => {
    await runAsync(
      async () => {
        const updated = await groupService.updateGroup(group.id, {
          leaderId: Number(userId),
        });
        return {
          ...updated,
          leaderId: userId,
          memberIds: Array.from(new Set([...group.memberIds, userId])),
        };
      },
      {
        successMessage: 'Group leader updated successfully',
        successTone: 'pine',
        errorMessage: 'Failed to update group leader',
        onSuccess: (updated) => setGroup(updated),
      }
    );
  };

  const handleRemoveMember = async (memberId: string) => {
    if (memberId === group.leaderId) {
      setLeaderReplaceOpen(true);
      return;
    }

    await runAsync(
      async () => {
        await groupService.removeUserFromGroup(group.id, memberId);
        const updated: AccessGroup = {
          ...group,
          memberIds: group.memberIds.filter((id) => id !== memberId),
        };
        return updated;
      },
      {
        successMessage: 'Member removed from group',
        successTone: 'pine',
        onSuccess: (updated) => setGroup(updated),
      }
    );
  };

  const handleConfirmLeaderReplacement = async (groupId: string, newLeaderId: string) => {
    await runAsync(
      async () => {
        await groupService.removeUserFromGroup(groupId, group.leaderId, newLeaderId);
        const refetched = await groupService.getGroupById(groupId);
        const updatedMembers = group.memberIds
          .filter((id) => id !== group.leaderId)
          .concat([newLeaderId]);
        const updated: AccessGroup = {
          ...refetched,
          leaderId: newLeaderId,
          memberIds: Array.from(new Set(updatedMembers)),
        };
        return updated;
      },
      {
        successMessage: 'Leader promoted and previous leader removed',
        successTone: 'pine',
        onSuccess: (updated) => {
          setGroup(updated);
          setLeaderReplaceOpen(false);
        },
      }
    );
  };

  const handleRequestAccessChange = (dbId: string, currentPermission: DbPermission) => {
    const db = databases.find((d) => d.id === dbId);
    if (!db) return;

    const nextPermission: DbPermission =
      currentPermission === 'view_edit' ? 'view_only' : 'view_edit';

    setPendingAccessChange({
      groupId: group.id,
      dbId,
      dbName: db.name,
      currentPermission,
      nextPermission,
    });
  };

  const handleConfirmAccessChange = async () => {
    if (!pendingAccessChange) return;

    await runAsync(
      async () => {
        const nextAccesses = group.databaseAccess.map((acc) => {
          if (acc.databaseId === pendingAccessChange.dbId) {
            return {
              databaseId: Number(acc.databaseId),
              accessType: pendingAccessChange.nextPermission,
            };
          }
          return {
            databaseId: Number(acc.databaseId),
            accessType: acc.permission,
          };
        });

        // If not already in group.databaseAccess, add it
        if (!nextAccesses.some((a) => String(a.databaseId) === pendingAccessChange.dbId)) {
          nextAccesses.push({
            databaseId: Number(pendingAccessChange.dbId),
            accessType: pendingAccessChange.nextPermission,
          });
        }

        const updated = await groupService.updateGroup(group.id, {
          databaseAccesses: nextAccesses,
        });

        return {
          ...updated,
          memberIds: group.memberIds,
        };
      },
      {
        successMessage: `Updated ${pendingAccessChange.dbName} access to ${pendingAccessChange.nextPermission}`,
        successTone: 'pine',
        onSuccess: (updated) => {
          setGroup(updated);
          setPendingAccessChange(null);
        },
      }
    );
  };

  const handleRequestRevokeAccess = (dbId: string) => {
    const db = databases.find((d) => d.id === dbId);
    if (!db) return;
    setPendingRevokeDb({ dbId, dbName: db.name });
  };

  const handleConfirmRevokeAccess = async () => {
    if (!pendingRevokeDb) return;
    const target = pendingRevokeDb;

    await runAsync(
      async () => {
        const nextAccesses = group.databaseAccess
          .filter((acc) => acc.databaseId !== target.dbId)
          .map((acc) => ({
            databaseId: Number(acc.databaseId),
            accessType: acc.permission,
          }));

        const updated = await groupService.updateGroup(group.id, {
          databaseAccesses: nextAccesses,
        });

        return {
          ...updated,
          memberIds: group.memberIds,
        };
      },
      {
        successMessage: `Revoked ${target.dbName} access from ${group.name}`,
        successTone: 'pine',
        errorMessage: 'Failed to revoke database access',
        onSuccess: (updated) => {
          setGroup(updated);
          setPendingRevokeDb(null);
        },
      }
    );
  };

  const handleGrantAccess = async (dbId: string, permission: DbPermission) => {
    const db = databases.find((d) => d.id === dbId);
    if (!db) return;

    await runAsync(
      async () => {
        const nextAccesses = [
          ...group.databaseAccess.map((acc) => ({
            databaseId: Number(acc.databaseId),
            accessType: acc.permission,
          })),
          {
            databaseId: Number(dbId),
            accessType: permission,
          },
        ];

        const updated = await groupService.updateGroup(group.id, {
          databaseAccesses: nextAccesses,
        });

        return {
          ...updated,
          memberIds: group.memberIds,
        };
      },
      {
        successMessage: `Granted ${permission} access on ${db.name} to ${group.name}`,
        successTone: 'pine',
        errorMessage: 'Failed to grant database access',
        onSuccess: (updated) => setGroup(updated),
      }
    );
  };

  return {
    group,
    leaderReplaceOpen,
    setLeaderReplaceOpen,
    pendingAccessChange,
    setPendingAccessChange,
    pendingRevokeDb,
    setPendingRevokeDb,
    handleAddMember,
    handleSetLeader,
    handleRemoveMember,
    handleConfirmLeaderReplacement,
    handleRequestAccessChange,
    handleConfirmAccessChange,
    handleRequestRevokeAccess,
    handleConfirmRevokeAccess,
    handleGrantAccess,
  };
}
