type Env = { CLOUDFRONT_BASE_URL: string };

type PagesContext<Bindings = Record<string, unknown>> = {
  request: Request;
  env: Bindings;
  params: Record<string, string>;
  waitUntil(promise: Promise<unknown>): void;
  next(): Promise<Response>;
};

type PagesFunction<Bindings = Record<string, unknown>> = (
  context: PagesContext<Bindings>,
) => Response | Promise<Response>;

export const onRequest: PagesFunction<Env> = async ({ request, env }) => {
  if (!env.CLOUDFRONT_BASE_URL) {
    return new Response('Missing CLOUDFRONT_BASE_URL binding', { status: 500 });
  }

  const incoming = new URL(request.url);
  const origin = new URL(env.CLOUDFRONT_BASE_URL);
  if (!origin.pathname.endsWith('/')) origin.pathname += '/';

  const upstreamUrl = new URL(incoming.pathname.replace(/^\/snapshots\//, ''), origin);
  upstreamUrl.search = incoming.search;

  const upstreamResp = await fetch(new Request(upstreamUrl.toString(), request));
  const headers = new Headers(upstreamResp.headers);
  if (!headers.has('cache-control')) {
    headers.set('cache-control', 'public, max-age=300');
  }

  return new Response(upstreamResp.body, {
    status: upstreamResp.status,
    statusText: upstreamResp.statusText,
    headers,
  });
};
