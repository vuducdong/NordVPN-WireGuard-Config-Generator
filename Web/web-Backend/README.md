# NordGen Backend Worker

A high-performance edge service for querying NordVPN WireGuard server topologies and generating configuration templates. Built on **Cloudflare Workers** and the **Hono** framework, this service handles caching, proxying, and configuration assembly with strict zero-knowledge isolation.

## Overview

This application serves as the API layer for the NordGen project. It is deployed as a serverless edge function that interfaces with NordVPN's infrastructure to retrieve server lists and exchange authentication tokens. 

By default, the backend operates as a public-key infrastructure layer. It returns structural configuration templates (`mode: "client"`) that allow frontends to hydrate cryptographic keys locally, ensuring private keys never traverse the backend network during configuration generation.

## Prerequisites

- [Bun](https://bun.sh/) or Node.js
- Cloudflare Wrangler CLI (`npm i -g wrangler`)
- A Cloudflare account with Workers and KV enabled

## Development

Install the required dependencies and start the local Wrangler development environment:

```bash
bun install
bun run dev
```

The local development server will start at `http://localhost:8787`.

## Cloudflare KV Integration

The worker depends on a Cloudflare KV namespace bound as `NORDGEN_KV` to cache NordVPN's server topology and public keys. 

Before deploying, ensure you have created a KV namespace and updated the `id` in `wrangler.jsonc`:

```jsonc
"kv_namespaces": [
  {
    "binding": "NORDGEN_KV",
    "id": "<YOUR_KV_NAMESPACE_ID>"
  }
]
```

### Database Synchronization

The server list and public keys are cached in KV. The worker uses a cron trigger (`*/15 * * * *`) to automatically refresh this data every 15 minutes. 

Alternatively, the cache can be flushed and resynced manually via the `/api/sync` endpoint using a deployment token.

## Deployment

To deploy the worker directly to Cloudflare:

```bash
bun run deploy
```

*Note: For the automated full-stack deployment pipeline involving both the worker and the frontend, use the `deploy` script located in the root workspace directory.*

## API Documentation

For detailed endpoint specifications, request/response formats, validation schemas, and client/server mode documentation, please refer to the API Documentation located in [API.md](./API.md).