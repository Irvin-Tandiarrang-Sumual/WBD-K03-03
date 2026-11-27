import React, { useState } from 'react';
import { useLoaderData } from 'react-router';
import { Crown, Edit3, Eye, Plus, Shield, ShieldCheck, Trash2, Users } from 'lucide-react';
import { PageHeader } from '../components/common/PageHeader';
import { ConfirmDialog } from '../components/feedback/ConfirmDialog';
import { AccessChangeConfirmModal } from '../components/modals/AccessChangeConfirmModal';
import { LeaderReplacementModal } from '../components/modals/LeaderReplacementModal';
import { useAdminGroupDetail } from '../hooks/useAdminGroupDetail';
import { databaseService } from '../services/database_service';
import { groupService } from '../services/group_service';
import { userService } from '../services/user_service';
import type { AccessGroup, DbPermission, PlatformUser, TargetDatabase } from '../types';

export async function clientLoader({ params }: { params: { id?: string } }) {
  const groupId = params.id || '1';
  const [group, users, databases] = await Promise.all([
    groupService.getGroupById(groupId),
    userService.getUsers(),
    databaseService.getAdminDatabases(),
  ]);

  if (!group) {
    throw new Response('Access Group Not Found', { status: 404 });
  }

  return { group, users, databases };
}

export default function AdminGroupDetailPage() {
  const {
    group: initialGroup,
    users,
    databases,
  } = useLoaderData() as {
    group: AccessGroup;
    users: PlatformUser[];
    databases: TargetDatabase[];
  };

  const d = useAdminGroupDetail(initialGroup, databases);

  return (
    <section className="space-y-5">
      <PageHeader
        title={d.group.name}
        subtitle={d.group.description}
        badge={{ label: `ID: ${d.group.id}`, variant: 'neutral' }}
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <MemberRosterCard
          memberIds={d.group.memberIds}
          leaderId={d.group.leaderId}
          users={users}
          onAddMember={d.handleAddMember}
          onSetLeader={d.handleSetLeader}
          onRemoveMember={d.handleRemoveMember}
        />

        <DatabaseAccessCard
          databaseAccess={d.group.databaseAccess}
          databases={databases}
          onRequestAccessChange={d.handleRequestAccessChange}
          onRequestRevokeAccess={d.handleRequestRevokeAccess}
          onGrantAccess={d.handleGrantAccess}
        />
      </div>

      <LeaderReplacementModal
        open={d.leaderReplaceOpen}
        group={d.group}
        users={users}
        onClose={() => d.setLeaderReplaceOpen(false)}
        onConfirm={d.handleConfirmLeaderReplacement}
      />

      <AccessChangeConfirmModal
        changeData={d.pendingAccessChange}
        onClose={() => d.setPendingAccessChange(null)}
        onConfirm={d.handleConfirmAccessChange}
      />

      <ConfirmDialog
        open={Boolean(d.pendingRevokeDb)}
        onClose={() => d.setPendingRevokeDb(null)}
        onConfirm={d.handleConfirmRevokeAccess}
        title="Revoke Database Access?"
        message={
          d.pendingRevokeDb
            ? `Are you sure you want to revoke access to ${d.pendingRevokeDb.dbName} from group "${d.group.name}"?`
            : ''
        }
        confirmText="Revoke Access"
        isDestructive={true}
      />
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface MemberRosterCardProps {
  memberIds: string[];
  leaderId: string;
  users: PlatformUser[];
  onAddMember: (userId: string) => void;
  onSetLeader: (userId: string) => void;
  onRemoveMember: (id: string) => void;
}

const MemberRosterCard: React.FC<MemberRosterCardProps> = ({
  memberIds,
  leaderId,
  users,
  onAddMember,
  onSetLeader,
  onRemoveMember,
}) => {
  const memberSet = new Set(memberIds);
  const availableUsers = users.filter((u) => !memberSet.has(u.id));
  const [selectedUserId, setSelectedUserId] = useState('');

  const activeUserId = availableUsers.some((u) => u.id === selectedUserId)
    ? selectedUserId
    : availableUsers[0]?.id ?? '';

  return (
    <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
      <div className="px-5 py-3.5 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex items-center justify-between">
        <h2 className="text-sm font-bold text-[#3A241A] flex items-center gap-2">
          <Users className="w-4 h-4 text-[#264D3B]" />
          <span>Member Roster ({memberIds.length})</span>
        </h2>
        <span className="text-[11px] font-semibold text-[#6B5444]">Mandatory leader rule</span>
      </div>

      <div className="divide-y divide-[#EFE5D1]">
        {memberIds.map((memberId) => {
          const member = users.find((u) => u.id === memberId);
          if (!member) return null;
          return (
            <MemberRosterRow
              key={member.id}
              member={member}
              isLeader={member.id === leaderId}
              onSetLeader={onSetLeader}
              onRemove={onRemoveMember}
            />
          );
        })}
      </div>

      {availableUsers.length > 0 && (
        <div className="px-5 py-3.5 bg-[#F5EEDD]/55 border-t border-[#D8C7A5] flex flex-wrap items-center gap-2">
          <select
            aria-label="Select user to add to group"
            value={activeUserId}
            onChange={(e) => setSelectedUserId(e.target.value)}
            className="flex-1 min-w-[160px] h-9 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] font-mono text-xs font-semibold px-2.5 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            {availableUsers.map((u) => (
              <option key={u.id} value={u.id}>
                @{u.username} ({u.role})
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={!activeUserId}
            onClick={() => activeUserId && onAddMember(activeUserId)}
            className="h-9 px-3 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-45"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Member</span>
          </button>
        </div>
      )}
    </div>
  );
};

interface MemberRosterRowProps {
  member: PlatformUser;
  isLeader: boolean;
  onSetLeader: (id: string) => void;
  onRemove: (id: string) => void;
}

const MemberRosterRow: React.FC<MemberRosterRowProps> = ({
  member,
  isLeader,
  onSetLeader,
  onRemove,
}) => (
  <div className="px-5 py-3.5 flex items-center justify-between gap-3 hover:bg-[#F5EEDD]/40 transition-colors">
    <div className="flex items-center gap-3 min-w-0">
      <div className="w-9 h-9 rounded-full bg-[#264D3B] text-[#F5EEDD] font-bold text-xs flex items-center justify-center shrink-0">
        {member.avatarInitials}
      </div>
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs font-bold text-[#3A241A] truncate">
            {member.username}
          </span>
          {isLeader && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70 text-[10px] font-bold">
              <Crown className="w-3 h-3 text-[#8A5A00]" />
              <span>Leader</span>
            </span>
          )}
        </div>
        <div className="text-xs text-[#6B5444] truncate">{member.email}</div>
      </div>
    </div>

    <div className="flex items-center gap-2 shrink-0">
      {!isLeader && (
        <button
          type="button"
          onClick={() => onSetLeader(member.id)}
          aria-label={`Set ${member.username} as leader`}
          className="h-9 px-3 rounded-xl border border-[#F3B738]/70 bg-[#F3B738]/15 hover:bg-[#F3B738]/30 text-[#6E4900] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
        >
          <Crown className="w-3.5 h-3.5 text-[#8A5A00]" />
          <span>Set Leader</span>
        </button>
      )}
      <button
        type="button"
        onClick={() => onRemove(member.id)}
        aria-label={`Remove ${member.username}`}
        className="h-9 px-3 rounded-xl border border-[#822433]/35 bg-[#822433]/8 hover:bg-[#822433] hover:text-white text-[#822433] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer shrink-0"
      >
        <Trash2 className="w-3.5 h-3.5" />
        <span>Remove</span>
      </button>
    </div>
  </div>
);

interface DatabaseAccessCardProps {
  databaseAccess: { databaseId: string; permission: DbPermission }[];
  databases: TargetDatabase[];
  onRequestAccessChange: (dbId: string, currentPermission: DbPermission) => void;
  onRequestRevokeAccess: (dbId: string) => void;
  onGrantAccess: (dbId: string, permission: DbPermission) => void;
}

const DatabaseAccessCard: React.FC<DatabaseAccessCardProps> = ({
  databaseAccess,
  databases,
  onRequestAccessChange,
  onRequestRevokeAccess,
  onGrantAccess,
}) => {
  const assignedIds = new Set(databaseAccess.map((acc) => acc.databaseId));
  const unassignedDbs = databases.filter((db) => !assignedIds.has(db.id));
  const [selectedDbId, setSelectedDbId] = useState('');
  const [selectedPermission, setSelectedPermission] = useState<DbPermission>('view_only');

  const activeSelectId = unassignedDbs.some((d) => d.id === selectedDbId)
    ? selectedDbId
    : unassignedDbs[0]?.id ?? '';

  return (
    <div className="bg-[#FFFDF9]/95 backdrop-blur-md rounded-2xl border border-[#D8C7A5] shadow-xs overflow-hidden">
      <div className="px-5 py-3.5 bg-[#EFE5D1]/55 border-b border-[#D8C7A5] flex items-center justify-between">
        <h2 className="text-sm font-bold text-[#3A241A] flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-[#1E6B8C]" />
          <span>Database Access Control</span>
        </h2>
        <span className="text-[11px] font-semibold text-[#6B5444]">Confirmation required</span>
      </div>

      <div className="divide-y divide-[#EFE5D1]">
        {databaseAccess.map((acc) => {
          const db = databases.find((d) => d.id === acc.databaseId);
          if (!db) return null;
          return (
            <DatabaseAccessRow
              key={acc.databaseId}
              database={db}
              permission={acc.permission}
              canRevoke={databaseAccess.length > 1}
              onRequestChange={onRequestAccessChange}
              onRequestRevoke={onRequestRevokeAccess}
            />
          );
        })}
      </div>

      {unassignedDbs.length > 0 && (
        <div className="px-5 py-3.5 bg-[#F5EEDD]/55 border-t border-[#D8C7A5] flex flex-wrap items-center gap-2">
          <select
            aria-label="Select database to grant access"
            value={activeSelectId}
            onChange={(e) => setSelectedDbId(e.target.value)}
            className="flex-1 min-w-[140px] h-9 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] font-mono text-xs font-semibold px-2.5 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            {unassignedDbs.map((db) => (
              <option key={db.id} value={db.id}>
                {db.name} ({db.engine})
              </option>
            ))}
          </select>
          <select
            aria-label="Select permission level"
            value={selectedPermission}
            onChange={(e) => setSelectedPermission(e.target.value as DbPermission)}
            className="h-9 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-2.5 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            <option value="view_only">view_only</option>
            <option value="view_edit">view_edit</option>
          </select>
          <button
            type="button"
            disabled={!activeSelectId}
            onClick={() => activeSelectId && onGrantAccess(activeSelectId, selectedPermission)}
            className="h-9 px-3 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-45"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Grant Access</span>
          </button>
        </div>
      )}
    </div>
  );
};

interface DatabaseAccessRowProps {
  database: TargetDatabase;
  permission: DbPermission;
  canRevoke: boolean;
  onRequestChange: (dbId: string, currentPermission: DbPermission) => void;
  onRequestRevoke: (dbId: string) => void;
}

const DatabaseAccessRow: React.FC<DatabaseAccessRowProps> = ({
  database,
  permission,
  canRevoke,
  onRequestChange,
  onRequestRevoke,
}) => {
  const isEdit = permission === 'view_edit';
  const badgeClass = isEdit
    ? 'bg-[#264D3B]/15 text-[#264D3B] border border-[#264D3B]/35'
    : 'bg-[#1E6B8C]/12 text-[#1E6B8C] border border-[#1E6B8C]/30';

  return (
    <div className="px-5 py-3.5 flex items-center justify-between gap-3 hover:bg-[#F5EEDD]/40 transition-colors">
      <div className="min-w-0">
        <div className="font-mono text-xs font-bold text-[#3A241A] truncate">
          {database.name}
        </div>
        <div className="mt-1 flex items-center gap-2">
          <span className="text-[11px] text-[#6B5444]">{database.engine}</span>
          <span
            className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${badgeClass}`}
          >
            {isEdit ? <Edit3 className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
            <span>{permission}</span>
          </span>
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <button
          type="button"
          onClick={() => onRequestChange(database.id, permission)}
          className="h-9 px-3 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
        >
          <Shield className="w-3.5 h-3.5 text-[#1E6B8C]" />
          <span>Change Access</span>
        </button>

        <button
          type="button"
          disabled={!canRevoke}
          title={canRevoke ? `Revoke access to ${database.name}` : 'A group must have at least 1 database'}
          onClick={() => onRequestRevoke(database.id)}
          aria-label={`Revoke ${database.name} access`}
          className="h-9 px-3 rounded-xl border border-[#822433]/35 bg-[#822433]/8 hover:bg-[#822433] hover:text-white text-[#822433] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <Trash2 className="w-3.5 h-3.5" />
          <span>Revoke</span>
        </button>
      </div>
    </div>
  );
};
