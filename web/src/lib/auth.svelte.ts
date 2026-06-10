// Auth controller. Holds the bearer token (persisted in localStorage) and the
// current user. `main.ts` calls `initAuth()` once at boot to validate a stored
// token via /me; components read `authStore` and call login/register/logout.
// Mirrors the shape of theme.svelte.ts.

import * as api from '$lib/api';
import type { User } from '$lib/types';

const STORAGE_KEY = 'hire.token';

function readStoredToken(): string | null {
  return localStorage.getItem(STORAGE_KEY);
}

function persist(token: string | null) {
  try {
    if (token) localStorage.setItem(STORAGE_KEY, token);
    else localStorage.removeItem(STORAGE_KEY);
  } catch {
    // best-effort: private mode / quota
  }
}

class AuthStore {
  token = $state<string | null>(readStoredToken());
  user = $state<User | null>(null);

  /** True only once a token has been confirmed by loading its user. A stored
   *  token alone does not count — see initAuth. */
  isAuthenticated = $derived(this.user !== null);

  async login(email: string, password: string) {
    this.adopt(await api.login(email, password));
  }

  async register(email: string, password: string) {
    this.adopt(await api.register(email, password));
  }

  logout() {
    this.token = null;
    this.user = null;
    persist(null);
  }

  private adopt(result: api.AuthResult) {
    this.token = result.token;
    this.user = result.user;
    persist(result.token);
  }
}

export const authStore = new AuthStore();

/** Validate a stored token on boot via /me. A rejected (expired/invalid) token
 *  is discarded so the user appears signed out. Safe to call without awaiting. */
export async function initAuth() {
  const token = authStore.token;
  if (!token) return;
  try {
    authStore.user = await api.me(token);
  } catch {
    authStore.logout();
  }
}
