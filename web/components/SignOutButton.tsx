"use client";

import { api } from "@/lib/api";

export default function SignOutButton() {
  async function signOut() {
    try {
      await api.logout();
    } finally {
      window.location.href = "/";
    }
  }

  return (
    <button type="button" onClick={signOut} className="btn-ghost btn-sm">
      Salir
    </button>
  );
}
