import { error } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

// Server-render the job detail: fetch by slug so the article content is in the
// initial HTML. A 404 from the API becomes a SvelteKit 404 page (not a 200
// shell); other failures bubble to the 500 page.
export const load: PageServerLoad = async ({ params, fetch }) => {
  try {
    const job = await serverApi(fetch).getJob(params.slug);
    return { job };
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      error(404, 'Job not found');
    }
    throw e;
  }
};
