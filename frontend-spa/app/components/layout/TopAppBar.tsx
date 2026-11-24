import React, { useState } from 'react';
import { Bell, Check, Menu } from 'lucide-react';
import { Link, useLocation, useNavigate } from 'react-router';
import { useAuth } from '../../context/auth-context';
import { APP_ROLES, type AppNotification, type AppRole } from '../../types';

export interface TopAppBarProps {
  onOpenMobileDrawer: () => void;
  sectionTitle?: string;
  pageTitle?: string;
}

export const TopAppBar: React.FC<TopAppBarProps> = ({
  onOpenMobileDrawer,
  sectionTitle = 'Console',
  pageTitle = 'Dashboard',
}) => {
  const {
    user,
    role,
    switchRole,
    notifications,
    unreadNotifCount,
    markAllNotificationsRead,
  } = useAuth();

  const location = useLocation();
  const navigate = useNavigate();

  const handleRoleSwitch = (nextRole: AppRole) => {
    switchRole(nextRole);
    navigate(resolveDefaultRoleRoute(nextRole));
  };

  return (
    <header className="h-16 bg-[#FFFDF9]/85 backdrop-blur-md border-b border-[#D8C7A5] px-4 lg:px-6 flex items-center justify-between gap-2 sm:gap-3 sticky top-0 z-30">
      <div className="flex items-center gap-2.5 sm:gap-3 min-w-0">
        <button
          type="button"
          onClick={onOpenMobileDrawer}
          aria-label="Open navigation menu"
          className="lg:hidden w-10 h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] flex items-center justify-center text-[#264D3B] cursor-pointer shrink-0 transition-colors"
        >
          <Menu className="w-5 h-5" />
        </button>

        <div className="min-w-0">
          <div className="text-sm font-bold text-[#3A241A] truncate">
            {sectionTitle} / {pageTitle}
          </div>
          <div className="text-xs text-[#6B5444] truncate hidden sm:block">
            Welcome to the Woods, {user.displayName}! 🌲
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1.5 sm:gap-2.5 shrink-0">
        <RoleSwitcherSegmented currentRole={role} onSelectRole={handleRoleSwitch} />

        <QuickViewNavigator currentPath={location.pathname} onNavigate={navigate} />

        <NotificationBellDropdown
          notifications={notifications}
          unreadCount={unreadNotifCount}
          onMarkAllRead={markAllNotificationsRead}
        />
      </div>
    </header>
  );
};

// --- Sub-Components & Technical Helpers (Stepdown Rule) ---

interface RoleSwitcherSegmentedProps {
  currentRole: AppRole;
  onSelectRole: (role: AppRole) => void;
}

const RoleSwitcherSegmented: React.FC<RoleSwitcherSegmentedProps> = ({
  currentRole,
  onSelectRole,
}) => (
  <div
    role="group"
    aria-label="Active role switcher"
    className="hidden md:inline-flex rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/80 p-0.5"
  >
    {APP_ROLES.map((role) => {
      const isActive = currentRole === role;
      const btnClass = isActive
        ? 'bg-[#264D3B] text-[#F5EEDD] shadow-xs'
        : 'text-[#3A241A] hover:bg-[#EFE5D1]';

      return (
        <button
          key={role}
          type="button"
          onClick={() => onSelectRole(role)}
          className={`h-8 px-3 rounded-[10px] text-xs font-semibold inline-flex items-center gap-1 transition-all cursor-pointer ${btnClass}`}
        >
          {isActive && <Check className="w-3.5 h-3.5" />}
          <span>{role}</span>
        </button>
      );
    })}
  </div>
);

interface QuickViewNavigatorProps {
  currentPath: string;
  onNavigate: (path: string) => void;
}

const QuickViewNavigator: React.FC<QuickViewNavigatorProps> = ({
  currentPath,
  onNavigate,
}) => (
  <div className="relative">
    <select
      aria-label="Navigate to screen"
      value={currentPath}
      onChange={(e) => onNavigate(e.target.value)}
      className="h-10 max-w-[138px] sm:max-w-[220px] rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] text-[#3A241A] text-xs font-semibold pl-2.5 sm:pl-3 pr-6 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
    >
      <option value="/login">00. Login Page</option>
      <option value="/admin/databases">01. Admin · Databases</option>
      <option value="/admin/users">02. Admin · Users</option>
      <option value="/admin/groups">03. Admin · Groups</option>
      <option value="/databases">05. Client DB Catalog</option>
      <option value="/drafts">10. Database Drafts</option>
      <option value="/approvals">12. Leader Approvals</option>
    </select>
  </div>
);

interface NotificationBellDropdownProps {
  notifications: AppNotification[];
  unreadCount: number;
  onMarkAllRead: () => void;
}

const NotificationBellDropdown: React.FC<NotificationBellDropdownProps> = ({
  notifications,
  unreadCount,
  onMarkAllRead,
}) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-label="Notifications"
        className="w-10 h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] flex items-center justify-center text-[#3A241A] relative cursor-pointer transition-colors"
      >
        <Bell className="w-4 h-4" />
        {unreadCount > 0 && (
          <span className="absolute top-2 right-2 w-2.5 h-2.5 rounded-full bg-[#F3B738] ring-2 ring-[#FFFDF9]" />
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 rounded-2xl bg-[#FFFDF9]/98 backdrop-blur-xl border border-[#D8C7A5] shadow-2xl overflow-hidden z-50">
          <div className="px-4 py-3 border-b border-[#EFE5D1] flex items-center justify-between bg-[#F5EEDD]/60">
            <span className="text-xs font-bold uppercase tracking-wider text-[#3A241A]">
              Journal Alerts
            </span>
            <button
              type="button"
              onClick={onMarkAllRead}
              className="text-[11px] font-semibold text-[#1E6B8C] hover:underline cursor-pointer"
            >
              Mark all read
            </button>
          </div>

          <div className="divide-y divide-[#EFE5D1] max-h-80 overflow-y-auto">
            {notifications.map((n) => (
              <NotificationItem key={n.id} notification={n} onSelect={() => setIsOpen(false)} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

interface NotificationItemProps {
  notification: AppNotification;
  onSelect: () => void;
}

const NotificationItem: React.FC<NotificationItemProps> = ({ notification, onSelect }) => (
  <Link
    to={notification.targetUrl || '/databases'}
    onClick={onSelect}
    className="w-full px-4 py-3 text-left hover:bg-[#F5EEDD]/60 transition-colors cursor-pointer flex items-start gap-2.5 block"
  >
    <span
      className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${
        notification.unread ? 'bg-[#F3B738]' : 'bg-transparent'
      }`}
    />
    <div className="min-w-0 flex-1">
      <div className="text-xs font-bold text-[#3A241A] truncate">{notification.title}</div>
      <div className="text-[11px] text-[#6B5444] line-clamp-2 mt-0.5">
        {notification.message || notification.body}
      </div>
      <div className="text-[10px] font-mono text-[#8A7361] mt-1">
        {notification.createdAtRelative || notification.time}
      </div>
    </div>
  </Link>
);

function resolveDefaultRoleRoute(role: AppRole): string {
  if (role === 'Admin') return '/admin/databases';
  if (role === 'Leader') return '/approvals';
  return '/databases';
}
