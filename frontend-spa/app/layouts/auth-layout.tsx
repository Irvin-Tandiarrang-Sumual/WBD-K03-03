import React from 'react';
import { Outlet } from 'react-router';
import { GravityFallsBackdrop } from '../assets/icons/GravityFallsBackdrop';

export default function AuthLayout() {
  return (
    <div className="min-h-screen relative overflow-x-hidden font-sans text-[#3A241A] bg-gradient-to-b from-[#F5EEDD] via-[#ECE1CB] to-[#DFD0B5] flex flex-col justify-center items-center p-4">
      <GravityFallsBackdrop scene="valley_trestle" />
      <div className="w-full max-w-md relative z-10">
        <Outlet />
      </div>
    </div>
  );
}
