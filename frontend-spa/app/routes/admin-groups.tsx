import React from 'react';
import { ArrowRight, Crown, Database, Shield, Users } from 'lucide-react';
import { Link, useLoaderData } from 'react-router';
import { fetchGroups } from '../services/dummy_groups';
import { fetchUsers } from '../services/dummy_users';
import type { AccessGroup, PlatformUser } from '../types';

export async function clientLoader() {
  const [groups, users] = await Promise.all([fetchGroups(), fetchUsers()]);
  return { groups, users };
}

interface AccessGroupCardProps {
  group: AccessGroup;
  leader?: PlatformUser;
}

const AccessGroupCard = ({ group, leader }: AccessGroupCardProps) => (
  <article
    data-testid="group-card"
    className="h-[220px] flex flex-col justify-between p-6 bg-[#FFFDF9]/92 backdrop-blur-md border border-[#D8C7A5] rounded-2xl shadow-sm hover:shadow-md transition"
  >
    <div className="flex items-start justify-between gap-2">
      <div className="flex items-center gap-2.5 min-w-0">
        <span className="w-8 h-8 rounded-lg bg-[#264D3B]/12 text-[#264D3B] flex items-center justify-center shrink-0">
          <Shield className="w-4 h-4" />
        </span>
        <h2 className="text-base font-bold text-[#3A241A] truncate">{group.name}</h2>
      </div>
      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full bg-[#F3B738]/25 text-[#6E4900] border border-[#F3B738]/70 text-[11px] font-bold shrink-0">
        <Crown className="w-3.5 h-3.5 text-[#8A5A00]" />
        <span>{leader?.username ?? 'leader'}</span>
      </span>
    </div>

    <div className="space-y-2.5">
      <p className="text-xs text-[#6B5444] line-clamp-1">{group.description}</p>
      <div className="flex items-center gap-2 flex-wrap">
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[#1E6B8C]/12 border border-[#1E6B8C]/25 text-[#1E6B8C] text-xs font-semibold">
          <Users className="w-3.5 h-3.5" />
          <span>{group.memberIds.length} members</span>
        </span>
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[#264D3B]/12 border border-[#264D3B]/25 text-[#264D3B] text-xs font-semibold">
          <Database className="w-3.5 h-3.5" />
          <span>{group.databaseAccess.length} DBs</span>
        </span>
      </div>
    </div>

    <div className="pt-3.5 border-t border-[#EFE5D1] flex items-center justify-between gap-2">
      <span className="text-[11px] font-mono text-[#6B5444]">Consensus Group</span>
      <Link
        to={`/admin/groups/${group.id}`}
        className="h-9 px-3.5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold inline-flex items-center gap-1.5 shadow-xs transition-colors cursor-pointer"
      >
        <span>Manage Details</span>
        <ArrowRight className="w-3.5 h-3.5" />
      </Link>
    </div>
  </article>
);

export default function AdminGroupsPage() {
  const { groups, users } = useLoaderData() as {
    groups: AccessGroup[];
    users: PlatformUser[];
  };

  return (
    <section className="space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[#3A241A]">Access Groups</h1>
          <p className="text-xs text-[#6B5444] mt-0.5">
            Groups define database permission scopes and leader consensus quorums.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        {groups.map((grp) => {
          const leader = users.find((u) => u.id === grp.leaderId);
          return <AccessGroupCard key={grp.id} group={grp} leader={leader} />;
        })}
      </div>
    </section>
  );
}
