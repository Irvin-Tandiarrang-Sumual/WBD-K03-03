import type { PlatformUser, AppRole } from '~/types/auth';
import { ROLE_PERSONA_MAP } from '~/types/auth';

export const INITIAL_USERS: PlatformUser[] = [
  {
    id: 'usr-1',
    username: 'admin',
    displayName: 'Admin',
    email: 'admin@wedebe.dev',
    role: 'admin',
    avatarInitials: 'AD',
    avatarColor: 'bg-[#264D3B] text-[#F5EEDD]',
  },
  {
    id: 'usr-2',
    username: 'leader_alice',
    displayName: 'Alice',
    email: 'alice@wedebe.dev',
    role: 'user',
    avatarInitials: 'LA',
    avatarColor: 'bg-[#1E6B8C] text-[#F5EEDD]',
  },
  {
    id: 'usr-3',
    username: 'hannah.okafor',
    displayName: 'Hannah',
    email: 'hannah.okafor@wedebe.dev',
    role: 'user',
    avatarInitials: 'HO',
    avatarColor: 'bg-[#3A241A] text-[#F5EEDD]',
  },
  {
    id: 'usr-4',
    username: 'irvin',
    displayName: 'Irvin',
    email: 'irvin@wedebe.dev',
    role: 'user',
    avatarInitials: 'IR',
    avatarColor: 'bg-[#264D3B] text-[#F5EEDD]',
  },
  {
    id: 'usr-5',
    username: 'soraia.mendes',
    displayName: 'Soraia',
    email: 'soraia.mendes@wedebe.dev',
    role: 'user',
    avatarInitials: 'SM',
    avatarColor: 'bg-[#F3B738] text-[#3A241A]',
  },
  {
    id: 'usr-6',
    username: 'devon.park',
    displayName: 'Devon',
    email: 'devon.park@wedebe.dev',
    role: 'user',
    avatarInitials: 'DP',
    avatarColor: 'bg-[#822433] text-[#F5EEDD]',
  },
];

let memoryUsers: PlatformUser[] = [...INITIAL_USERS];

export const fetchUsers = async (): Promise<PlatformUser[]> => {
  await new Promise((resolve) => setTimeout(resolve, 60));
  return [...memoryUsers];
};

export const fetchUserById = async (id: string): Promise<PlatformUser | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 40));
  return memoryUsers.find((u) => u.id === id);
};

export const fetchUserByUsername = async (username: string): Promise<PlatformUser | undefined> => {
  await new Promise((resolve) => setTimeout(resolve, 40));
  const cleaned = username.trim().toLowerCase();
  return memoryUsers.find((u) => u.username.toLowerCase() === cleaned);
};

export const createUser = async (input: {
  username: string;
  displayName: string;
  email: string;
  role: 'admin' | 'user';
}): Promise<PlatformUser> => {
  await new Promise((resolve) => setTimeout(resolve, 100));
  const initials = input.displayName
    .split(' ')
    .map((w) => w[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();

  const colors = [
    'bg-[#264D3B] text-[#F5EEDD]',
    'bg-[#1E6B8C] text-[#F5EEDD]',
    'bg-[#F3B738] text-[#3A241A]',
    'bg-[#822433] text-[#F5EEDD]',
  ];
  const avatarColor = colors[memoryUsers.length % colors.length];

  const newUser: PlatformUser = {
    id: `usr-${Date.now()}`,
    username: input.username.toLowerCase().replace(/\s+/g, '.'),
    displayName: input.displayName,
    email: input.email,
    role: input.role,
    avatarInitials: initials || 'US',
    avatarColor,
  };

  memoryUsers = [newUser, ...memoryUsers];
  return newUser;
};

export const getUserByRolePersona = (role: AppRole): PlatformUser => {
  const userId = ROLE_PERSONA_MAP[role];
  return memoryUsers.find((u) => u.id === userId) || memoryUsers[0];
};
