import React from 'react';
import {
  Crown,
  Database,
  GitPullRequest,
  LogOut,
  Server,
  Shield,
  Terminal,
  Users,
  X,
} from 'lucide-react';
import { NavLink, useNavigate } from 'react-router';
import { GopherDipperCutout } from '../../assets/icons/GopherDipperCutout';
import { useAuth } from '../../context/auth-context';
import type { AppRole, PlatformUser } from '../../types';

export interface SidebarProps {
  closeMobile?: () => void;
  pendingApprovalsCount?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  closeMobile,
  pendingApprovalsCount = 1,
}) => {
  const { user, role, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
    if (closeMobile) closeMobile();
  };

  return (
    <div className="h-full flex flex-col justify-between p-4 sm:p-5 overflow-y-auto">
      <div>
        <SidebarBrandHeader onCloseMobile={closeMobile} />

        <nav aria-label="Sidebar Navigation" className="space-y-5">
          <SidebarNavSections
            role={role}
            pendingApprovalsCount={pendingApprovalsCount}
            onItemClick={closeMobile}
          />
        </nav>
      </div>

      <div className="pt-4 mt-4 border-t border-[#F5EEDD]/15 space-y-3">
        <SidebarUserCard user={user} role={role} onLogout={handleLogout} />
      </div>
    </div>
  );
};

// --- Sub-Components (Stepdown Rule) ---

interface SidebarBrandHeaderProps {
  onCloseMobile?: () => void;
}

const SidebarBrandHeader: React.FC<SidebarBrandHeaderProps> = ({ onCloseMobile }) => (
  <div className="flex items-center justify-between gap-3 mb-6 pb-4 border-b border-[#F5EEDD]/15">
    <NavLink
      to="/databases"
      onClick={onCloseMobile}
      className="flex items-center gap-2.5 text-left cursor-pointer group min-w-0"
    >
      <GopherDipperCutout className="w-11 h-11" />
      <div className="min-w-0">
        <span className="text-[19px] font-bold tracking-tight text-[#F5EEDD] block">
          Wedebe
        </span>
        <span className="mt-0.5 inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-[#F3B738] text-[#3A241A] text-[10px] font-bold tracking-wide">
          Mystery Console 🌲
        </span>
      </div>
    </NavLink>
    {onCloseMobile && (
      <button
        type="button"
        onClick={onCloseMobile}
        aria-label="Close navigation menu"
        className="w-11 h-11 min-w-[44px] min-h-[44px] rounded-full flex items-center justify-center text-[#F5EEDD]/80 hover:text-white hover:bg-[#1B3B2B] cursor-pointer focus-visible:ring-2 focus-visible:ring-[#F3B738]"
      >
        <X className="w-5 h-5" />
      </button>
    )}
  </div>
);

interface SidebarNavSectionsProps {
  role: AppRole;
  pendingApprovalsCount: number;
  onItemClick?: () => void;
}

const SidebarNavSections: React.FC<SidebarNavSectionsProps> = ({
  role,
  pendingApprovalsCount,
  onItemClick,
}) => {
  if (role === 'Admin') {
    return (
      <>
        <div className="space-y-1">
          <SectionLabel text="Administration" />
          <NavLinkItem label="Databases" to="/admin/databases" icon={<Server className="w-4 h-4" />} onClick={onItemClick} />
          <NavLinkItem label="Users" to="/admin/users" icon={<Users className="w-4 h-4" />} onClick={onItemClick} />
          <NavLinkItem label="Groups" to="/admin/groups" icon={<Shield className="w-4 h-4" />} onClick={onItemClick} />
        </div>
        <div className="space-y-1">
          <SectionLabel text="Client Workspace" />
          <NavLinkItem label="Client DB Catalog" to="/databases" icon={<Database className="w-4 h-4" />} onClick={onItemClick} />
          <NavLinkItem label="SQL Console" to="/console" icon={<Terminal className="w-4 h-4" />} onClick={onItemClick} />
          <NavLinkItem label="Draft Proposals" to="/drafts" icon={<GitPullRequest className="w-4 h-4" />} onClick={onItemClick} />
          <NavLinkItem label="Approvals" to="/approvals" icon={<Crown className="w-4 h-4" />} badgeCount={pendingApprovalsCount} onClick={onItemClick} />
        </div>
      </>
    );
  }

  if (role === 'Leader') {
    return (
      <div className="space-y-1">
        <SectionLabel text="Leader Workspace" />
        <NavLinkItem label="Client DB Catalog" to="/databases" icon={<Database className="w-4 h-4" />} onClick={onItemClick} />
        <NavLinkItem label="SQL Console" to="/console" icon={<Terminal className="w-4 h-4" />} onClick={onItemClick} />
        <NavLinkItem label="Draft Proposals" to="/drafts" icon={<GitPullRequest className="w-4 h-4" />} onClick={onItemClick} />
        <NavLinkItem
          label="Approvals"
          to="/approvals"
          icon={<Crown className="w-4 h-4" />}
          badgeCount={pendingApprovalsCount}
          onClick={onItemClick}
        />
      </div>
    );
  }

  return (
    <div className="space-y-1">
      <SectionLabel text="Workspace" />
      <NavLinkItem label="Client DB Catalog" to="/databases" icon={<Database className="w-4 h-4" />} onClick={onItemClick} />
      <NavLinkItem label="SQL Console" to="/console" icon={<Terminal className="w-4 h-4" />} onClick={onItemClick} />
      <NavLinkItem label="Draft Proposals" to="/drafts" icon={<GitPullRequest className="w-4 h-4" />} onClick={onItemClick} />
    </div>
  );
};

const SectionLabel: React.FC<{ text: string }> = ({ text }) => (
  <div className="px-3.5 pb-1.5 text-[11px] font-semibold uppercase tracking-wider text-[#F5EEDD]/65">
    {text}
  </div>
);

interface NavLinkItemProps {
  label: string;
  to: string;
  icon: React.ReactNode;
  badgeCount?: number;
  onClick?: () => void;
}

const NavLinkItem: React.FC<NavLinkItemProps> = ({ label, to, icon, badgeCount, onClick }) => (
  <NavLink
    to={to}
    onClick={onClick}
    className={({ isActive }) =>
      `w-full min-h-[44px] flex items-center justify-between px-3.5 py-2.5 rounded-xl text-xs font-semibold transition-all focus-visible:ring-2 focus-visible:ring-[#F3B738] ${
        isActive
          ? 'bg-[#1B3B2B] text-[#F5EEDD] shadow-inner font-bold'
          : 'text-[#F5EEDD]/80 hover:text-white hover:bg-[#1B3B2B]/60'
      }`
    }
  >
    <div className="flex items-center gap-2.5 min-w-0">
      <span className="shrink-0">{icon}</span>
      <span className="truncate">{label}</span>
    </div>
    {typeof badgeCount === 'number' && badgeCount > 0 && (
      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-[#F3B738] text-[#3A241A] shrink-0">
        {badgeCount}
      </span>
    )}
  </NavLink>
);

interface SidebarUserCardProps {
  user: PlatformUser;
  role: AppRole;
  onLogout: () => void;
}

const SidebarUserCard: React.FC<SidebarUserCardProps> = ({ user, role, onLogout }) => (
  <div className="rounded-2xl bg-[#1B3B2B] border border-[#F5EEDD]/20 p-3 flex items-center justify-between gap-2.5">
    <div className="flex items-center gap-2.5 min-w-0">
      <div className="w-9 h-9 rounded-full bg-[#F5EEDD] text-[#264D3B] font-bold text-xs flex items-center justify-center shrink-0">
        {user.avatarInitials}
      </div>
      <div className="min-w-0">
        <div className="text-xs font-bold text-[#F5EEDD] truncate">{user.username}</div>
        <div className="inline-flex items-center gap-1 text-[10px] font-semibold text-[#F3B738]">
          {role === 'Leader' && <Crown className="w-3 h-3" />}
          <span>{role}</span>
        </div>
      </div>
    </div>
    <button
      type="button"
      onClick={onLogout}
      aria-label="Sign out"
      title="Sign out"
      className="w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl flex items-center justify-center text-[#F5EEDD]/75 hover:text-white hover:bg-[#264D3B] transition-colors cursor-pointer shrink-0 focus-visible:ring-2 focus-visible:ring-[#F3B738]"
    >
      <LogOut className="w-4 h-4" />
    </button>
  </div>
);
