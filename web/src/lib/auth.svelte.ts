// Auth controller. The session lives in an httpOnly cookie the browser manages,
// so this store holds no token — only the current user. `main.ts` calls
// `initAuth()` once at boot to resolve the cookie via /me; components read
// `authStore` and call login/register/logout.

import * as api from '$lib/api';
import type { User } from '$lib/types';

class AuthStore {
  user = $state<User | null>(null);

  /** True once the session cookie has been confirmed by loading its user. */
  isAuthenticated = $derived(this.user !== null);

  async login(email: string, password: string) {
    this.user = await api.login(email, password);
  }

  async register(email: string, password: string) {
    this.user = await api.register(email, password);
  }

  async logout() {
    // Best-effort: drop local session state even if the network call fails.
    try {
      await api.logout();
    } catch {
      // ignore
    }
    this.user = null;
  }
}

export const authStore = new AuthStore();

/** Resolve the session cookie on boot via /me. No cookie (or a rejected one)
 *  just leaves the user signed out. Safe to call without awaiting. */
export async function initAuth() {
  try {
    authStore.user = await api.me();
  } catch {
    authStore.user = null;
  }
}
