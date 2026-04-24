# Qorvi Domestic Pre-listing Monitor Plan

## 0. 목적

이 문서는 Qorvi에 `업비트/빗썸 미상장 토큰 중 상장 전형 시그널을 보이는 후보를 탐지하고 운영자가 추적할 수 있는 token-first monitor`를 추가하기 위한 실행 계획서다.

핵심 목표는 단순한 `상장 예측`이 아니다. 다음 세 가지를 동시에 만족해야 한다.

1. `상장 여부 정확성`
   - 업비트/빗썸에 이미 상장된 자산을 후보로 잘못 노출하지 않는다.
2. `후보 품질`
   - 미상장 토큰 중에서도 실제로 볼 가치가 있는 `대규모 이동`, `quality wallet overlap`, `거래량/유동성 급증` 후보만 상위에 올린다.
3. `운영 가능성`
   - 운영자가 `/admin`과 전용 feed에서 후보를 검토하고, false positive를 억제하고, watchlist/alert로 넘길 수 있어야 한다.

## 1. 제품 정의

이 기능의 정식 의미는 아래와 같다.

`국내 거래소(업비트/빗썸) 미상장 토큰 중, 온체인과 DEX activity 기준으로 listing-adjacent behavior를 보이는 후보를 랭킹한다.`

중요한 점:

- 이 기능은 `상장 확정`을 말하지 않는다.
- 이 기능은 `상장 가능성이 높아 보이는 전형적 사전 시그널`을 보여준다.
- UI와 AI explanation 모두 `prediction`보다 `evidence-led candidate monitor` 톤을 유지해야 한다.

## 2. 왜 이 기능이 필요한가

현재 Qorvi는 wallet-first 분석이 강하다.

- smart money / seed whale / shadow exit / first connection은 wallet-centric engine이다.
- 하지만 한국 시장에서 실사용 가치는 `어떤 토큰이 아직 국내 거래소에 없는데, 상장 전에 보이는 패턴을 형성하고 있느냐`에 크게 좌우된다.

즉 다음 전환이 필요하다.

`wallet-first alpha surface -> token-first domestic pre-listing monitor`

이 기능은 다음 유저에게 직접적인 가치를 준다.

1. 한국 거래소 상장 전 움직임을 보고 싶은 trader
2. smart money의 초기 진입을 token 단위로 보고 싶은 researcher
3. 운영자가 `이 토큰은 왜 올라왔는가`를 evidence로 검토하고 싶은 analyst

## 3. 제품 원칙

### 3.1 식별 원칙

반드시 `chain + token contract` 기준으로 식별한다.

- symbol 기준 비교 금지
- name 기준 fuzzy match 금지
- 거래소 상장 여부는 가능한 한 canonical asset registry를 통해 연결

### 3.2 노출 원칙

후보 노출은 다음 조건을 같이 만족해야 한다.

1. 국내 거래소 미상장
2. 토큰 activity 관점에서 최근 시그널 존재
3. 단순 noise가 아니라 threshold 이상 evidence 존재

### 3.3 wording 원칙

다음 wording은 피한다.

- `상장 예정`
- `곧 업비트 상장`
- `빗썸 상장 확률`

다음 wording을 사용한다.

- `국내 미상장 후보`
- `listing-adjacent activity`
- `pre-listing accumulation`
- `large onchain movement before domestic listing`

## 4. 데이터 소스

### 4.1 국내 거래소 상장 목록

상장 여부는 거래소 공식 거래 대상 목록 API를 기준으로 관리한다.

- Upbit: `GET /v1/market/all`
- Bithumb: `GET /v1/market/all`

필요 역할:

1. 거래소별 현재 상장 자산 universe 수집
2. KRW/BTC/USDT 등 마켓 prefix 제거 후 base asset 정규화
3. 내부 token registry와 매핑

### 4.2 토큰 메타데이터 / contract 보강

내부 tokens 테이블만으로 부족할 수 있으므로 contract registry 보강이 필요하다.

우선순위:

1. internal observed tokens
2. CoinGecko contract map
3. DEX Screener token/pair metadata

### 4.3 activity / signal 소스

후보 랭킹의 입력은 기존 Qorvi 엔진을 재사용한다.

- large transfer / large holder movement
- smart money overlap
- first-entry / early-conviction signal
- volume / liquidity expansion
- deployer / treasury / MM linked flow
- wallet-quality overlap

## 5. 시스템 구조

전체 흐름:

`exchange market registry -> token registry -> domestic unlisted filter -> token activity scoring -> candidate feed -> watchlist / alert / analyst drilldown`

### 5.1 Exchange Listing Registry

새 read model:

- `exchange_listing_registry`

필드 초안:

- `exchange`
- `market`
- `base_symbol`
- `quote_symbol`
- `display_name`
- `chain_hint`
- `token_address`
- `normalized_asset_key`
- `listed`
- `listed_at_detected`
- `last_checked_at`
- `metadata`

역할:

- 업비트/빗썸 상장 여부의 canonical source
- 미상장 필터의 기준 데이터

### 5.2 Domestic Unlisted Token Registry

새 read model:

- `domestic_unlisted_tokens_daily`

필드 초안:

- `chain`
- `token_address`
- `symbol`
- `name`
- `coingecko_id`
- `dexscreener_primary_pair`
- `listed_upbit`
- `listed_bithumb`
- `domestic_listed`
- `first_seen_at`
- `last_seen_at`
- `metadata`

역할:

- 내부 token universe 중 국내 미상장 토큰 집합을 daily snapshot으로 유지

### 5.3 Token Activity Feature Store

새 feature store:

- `token_entry_features_daily`
- `token_large_movement_features_daily`
- `token_market_activity_features_daily`

핵심 feature:

- `large_transfer_count_24h`
- `large_transfer_usd_24h`
- `smart_money_wallet_count_24h`
- `quality_wallet_overlap_score`
- `new_wallet_entry_count_24h`
- `repeat_entry_quality`
- `dex_volume_usd_24h`
- `dex_volume_change_24h`
- `liquidity_usd`
- `liquidity_change_24h`
- `deployer_treasury_outflow_score`
- `exchange_adjacent_flow_score`
- `first_entry_before_crowding_score`

### 5.4 Candidate Feed

새 finding/feed 계층:

- `pre_listing_candidate`

노출 기준:

- `domestic_listed = false`
- 최근 24h 또는 7d 내 feature threshold 충족
- suppression rule 비적용

feed item 포함 항목:

- token name / symbol / chain
- contract
- why this token surfaced
- recent large movement summary
- smart money overlap
- DEX activity summary
- confidence / caution
- next watch

## 6. 우선 구현 범위

### Phase 1

목표: `국내 미상장 여부 + 큰 움직임 기반 후보 목록`

포함:

1. 업비트/빗썸 상장 목록 sync
2. 국내 미상장 토큰 필터
3. 최근 큰 이동 / 큰 거래 / quality wallet overlap 기반 간단 스코어
4. 읽기 전용 feed API
5. `/discover` 또는 `/signals/pre-listing` UI

제외:

- listing probability 수치화
- AI narrative 강화
- 자동 alert

### Phase 2

목표: `token-first early-entry evidence engine`

포함:

1. `token_entry_features_daily`
2. `first_entry_before_listing`
3. `quality_wallet_overlap`
4. `entry_persistence`
5. `listing_event_linkage` placeholder

### Phase 3

목표: `운영/알림/검증 체계`

포함:

1. watchlist 편입
2. alert 조건
3. analyst explanation
4. backtest / review set

## 7. Workstreams

## 7.1 Workstream A: Exchange Listing Registry

### 목표

업비트/빗썸 상장 여부를 안정적으로 관리하는 canonical registry를 만든다.

### 해야 할 일

1. Upbit listing sync worker
2. Bithumb listing sync worker
3. market symbol -> canonical asset mapping
4. `listed on upbit / bithumb` read model

### 완료 기준

- 하루 단위가 아니라 반복 실행 가능한 sync가 있음
- 토큰별 국내 상장 여부를 deterministic하게 판정 가능함

## 7.2 Workstream B: Domestic Unlisted Universe

### 목표

내부가 관측한 토큰 중 `국내 미상장 토큰 universe`를 생성한다.

### 해야 할 일

1. internal token table 정리
2. CoinGecko contract map optional enrichment
3. DEX Screener metadata optional enrichment
4. domestic listed flag materialization

### 완료 기준

- chain + contract 기준으로 국내 미상장 집합을 daily snapshot으로 생성 가능함

## 7.3 Workstream C: Token Movement Scoring

### 목표

미상장 토큰 중 실제 볼 가치가 있는 후보만 상위에 노출한다.

### 해야 할 일

1. `large movement` 정의
   - 절대 USD 규모
   - circulating / liquidity 대비 상대 규모
2. `quality wallet overlap` 계산
   - tracked/labeled/scored wallet 유입 수
3. `DEX activity` 계산
   - volume jump
   - liquidity change
4. `candidate score`
   - evidence sufficiency
   - contradiction / suppression

### 완료 기준

- 단순 신규 토큰 noise가 아니라 evidence 있는 토큰이 상위에 옴

## 7.4 Workstream D: Candidate Feed & Operator UX

### 목표

운영자와 사용자 모두가 읽을 수 있는 token-first surface를 제공한다.

### 해야 할 일

1. `GET /v1/signals/pre-listing` 또는 `GET /v1/discover/domestic-unlisted`
2. token card UI
3. why surfaced / next watch copy
4. `/admin`에서 suppression / review action

### 완료 기준

- 운영자가 false positive를 쉽게 제거할 수 있음
- 유저가 token-first로 후보를 탐색할 수 있음

## 7.5 Workstream E: Alert / Watchlist / Analyst

### 목표

후보를 운영 파이프라인으로 연결한다.

### 해야 할 일

1. token watchlist item type 추가
2. pre-listing candidate alert rule
3. analyst explanation context 추가
4. candidate -> watchlist conversion tracking

### 완료 기준

- 후보를 저장/알림/분석 워크플로우로 넘길 수 있음

## 8. 주요 리스크

### 8.1 false positive

문제:

- 상장과 무관한 DEX volume spike
- meme coin rotation
- airdrop / farming / bot noise

대응:

- quality wallet overlap 가중치
- liquidity-relative threshold
- deployer-only / retail-only suppressor

### 8.2 asset mapping mismatch

문제:

- 거래소 symbol과 onchain contract mapping 불일치

대응:

- symbol이 아니라 contract-first
- ambiguous symbol은 `unknown mapping`으로 보수 처리

### 8.3 prediction framing risk

문제:

- 사용자가 `상장 확정`처럼 받아들일 수 있음

대응:

- UI wording에서 prediction 제거
- evidence/caution/unknowns 분리

## 9. Definition of Done

다음 조건을 만족하면 1차 출시 가능 상태로 본다.

1. 업비트/빗썸 상장 목록 sync가 자동으로 갱신된다.
2. 내부 token universe에서 국내 미상장 여부를 deterministic하게 계산할 수 있다.
3. 미상장 토큰 중 큰 이동/quality overlap/market activity 기반 후보 피드가 나온다.
4. 후보 카드에 `왜 올라왔는지`가 명시된다.
5. 운영자가 `/admin`에서 후보 suppression 또는 후속 추적을 할 수 있다.
6. UI는 `상장 예측`이 아니라 `국내 미상장 후보 모니터`로 읽힌다.

## 10. Immediate Next Tasks

다음 구현 순서는 이 순서로 고정한다.

1. `exchange_listing_registry` schema + sync worker
2. token registry에서 `domestic_listed` 계산
3. 간단한 `large movement + quality overlap` 기준 candidate scorer
4. read API
5. discover/pre-listing feed UI
