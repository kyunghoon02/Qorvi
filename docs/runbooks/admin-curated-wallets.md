# Admin Curated Wallets

Qorvi의 `주요 지갑 자동 인덱싱` canonical source는 Postgres의 admin curated lists입니다.

운영자가 실제로 채우는 경로는 두 가지입니다.

1. bootstrap/import
- `configs/curated-wallet-seeds.json` 같은 seed JSON을 준비
- import worker를 한 번 실행
- admin curated lists가 생성되고, discover/seed bootstrap/curated entity index가 같은 source를 사용하게 됨

2. manual admin maintenance
- 이후에는 admin curated list API를 통해 list/item을 추가
- canonical source는 계속 admin curated lists

## Recommended production workflow

### 1. seed JSON 준비

기본 파일:
- `/Users/kh/Github/Qorvi/configs/curated-wallet-seeds.json`
- `/Users/kh/Github/Qorvi/configs/probable-wallet-seeds.json`

현재 starter set은 production 보수 기준으로 다음만 포함합니다.
- exchange: public explorer-labeled wallets
- bridge: official docs에 공개된 bridge contracts
- treasury: official treasury / collector / multisig addresses

즉 founder/smart-money 같은 inferred seed는 기본 starter set에서 제외했고, 필요하면 이후 `probable` 또는 `inferred`로 별도 추가하는 방식이 맞습니다.

`probable` tier는 별도 파일로 운영합니다.
- `configs/probable-wallet-seeds.json`
- public explorer label 또는 public ENS가 있는 fund / smart-money 후보를 따로 적재
- verified starter set과 섞지 않는 것이 좋습니다

필드:
- `chain`: `evm` 또는 `solana`
- `address`
- `displayName`
- `description`
- `category`
- `trackingPriority` optional
- `candidateScore` optional
- `confidence` optional
- `tags`

예시:

```json
[
  {
    "chain": "evm",
    "address": "0x28C6c06298d514Db089934071355E5743bf21d60",
    "displayName": "Binance Hot Wallet",
    "description": "Curated exchange wallet for warm indexing.",
    "category": "exchange",
    "tags": ["featured", "exchange"]
  },
  {
    "chain": "solana",
    "address": "5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1",
    "displayName": "Smart Money Seed",
    "description": "Curated smart money wallet.",
    "category": "smart-money",
    "tags": ["featured", "smart-money"]
  }
]
```

## 2. import 실행

필수 env:
- `POSTGRES_URL`
- `NEO4J_URL`
- `NEO4J_USERNAME`
- `NEO4J_PASSWORD`
- `REDIS_URL`

optional:
- `QORVI_CURATED_WALLET_SEEDS_PATH`
  - 미지정 시 `configs/curated-wallet-seeds.json`

명령:

```bash
corepack pnpm curated:import
```

또는 production env 파일 기준:

```bash
set -a
source .env.production.seeded.draft
set +a
corepack pnpm prod:curated:import
```

probable tier를 따로 import하려면:

```bash
corepack pnpm probable:import
```

또는 production env 파일 기준:

```bash
set -a
source .env.production.seeded.draft
set +a
corepack pnpm prod:probable:import
```

이 작업이 하는 일:
- category별 admin curated watchlist 생성 또는 재사용
- wallet item 추가
- curated entity index sync

기본 list 생성 규칙:
- list name: `Curated wallets · <Category>`
- list tags: `admin-curated`, `wallet-seeds`, `seed-import`, `<category>`
- item tags: seed JSON의 `tags` + `<category>`

중복 wallet item은 skip됩니다.

운영 의미:
- `curated:import`는 bootstrap/import 전용 one-shot입니다
- 이 명령은 기존 admin curated lists를 채우는 역할이고, 주기 실행 worker를 대체하지 않습니다

## 3. discover/worker 반영

import 후에는 다음이 같은 source를 사용합니다.
- `/v1/discover/featured-wallets`
- `curated-wallet-seed-enqueue` worker
- admin curated entity index

즉 검색하지 않아도 주요 지갑을 warm state로 유지할 수 있습니다.

production에서는 `curated-wallet-seed-enqueue`를 별도 worker로 주기 실행해야 합니다.

- Render service name: `qorvi-worker-curated-seeds`
- worker mode: `curated-wallet-seed-enqueue`
- 기본 loop interval: `21600` seconds (6 hours)

수동으로 바로 enqueue만 다시 걸고 싶으면:

```bash
corepack pnpm curated:enqueue
```

또는 production env 파일 기준:

```bash
set -a
source .env.production.seeded.draft
set +a
corepack pnpm prod:curated:enqueue
```

## 4. 운영 중 수동 수정

admin curated lists API:
- `GET /v1/admin/curated-lists`
- `POST /v1/admin/curated-lists`
- `POST /v1/admin/curated-lists/:id/items`
- `DELETE /v1/admin/curated-lists/:id`
- `DELETE /v1/admin/curated-lists/:id/items/:itemId`

wallet item payload 예시:

```json
{
  "itemType": "wallet",
  "itemKey": "evm:0x28C6c06298d514Db089934071355E5743bf21d60",
  "tags": ["featured", "exchange"],
  "notes": "Large exchange wallet"
}
```

`featured` tag가 붙은 item만 discover featured surface에 우선 노출됩니다.

중요:
- 현재 `/admin` 화면은 curated lists를 읽어 보여주는 baseline만 있고, 수정 UI는 아직 없습니다
- 따라서 지금 운영자가 실제로 채우는 곳은 두 가지입니다
  1. `/Users/kh/Github/Qorvi/configs/curated-wallet-seeds.json` 수정 후 `corepack pnpm curated:import`
  2. `/Users/kh/Github/Qorvi/configs/probable-wallet-seeds.json` 수정 후 `corepack pnpm probable:import`
  3. admin curated list API 직접 호출

`entity` 라벨까지 같이 쓰고 싶으면 같은 curated list 안에 entity item도 추가해야 합니다.

```json
{
  "itemType": "entity",
  "itemKey": "curated:exchange:binance",
  "tags": ["exchange", "featured"],
  "notes": "Binance curated entity"
}
```

## Recommendation

- 초기 bootstrap은 JSON import로 한 번에 넣기
- 이후 운영 중 수정은 admin curated API로 하기
- discover에서 항상 보여야 할 지갑에는 `featured` tag를 유지하기
- production에서는 `qorvi-worker-curated-seeds`를 계속 돌려 warm indexing 상태를 유지하기
- verified-public과 probable tier는 seed 파일을 분리해 관리하기
