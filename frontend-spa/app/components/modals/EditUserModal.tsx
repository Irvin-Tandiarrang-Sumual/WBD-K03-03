import React, { useEffect, useState } from 'react';
import type { PlatformUser } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

export interface EditUserPayload {
  username: string;
  password?: string;
  role: 'admin' | 'user';
}

interface EditUserModalProps {
  user: PlatformUser | null;
  onClose: () => void;
  onSaveUser: (userId: string, payload: EditUserPayload) => void;
}

export const EditUserModal: React.FC<EditUserModalProps> = ({ user, onClose, onSaveUser }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState<'admin' | 'user'>('user');
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (user) {
      setUsername(user.username);
      setPassword('');
      setRole(user.role === 'admin' ? 'admin' : 'user');
      setErrorMessage('');
    }
  }, [user]);

  const handleFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!user) return;

    const cleanUsername = username.trim().toLowerCase();
    if (!cleanUsername) {
      setErrorMessage('Username is required.');
      return;
    }

    setErrorMessage('');
    onSaveUser(user.id, {
      username: cleanUsername,
      password: password.trim() ? password : undefined,
      role,
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

        {errorMessage && <p className="text-xs font-semibold text-[#822433]">{errorMessage}</p>}
      </form>
    </M3Dialog>
  );
};
