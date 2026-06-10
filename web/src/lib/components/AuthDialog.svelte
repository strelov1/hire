<script lang="ts">
  import { authStore } from '$lib/auth.svelte';
  import { ApiError } from '$lib/api';
  import { Button } from '$lib/ui';

  // `mode` is bindable so the in-dialog toggle can switch between sign in and
  // register without the parent re-opening it.
  let { mode = $bindable(), onClose }: { mode: 'login' | 'register'; onClose: () => void } =
    $props();

  let email = $state('');
  let password = $state('');
  let error = $state<string | null>(null);
  let submitting = $state(false);

  const title = $derived(mode === 'login' ? 'Sign in' : 'Create account');

  function messageFor(e: unknown): string {
    if (e instanceof ApiError) {
      if (e.status === 401) return 'Invalid email or password.';
      if (e.status === 409) return 'That email is already registered.';
      if (e.status === 400) return 'Enter a valid email and a password of at least 8 characters.';
    }
    return 'Something went wrong. Please try again.';
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    submitting = true;
    try {
      await authStore[mode](email, password);
      onClose();
    } catch (err) {
      error = messageFor(err);
    } finally {
      submitting = false;
    }
  }

  function toggleMode() {
    mode = mode === 'login' ? 'register' : 'login';
    error = null;
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
  <!-- Backdrop is a real button so closing on click is keyboard-accessible. -->
  <button type="button" aria-label="Close dialog" class="absolute inset-0 bg-black/50" onclick={onClose}
  ></button>

  <div
    role="dialog"
    aria-modal="true"
    aria-label={title}
    class="relative w-full max-w-sm rounded-lg border border-border bg-background p-6 shadow-lg"
  >
    <h2 class="mb-4 text-base font-semibold tracking-tight">{title}</h2>

    <form class="flex flex-col gap-3" onsubmit={submit}>
      <label class="flex flex-col gap-1 text-sm">
        <span class="text-muted-foreground">Email</span>
        <input
          type="email"
          bind:value={email}
          required
          autocomplete="email"
          class="rounded-md border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>

      <label class="flex flex-col gap-1 text-sm">
        <span class="text-muted-foreground">Password</span>
        <input
          type="password"
          bind:value={password}
          required
          minlength={mode === 'register' ? 8 : undefined}
          autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
          class="rounded-md border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </label>

      {#if error}
        <p class="text-sm text-destructive">{error}</p>
      {/if}

      <Button type="submit" variant="primary" disabled={submitting} class="mt-1">
        {submitting ? 'Please wait…' : title}
      </Button>
    </form>

    <p class="mt-4 text-center text-sm text-muted-foreground">
      {mode === 'login' ? 'No account?' : 'Already have an account?'}
      <button
        type="button"
        onclick={toggleMode}
        class="font-medium text-foreground underline-offset-4 hover:underline"
      >
        {mode === 'login' ? 'Create one' : 'Sign in'}
      </button>
    </p>
  </div>
</div>
