import React, { useEffect, useState } from 'react';
import type { AccessGroup, PlatformUser } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

export interface EditUserPayload {
  username: string;
  password?: string;
  role: 'admin' | 'user';
  groupIds: string[];
}

interface EditUserModalProps {
  user: PlatformUser | null;
  groups: AccessGroup[];
  onClose: () => void;
  onSaveUser: (userId: string, payload: EditUserPayload) => void;
}

export const EditUserModal: React.FC<EditUserModalProps> = ({
  user,
  groups,
  onClose,
  onSaveUser,
}) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState<'admin' | 'user'>('user');
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (user) {
      setUsername(user.username);
      setPassword('');
      setRole(user.role === 'admin' ? 'admin' : 'user');
      setSelectedGroupIds(
        groups.filter((g) => g.memberIds.includes(user.id)).map((g) => g.id)
      );
      setErrorMessage('');
    }
  }, [user, groups]);

  const toggleGroup = (groupId: string) => {
    setSelectedGroupIds((prev) =>
      prev.includes(groupId) ? prev.filter((id) => id !== groupId) : [...prev, groupId]
    );
  };

  const handleFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!user) return;

    const cleanUsername = username.trim().toLowerCase();
    if (!cleanUsername) {
      setErrorMessage('Username is required.');
      return;
    }
    if (role === 'user' && selectedGroupIds.length === 0) {
      setErrorMessage('Regular users must belong to at least 1 group.');
      return;
    }

    setErrorMessage('');
    onSaveUser(user.id, {
      username: cleanUsername,
      password: password.trim() ? password : undefined,
      role,
      groupIds: selectedGroupIds,
    });
  };

  return (
    <M3Dialog
      open={Boolean(user)}
      onClose={onClose}
      title={user ? `Edit User — @${user.username}` : 'Edit User'}
      maxWidth="max-w-md"
      footer={
        <>
          <button
            type="button"
            onClick={onClose}
            className="h-10 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="submit"
            form="edit-user-form"
            className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold cursor-pointer"
          >
            Save Changes
          </button>
        </>
      }
    >
      <form id="edit-user-form" onSubmit={handleFormSubmit} className="space-y-3.5">
        <div>
          <label htmlFor="edit-u-username" className="block text-xs font-bold text-[#3A241A] mb-1">
            Username
          </label>
          <input
            id="edit-u-username"
            name="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] font-mono text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>

        <div>
          <label htmlFor="edit-u-password" className="block text-xs font-bold text-[#3A241A] mb-1">
            New Password <span className="font-normal text-[#8A7361]">(optional)</span>
          </label>
          <input
            id="edit-u-password"
            name="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Leave blank to keep current password"
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>

        <div>
          <label htmlFor="edit-u-role" className="block text-xs font-bold text-[#3A241A] mb-1">
            Global Role
          </label>
          <select
            id="edit-u-role"
            name="role"
            value={role}
            onChange={(e) => setRole(e.target.value === 'admin' ? 'admin' : 'user')}
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </div>

        <div>
          <span className="block text-xs font-bold text-[#3A241A] mb-1.5">
            Group Memberships
          </span>
          <div className="max-h-36 overflow-y-auto rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] p-2.5 space-y-1.5">
            {groups.map((g) => {
              const checked = selectedGroupIds.includes(g.id);
              const isGroupLeader = user ? g.leaderId === user.id : false;
              return (
                <label
                  key={g.id}
                  className="flex items-center justify-between gap-2 px-2 py-1 rounded-lg hover:bg-[#F5EEDD]/60 text-xs cursor-pointer"
                >
                  <span className="flex items-center gap-2 text-[#3A241A] font-medium">
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={isGroupLeader}
                      onChange={() => toggleGroup(g.id)}
                      className="rounded border-[#D8C7A5] text-[#264D3B] focus:ring-[#264D3B]"
                    />
                    <span className="font-mono">{g.name}</span>
                  </span>
                  {isGroupLeader && (
                    <span className="text-[10px] font-bold text-[#8A5A00]">Leader</span>
                  )}
                </label>
              );
            })}
          </div>
        </div>

        {errorMessage && <p className="text-xs font-semibold text-[#822433]">{errorMessage}</p>}
      </form>
    </M3Dialog>
  );
};
