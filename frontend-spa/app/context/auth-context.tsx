import React, { createContext, useContext, useEffect, useState } from 'react';
import type { AppNotification, AppRole, PlatformUser } from '../types';
import { INITIAL_USERS } from '../services/dummy_users';

export const INITIAL_NOTIFICATIONS: AppNotification[] = [
  {
    id: 'notif-1',
    title: 'New Migration Proposal',
    body: 'Dipper Pines submitted DRF-2026-003 for inventory_prod_pg.',
    message: 'Dipper Pines submitted DRF-2026-003 for inventory_prod_pg.',
    time: '8m ago',
    targetUrl: '/approvals',
    unread: true,
    createdAtRelative: '8m ago',
  },
  {
    id: 'notif-2',
    title: 'Quarantine Auto-Recovery',
    body: 'analytics_clickhouse locked on state 423. Linear rollback required.',
    message: 'analytics_clickhouse locked on state 423. Linear rollback required.',
    time: '24m ago',
    targetUrl: '/databases/db-analytics-clickhouse',
    unread: true,
    createdAtRelative: '24m ago',
  },
  {
    id: 'notif-3',
    title: 'Unanimous Consensus Reached',
    body: 'DRF-2026-001 has been approved by 2/2 leaders and applied.',
    message: 'DRF-2026-001 has been approved by 2/2 leaders and applied.',
    time: '2h ago',
    targetUrl: '/drafts/DRF-2026-001',
    unread: false,
    createdAtRelative: '2h ago',
  },
];

interface SnackbarState {
  message: string;
  tone: 'pine' | 'maroon' | 'blue';
}

interface AuthContextType {
  user: PlatformUser;
  role: AppRole;
  isAuthenticated: boolean;
  switchRole: (role: AppRole) => void;
  login: (username: string) => boolean;
  logout: () => void;
  notifications: AppNotification[];
  unreadNotifCount: number;
  markAllNotificationsRead: () => void;
  snackbar: SnackbarState | null;
  showSnackbar: (message: string, tone?: 'pine' | 'maroon' | 'blue') => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [currentUser, setCurrentUser] = useState<PlatformUser>(INITIAL_USERS[0]); // default admin
  const [currentRole, setCurrentRole] = useState<AppRole>('Admin');
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(true);
  const [notifications, setNotifications] = useState<AppNotification[]>(INITIAL_NOTIFICATIONS);
  const [snackbar, setSnackbar] = useState<SnackbarState | null>(null);

  const showSnackbar = (message: string, tone: 'pine' | 'maroon' | 'blue' = 'pine') => {
    setSnackbar({ message, tone });
    setTimeout(() => {
      setSnackbar((prev) => (prev?.message === message ? null : prev));
    }, 3800);
  };

  const switchRole = (nextRole: AppRole) => {
    setCurrentRole(nextRole);
    const matchedUser =
      INITIAL_USERS.find((u) => {
        if (nextRole === 'Admin') return u.role === 'admin';
        if (nextRole === 'Leader') return u.id === 'usr-alice-leader';
        return u.role === 'user' && u.id !== 'usr-alice-leader';
      }) || INITIAL_USERS[0];
    setCurrentUser(matchedUser);
    showSnackbar(`Switched role to ${nextRole} (${matchedUser.displayName})`, 'blue');
  };

  const login = (username: string): boolean => {
    const found = INITIAL_USERS.find(
      (u) => u.username.toLowerCase() === username.trim().toLowerCase()
    );
    if (!found) return false;
    setCurrentUser(found);
    if (found.role === 'admin') setCurrentRole('Admin');
    else if (found.id === 'usr-alice-leader') setCurrentRole('Leader');
    else setCurrentRole('User');
    setIsAuthenticated(true);
    showSnackbar(`Welcome back, ${found.displayName}! 🌲`, 'pine');
    return true;
  };

  const logout = () => {
    setIsAuthenticated(false);
    showSnackbar('Signed out from Wedebe Console', 'maroon');
  };

  const markAllNotificationsRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, unread: false })));
  };

  const unreadNotifCount = notifications.filter((n) => n.unread).length;

  return (
    <AuthContext.Provider
      value={{
        user: currentUser,
        role: currentRole,
        isAuthenticated,
        switchRole,
        login,
        logout,
        notifications,
        unreadNotifCount,
        markAllNotificationsRead,
        snackbar,
        showSnackbar,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
};
