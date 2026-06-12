<script lang="ts">
  import { Globe } from '@lucide/svelte';

  // A company's logo from logo.dev's name endpoint (resolves by company name, no domain
  // needed). `fallback=404` makes the API return 404 — instead of a generic monogram —
  // when it has no logo, so the image fails to load and we show the globe icon instead.
  // The globe therefore covers both "no logo found" and "request failed" (offline, API
  // down). Shared by the job row, the job page, and the company page for one look.
  // The token is a logo.dev publishable key — designed to live in the client img src.
  const LOGO_TOKEN = 'pk_OywHLHTfSOuQesyeqep_nw';

  let { name, size = 'size-4' }: { name: string; size?: string } = $props();

  let failed = $state(false);
  const src = $derived(
    `https://img.logo.dev/name/${encodeURIComponent(name)}?token=${LOGO_TOKEN}&fallback=404`,
  );

  // A new company means a fresh attempt — clear a prior failure when the name changes
  // (the company page reuses this instance across navigations).
  $effect(() => {
    void name; // re-run when the company changes
    failed = false;
  });
</script>

{#if !name || failed}
  <Globe class="{size} shrink-0 text-muted-foreground" />
{:else}
  <img
    {src}
    alt=""
    class="{size} shrink-0 rounded object-contain"
    loading="lazy"
    onerror={() => (failed = true)}
  />
{/if}
