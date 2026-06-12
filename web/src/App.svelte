<script lang="ts">
  import { router } from '$lib/router.svelte';
  import TopBar from '$lib/components/TopBar.svelte';
  import JobsView from '$lib/components/JobsView.svelte';
  import JobView from '$lib/components/JobView.svelte';
  import CompaniesView from '$lib/components/CompaniesView.svelte';
  import CompanyView from '$lib/components/CompanyView.svelte';

  const route = $derived(router.route);

  // Delegate same-origin anchor clicks to the client router so internal links
  // navigate without a full reload. External links (e.g. a job's Apply URL) and
  // modified clicks fall through to default browser behavior.
  function onClick(e: MouseEvent) {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
      return;
    }
    const anchor = (e.target as HTMLElement).closest('a');
    if (!anchor) return;
    const href = anchor.getAttribute('href');
    if (!href || anchor.target === '_blank' || anchor.hasAttribute('download')) return;
    if (anchor.origin !== window.location.origin) return;
    // API URLs are never client routes: the OAuth start links must perform a
    // real navigation so the server can redirect to the provider.
    if (anchor.pathname.startsWith('/api/')) return;

    e.preventDefault();
    router.navigate(anchor.pathname);
    window.scrollTo(0, 0);
  }
</script>

<svelte:window onclick={onClick} onpopstate={() => router.syncFromLocation()} />

<!-- Column layout with min-h-svh (small-viewport height, so the mobile address
     bar never forces extra scroll) keeps the footer pinned to the bottom on
     sparse pages while letting main grow. -->
<div class="flex min-h-svh flex-col">
  <TopBar />

  <!-- The outer container is the same width on every page so the header never
       jumps between routes; narrow reading views center themselves inside it. -->
  <main class="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
    {#if route.name === 'jobs'}
      <JobsView />
    {:else}
      <div class="mx-auto max-w-3xl">
        {#if route.name === 'job'}
          <JobView slug={route.slug} />
        {:else if route.name === 'companies'}
          <CompaniesView />
        {:else if route.name === 'company'}
          <CompanyView slug={route.slug} />
        {:else}
          <p class="py-12 text-center text-sm text-muted-foreground">Page not found.</p>
        {/if}
      </div>
    {/if}
  </main>

  <footer class="border-t border-border">
    <div
      class="mx-auto flex w-full max-w-6xl flex-col gap-2 px-4 py-6 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between"
    >
      <p>An open-source aggregator for IT jobs, normalized and deduplicated from company boards.</p>
      <a
        href="https://github.com/strelov1/freehire"
        target="_blank"
        rel="noopener noreferrer"
        class="shrink-0 font-medium text-foreground transition-colors hover:text-muted-foreground"
      >
        GitHub ↗
      </a>
    </div>
  </footer>
</div>
