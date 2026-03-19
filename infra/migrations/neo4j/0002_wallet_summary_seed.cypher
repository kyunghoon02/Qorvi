MERGE (w:Wallet {
  id: "11111111-1111-1111-1111-111111111111",
  chain: "evm",
  address: "0x1234567890abcdef1234567890abcdef12345678"
})
SET w.displayName = "Seed Whale";

MERGE (c:Cluster {
  id: "cluster_seed_whales",
  clusterKey: "cluster_seed_whales"
})
SET c.clusterType = "whale",
    c.clusterScore = 82;

MERGE (w)-[:MEMBER_OF]->(c);

MERGE (w)-[:INTERACTED_WITH {
  counterpartyCount: 11,
  lastObservedAt: datetime("2026-03-19T01:02:03Z")
}]->(:Wallet {
  id: "22222222-2222-2222-2222-222222222222",
  chain: "evm",
  address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
});
