export function buildWireGuardConfig(
  privateKey: string,
  dns: string,
  publicKey: string,
  endpoint: string,
  keepalive: number,
): string {
  return `[Interface]
PrivateKey=${privateKey}
Address=10.5.0.2/16
DNS=${dns}

[Peer]
PublicKey=${publicKey}
AllowedIPs=0.0.0.0/0,::/0
Endpoint=${endpoint}:51820
PersistentKeepalive=${keepalive}`;
}