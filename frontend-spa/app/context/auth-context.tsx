import React, { createContext, useContext, useEffect, useState } from 'react';
import type { AppNotification, AppRole, PlatformUser } from '../types';
import { authService } from '../services/auth_service';

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

const GUEST_USER: PlatformUser = {
  id: '',
  username: 'guest',
  displayName: 'Guest',
  email: '',
  role: 'user',
  avatarInitials: 'GU',
  avatarColor: 'bg-[#264D3B] text-[#F5EEDD]',
};

interface AuthContextType {
  user: PlatformUser;
  role: AppRole;
  isAuthenticated: boolean;
  isLoading: boolean;
  switchRole: (role: AppRole) => void;
  login: (username: string, password?: string) => Promise<boolean>;
  logout: () => Promise<void>;
  notifications: AppNotification[];
  unreadNotifCount: number;
  markAllNotificationsRead: () => void;
  snackbar: SnackbarState | null;
  showSnackbar: (message: string, tone?: 'pine' | 'maroon' | 'blue') => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [currentUser, setCurrentUser] = useState<PlatformUser>(GUEST_USER);
  const [currentRole, setCurrentRole] = useState<AppRole>('User');
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(true);
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
    showSnackbar(`Switched role to ${nextRole}`, 'blue');
  };

  useEffect(() => {
    let isMounted = true;
    authService
      .getMe()
      .then((user) => {
        if (!isMounted) return;
        setCurrentUser(user);
        setIsAuthenticated(true);
        setCurrentRole(user.role === 'admin' ? 'Admin' : 'User');
      })
      .catch(() => {
        if (!isMounted) return;
        setCurrentUser(GUEST_USER);
        setIsAuthenticated(false);
      })
      .finally(() => {
        if (isMounted) {
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, []);

  const login = async (username: string, password = ''): Promise<boolean> => {
    try {
      const user = await authService.login(username, password);
      setCurrentUser(user);
      setIsAuthenticated(true);
      setCurrentRole(user.role === 'admin' ? 'Admin' : 'User');
      showSnackbar(`Welcome back, ${user.displayName}! 🌲`, 'pine');
      return true;
    } catch (err: any) {
      showSnackbar(err.message || 'Login failed', 'maroon');
      throw err;
    }
  };

  const logout = async (): Promise<void> => {
    try {
      await authService.logout();
    } catch {
      // Continue cleanup on network failure
    } finally {
      setCurrentUser(GUEST_USER);
      setIsAuthenticated(false);
      setCurrentRole('User');
      showSnackbar('Signed out from Wedebe Console', 'maroon');
    }
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
        isLoading,
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
