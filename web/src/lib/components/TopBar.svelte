<script lang="ts">
  import { router } from '$lib/router.svelte';
  import { authStore } from '$lib/auth.svelte';
  import { cn } from '$lib/utils';
  import { Button } from '$lib/ui';
  import ThemeToggle from './ThemeToggle.svelte';
  import AuthDialog from './AuthDialog.svelte';

  const name = $derived(router.route.name);

  const links = [
    { href: '/', label: 'Jobs', match: ['jobs', 'job'] },
    { href: '/companies', label: 'Companies', match: ['companies', 'company'] },
  ];

  // The auth dialog lives at the layout level; the top bar buttons open it in
  // the requested mode. Open state is separate from mode so `mode` stays a
  // non-null value the dialog can two-way bind for its sign-in/register toggle.
  let dialogOpen = $state(false);
  let dialogMode = $state<'login' | 'register'>('login');

  function open(mode: 'login' | 'register') {
    dialogMode = mode;
    dialogOpen = true;
  }
</script>

<header class="border-b border-border">
  <div class="mx-auto flex h-14 max-w-3xl items-center gap-6 px-4">
    <a href="/" class="text-sm font-semibold tracking-tight">hire</a>

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
        <span class="text-sm text-muted-foreground">{authStore.user?.email}</span>
        <Button variant="ghost" size="sm" onclick={() => authStore.logout()}>Log out</Button>
      {:else}
        <Button variant="ghost" size="sm" onclick={() => open('login')}>Sign in</Button>
        <Button variant="primary" size="sm" onclick={() => open('register')}>Register</Button>
      {/if}
      <ThemeToggle />
    </div>
  </div>
</header>

{#if dialogOpen}
  <AuthDialog bind:mode={dialogMode} onClose={() => (dialogOpen = false)} />
{/if}
