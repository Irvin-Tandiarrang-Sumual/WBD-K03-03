import React, { useState } from 'react';
import { Bell, Menu } from 'lucide-react';
import { Link } from 'react-router';
import { useAuth } from '../../context/auth-context';
import type { AppNotification } from '../../types';

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
    notifications,
    unreadNotifCount,
    markAllNotificationsRead,
  } = useAuth();

  return (
    <header className="h-16 bg-[#FFFDF9]/85 backdrop-blur-md border-b border-[#D8C7A5] px-4 lg:px-6 flex items-center justify-between gap-2 sm:gap-3 sticky top-0 z-30">
      <div className="flex items-center gap-2.5 sm:gap-3 min-w-0">
        <button
          type="button"
          onClick={onOpenMobileDrawer}
          aria-label="Open navigation menu"
          className="lg:hidden w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] flex items-center justify-center text-[#264D3B] cursor-pointer shrink-0 transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B]"
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
        <NotificationBellDropdown
          notifications={notifications}
          unreadCount={unreadNotifCount}
          onMarkAllRead={markAllNotificationsRead}
        />
      </div>
    </header>
  );
};

// --- Sub-Components (Notification Dropdown) ---

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
        className="w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] flex items-center justify-center text-[#3A241A] relative cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B]"
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
