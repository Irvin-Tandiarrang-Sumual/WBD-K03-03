import React from 'react';
import { useLoaderData } from 'react-router';
import { Crown, Edit3, Eye, Shield, ShieldCheck, Trash2, Users } from 'lucide-react';
import { PageHeader } from '../components/common/PageHeader';
import { AccessChangeConfirmModal } from '../components/modals/AccessChangeConfirmModal';
import { LeaderReplacementModal } from '../components/modals/LeaderReplacementModal';
import { useAdminGroupDetail } from '../hooks/useAdminGroupDetail';
import { fetchDatabases } from '../services/dummy_databases';
import { fetchGroupById } from '../services/dummy_groups';
import { fetchUsers } from '../services/dummy_users';
import type { AccessGroup, DbPermission, PlatformUser, TargetDatabase } from '../types';

export async function clientLoader({ params }: { params: { id?: string } }) {
  const groupId = params.id || 'grp-core-services';
  const [group, users, databases] = await Promise.all([
    fetchGroupById(groupId),
    fetchUsers(),
    fetchDatabases(),
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
          onRemoveMember={d.handleRemoveMember}
        />

        <DatabaseAccessCard
          databaseAccess={d.group.databaseAccess}
          databases={databases}
          onRequestAccessChange={d.handleRequestAccessChange}
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
    </section>
  );
}

// --- Sub-Components (Stepdown Rule) ---

interface MemberRosterCardProps {
  memberIds: string[];
  leaderId: string;
  users: PlatformUser[];
  onRemoveMember: (id: string) => void;
}

const MemberRosterCard: React.FC<MemberRosterCardProps> = ({
  memberIds,
  leaderId,
  users,
  onRemoveMember,
}) => (
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
            onRemove={onRemoveMember}
          />
        );
      })}
    </div>
  </div>
);

interface MemberRosterRowProps {
  member: PlatformUser;
  isLeader: boolean;
  onRemove: (id: string) => void;
}

const MemberRosterRow: React.FC<MemberRosterRowProps> = ({ member, isLeader, onRemove }) => (
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
);

interface DatabaseAccessCardProps {
  databaseAccess: { databaseId: string; permission: DbPermission }[];
  databases: TargetDatabase[];
  onRequestAccessChange: (dbId: string, currentPermission: DbPermission) => void;
}

const DatabaseAccessCard: React.FC<DatabaseAccessCardProps> = ({
  databaseAccess,
  databases,
  onRequestAccessChange,
}) => (
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
            onRequestChange={onRequestAccessChange}
          />
        );
      })}
    </div>
  </div>
);

interface DatabaseAccessRowProps {
  database: TargetDatabase;
  permission: DbPermission;
  onRequestChange: (dbId: string, currentPermission: DbPermission) => void;
}

const DatabaseAccessRow: React.FC<DatabaseAccessRowProps> = ({
  database,
  permission,
  onRequestChange,
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

      <button
        type="button"
        onClick={() => onRequestChange(database.id, permission)}
        className="h-9 px-3 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#F5EEDD] text-[#3A241A] text-xs font-semibold inline-flex items-center gap-1.5 transition-colors cursor-pointer"
      >
        <Shield className="w-3.5 h-3.5 text-[#1E6B8C]" />
        <span>Change Access</span>
      </button>
    </div>
  );
};
