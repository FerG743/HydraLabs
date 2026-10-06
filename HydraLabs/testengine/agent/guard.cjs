// Read-only network guard for the Playwright MCP browser (loaded with --init-page).
// Enforced in the browser process, outside the model's control: every request that is not GET/HEAD/OPTIONS is
// aborted and recorded. Set HYDRA_ALLOW_WRITES=1 to let writes through (never the default).
const ALLOW = process.env.HYDRA_ALLOW_WRITES === '1';

module.exports.default = async ({ page }) => {
  const context = page.context();
  if (context.__hydraGuard) return;          // one handler per context, not one per page
  context.__hydraGuard = true;
  await context.route('**/*', route => {
    const method = route.request().method();
    if (ALLOW || method === 'GET' || method === 'HEAD' || method === 'OPTIONS')
      return route.continue();
    console.error(`[hydra-guard] blocked ${method} ${route.request().url()}`);
    return route.abort('blockedbyclient');
  });
};
