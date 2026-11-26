import { useMemo, useState } from 'react';
import { useAuth } from '../context/auth-context';
import { userService } from '../services/user_service';
import type { AppRole, PlatformUser } from '../types';
import { useAsyncAction } from './useAsyncAction';

export function useAdminUsers(initialUsers: PlatformUser[]) {
  const { showSnackbar } = useAuth();
  const { execute: runAsync } = useAsyncAction(showSnackbar);

  const [users, setUsers] = useState<PlatformUser[]>(initialUsers);
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<'all' | 'admin' | 'user'>('all');
  const [addUserOpen, setAddUserOpen] = useState(false);

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

  const handleCreateUser = async (data: {
    username: string;
    displayName: string;
    email: string;
    role: AppRole;
  }) => {
    await runAsync(
      async () => {
        const created = await userService.createUser({
          username: data.username,
          password: `${data.username}123`,
          role: data.role === 'Admin' ? 'admin' : 'user',
          groupIds: [1],
        });
        return created;
      },
      {
        successMessage: (created) => `User ${created.displayName} created successfully!`,
        successTone: 'pine',
        onSuccess: (created) => {
          setUsers((prev) => [created, ...prev]);
          setAddUserOpen(false);
        },
      }
    );
  };

  return {
    users,
    filteredUsers,
    search,
    setSearch,
    roleFilter,
    setRoleFilter,
    addUserOpen,
    setAddUserOpen,
    handleCreateUser,
  };
}
