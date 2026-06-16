export type AddressLabel = {
  name: string;
  category: "defi" | "cex" | "bridge" | "token" | "known_contract";
};

export const protocolLabels: Record<string, AddressLabel> = {
  "0x1f98431c8ad98523631ae4a59f267346ea31f984": {
    name: "Uniswap V3 Factory",
    category: "defi",
  },
  "0xe592427a0aece92de3edee1f18e0157c05861564": {
    name: "Uniswap V3 Router",
    category: "defi",
  },
  "0x1111111254eeb25477b68fb85ed929f73a960582": {
    name: "1inch Router",
    category: "defi",
  },
  "0x7be8076f4ea4a4ad08075c2508e481d6c946d12b": {
    name: "OpenSea Seaport",
    category: "known_contract",
  },
  "0x7d2768de32b0b80b7a3454c06bdac6c6a9a7f2b7": {
    name: "Aave V2 Lending Pool",
    category: "defi",
  },
  "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2": {
    name: "Aave V3 Pool",
    category: "defi",
  },
  "0xae7ab96520de3a18e5e111b5eaab095312d7fe84": {
    name: "Lido stETH",
    category: "defi",
  },
  "0xdc24316b9ae028f1497c275eb9192a3ea0f67022": {
    name: "Curve stETH Pool",
    category: "defi",
  },
  "0x3d9819210a31b4961b30ef54be2aed79b9c9cd3b9": {
    name: "Compound Comptroller",
    category: "defi",
  },
  "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2": {
    name: "Maker MKR",
    category: "defi",
  },
  "0xd9e1ce17f2641f24ae83637ab66a2cca9c378b9f": {
    name: "SushiSwap Router",
    category: "defi",
  },
  "0x858646372cc42e1a627fce94aa7a7033e7cf075a": {
    name: "EigenLayer Strategy Manager",
    category: "defi",
  },
};

export const cexLabels: Record<string, AddressLabel> = {
  "0x28c6c06298d514db089934071355e5743bf21d60": {
    name: "Binance 14",
    category: "cex",
  },
  "0x21a31ee1afc51d94c2efccaa2092ad1028285549": {
    name: "Binance 15",
    category: "cex",
  },
  "0xdfd5293d8e347dfe59e90efd55b2956a1343963d": {
    name: "Binance 16",
    category: "cex",
  },
  "0x3f5ce5fbfe3e9af3971d580e5b1eefa7cc63dc72": {
    name: "Binance",
    category: "cex",
  },
  "0x71660c4005ba85c37ccec55d0c4493e66fe775d3": {
    name: "Coinbase",
    category: "cex",
  },
  "0x503828976d22510aad0201ac7ec88293211d23da": {
    name: "Coinbase",
    category: "cex",
  },
  "0x5e575279bf9f4acf0a130c186861454247394c06": {
    name: "Kraken",
    category: "cex",
  },
  "0xf89d7b9c864f589bbf53a82105107622b35eaa40": {
    name: "Bybit",
    category: "cex",
  },
};

// Ethereum mainnet L1 contracts from each bridge's published deployment list.
// Only an exact allowlist match is surfaced as confirmed bridge activity.
export const bridgeLabels: Record<
  string,
  AddressLabel & { destinationChain: string }
> = {
  "0xbeb5fc579115071764c7423a4f12edde41f106ed": {
    name: "Optimism Portal",
    category: "bridge",
    destinationChain: "OP Mainnet",
  },
  "0x25ace71c97b33cc4729cf772ae268934f7ab5fa1": {
    name: "Optimism L1 CrossDomainMessenger",
    category: "bridge",
    destinationChain: "OP Mainnet",
  },
  "0x99c9fc46f92e8a1c0dec1b1747d010903e884be1": {
    name: "Optimism Standard Bridge",
    category: "bridge",
    destinationChain: "OP Mainnet",
  },
  "0x49048044d57e1c92a77f79988d21fa8faf74e97e": {
    name: "Base OptimismPortal",
    category: "bridge",
    destinationChain: "Base",
  },
  "0x866e82a600a1414e583f7f13623f1ac5d58b0afa": {
    name: "Base L1 CrossDomainMessenger",
    category: "bridge",
    destinationChain: "Base",
  },
  "0x3154cf16ccdb4c6d922629664174b904d80f2c35": {
    name: "Base Standard Bridge",
    category: "bridge",
    destinationChain: "Base",
  },
  "0x72ce9c846789fdb6fc1f34ac4ad25dd9ef7031ef": {
    name: "Arbitrum One L1 Gateway Router",
    category: "bridge",
    destinationChain: "Arbitrum One",
  },
  "0x4dbd4fc535ac27206064b68ffcf827b0a60bab3f": {
    name: "Arbitrum One Inbox",
    category: "bridge",
    destinationChain: "Arbitrum One",
  },
  "0x0b9857ae2d4a3dbe74ffe1d7df045bb7f96e4840": {
    name: "Arbitrum One Outbox",
    category: "bridge",
    destinationChain: "Arbitrum One",
  },
  "0x5c7bcd6e7de5423a257d81b442095a1a6ced35c5": {
    name: "Across Ethereum SpokePool",
    category: "bridge",
    destinationChain: "Destination encoded by Across route",
  },
  "0x8731d54e9d02c286767d56ac03e8037c07e01e98": {
    name: "Stargate Router",
    category: "bridge",
    destinationChain: "Destination encoded by Stargate route",
  },
  "0x150f94b44927f078737562f0fcf3c95c01cc2376": {
    name: "Stargate RouterETH",
    category: "bridge",
    destinationChain: "Destination encoded by Stargate route",
  },
  "0x296f55f8fb28e498b858d0bcda06d955b2cb3f97": {
    name: "Stargate Bridge",
    category: "bridge",
    destinationChain: "Destination encoded by Stargate route",
  },
};

export function normalizeAddress(address: string): string {
  return address.trim().toLowerCase();
}

export function compactAddress(address: string): string {
  const normalized = address.trim();
  if (normalized.length <= 14) {
    return normalized;
  }
  return `${normalized.slice(0, 6)}...${normalized.slice(-4)}`;
}

export function getKnownLabel(address: string): AddressLabel | null {
  const key = normalizeAddress(address);
  return protocolLabels[key] ?? cexLabels[key] ?? bridgeLabels[key] ?? null;
}
