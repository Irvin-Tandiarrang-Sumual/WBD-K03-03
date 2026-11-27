import React, { useState } from 'react';
import { AlertTriangle, CheckCircle2 } from 'lucide-react';
import { Outlet, redirect, useLocation, useNavigation } from 'react-router';
import { GravityFallsBackdrop } from '../assets/icons/GravityFallsBackdrop';
import { Sidebar } from '../components/layout/Sidebar';
import { TopAppBar } from '../components/layout/TopAppBar';
import { useAuth } from '../context/auth-context';
import { authService } from '../services/auth_service';
import { getScenicSceneForView } from '../utils/scenes';
import type { ScenicSceneId, SnackbarData } from '../types';

export async function clientLoader({ request }: { request: Request }) {
  try {
    const user = await authService.getMe();
    const url = new URL(request.url);
    if (url.pathname.startsWith('/admin') && user.role !== 'admin') {
      throw redirect('/databases');
    }
    return { user };
  } catch (err) {
    if (err instanceof Response) throw err;
    throw redirect('/login');
  }
}

const computeSectionAndTitle = (path: string): { section: string; title: string } => {
  if (path.startsWith('/admin/databases')) {
    return { section: 'Administration', title: 'Target Databases' };
  }
  if (path.startsWith('/admin/users')) {
    return { section: 'Administration', title: 'Users Management' };
  }
  if (path.startsWith('/admin/groups')) {
    return { section: 'Administration', title: 'Access Groups' };
  }
  if (path.startsWith('/databases') && path.includes('/drafts/new')) {
    return { section: 'Proposals', title: 'New Draft' };
  }
  if (path.startsWith('/databases') && path !== '/databases') {
    return { section: 'Database', title: 'Traversal & History' };
  }
  if (path.startsWith('/drafts') && path !== '/drafts') {
    return { section: 'Proposals', title: 'Draft Consensus' };
  }
  if (path.startsWith('/drafts')) {
    return { section: 'Workspace', title: 'Draft Proposals' };
  }
  if (path.startsWith('/approvals')) {
    return { section: 'Governance', title: 'Leader Approvals' };
  }
  return { section: 'Workspace', title: 'Catalog' };
};

interface MobileNavDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

const MobileNavDrawer = ({ isOpen, onClose }: MobileNavDrawerProps) => (
  <div
    aria-hidden={!isOpen}
    className={`fixed inset-0 z-50 lg:hidden transition-opacity duration-200 ${
      isOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'
    }`}
  >
    <div
      className="absolute inset-0 bg-[#3A241A]/50 backdrop-blur-xs"
      onClick={onClose}
    />
    <aside
      aria-label="Mobile navigation drawer"
      className={`relative w-64 max-w-[82vw] bg-[#264D3B] text-[#F5EEDD] h-full shadow-2xl transform transition-transform duration-200 ease-out ${
        isOpen ? 'translate-x-0' : '-translate-x-full'
      }`}
    >
      <Sidebar closeMobile={onClose} />
    </aside>
  </div>
);

const getSnackbarStyle = (tone?: SnackbarData['tone']) => {
  switch (tone) {
    case 'maroon':
      return 'bg-[#822433] text-white border-[#691B28]';
    case 'blue':
      return 'bg-[#1E6B8C] text-white border-[#14506B]';
    case 'pine':
    default:
      return 'bg-[#264D3B] text-[#F5EEDD] border-[#1B3B2B]';
  }
};

const GlobalSnackbarToast = ({ snackbar }: { snackbar: SnackbarData }) => {
  const isMaroon = snackbar.tone === 'maroon';
  const styleClass = getSnackbarStyle(snackbar.tone);

  return (
    <div
      role="status"
      className={`fixed bottom-5 right-5 z-50 px-4 py-3 rounded-2xl shadow-xl text-xs font-bold flex items-center gap-2.5 border ${styleClass}`}
    >
      {isMaroon ? (
        <AlertTriangle className="w-4 h-4 text-[#F3B738]" />
      ) : (
        <CheckCircle2 className="w-4 h-4 text-[#F3B738]" />
      )}
      <span>{snackbar.message}</span>
    </div>
  );
};

export default function AppLayout() {
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const { snackbar, isLoading } = useAuth();
  const location = useLocation();
  const navigation = useNavigation();
  const isNavigating = navigation.state === 'loading';

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#F5EEDD] font-sans text-[#3A241A]">
        <div className="flex flex-col items-center gap-3">
          <div className="w-10 h-10 border-4 border-[#264D3B] border-t-transparent rounded-full animate-spin" />
          <p className="text-sm font-semibold">Connecting to Wedebe...</p>
        </div>
      </div>
    );
  }

  const scene: ScenicSceneId = getScenicSceneForView(true, location.pathname);
  const { section, title } = computeSectionAndTitle(location.pathname);

  return (
    <div className="min-h-screen relative overflow-x-clip font-sans text-[#3A241A] bg-gradient-to-b from-[#F5EEDD] via-[#ECE1CB] to-[#DFD0B5] flex">
      {isNavigating && (
        <div className="fixed top-0 left-0 right-0 z-50 h-1 bg-[#D8C7A5]/30 overflow-hidden">
          <div className="h-full bg-gradient-to-r from-[#264D3B] via-[#F3B738] to-[#1E6B8C] animate-pulse w-full" />
        </div>
      )}
      <GravityFallsBackdrop scene={scene} />

      <aside className="hidden lg:flex w-64 bg-[#264D3B] text-[#F5EEDD] border-r border-[#1B3B2B] flex-col justify-between shrink-0 shadow-xl sticky top-0 h-screen z-20">
        <Sidebar />
      </aside>

      <MobileNavDrawer
        isOpen={mobileDrawerOpen}
        onClose={() => setMobileDrawerOpen(false)}
      />

      <div className="flex-1 min-w-0 flex flex-col relative z-10">
        <TopAppBar
          onOpenMobileDrawer={() => setMobileDrawerOpen(true)}
          sectionTitle={section}
          pageTitle={title}
        />

        <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl w-full mx-auto">
          <Outlet />
        </main>
      </div>

      {snackbar && <GlobalSnackbarToast snackbar={snackbar} />}
    </div>
  );
}
