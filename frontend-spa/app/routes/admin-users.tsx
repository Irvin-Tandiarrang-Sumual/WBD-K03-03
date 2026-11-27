import React from 'react';
import { useLoaderData } from 'react-router';
import { Pencil, Trash2, UserPlus } from 'lucide-react';
import { PageHeader } from '../components/common/PageHeader';
import { SearchInput } from '../components/common/SearchInput';
import { ConfirmDialog } from '../components/feedback/ConfirmDialog';
import { AddUserModal } from '../components/modals/AddUserModal';
import { EditUserModal } from '../components/modals/EditUserModal';
import { useAdminUsers } from '../hooks/useAdminUsers';
import { groupService } from '../services/group_service';
import { userService } from '../services/user_service';
import type { AccessGroup, PlatformUser } from '../types';

export async function clientLoader() {
  const [users, groups] = await Promise.all([
    userService.getUsers(),
    groupService.getGroups(),
  ]);
  return { users, groups };
}

export default function AdminUsersPage() {
  const { users: initialUsers, groups: initialGroups } = useLoaderData() as {
    users: PlatformUser[];
    groups: AccessGroup[];
  };

  const d = useAdminUsers(initialUsers, initialGroups);

  return (
    <section className="space-y-5">
      <PageHeader
        title="Users & Access Control"
        subtitle="Manage identities, global roles, and team roster assignments"
        actions={
          <button
            type="button"
            onClick={() => d.setAddUserOpen(true)}
            className="h-10 px-4 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-sm transition-colors cursor-pointer"
          >
            <UserPlus className="w-4 h-4" />
            <span>Add User</span>
          </button>
        }
      />

      <AdminUsersTable
        users={d.filteredUsers}
        groups={d.groups}
        search={d.search}
        onSearchChange={d.setSearch}
        roleFilter={d.roleFilter}
        onRoleFilterChange={d.setRoleFilter}
        onEditUser={d.setEditingUser}
        onDeleteUser={d.setDeletingUser}
      />

      <AddUserModal
        open={d.addUserOpen}
        groups={d.groups}
        onClose={() => d.setAddUserOpen(false)}
        onAddUser={d.handleCreateUser}
      />

      <EditUserModal
        user={d.editingUser}
        groups={d.groups}
        onClose={() => d.setEditingUser(null)}
        onSaveUser={d.handleUpdateUser}
      />

      <ConfirmDialog
        open={Boolean(d.deletingUser)}
        onClose={() => d.setDeletingUser(null)}
        onConfirm={d.handleDeleteUser}
        title="Delete User?"
        message={
          d.deletingUser
            ? `Are you sure you want to permanently delete user @${d.deletingUser.username}? This action cannot be undone.`
            : ''
        }
        confirmText="Delete User"
        isDestructive={true}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface AdminUsersTableProps {
  users: PlatformUser[];
  groups: AccessGroup[];
  search: string;
  onSearchChange: (value: string) => void;
  roleFilter: 'all' | 'admin' | 'user';
  onRoleFilterChange: (value: 'all' | 'admin' | 'user') => void;
  onEditUser: (user: PlatformUser) => void;
  onDeleteUser: (user: PlatformUser) => void;
}

const AdminUsersTable: React.FC<AdminUsersTableProps> = ({
  users,
  groups,
  search,
  onSearchChange,
  roleFilter,
  onRoleFilterChange,
  onEditUser,
  onDeleteUser,
}) => (
  <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs">
    <div className="p-4 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex flex-wrap items-center justify-between gap-3 rounded-t-2xl relative z-20">
      <SearchInput
        value={search}
        onChange={onSearchChange}
        placeholder="Search users..."
        className="max-w-xs"
      />
      <select
        aria-label="Filter users by role"
        value={roleFilter}
        onChange={(e) => onRoleFilterChange(e.target.value as 'all' | 'admin' | 'user')}
        className="h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
      >
        <option value="all">Role: All</option>
        <option value="user">User</option>
        <option value="admin">Admin</option>
      </select>
    </div>

    <div className="overflow-x-auto rounded-b-2xl">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-[#EFE5D1] text-[11px] font-bold uppercase tracking-wider text-[#6B5444] bg-[#F5EEDD]/40">
            <th className="py-3 px-4">User</th>
            <th className="py-3 px-4">Email</th>
            <th className="py-3 px-4">Global Role</th>
            <th className="py-3 px-4">Group Memberships</th>
            <th className="py-3 px-4 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#EFE5D1] text-sm">
          {users.map((u) => {
            const userGroups = groups.filter((g) => g.memberIds.includes(u.id));
            return (
              <AdminUserTableRow
                key={u.id}
                user={u}
                userGroups={userGroups}
                onEdit={() => onEditUser(u)}
                onDelete={() => onDeleteUser(u)}
              />
            );
          })}
        </tbody>
      </table>
    </div>
  </div>
);

interface AdminUserTableRowProps {
  user: PlatformUser;
  userGroups: AccessGroup[];
  onEdit: () => void;
  onDelete: () => void;
}

const AdminUserTableRow: React.FC<AdminUserTableRowProps> = ({
  user,
  userGroups,
  onEdit,
  onDelete,
}) => (
  <tr className="hover:bg-[#F5EEDD]/45 transition-colors">
    <td className="py-3.5 px-4">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-full bg-[#264D3B] text-[#F5EEDD] font-bold text-xs flex items-center justify-center shrink-0">
          {user.avatarInitials}
        </div>
        <div>
          <div className="font-semibold text-xs text-[#3A241A]">{user.displayName}</div>
          <div className="font-mono text-[11px] text-[#6B5444]">@{user.username}</div>
        </div>
      </div>
    </td>
    <td className="py-3.5 px-4 font-mono text-xs text-[#6B5444]">{user.email}</td>
    <td className="py-3.5 px-4">
      <span
        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${getUserRoleBadgeStyle(
          user.role
        )}`}
      >
        {user.role.toUpperCase()}
      </span>
    </td>
    <td className="py-3.5 px-4">
      <div className="flex items-center gap-1.5 flex-wrap">
        {userGroups.length === 0 ? (
          <span className="text-xs text-[#8A7361] italic">None</span>
        ) : (
          userGroups.map((g) => (
            <span
              key={g.id}
              className="px-2 py-0.5 rounded-md text-[11px] font-semibold bg-[#EFE5D1] text-[#3A241A] border border-[#D8C7A5]"
            >
              {g.name}
              {g.leaderId === user.id ? ' (Leader)' : ''}
            </span>
          ))
        )}
      </div>
    </td>
    <td className="py-3.5 px-4 text-right">
      <div className="inline-flex items-center gap-1.5">
        <button
          type="button"
          onClick={onEdit}
          aria-label={`Edit user ${user.username}`}
          className="h-8 px-2.5 rounded-lg border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1 transition-colors cursor-pointer"
        >
          <Pencil className="w-3.5 h-3.5 text-[#1E6B8C]" />
          <span>Edit</span>
        </button>
        <button
          type="button"
          onClick={onDelete}
          aria-label={`Delete user ${user.username}`}
          className="h-8 px-2.5 rounded-lg border border-[#822433]/30 bg-[#822433]/8 hover:bg-[#822433] text-[#822433] hover:text-white text-xs font-semibold inline-flex items-center gap-1 transition-colors cursor-pointer"
        >
          <Trash2 className="w-3.5 h-3.5" />
          <span>Delete</span>
        </button>
      </div>
    </td>
  </tr>
);

function getUserRoleBadgeStyle(role: string): string {
  return role === 'admin'
    ? 'bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70'
    : 'bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/30';
}
