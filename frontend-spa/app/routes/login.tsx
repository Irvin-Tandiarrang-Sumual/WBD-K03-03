import React, { useState } from 'react';
import { ArrowRight } from 'lucide-react';
import { useNavigate } from 'react-router';
import { GopherDipperCutout } from '../assets/icons/GopherDipperCutout';
import { useAuth } from '../context/auth-context';
import { APP_ROLES, type AppRole } from '../types';

export default function LoginPage() {
  const [username, setUsername] = useState('irvin');
  const [password, setPassword] = useState('••••••••••••');
  const [errorMessage, setErrorMessage] = useState('');

  const { login, switchRole } = useAuth();
  const navigate = useNavigate();

  const handleLoginSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmedUsername = username.trim();

    if (!trimmedUsername) {
      setErrorMessage('Username is required.');
      return;
    }

    const isLoginSuccess = login(trimmedUsername);
    if (!isLoginSuccess) {
      setErrorMessage(`User "${trimmedUsername}" not found. Try 'irvin', 'leader_alice', or 'wendy'.`);
      return;
    }

    navigate('/databases');
  };

  const handleQuickLogin = (role: AppRole) => {
    executeRoleQuickLogin(role, login, switchRole, navigate);
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
            placeholder="irvin"
            className="w-full h-12 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-sm px-3.5 focus:outline-none focus:border-[#1E6B8C] focus:ring-2 focus:ring-[#1E6B8C]/20 transition-colors"
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
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••••••"
            className="w-full h-12 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] text-sm px-3.5 focus:outline-none focus:border-[#1E6B8C] focus:ring-2 focus:ring-[#1E6B8C]/20 transition-colors"
          />
        </div>

        {errorMessage && (
          <div className="rounded-xl px-3.5 py-2.5 text-xs font-medium bg-[#822433]/12 border border-[#822433]/35 text-[#822433]">
            {errorMessage}
          </div>
        )}

        <button
          type="submit"
          className="w-full h-12 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-white font-semibold text-sm shadow-md transition-colors cursor-pointer inline-flex items-center justify-center gap-2"
        >
          <span>Enter Console</span>
          <ArrowRight className="w-4 h-4" />
        </button>
      </form>

      <QuickTestLogin onQuickLogin={handleQuickLogin} />
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
}

const QuickTestLogin: React.FC<QuickTestLoginProps> = ({ onQuickLogin }) => (
  <div className="mt-6 pt-5 border-t border-[#EFE5D1]">
    <p className="text-[11px] font-semibold uppercase tracking-wider text-[#6B5444] text-center mb-3">
      Quick Test Login
    </p>
    <div className="grid grid-cols-3 gap-2">
      {APP_ROLES.map((role) => (
        <button
          key={role}
          type="button"
          onClick={() => onQuickLogin(role)}
          className="h-11 rounded-xl border border-[#D8C7A5] bg-[#F5EEDD]/70 hover:bg-[#264D3B] hover:text-[#F5EEDD] hover:border-[#264D3B] text-[#3A241A] text-xs font-semibold transition-colors cursor-pointer"
        >
          {role}
        </button>
      ))}
    </div>
  </div>
);

function executeRoleQuickLogin(
  role: AppRole,
  login: (username: string) => boolean,
  switchRole: (role: AppRole) => void,
  navigate: (to: string) => void
): void {
  if (role === 'Admin') {
    login('wendy');
    switchRole('Admin');
    navigate('/admin/databases');
    return;
  }

  if (role === 'Leader') {
    login('leader_alice');
    switchRole('Leader');
    navigate('/approvals');
    return;
  }

  login('irvin');
  switchRole('User');
  navigate('/databases');
}
