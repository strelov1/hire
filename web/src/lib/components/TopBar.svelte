<script lang="ts">
  import { router } from '$lib/router.svelte';
  import { authStore } from '$lib/auth.svelte';
  import { cn } from '$lib/utils';
  import { Button } from '$lib/ui';
  import ThemeToggle from './ThemeToggle.svelte';
  import AuthDialog from './AuthDialog.svelte';
  import UserMenu from './UserMenu.svelte';

  const name = $derived(router.route.name);

  const links = [
    { href: '/', label: 'Jobs', match: ['jobs', 'job'] },
    { href: '/companies', label: 'Companies', match: ['companies', 'company'] },
  ];

  // The auth dialog lives at the layout level and always opens in sign-in
  // mode (its own footer toggle switches to register). Open state is separate
  // from mode so `mode` stays a non-null value the dialog can two-way bind.
  let dialogOpen = $state(false);
  let dialogMode = $state<'login' | 'register'>('login');
  // A failed OAuth callback redirects back with ?auth_error; surface it by
  // reopening the dialog with an inline message and cleaning the URL.
  let dialogError = $state<string | null>(null);

  function openDialog() {
    dialogMode = 'login';
    dialogError = null;
    dialogOpen = true;
  }

  if (new URLSearchParams(window.location.search).has('auth_error')) {
    dialogError = 'Sign-in failed. Please try again.';
    dialogMode = 'login';
    dialogOpen = true;
    window.history.replaceState(window.history.state, '', window.location.pathname);
  }
</script>

<header class="border-b border-border">
  <div class="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4 sm:gap-6">
    <a href="/" class="text-sm font-semibold tracking-tight">FreeHire</a>

    <nav class="flex items-center gap-4 text-sm">
      {#each links as link (link.href)}
        <a
          href={link.href}
          class={cn(
            'transition-colors hover:text-foreground',
            link.match.includes(name) ? 'text-foreground' : 'text-muted-foreground',
          )}
        >
          {link.label}
        </a>
      {/each}
    </nav>

    <div class="ml-auto flex items-center gap-3">
      {#if authStore.isAuthenticated}
        <UserMenu />
      {:else}
        <Button variant="primary" size="sm" onclick={openDialog}>Sign in</Button>
      {/if}
      <ThemeToggle />
    </div>
  </div>
</header>

{#if dialogOpen}
  <AuthDialog bind:mode={dialogMode} initialError={dialogError} onClose={() => (dialogOpen = false)} />
{/if}
