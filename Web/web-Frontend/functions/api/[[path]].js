export async function onRequest(context) {
  const { request } = context;
  const url = new URL(request.url);
  const backendUrl = `https://api.nordgen.selfhoster.win${url.pathname}${url.search}`;

  const headers = new Headers(request.headers);
  headers.delete('host');

  const clientIP = request.headers.get('cf-connecting-ip') || request.headers.get('x-real-ip') || request.headers.get('x-forwarded-for');
  if (clientIP) {
    headers.set('X-Client-IP', clientIP.split(',')[0].trim());
  }

  const options = {
    method: request.method,
    headers,
    redirect: 'manual',
  };

  if (request.method !== 'GET' && request.method !== 'HEAD') {
    options.body = request.body;
  }

  try {
    const backendResponse = await fetch(backendUrl, options);
    const responseHeaders = new Headers();
    const allowedHeaders = new Set([
      'content-type',
      'content-disposition',
      'cache-control',
      'etag',
      'content-encoding',
      'vary',
      'content-length'
    ]);

    for (const [key, value] of backendResponse.headers.entries()) {
      if (allowedHeaders.has(key.toLowerCase())) {
        responseHeaders.set(key, value);
      }
    }

    const nullBodyStatuses = [101, 204, 205, 304];
    const useNullBody = nullBodyStatuses.includes(backendResponse.status);

    return new Response(useNullBody ? null : backendResponse.body, {
      status: backendResponse.status,
      statusText: backendResponse.statusText,
      headers: responseHeaders,
    });
  } catch (error) {
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}