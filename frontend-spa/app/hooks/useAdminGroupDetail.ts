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

  return {
    group,
    leaderReplaceOpen,
    setLeaderReplaceOpen,
    pendingAccessChange,
    setPendingAccessChange,
    handleRemoveMember,
    handleConfirmLeaderReplacement,
    handleRequestAccessChange,
    handleConfirmAccessChange,
  };
}
