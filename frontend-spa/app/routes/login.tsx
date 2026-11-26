import React, { useState } from 'react';
import { ArrowRight, Loader2 } from 'lucide-react';
import { redirect, useNavigate } from 'react-router';
import { GopherDipperCutout } from '../assets/icons/GopherDipperCutout';
import { useAuth } from '../context/auth-context';
import { authService } from '../services/auth_service';
import { APP_ROLES, type AppRole } from '../types';

export async function clientLoader() {
  try {
    const user = await authService.getMe();
    if (user) {
      return redirect('/databases');
    }
  } catch {
    // Unauthenticated, stay on login page
  }
  return null;
}

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin123');
  const [errorMessage, setErrorMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { login, switchRole } = useAuth();
  const navigate = useNavigate();

  const handleLoginSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const trimmedUsername = username.trim();

    if (!trimmedUsername) {
      setErrorMessage('Username is required.');
      return;
    }

    if (!password) {
      setErrorMessage('Password is required.');
      return;
    }

    try {
      setIsSubmitting(true);
      setErrorMessage('');
      await login(trimmedUsername, password);
      navigate('/databases');
    } catch (err: any) {
      setErrorMessage(err.message || 'Login failed. Please check your credentials.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleQuickLogin = (role: AppRole) => {
    executeRoleQuickLogin(role, login, switchRole, navigate, setUsername, setPassword);
  };

  return (
    <div className="w-full bg-[#FFFDF9]/95 backdrop-blur-xl border-2 border-[#D8C7A5] rounded-3xl shadow-2xl p-6 sm:p-8">
      <LoginBrandingHeader />

      <form onSubmit={handleLoginSubmit} className="space-y-4">
        <div>
          <label htmlFor="login-username" className="block text-xs font-semibold text-[#3A241A] mb-1.5">
            Username
          </label>
          <input
            id="login-username"
            type="text"
            value={username}
            onChange={(e) => {
              setUsername(e.target.value);
              if (errorMessage) setErrorMessage('');
            }}
            placeholder="admin"
            disabled={isSubmitting}
            className="w-full h-12 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-sm px-3.5 focus:outline-none focus:border-[#1E6B8C] focus:ring-2 focus:ring-[#1E6B8C]/20 transition-colors disabled:opacity-60"
          />
        </div>

        <div>
          <label htmlFor="login-password" className="block text-xs font-semibold text-[#3A241A] mb-1.5">
            Password
          </label>
          <input
            id="login-password"
            type="password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              if (errorMessage) setErrorMessage('');
            }}
            placeholder="••••••••••••"
            disabled={isSubmitting}
            className="w-full h-12 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-sm px-3.5 focus:outline-none focus:border-[#1E6B8C] focus:ring-2 focus:ring-[#1E6B8C]/20 transition-colors disabled:opacity-60"
          />
        </div>

        {errorMessage && (
          <div className="rounded-xl px-3.5 py-2.5 text-xs font-medium bg-[#822433]/12 border border-[#822433]/35 text-[#822433]">
            {errorMessage}
          </div>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="w-full h-12 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-white font-semibold text-sm shadow-md transition-colors cursor-pointer inline-flex items-center justify-center gap-2 disabled:opacity-70"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" />
              <span>Authenticating...</span>
            </>
          ) : (
            <>
              <span>Enter Console</span>
              <ArrowRight className="w-4 h-4" />
            </>
          )}
        </button>
      </form>

      <QuickTestLogin onQuickLogin={handleQuickLogin} disabled={isSubmitting} />
    </div>
  );
}

// --- Sub-Components & Technical Helpers (Stepdown Rule) ---

const LoginBrandingHeader = () => (
  <div className="flex flex-col items-center text-center mb-7">
    <GopherDipperCutout className="w-24 h-24 mb-2" />
    <div className="inline-flex items-center gap-2">
      <h1 className="text-3xl font-bold tracking-tight text-[#3A241A]">Wedebe</h1>
      <span className="px-2 py-0.5 rounded-full bg-[#F3B738] text-[#3A241A] text-[10px] font-bold uppercase tracking-wider shadow-xs">
        Mystery Console 🌲
      </span>
    </div>
    <p className="text-xs font-medium text-[#6B5444] mt-1.5">
      The Mystery Shack Migration Console
    </p>
  </div>
);

interface QuickTestLoginProps {
  onQuickLogin: (role: AppRole) => void;
  disabled?: boolean;
}

const QuickTestLogin: React.FC<QuickTestLoginProps> = ({ onQuickLogin, disabled }) => (
  <div className="mt-6 pt-5 border-t border-[#EFE5D1]">
    <p className="text-[11px] font-semibold uppercase tracking-wider text-[#6B5444] text-center mb-3">
      Quick Test Login
    </p>
    <div className="grid grid-cols-3 gap-2">
      {APP_ROLES.map((role) => (
        <button
          key={role}
          type="button"
          disabled={disabled}
          onClick={() => onQuickLogin(role)}
          className="h-11 rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/70 hover:bg-[#264D3B] hover:text-[#F5EEDD] hover:border-[#264D3B] text-[#3A241A] text-xs font-semibold transition-colors cursor-pointer disabled:opacity-60"
        >
          {role}
        </button>
      ))}
    </div>
  </div>
);

async function executeRoleQuickLogin(
  role: AppRole,
  login: (username: string, password?: string) => Promise<boolean>,
  switchRole: (role: AppRole) => void,
  navigate: (to: string) => void,
  setUsername: (u: string) => void,
  setPassword: (p: string) => void
): Promise<void> {
  if (role === 'Admin') {
    setUsername('admin');
    setPassword('admin123');
    try {
      await login('admin', 'admin123');
      navigate('/admin/databases');
    } catch {
      // Keep credentials in form so user sees error
    }
    return;
  }

  if (role === 'Leader') {
    setUsername('leader');
    setPassword('leader123');
    try {
      await login('leader', 'leader123');
      switchRole('Leader');
      navigate('/approvals');
    } catch {
      // Keep credentials in form so user sees error
    }
    return;
  }

  const u = 'user';
  const p = 'user123';
  setUsername(u);
  setPassword(p);
  try {
    await login(u, p);
    switchRole('User');
    navigate('/databases');
  } catch {
    // Keep credentials in form so user sees error
  }
}
