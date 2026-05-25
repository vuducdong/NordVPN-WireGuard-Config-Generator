export const KEEPALIVE_MIN = 15;
export const KEEPALIVE_MAX = 120;
export const KEEPALIVE_DEFAULT = 25;
export const DEFAULT_DNS = "103.86.96.100";

export const RATE_LIMIT_CONFIG = 100;
export const RATE_LIMIT_BATCH = 5;
export const RATE_LIMIT_WINDOW_MS = 60_000;

export const KV_DATABASE_KEY = "global:database";
export const KV_SERVERS_JSON_KEY = "global:servers_json";
export const KV_VERSION_KEY = "global:version";
export const KV_DEPLOY_TOKEN_KEY = "global:deploy_token";

export const NORDVPN_SERVERS_URL =
  "https://api.nordvpn.com/v1/servers?limit=16384&filters[servers_technologies][identifier]=wireguard_udp";
export const NORDVPN_CREDENTIALS_URL =
  "https://api.nordvpn.com/v1/users/services/credentials";

export const UPSTREAM_USER_AGENT =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36";