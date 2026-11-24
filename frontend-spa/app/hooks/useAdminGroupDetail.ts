import { useState } from 'react';
import { useAuth } from '../context/auth-context';
import {
  promoteAndRemoveLeader,
  removeGroupMember,
  updateGroupDatabaseAccess,
} from '../services/dummy_groups';
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
      async () => removeGroupMember(group.id, memberId),
      {
        successMessage: 'Member removed from group',
        successTone: 'pine',
        onSuccess: (updated) => setGroup(updated),
      }
    );
  };

  const handleConfirmLeaderReplacement = async (groupId: string, newLeaderId: string) => {
    await runAsync(
      async () => promoteAndRemoveLeader(groupId, newLeaderId),
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
      async () =>
        updateGroupDatabaseAccess(
          pendingAccessChange.groupId,
          pendingAccessChange.dbId,
          pendingAccessChange.nextPermission
        ),
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
