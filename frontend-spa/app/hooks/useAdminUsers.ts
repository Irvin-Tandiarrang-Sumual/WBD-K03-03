import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { groupService } from '../services/group_service';
import { userService } from '../services/user_service';
import type { AddUserPayload } from '../components/modals/AddUserModal';
import type { EditUserPayload } from '../components/modals/EditUserModal';
import type { AccessGroup, PlatformUser } from '../types';
import { useAsyncAction } from './useAsyncAction';

export function useAdminUsers(initialUsers: PlatformUser[], initialGroups: AccessGroup[]) {
  const { showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [users, setUsers] = useState<PlatformUser[]>(initialUsers);
  const [groups, setGroups] = useState<AccessGroup[]>(initialGroups);
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<'all' | 'admin' | 'user'>('all');
  const [addUserOpen, setAddUserOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<PlatformUser | null>(null);
  const [deletingUser, setDeletingUser] = useState<PlatformUser | null>(null);

  const filteredUsers = useMemo(() => {
    return users.filter((u) => {
      if (
        search &&
        !u.username.toLowerCase().includes(search.toLowerCase()) &&
        !u.displayName.toLowerCase().includes(search.toLowerCase()) &&
        !u.email.toLowerCase().includes(search.toLowerCase())
      ) {
        return false;
      }
      if (roleFilter !== 'all' && u.role !== roleFilter) return false;
      return true;
    });
  }, [users, search, roleFilter]);

  const handleCreateUser = async (data: AddUserPayload) => {
    await runAsync(
      async () => {
        const created = await userService.createUser({
          username: data.username,
          password: data.password || `${data.username}123`,
          role: data.role === 'Admin' ? 'admin' : 'user',
          groupIds: [data.groupId],
        });
        const refreshedGroups = await groupService.getGroups().catch(() => null);
        return { created, refreshedGroups };
      },
      {
        successMessage: ({ created }) => `User ${created.displayName} created successfully!`,
        successTone: 'pine',
        onSuccess: ({ created, refreshedGroups }) => {
          setUsers((prev) => [created, ...prev]);
          if (refreshedGroups) {
            setGroups(refreshedGroups);
          }
          setAddUserOpen(false);
        },
      }
    );
  };

  const handleUpdateUser = async (userId: string, data: EditUserPayload) => {
    const existing = users.find((u) => u.id === userId);
    if (!existing) return;

    await runAsync(
      async () => {
        let updated: PlatformUser = existing;

        const profilePayload: { username?: string; password?: string } = {};
        if (data.username !== existing.username) {
          profilePayload.username = data.username;
        }
        if (data.password) {
          profilePayload.password = data.password;
        }
        if (Object.keys(profilePayload).length > 0) {
          updated = await userService.updateUser(userId, profilePayload);
        }

        if (data.role !== existing.role) {
          updated = await userService.updateUserRole(userId, data.role);
        }

        const currentGroupIds = new Set(
          groups.filter((g) => g.memberIds.includes(userId)).map((g) => g.id)
        );
        const targetGroupIds = new Set(data.groupIds);

        for (const gid of targetGroupIds) {
          if (!currentGroupIds.has(gid)) {
            await groupService.addUserToGroup(gid, userId);
          }
        }
        for (const gid of currentGroupIds) {
          if (!targetGroupIds.has(gid)) {
            await groupService.removeUserFromGroup(gid, userId);
          }
        }

        const refreshedGroups = await groupService.getGroups().catch(() => null);
        return { updated, refreshedGroups };
      },
      {
        successMessage: ({ updated }) => `User @${updated.username} updated successfully!`,
        successTone: 'pine',
        errorMessage: 'Failed to update user',
        onSuccess: ({ updated, refreshedGroups }) => {
          setUsers((prev) => prev.map((u) => (u.id === userId ? updated : u)));
          if (refreshedGroups) {
            setGroups(refreshedGroups);
          }
          setEditingUser(null);
        },
      }
    );
  };

  const handleDeleteUser = async () => {
    if (!deletingUser) return;
    const target = deletingUser;

    await runAsync(
      async () => {
        await userService.deleteUser(target.id);
        return target;
      },
      {
        successMessage: (deleted) => `User @${deleted.username} deleted successfully!`,
        successTone: 'pine',
        errorMessage: 'Failed to delete user',
        onSuccess: (deleted) => {
          setUsers((prev) => prev.filter((u) => u.id !== deleted.id));
          setDeletingUser(null);
        },
      }
    );
  };

  return {
    users,
    groups,
    filteredUsers,
    search,
    setSearch,
    roleFilter,
    setRoleFilter,
    addUserOpen,
    setAddUserOpen,
    editingUser,
    setEditingUser,
    deletingUser,
    setDeletingUser,
    handleCreateUser,
    handleUpdateUser,
    handleDeleteUser,
  };
}
