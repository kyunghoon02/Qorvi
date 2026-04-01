# [PRD] Qorvi
## Onchain AI Analyst for Smart Money, VC, MM, Treasury, and Bridge Flow

**버전:** 2.0 (Production Draft)  
**상태:** Draft for Build  
**작성일:** 2026-03-18  
**제품 유형:** 멀티체인 온체인 인텔리전스 SaaS  
**지원 체인:** EVM, Solana  
**현재 운영 가정:** 실제 product deployment를 전제로 한다. 핵심 공급자는 production tier 또는 동등한 운영 안정성을 확보한 계약/예산 기준으로 운용하며, launch readiness와 운영 신뢰성을 기능 우선순위와 동일 선상에서 다룬다.

---

## 1. 문서 목적

이 문서는 Qorvi의 정식 제품 버전을 위한 PRD다.
기존 아이디어 문서를 실제 빌드 가능한 제품 문서로 확장하며, 아래 항목을 명확히 정의한다.

- 제품이 해결하는 문제와 목표 사용자
- 핵심 기능의 범위와 출력 형식
- 탐지 규칙과 점수화 방식
- 멀티체인 데이터 수집 및 저장 구조
- 실시간 알림 파이프라인과 운영 제약
- API/UX/과금/지표/리스크까지 포함한 출시 기준

이 문서는 **“고래 그래프를 보여주는 시각화 앱”** 이 아니라,  
**“온체인 활동을 읽고 smart money, VC, MM, treasury, bridge flow의 의미를 설명하는 AI analyst product”** 로 Qorvi을 정의한다.

---

## 2. 제품 개요

### 2.1 비전

개별 트랜잭션이 아니라 **자금 흐름의 의미**를 읽는다.

### 2.2 문제 정의

기존 온체인 도구 대부분은 다음 한계가 있다.

1. 단일 지갑 중심이라 **협업 또는 동조 매매**를 잡기 어렵다.
2. 단순 입출금 트래킹에 그쳐, **매도 준비용 자금 분산** 같은 사전 신호를 놓친다.
3. 특정 토큰을 누가 샀는지는 보여주지만, **서로 무관해 보이던 고래들이 같은 신규 자산으로 모이는 현상**을 구조적으로 탐지하지 못한다.
4. 실시간 알림이 있어도 “무슨 일이 일어났는지”는 알려주지만, **왜 중요한지**를 설명하지 못한다.

### 2.3 제품 가설

아래 세 가지를 동시에 제공하면, 온체인 트레이더·리서처·프로토콜 팀에게 높은 가치를 준다.

- **AI Findings Layer:** 온체인 이벤트와 엔터티 맥락을 종합해 중요한 움직임의 의미를 설명
- **Behavior Cohort Intelligence:** 같은 행위자 또는 강한 동조 그룹으로 추정되는 행동 코호트 탐지
- **Distribution & Exit Risk / Early Convergence:** 분산 출구 준비와 신규 공통 진입을 조기에 탐지

### 2.4 핵심 가치 제안

Qorvi은 단순한 주소 조회 도구가 아니라 다음 질문에 답하는 제품이다.

- 이 움직임은 왜 중요한가?
- 이 자금 흐름은 smart money, VC, MM, treasury, bridge activity 중 무엇을 암시하는가?
- 이것은 단순 이체인가, 운영 분산인가, 분배 준비인가, 시장 메이킹 핸드오프인가?
- 서로 관련 없어 보이던 지갑들이 왜 같은 토큰이나 프로토콜에 동시에 모이는가?
- 지금 어떤 행동 패턴이 다음 시장 이벤트로 이어질 가능성이 높은가?

---

## 3. 목표 사용자

### 3.1 주요 사용자

**1) 온체인 트레이더 / 리서처**  
smart money 움직임, convergence, exit risk를 빠르게 읽는 것이 목적.

**2) 프로젝트 팀 / BD / Growth 팀**  
VC, MM, treasury, exchange pressure와 신규 유입 질을 해석하는 것이 목적.

**3) 소규모 펀드 / 온체인 데스크**  
고품질 행동 신호와 엔터티 단위 흐름 해석을 통한 조사 효율 향상이 목적.

### 3.2 비목표 사용자

- 일반 소비자용 초보자 교육 툴
- 지갑 생성/서명/송금 기능이 필요한 월렛 제품
- 법적 포렌식 수준의 실명 확정 도구

---

## 4. 제품 원칙

### 4.1 Evidence-first
모든 탐지 결과는 반드시 **증거 목록**과 함께 제공한다.

### 4.2 Probabilistic, not absolute
Qorvi은 “확정 판정”보다 **점수 기반 추정**을 기본으로 한다.

### 4.3 Multichain by design
EVM과 Solana를 동등한 1급 체인으로 취급한다.

### 4.4 Budget-aware architecture
무료 티어 운영 가정 하에서 동작 가능해야 하며, 기능 설계에도 호출 예산 제한이 반영되어야 한다.

### 4.5 Operator-friendly
내부 운영자가 수동 라벨링, watchlist 편집, false positive 교정, alert mute를 쉽게 할 수 있어야 한다.

---

## 5. 제품 목표와 비목표

### 5.1 제품 목표

1. 사용자는 홈에서 **AI Findings Feed** 를 통해 오늘 중요한 smart money / VC / MM / treasury / bridge 움직임을 먼저 볼 수 있다.
2. 사용자가 특정 주소를 검색하면 **AI Wallet Brief** 가 먼저 반환되고, 그 아래에 evidence timeline, counterparties, graph evidence가 이어진다.
3. 시스템은 **Behavior Cohort Intelligence**, **Distribution & Exit Risk**, **Early Convergence** 를 지속적으로 계산하고 findings로 서빙한다.
4. 엔터티와 행동 라벨을 분리해 **verified / probable / behavioral** 맥락을 함께 노출한다.
5. public web app + 내부 운영 콘솔 + findings/brief 중심 API 구조를 만든다.

### 5.2 비목표

1. 실명 KYC 수준의 신원 확정
2. 체인 전체를 무제한 풀 인덱싱하는 범용 노드/데이터 웨어하우스 구축
3. 모든 체인 지원
4. 자동 매매 봇 / 주문 실행 기능

---

## 6. 핵심 사용자 시나리오

### 6.1 홈 발견
사용자는 홈에서 오늘 중요한 AI findings, rising exit risk, cross-chain rotation, early convergence를 먼저 확인한다.

### 6.2 주소 검색
사용자는 특정 EVM 또는 Solana 주소를 검색하고, 그래프보다 먼저 AI brief와 key findings를 읽는다.

### 6.3 엔터티 해석
사용자는 VC, MM, treasury, exchange, bridge와 연결된 주소의 의미를 entity/page 단위로 해석한다.

### 6.4 행동 코호트 탐지
사용자는 quality wallet들이 같은 토큰/프로토콜에 동시에 유입되는 behavior convergence를 추적한다.

### 6.5 리스크 및 알림
사용자는 distribution/exit risk, exchange pressure, suspected MM handoff 같은 findings를 저장하고 알림으로 구독한다.

---

## 7. 기능 요구사항

## 7.0 AI Findings Layer

### 목적
온체인 이벤트, 주소 관계, 라벨, 브릿지 이동, counterparty 패턴을 종합해 사람이 이해할 수 있는 분석 결과를 생성한다.

### 입력
- normalized transactions
- wallet edges
- entity labels
- cluster membership
- bridge links
- alert candidates
- historical analogs

### 출력
- finding_type
- summary
- importance_reason
- confidence
- evidence list
- next addresses/tokens to watch

### 대표 finding 유형
- Suspected MM handoff
- Cross-chain rotation detected
- Coordinated accumulation
- Exit preparation rising
- New smart money convergence
- Treasury redistribution
- Exchange deposit pressure

### 원칙
- AI는 truth engine이 아니라 evidence interpretation layer다.
- finding은 규칙 엔진과 증거 번들을 기반으로 생성해야 한다.
- AI 요약은 observed facts와 inferred interpretation을 분리해야 한다.

---

## 7.1 AI Findings Feed

### 목적
사용자가 주소를 직접 검색하지 않아도 오늘 중요한 smart money / VC / MM / treasury / bridge 흐름을 바로 읽을 수 있게 한다.

### 제공 정보
- finding type
- 요약 문장
- 중요 이유
- confidence
- 관련 wallet / entity / token / chain
- evidence preview
- next to watch

### 우선순위 기준
- severity
- signal novelty
- source quality
- entity confidence
- historical analog score

---

## 7.2 AI Wallet Brief

### 목적
개별 주소를 중심으로 기본 프로필이 아니라, 해당 주소 활동의 의미를 먼저 설명한다.

### 제공 정보
- AI summary
- key findings
- verified / probable / behavioral labels
- 최근 24시간 / 7일 / 30일 activity
- 주요 counterparties 상위 N개
- 유입/유출 비율
- 토큰 보유 구성
- 관련 behavior cohort 후보
- 최근 findings 및 위험 점수

### 표시 조건
- 주소 검색 즉시 brief는 3초 이내 반환
- evidence timeline, counterparties, graph evidence는 비동기 확장 가능
- 그래프는 summary 아래의 evidence view로 배치

### 예외 처리
- 신규 주소, 휴면 주소, 거래량 없는 주소도 결과를 반환해야 함
- 미라벨 주소는 “Unknown”으로 두되, probable label은 confidence와 evidence를 함께 표시

---

## 7.3 Behavior Cohort Intelligence

### 목적
서로 다른 지갑들이 같은 행위자 또는 강한 동조 그룹일 가능성을 점수화하고 해석한다.

### 입력
- Seed smart money 집합
- 지갑별 historical transfers / swaps / funding source / counterparties
- verified / probable entity labels
- 시간축 기반 행동 시퀀스

### 핵심 정의
**Behavior Cohort** 는 확정 실체가 아니라, 일정 증거가 누적된 **행동적 코호트** 다.

### cohort 점수
`cohort_score = 0.35*same_funder + 0.25*co_trading + 0.15*shared_counterparties + 0.10*cex_pattern + 0.10*temporal_sync + 0.05*bridge_similarity`

### 출력
- cohort_id
- cohort_score
- member wallets
- 핵심 증거 목록
- 최근 30일 공통 행동
- 합산 순유입/순유출
- 공통 신규 진입 자산

---

## 7.4 Distribution & Exit Risk

### 목적
대량 매도가 아니라 분산, 거래소 근접, 브릿지 이탈을 통해 드러나는 출구 준비 신호를 조기 탐지한다.

### 핵심 정의
**Distribution / Exit Risk** 는 매도 확정이 아니라, 출구 준비 또는 pressure 상승 가능성을 나타내는 점수다.

### exit_risk 점수
`exit_risk = 0.30*fanout_score + 0.25*cex_proximity + 0.15*bridge_escape + 0.15*outflow_ratio + 0.10*fresh_wallet_usage + 0.05*timing_intensity`

### finding 예시
- Treasury redistribution
- Exchange deposit pressure
- Exit preparation rising

---

## 7.5 Early Convergence

### 목적
quality wallet들이 동일 신규 토큰/프로토콜에 거의 동시에 처음 진입하는 순간을 포착한다.

### 핵심 정의
**Early Convergence** 는 아래 조건을 만족하는 신규 공통 진입 이벤트다.

- 동일 토큰 혹은 동일 프로토콜에 대해
- 최근 90일간 해당 지갑의 보유/거래 이력이 없고
- 24시간 이내 2개 이상 quality wallet 또는 cohort member가 처음 진입

### convergence 점수
`convergence_score = 0.35*wallet_quality + 0.25*novelty + 0.20*time_convergence + 0.10*size_signal + 0.10*cross_cohort_diversity`

---

## 7.6 Entity Interpretation & Handoff Detection

### 목적
VC / MM / treasury / exchange / bridge와 연결된 흐름을 entity 단위로 해석한다.

### 핵심 finding
- Suspected MM handoff
- Treasury redistribution
- Cross-chain rotation
- Exchange pressure

### 설명 원칙
- 실명 확정이 아니라 verified / probable entity label과 evidence를 함께 표시
- handoff, redistribution, rotation은 항상 증거 번들을 같이 제시

---

## 7.7 Graph Evidence View

### 목적
그래프는 주인공이 아니라, AI finding과 wallet brief가 왜 그렇게 해석됐는지를 보여주는 증거 UI다.

### 원칙
- 기본 화면에서 그래프는 summary 아래에 위치
- 1-hop 기본, progressive expansion 허용
- evidence caption, entity badge, confidence, token flow를 함께 보여준다

---

## 7.8 Alerting

### 지원 대상
- Findings
- Wallet
- Entity
- Behavior Cohort
- Distribution / Exit Risk
- Early Convergence

### 알림 채널
- In-app
- Email
- Telegram
- Discord Webhook

### 시스템 요구사항
- 중복 알림 방지
- alert cooldown 지원
- 동일 finding severity 상승 시 재알림 가능
- alert는 항상 왜 중요한지와 다음 watch 대상을 포함해야 한다

---

## 7.9 Watchlists & Discovery

### 기능
- 사용자는 주소, 엔티티, 코호트, 토큰, finding type을 watchlist에 저장할 수 있다.
- watchlist별로 태그와 설명을 붙일 수 있다.
- public curated lists 제공: Smart Money, Early Rotation, Exit Risk, Solana Whales, EVM Funds

### 운영 기능
- 내부 운영자가 curated list를 수동 편집 가능해야 한다.
- false positive가 많은 주소는 suppress 가능해야 한다.

---

## 7.10 Public API

### 목적
유료 고객과 내부 파트너가 Qorvi의 해석 결과를 서비스에 재사용할 수 있게 한다.

### 제공 엔드포인트
- `/findings`
- `/findings/:id`
- `/wallet/:chain/:address/brief`
- `/wallet/:chain/:address/graph`
- `/entity/:id`
- `/signals/convergence`
- `/signals/exit-risk`
- `/signals/cross-chain-rotation`
- `/signals/mm-handoffs`
- `/alerts`

### API 정책
- Free 플랜은 지연 데이터와 낮은 rate limit 제공
- Pro/Team 플랜은 실시간성, 더 깊은 evidence, 더 많은 findings 결과 수를 제공

---

## 8. 데이터 전략 및 외부 데이터 공급자

## 8.1 운영 원칙
Qorvi은 현재 단계에서 외부 데이터 공급자의 **무료 티어만 사용**한다.
따라서 전체 체인 풀 인덱싱이 아니라, **seed whale 중심의 선별 인제스천 + 검색 시 온디맨드 확장 + 실시간 watchlist 감시** 구조를 채택한다.

## 8.2 공급자 역할

### Dune
- 역할: seed whale discovery, 배치 리서치, 후보 지갑 생성
- 사용 방식: 저장된 쿼리 결과 export, 정기 리랭킹
- 제한: 복잡한 쿼리 남발 금지, export 크기 제한 고려

### Alchemy
- 역할: EVM historical transfer, realtime activity webhook
- 사용 방식: 주소별 1-hop/2-hop 관계 추출, watchlist 실시간 모니터링
- 제한: internal transfer historical coverage 차이 고려

### Helius
- 역할: Solana wallet history, transfers, identity, funded-by, realtime webhook
- 사용 방식: Solana 지갑 프로필, counterparty 추출, funder 추적, 실시간 감시
- 제한: Wallet API는 beta이며 응답 포맷 변화 가능성 고려

### Moralis
- 역할: 온디맨드 enrichment
- 사용 방식: 사용자가 상세 화면을 열 때 순자산/포지션/PnL 보강
- 제한: 실시간 핵심 원장이 아니라 보조 계층으로 사용

## 8.3 제외 범위
현재 버전에서는 아래를 코어 경로에 넣지 않는다.
- 유료 전용 라벨 데이터셋
- Birdeye 유료 플랜 의존 기능
- 전 체인 firehose ingest
- 실명 기반 attribution 데이터 구매

---

## 9. 데이터 파이프라인

## 9.1 상위 구조
1. Seed Discovery Batch
2. Historical Backfill
3. Realtime Ingestion
4. Enrichment
5. Graph Materialization
6. Scoring
7. Findings Generation
8. Alert Delivery

## 9.2 Seed Discovery Batch
- Dune 쿼리 결과를 정기적으로 export
- 상위 PnL, 회전율, 초기 진입 성공률, 자금 규모 기준으로 seed whale 후보 생성
- candidate score 기준 상위 주소만 watchlist에 편입

## 9.3 Historical Backfill

### EVM
- 주소의 inbound / outbound / token transfers 조회
- 90일 기본, 365일 확장 옵션
- 1-hop 상대 주소를 우선 수집하고 필요 시 상위 N개에 대해 2-hop 확장

### Solana
- wallet history / transfers / funded-by / identity 조회
- direction, counterparty, signature, funding source를 정규화
- 1-hop, 2-hop 확장은 예산 제한 하에 수행

## 9.4 Realtime Ingestion
- watchlist 등록 주소에 대해서만 webhook 적용
- chain별 raw event를 수신 후 바로 분석하지 않고 먼저 원본 저장
- 동일 이벤트의 중복 수신을 고려해 idempotency 키로 dedup 수행

## 9.5 Entity Attribution & Behavior Labeling
- verified entity labels
- probable entity labels
- behavior labels
- USD pricing normalization
- token metadata 정규화
- protocol / exchange category 매핑
- 모든 추정 라벨은 confidence와 evidence를 함께 저장

### 라벨 3층 구조
1. Verified Entity Label
   - exchange
   - bridge
   - protocol
   - fund
   - market_maker
   - treasury
   - deployer
2. Probable Entity Label
   - suspected_market_maker
   - suspected_cex_deposit
   - suspected_treasury_cluster
   - suspected_fund_adjacent
3. Behavior Label
   - early_rotator
   - coordinated_buyer
   - bridge_rotator
   - distribution_pattern
   - high_conviction_holder

### 9.5.1 Precision-first Engine Hardening

Behavior label과 interpretation finding은 이름만 강한 힌트가 아니라, 반복성 있는
증거와 path confirmation을 가진 엔진으로 점진적으로 고도화한다.

우선순위는 아래와 같다.

1. Bridge Rotation Engine
   - bridge touch만이 아니라 destination chain activity, post-bridge first entry, bridge recurrence를 evidence로 사용
2. Exchange Pressure Engine
   - exchange-linked outflow, deposit-like path, distribution fanout, pressure ratio를 evidence로 사용
3. Treasury Redistribution Engine
   - treasury anchor, multisig/foundation proximity, treasury fanout signature, internal rebalance discount를 evidence로 사용
4. MM Handoff Engine
   - project-to-MM path, inventory rotation, repeat MM counterparty, post-handoff distribution을 evidence로 사용
5. Early Rotator / High Conviction Entry Engine
   - early convergence 점수 alias가 아니라 first-entry timing, quality wallet overlap, persistence, historical outcome을 evidence로 사용

초기 출시 단계에서는 behavioral label을 triage hint로 사용하되, stronger finding은 richer evidence bundle을 요구한다. production surface에서는 항상 confidence와 evidence refs를 함께 노출한다.

## 9.6 Materialization
- 정규화 트랜잭션을 기반으로 graph edge 생성
- cohort / distribution risk / convergence 점수는 별도 계산 후 캐시

## 9.7 Findings Generation
- candidate finding 생성
- evidence bundle 생성
- confidence 계산
- AI summary 생성
- duplicate finding merge
- feed priority 계산
- Bridge / Exchange / Treasury / MM / Early-conviction 엔진별 golden set과 false positive case를 별도 유지

---

## 10. 데이터 모델

Qorvi은 **하이브리드 저장 구조**를 사용한다.

- **PostgreSQL:** 제품 도메인 데이터, 정규화 트랜잭션, 사용자/알림/과금
- **Neo4j:** 관계망 탐색, 클러스터 분석, 시각화용 그래프
- **Redis:** dedup, rate limit, hot cache, job queue state
- **Object Storage:** raw webhook payload, backfill snapshot, audit logs

## 10.1 핵심 엔티티

### Wallet
- wallet_id
- chain
- address
- label
- label_source
- label_confidence
- is_seed_whale
- first_seen_at
- last_seen_at

### Token
- token_id
- chain
- address_or_mint
- symbol
- name
- decimals
- is_verified

### Entity
- entity_id
- type (`exchange`, `protocol`, `bridge`, `fund`, `market_maker`, `unknown`)
- name
- confidence

### Finding
- finding_id
- finding_type
- importance_reason
- confidence
- summary
- evidence_json
- next_watch_json
- created_at

### Cluster
- cluster_id
- score
- type (`strong`, `weak`, `emerging`)
- reason_summary
- member_count
- updated_at

### Transaction
- tx_id
- chain
- block_time
- tx_type
- from_wallet_id
- to_wallet_id
- token_id
- amount_raw
- amount_usd
- direction
- source_provider
- raw_ref

### AlertRule
- rule_id
- user_id
- subject_type
- subject_id
- condition_json
- channel
- cooldown_seconds
- is_active

### AlertEvent
- event_id
- rule_id
- severity
- evidence_json
- sent_at
- dedup_key

## 10.2 Neo4j 노드
- `(:Wallet)`
- `(:Token)`
- `(:Entity)`
- `(:Cluster)`

## 10.3 Neo4j 관계
- `(:Wallet)-[:TRANSFERRED_TO]->(:Wallet)`
- `(:Wallet)-[:FUNDED_BY]->(:Wallet|:Entity)`
- `(:Wallet)-[:INTERACTED_WITH]->(:Token)`
- `(:Wallet)-[:DEPOSITED_TO]->(:Entity)`
- `(:Wallet)-[:BRIDGED_TO]->(:Wallet)`
- `(:Wallet)-[:MEMBER_OF {score,evidence}]->(:Cluster)`
- `(:Cluster)-[:ACQUIRED]->(:Token)`

---

## 11. 탐지 엔진 설계 세부사항

## 11.1 연관 주소 추출 규칙

### EVM
- 특정 주소의 inbound/outbound transfer 상대 주소를 집계
- 상대 주소별 tx_count, total_usd, first_seen, last_seen 계산
- 거래소/프로토콜 라벨이 붙은 주소는 별도 class로 분리

### Solana
- transfers endpoint의 counterparty 기반으로 연관 주소 수집
- funded-by 결과는 별도 강한 관계로 저장
- identity 결과가 존재하면 entity edge로 연결

## 11.2 graph 확장 제한
무제한 graph expansion은 금지한다.

- 기본 기간: 최근 90일
- 기본 깊이: 1-hop
- 확장 깊이: 최대 2-hop
- hop당 상위 N개 상대 주소만 확장
- 거래소/브릿지/고밀도 서비스 주소는 탐색 중단점으로 활용

## 11.3 false positive 제어
- 내부 treasury rebalancing 패턴 화이트리스트
- 라벨 confidence 낮은 엔티티는 확정 문구 금지
- 단일 신호만으로 alert 발송 금지
- human override 및 suppression 리스트 지원

---

## 12. 시스템 아키텍처

## 12.1 주요 컴포넌트

### Collector Service
- EVM collector
- Solana collector
- batch discovery worker

### Normalization Service
- 체인별 payload를 공통 transaction schema로 변환
- decimals, symbol, USD price 정규화

### Graph Service
- Neo4j materialization
- shortest path / neighborhood / evidence graph 조회

### Scoring Service
- cohort_score
- exit_risk
- convergence_score 계산

### Findings Service
- finding classification
- evidence summarization
- importance explanation
- next watch recommendation

### Alert Service
- dedup
- severity evaluation
- multi-channel delivery

### API Service
- public web app용 API
- internal ops admin API
- partner API

### Frontend
- Home / Discover
- Search
- Wallet Brief page
- Entity page
- AI Findings feed
- Alert center
- Watchlists
- Admin console

## 12.2 이벤트 처리 원칙
실시간 이벤트는 다음 순서를 반드시 따른다.

1. raw 저장
2. idempotency 확인
3. normalization
4. enrichment
5. graph update
6. score recompute
7. findings generation
8. alert evaluation
9. delivery

---

## 13. API 요구사항

## 13.1 응답 원칙
- summary 응답은 빠르게, graph/detail 응답은 비동기로 확장
- score는 항상 evidence와 함께 반환
- chain field는 모든 응답에 포함

## 13.2 예시 응답

### Wallet Brief
- address
- chain
- ai_summary
- key_findings
- verified_labels
- probable_labels
- behavioral_labels
- netflow_7d
- top_counterparties
- latest_signals

### Entity / Cohort Detail
- entity_or_cohort_id
- score
- members
- common_tokens_or_flows
- common_funders_or_destinations
- recent_actions
- related_alerts

### Finding Feed Item
- finding_type
- severity
- title
- evidence
- wallets_or_entities
- token_or_protocol
- timestamp

---

## 14. UX 요구사항

## 14.1 정보 설계

### Global Search
- address / ENS / Solana address / token / entity 검색

### Home / Discover
- AI Findings Feed
- Emerging Smart Money Signals
- Suspected MM / VC / Treasury Activity
- Cross-chain Rotations
- Exit Risk Rising

### Wallet Page
- AI Summary
- Key Findings
- Evidence Timeline
- counterparties 테이블
- graph evidence
- raw transactions

### Entity Page
- VC / MM / Fund / Treasury / Exchange 엔티티 단위 분석
- linked wallets
- interpreted findings
- evidence timeline

### Signal Page
- 모든 AI findings 피드
- 필터: smart money / vc / mm / bridge / exit / token

### Alert Center
- active rules
- triggered events
- mute / snooze / severity filter

### Admin Console
- label editor
- watchlist seeding
- suppression list
- query budget monitor

## 14.2 시각화 원칙
- 입력한 주소는 항상 시각적 중심축으로 강조되어야 한다.
- 관련 주소는 “왜 연결됐는지”가 보이도록 node/edge caption, entity badge, confidence 표현을 함께 제공해야 한다.
- 기본 그래프 경험은 `탐색형 관계 요약`과 `분석형 흐름 힌트`를 동시에 제공해야 한다.
- 레이아웃은 완전한 방사형 네트워크보다 `hub-and-spoke`, `layered neighborhood`, `partial flow hint`를 우선한다.
- 특정 패턴에서 inbound/outbound 방향성은 좌우 또는 상하 bias로 암시할 수 있지만, 단일 고정 레이아웃만 강제하지 않는다.
- 노드 크기: 최근 activity 또는 자산 규모 기반
- 엣지 두께: volume / 빈도 기반
- 색상: entity type 또는 cluster membership 기준
- 신뢰도 낮은 추정은 점선/연한 색으로 표시

---

## 15. 비기능 요구사항

## 15.1 성능
- AI wallet brief p95 3초 이내
- 캐시된 entity/cohort detail p95 5초 이내
- alert ingest 후 60초 이내 사용자 표시 목표

## 15.2 안정성
- webhook duplicate safe
- provider 장애 시 재시도 및 지연 처리
- score 계산 실패가 전체 ingest를 막지 않아야 함

## 15.3 보안
- API key 분리 저장
- admin 기능 role-based access control 적용
- 감사 로그 저장

## 15.4 관측성
- provider별 호출량 및 실패율 추적
- chain별 ingest lag 추적
- false positive / user feedback 이벤트 저장

---

## 16. 과금 및 상품 구조

## 16.1 Free
- AI findings 일부 열람
- 단일 지갑 brief 검색
- 1-hop graph evidence
- 지연 데이터
- 제한된 alert 수
- public findings 일부 열람

## 16.2 Pro
- 실시간 alert
- cohort / entity detail 전체 열람
- 2-hop graph
- exit risk history
- telegram/discord 통합

## 16.3 Team / API
- 더 높은 rate limit
- watchlist 공유
- API access
- export 기능
- 팀 단위 alert routing

---

## 17. 성공 지표

## 17.1 제품 지표
- weekly active analysts
- search to watchlist conversion
- alert open rate
- paid conversion
- retention D30 / D90

## 17.2 모델 품질 지표
- user-confirmed useful alert ratio
- false positive rate
- suppressed alert ratio
- finding explanation click-through

## 17.3 데이터 운영 지표
- provider quota usage
- webhook processing lag
- dedup hit rate
- cache hit rate

---

## 18. 리스크 및 대응

### 18.1 무료 티어 한계
**리스크:** 호출량 증가 시 quota 초과  
**대응:** seed 중심 설계, watchlist 제한, 온디맨드 확장 상한, 캐시 적극 사용

### 18.2 라벨 품질 부족
**리스크:** 거래소/프로토콜 오분류  
**대응:** label confidence 도입, operator review, 추정/확정 구분

### 18.3 false positive
**리스크:** 동조가 아닌 우연한 동시 행동을 과대 해석  
**대응:** 단일 규칙 경보 금지, evidence 공개, user feedback 반영

### 18.4 Solana API 변동성
**리스크:** beta API 응답 포맷 변경  
**대응:** adapter layer 분리, schema versioning 도입

### 18.5 복잡한 graph 시각화
**리스크:** 프론트 성능 저하  
**대응:** progressive rendering, graph summarization, pre-computed neighborhoods, 중심 주소 우선 시각화, dense neighborhood는 묶음/요약 카드로 축약

---

## 19. 출시 기준

정식 product launch 전 아래 조건을 만족해야 한다.

1. EVM/Solana 모두 AI wallet brief 동작
2. seed whale 자동 갱신 배치와 tracked wallet 증분 추적이 안정적으로 동작
3. cohort_score / exit_risk / convergence_score 와 interpretation finding이 production-grade evidence bundle과 함께 계산 가능
4. alert dedup, cooldown, delivery retry, operator mute/suppression 동작
5. admin label editor, review queue, suppression 기능 동작
6. provider 호출량 / ingest freshness / finding freshness 대시보드 구축
7. 최소 5개 finding type 자동 생성 가능
8. AI summary와 evidence bundle 간 일관성 검증 완료
9. target production environment preflight, rollback, replay runbook 완료

---

## 20. 향후 확장

- 추가 체인 지원(Base, BNB Chain, Sui 등)
- 유료 라벨 데이터셋 도입
- fund / MM / DAO / treasury 타입별 모델 분화
- 팀 협업 기능 및 조사 리포트 생성
- 자동 트레이딩 / execution

### AI가 하지 않는 것
- raw tx만 보고 근거 없이 결론 내리기
- 실명 신원 확정
- 자동 트레이딩
- 독립적인 truth engine 역할

---

## 21. 한 줄 정의

**Qorvi은 raw wallet activity를 읽어 smart money, VC, MM, treasury, bridge flow의 의미를 설명하는 온체인 AI analyst다.**
