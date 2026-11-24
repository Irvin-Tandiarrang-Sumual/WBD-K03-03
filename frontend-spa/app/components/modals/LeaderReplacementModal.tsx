import React, { useState } from 'react';
import type { AccessGroup, PlatformUser } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

interface LeaderReplacementModalProps {
  open: boolean;
  group: AccessGroup | null;
  users: PlatformUser[];
  onClose: () => void;
  onConfirm: (groupId: string, newLeaderId: string) => void;
}

export const LeaderReplacementModal: React.FC<LeaderReplacementModalProps> = ({
  open,
  group,
  users,
  onClose,
  onConfirm,
}) => {
  const [selectedLeaderId, setSelectedLeaderId] = useState('');
  const [error, setError] = useState('');

  if (!group) return null;

  const eligibleMembers = group.memberIds.filter((id) => id !== group.leaderId);

  const handleConfirm = () => {
    const leaderToPromote = selectedLeaderId || eligibleMembers[0];
    if (!leaderToPromote) {
      setError('A replacement leader must be selected.');
      return;
    }
    setError('');
    onConfirm(group.id, leaderToPromote);
  };

  return (
    <M3Dialog
      open={open}
      onClose={onClose}
      title="Mandatory Leader Replacement"
      subtitle={group.name}
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
            type="button"
            onClick={handleConfirm}
            className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold shadow-sm transition-colors cursor-pointer"
          >
            Promote & Remove Leader
          </button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="rounded-xl p-3.5 bg-[#F3B738]/20 border border-[#F3B738]/70 text-xs font-medium text-[#3A241A]">
          A group must have a leader. Select an existing member to promote as leader before removing
          the current leader.
        </div>
        <div>
          <label htmlFor="replacement-leader-select" className="block text-xs font-bold text-[#3A241A] mb-1.5">
            New Group Leader
          </label>
          <select
            id="replacement-leader-select"
            value={selectedLeaderId || eligibleMembers[0] || ''}
            onChange={(e) => {
              setSelectedLeaderId(e.target.value);
              if (error) setError('');
            }}
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            {eligibleMembers.map((id) => {
              const u = users.find((user) => user.id === id);
              return (
                <option key={id} value={id}>
                  {u ? `${u.displayName} (@${u.username})` : id}
                </option>
              );
            })}
          </select>
          {error && <p className="text-xs font-semibold text-[#822433] mt-1.5">{error}</p>}
        </div>
      </div>
    </M3Dialog>
  );
};
