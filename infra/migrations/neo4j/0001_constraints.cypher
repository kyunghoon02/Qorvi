CREATE CONSTRAINT wallet_id IF NOT EXISTS
FOR (w:Wallet)
REQUIRE w.id IS UNIQUE;

CREATE CONSTRAINT wallet_chain_address IF NOT EXISTS
FOR (w:Wallet)
REQUIRE (w.chain, w.address) IS UNIQUE;

CREATE CONSTRAINT token_id IF NOT EXISTS
FOR (t:Token)
REQUIRE t.id IS UNIQUE;

CREATE CONSTRAINT token_chain_address IF NOT EXISTS
FOR (t:Token)
REQUIRE (t.chain, t.address) IS UNIQUE;

CREATE CONSTRAINT entity_id IF NOT EXISTS
FOR (e:Entity)
REQUIRE e.id IS UNIQUE;

CREATE CONSTRAINT entity_key IF NOT EXISTS
FOR (e:Entity)
REQUIRE e.entityKey IS UNIQUE;

CREATE CONSTRAINT cluster_id IF NOT EXISTS
FOR (c:Cluster)
REQUIRE c.id IS UNIQUE;

CREATE CONSTRAINT cluster_key IF NOT EXISTS
FOR (c:Cluster)
REQUIRE c.clusterKey IS UNIQUE;

CREATE INDEX wallet_chain_index IF NOT EXISTS
FOR (w:Wallet)
ON (w.chain);

CREATE INDEX wallet_address_index IF NOT EXISTS
FOR (w:Wallet)
ON (w.address);

CREATE INDEX entity_type_index IF NOT EXISTS
FOR (e:Entity)
ON (e.entityType);
