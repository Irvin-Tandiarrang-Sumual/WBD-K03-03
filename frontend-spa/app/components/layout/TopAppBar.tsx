import React from 'react';
import { Bell, Menu } from 'lucide-react';
import { useAuth } from '../../context/auth-context';

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
  const { user, showSnackbar } = useAuth();

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
        <button
          type="button"
          onClick={() => showSnackbar('Feature under development', 'blue')}
          aria-label="Notifications"
          title="Notifications (Feature under development)"
          className="w-11 h-11 min-w-[44px] min-h-[44px] rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:border-[#1E6B8C] flex items-center justify-center text-[#3A241A] relative cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-[#264D3B]"
        >
          <Bell className="w-4 h-4" />
        </button>
      </div>
    </header>
  );
};
