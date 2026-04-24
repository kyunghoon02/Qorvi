# Qorvi Engine Hardening Roadmap

이 문서는 Qorvi의 핵심 행동 해석 엔진을 `baseline heuristic` 단계에서
`evidence-backed interpretation engine` 단계로 끌어올리기 위한 실제 구현
로드맵이다.

목표는 LLM을 먼저 고도화하는 것이 아니라, deterministic evidence / feature /
finding pipeline의 정밀도를 높여 `Bridge`, `Exchange`, `Treasury`, `MM`,
`Early Convergence / High Conviction` 해석을 실사용 가능한 수준으로 만드는 데 있다.

## 1. 공통 원칙

1. 모든 엔진은 `raw tx -> normalized feature -> evidence -> finding -> explanation` 순서로 동작한다.
2. `verified / probable / behavioral` 경계는 유지한다.
3. behavioral label은 hint로 시작하되, finding은 richer evidence bundle을 요구한다.
4. confidence는 단순 이름 매핑이 아니라 evidence count, path confirmation, recurrence, threshold 충족 여부를 반영해야 한다.
5. 각 엔진은 golden set(true positive / false positive / ambiguous case)을 별도로 유지한다.

## 2. 우선순위

1. `Bridge Rotation Engine`
2. `Exchange Pressure Engine`
3. `Treasury Redistribution Engine`
4. `MM Handoff Engine`
5. `Early Rotator Engine`
6. `High Conviction Entry Engine`

## 3. Bridge Rotation Engine

### 현재 baseline

- 현재 `behavioral:bridge_escape_pattern`은 outbound activity가 bridge-like counterparty를 한 번이라도 터치하면 붙는다.
- 목적 체인 도착지, bridge 이후 downstream path, 신규 토큰 진입은 아직 확인하지 않는다.

### 추가할 evidence

1. `bridge_outbound_touch`
   - bridge-like counterparty로 향하는 outbound transfer
2. `bridge_link_confirmation`
   - 동일 wallet 기준 bridge 직후 destination chain activity가 시간창 내 관찰됨
3. `post_bridge_first_entry`
   - bridge 직후 destination chain 신규 토큰/프로토콜 진입
4. `post_bridge_counterparty_shift`
   - bridge 이후 counterparty set이 source chain 대비 의미 있게 변함
5. `bridge_recurrence`
   - 동일 wallet이 N회 이상 유사 bridge path를 반복

### 필요한 테이블 / feature

1. `wallet_bridge_links`
   - `wallet_id`
   - `source_chain`
   - `destination_chain`
   - `bridge_entity_key`
   - `source_tx_hash`
   - `destination_tx_hash`
   - `linked_at`
   - `confidence`
2. `wallet_bridge_features_daily`
   - `wallet_id`
   - `observed_date`
   - `bridge_touch_count`
   - `confirmed_link_count`
   - `post_bridge_first_entry_count`
   - `bridge_recurrence_count`
3. `wallet_evidence`
   - `bridge_link_confirmation`
   - `post_bridge_first_entry`
   - `destination_chain_shift`

### worker / materialization touchpoints

1. webhook ingest / backfill 후 chain-local transfer를 `wallet_bridge_links` 후보로 기록
2. enrichment/materialization batch가 source-destination 후보를 시간창 내 매칭
3. findings worker가 `cross_chain_rotation` 및 `bridge_escape_pattern`을 재평가

### API / finding 출력

- behavioral label:
  - `behavioral:bridge_linked_outflow` 로 downgrade 후 사용
- stronger finding:
  - `cross_chain_rotation`
  - `bridge_path_resolved`
- `GET /v1/analyst/findings/:id/evidence-timeline`
  - source tx
  - linked destination tx
  - destination chain first entry

## 4. Exchange Pressure Engine

### 현재 baseline

- 현재 `behavioral:exchange_distribution_pattern`은 outbound activity가 exchange-like counterparty를 한 번 터치하면 붙는다.
- deposit-like path, fan-in/fan-out, 반복성, amount threshold는 없다.

### 추가할 evidence

1. `exchange_outbound_touch`
2. `exchange_path_recurrence`
   - 동일 exchange-adjacent 경로 반복
3. `deposit_like_path`
   - outbound -> exchange-adjacent wallet -> aggregation
4. `distribution_fanout`
   - short window fan-out 이후 exchange touch 증가
5. `cex_pressure_ratio`
   - exchange-adjacent outflow / total outflow 비율

### 필요한 테이블 / feature

1. `wallet_exchange_flow_features_daily`
   - `wallet_id`
   - `observed_date`
   - `exchange_touch_count`
   - `distinct_exchange_counterparties`
   - `deposit_like_path_count`
   - `distribution_fanout_count`
   - `cex_pressure_ratio`
2. `wallet_counterparty_features_daily`
   - `wallet_id`
   - `counterparty_wallet_id`
   - `direction`
   - `tx_count`
   - `notional_usd`
   - `first_seen_at`
   - `last_seen_at`
3. `wallet_evidence`
   - `deposit_like_path`
   - `exchange_pressure_ratio`
   - `distribution_fanout`

### worker / materialization touchpoints

1. summary aggregate job가 wallet-counterparty 흐름에서 exchange-adjacent path feature 산출
2. shadow-exit / findings worker가 exchange feature를 읽어 `cex_deposit_pressure` 평가

### API / finding 출력

- behavioral label:
  - `behavioral:exchange_linked_outflow`
- stronger finding:
  - `cex_deposit_pressure`
  - `exchange_distribution`
- evidence timeline:
  - fan-out
  - exchange-linked outflow
  - deposit-like path

## 5. Treasury Redistribution Engine

### 현재 baseline

- treasury/foundation 문자열 매칭으로 inferred treasury를 붙이고, fan-out/exit-like context가 있으면 `treasury_redistribution` finding을 생성한다.
- treasury anchor proximity, multisig signer, 정기 운영 분배 패턴은 아직 약하다.

### 추가할 evidence

1. `treasury_anchor_match`
   - curated treasury / deployer / multisig anchor와 직접 연결
2. `treasury_fanout_signature`
   - 짧은 기간 다수 subwallet로 split
3. `operational_distribution_pattern`
   - 반복적 소액/정기 분배
4. `treasury_rebalance_discount`
   - 내부 이동으로 보이는 경우 점수 차감
5. `treasury_to_market_path`
   - treasury -> probable MM/exchange/bridge path

### 필요한 테이블 / feature

1. `wallet_treasury_features_daily`
   - `wallet_id`
   - `observed_date`
   - `anchor_match_count`
   - `fanout_signature_count`
   - `operational_distribution_count`
   - `rebalance_discount_count`
   - `treasury_to_market_path_count`
2. `wallet_evidence`
   - `treasury_anchor_match`
   - `treasury_fanout_signature`
   - `internal_rebalance_discount`

### worker / materialization touchpoints

1. curated entity sync와 wallet label read model을 같이 사용
2. shadow-exit worker는 treasury-discount와 redistribution을 분리 계산
3. findings worker는 `treasury_redistribution`를 별도 finding으로 유지

### API / finding 출력

- probable label:
  - `probable_treasury_cluster`
- finding:
  - `treasury_redistribution`
  - `treasury_to_market_path`

## 6. MM Handoff Engine

### 현재 baseline

- inferred market_maker label + shadow-exit-like 흐름이 있으면 `suspected_mm_handoff`가 생성된다.
- inventory rotation, 양방향 venue touch, 프로젝트별 반복 handoff는 아직 보지 않는다.

### 추가할 evidence

1. `mm_anchor_match`
   - curated/verified MM entity proximity
2. `inventory_rotation_pattern`
   - 동일 자산 반복 양방향 이동
3. `project_to_mm_path`
   - treasury / fund-adjacent -> probable MM path
4. `post_handoff_distribution`
   - handoff 후 exchange/venue 분산
5. `repeat_mm_counterparty`
   - 여러 프로젝트에서 동일 MM-like counterparty 반복 등장

### 필요한 테이블 / feature

1. `wallet_mm_features_daily`
   - `wallet_id`
   - `observed_date`
   - `mm_anchor_match_count`
   - `inventory_rotation_count`
   - `project_to_mm_path_count`
   - `post_handoff_distribution_count`
   - `repeat_mm_counterparty_count`
2. `wallet_evidence`
   - `inventory_rotation_pattern`
   - `project_to_mm_path`
   - `post_handoff_distribution`

### worker / materialization touchpoints

1. wallet-counterparty aggregate에서 양방향 venue touch 추출
2. findings worker가 `suspected_mm_handoff`를 label-presence가 아니라 path bundle 기반으로 생성

### API / finding 출력

- probable label:
  - `suspected_market_maker`
- finding:
  - `suspected_mm_handoff`
  - `market_maker_inventory_rotation`

## 7. Early Rotator / High Conviction Entry Engine

### 현재 baseline

- `early_rotation_pattern`은 사실상 alpha score의 summary-time alias다.
- `high_conviction_entry`는 finding으로만 있고, 실제 holder persistence 엔진은 아니다.

### 추가할 evidence

1. `first_entry_before_crowding`
   - wallet가 테마/토큰에 crowd 이전 조기 진입
2. `repeat_early_entry_success`
   - 과거 early entry 후 성과 좋은 반복 사례
3. `persistence_after_entry`
   - 진입 후 일정 기간 보유 지속
4. `quality_wallet_overlap`
   - high-quality wallet cohort와 동시 진입
5. `theme_convergence`
   - 같은 테마/프로토콜로 반복 조기 진입

### 필요한 테이블 / feature

1. `wallet_entry_features_daily`
   - `wallet_id`
   - `token_or_protocol_key`
   - `entry_date`
   - `is_first_entry`
   - `quality_overlap_count`
   - `holding_persistence_hours`
   - `subsequent_outcome_score`
2. `signal_outcomes`
   - 기존 outcome schema를 early/high-conviction engine calibration에 연결

### worker / materialization touchpoints

1. first-connection worker가 alpha snapshot만 만들지 말고 entry feature도 같이 남김
2. findings worker가 `high_conviction_entry`를 persistence / overlap evidence와 함께 생성
3. wallet brief의 behavioral label은 `early_convergence_signal`로 재정의

### API / finding 출력

- behavioral label:
  - `early_convergence_signal`
- finding:
  - `high_conviction_entry`
  - later: `high_conviction_holder`

## 8. Cross-cutting API additions

1. `GET /v1/analyst/tools/wallets/:chain/:address/path-signals`
   - bridge / exchange / treasury / MM path evidence
2. `GET /v1/analyst/findings/:findingId/evidence-timeline`
   - tx ref / path ref / entity ref 강화
3. `GET /v1/analyst/findings/:findingId/historical-analogs`
   - outcome score / similar pattern count / follow-on market reaction

## 9. Suggested implementation order

1. `wallet_counterparty_features_daily` + `wallet_bridge_links`
2. `Bridge Rotation Engine`
3. `Exchange Pressure Engine`
4. `Treasury Redistribution Engine`
5. `MM Handoff Engine`
6. `Early Rotator / High Conviction Entry` 재정의
7. golden set / analyst review loop 추가

