import React, { useState } from 'react';
import type { AppRole } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

interface AddUserPayload {
  username: string;
  displayName: string;
  email: string;
  role: AppRole;
}

interface AddUserModalProps {
  open: boolean;
  onClose: () => void;
  onAddUser: (userData: AddUserPayload) => void;
}

export const AddUserModal: React.FC<AddUserModalProps> = ({ open, onClose, onAddUser }) => {
  const [errorMessage, setErrorMessage] = useState('');

  const handleFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = extractUserFormData(event.currentTarget);
    const validationError = validateUserPayload(payload);

    if (validationError) {
      setErrorMessage(validationError);
      return;
    }

    setErrorMessage('');
    onAddUser(payload);
    event.currentTarget.reset();
  };

  return (
    <M3Dialog
      open={open}
      onClose={onClose}
      title="Add New User"
      maxWidth="max-w-md"
      footer={
        <ModalActionButtons onClose={onClose} submitFormId="add-user-form" submitLabel="Add User" />
      }
    >
      <form id="add-user-form" onSubmit={handleFormSubmit} className="space-y-3.5">
        <div>
          <label htmlFor="u-username" className="block text-xs font-bold text-[#3A241A] mb-1">
            Username
          </label>
          <input
            id="u-username"
            name="username"
            type="text"
            placeholder="dipper.pines"
            required
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] font-mono text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>

        <div>
          <label htmlFor="u-display" className="block text-xs font-bold text-[#3A241A] mb-1">
            Display Name
          </label>
          <input
            id="u-display"
            name="displayName"
            type="text"
            placeholder="Dipper Pines"
            required
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>

        <div>
          <label htmlFor="u-email" className="block text-xs font-bold text-[#3A241A] mb-1">
            Email
          </label>
          <input
            id="u-email"
            name="email"
            type="email"
            placeholder="dipper@wedebe.dev"
            required
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>

        <div>
          <label htmlFor="u-role" className="block text-xs font-bold text-[#3A241A] mb-1">
            Global Role
          </label>
          <select
            id="u-role"
            name="role"
            defaultValue="User"
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            <option value="User">User</option>
            <option value="Leader">Leader</option>
            <option value="Admin">Admin</option>
          </select>
        </div>

        {errorMessage && <p className="text-xs font-semibold text-[#822433]">{errorMessage}</p>}
      </form>
    </M3Dialog>
  );
};

// --- Low-Level Technical Helpers (Stepdown Rule) ---

function extractUserFormData(formElement: HTMLFormElement): AddUserPayload {
  const formData = new FormData(formElement);
  return {
    username: String(formData.get('username') || '').trim().toLowerCase(),
    displayName: String(formData.get('displayName') || '').trim(),
    email: String(formData.get('email') || '').trim().toLowerCase(),
    role: (formData.get('role') as AppRole) || 'User',
  };
}

function validateUserPayload(payload: AddUserPayload): string | null {
  if (!payload.username) return 'Username is required.';
  if (!payload.displayName) return 'Display name is required.';
  if (!payload.email || !payload.email.includes('@')) return 'Valid email is required.';
  return null;
}

interface ModalActionButtonsProps {
  onClose: () => void;
  submitFormId: string;
  submitLabel: string;
}

const ModalActionButtons: React.FC<ModalActionButtonsProps> = ({
  onClose,
  submitFormId,
  submitLabel,
}) => (
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
      form={submitFormId}
      className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold shadow-sm transition-colors cursor-pointer"
    >
      {submitLabel}
    </button>
  </>
);
