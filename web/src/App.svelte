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

    e.preventDefault();
    router.navigate(anchor.pathname);
    window.scrollTo(0, 0);
  }
</script>

<svelte:window onclick={onClick} onpopstate={() => router.syncFromLocation()} />

<TopBar />

<!-- The outer container is the same width on every page so the header never
     jumps between routes; narrow reading views center themselves inside it. -->
<main class="mx-auto max-w-6xl px-4 py-6">
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
