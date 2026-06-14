import { redirect } from '@sveltejs/kit';

// The bare /webpage-builder route redirects to the default page's per-page
// route (/webpage-builder/10) so the URL always carries an explicit PageID.
export const load = () => {
  redirect(307, '/webpage-builder/10');
};
