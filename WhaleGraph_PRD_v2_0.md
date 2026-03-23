# [PRD] WhaleGraph
## Behavioral Network Intelligence for EVM & Solana Whales

**버전:** 2.0 (Production Draft)  
**상태:** Draft for Build  
**작성일:** 2026-03-18  
**제품 유형:** 멀티체인 온체인 인텔리전스 SaaS  
**지원 체인:** EVM, Solana  
**현재 운영 가정:** 외부 데이터 공급자는 무료 티어만 사용하며, 유료 전환 없이도 public beta까지 운영 가능한 구조를 우선 채택한다.

---

## 1. 문서 목적

이 문서는 WhaleGraph의 정식 제품 버전을 위한 PRD다.  
기존 아이디어 문서를 실제 빌드 가능한 제품 문서로 확장하며, 아래 항목을 명확히 정의한다.

- 제품이 해결하는 문제와 목표 사용자
- 핵심 기능의 범위와 출력 형식
- 탐지 규칙과 점수화 방식
- 멀티체인 데이터 수집 및 저장 구조
- 실시간 알림 파이프라인과 운영 제약
- API/UX/과금/지표/리스크까지 포함한 출시 기준

이 문서는 **“고래 지갑을 보여주는 시각화 앱”** 이 아니라,  
**“고래 간 관계, 동조 행동, 자금 분산, 신규 공통 진입을 증거 기반으로 해석하는 behavioral market intelligence product”** 로 WhaleGraph를 정의한다.

---

## 2. 제품 개요

### 2.1 비전

개별 지갑이 아니라 **세력의 행동 네트워크**를 본다.

### 2.2 문제 정의

기존 온체인 도구 대부분은 다음 한계가 있다.

1. 단일 지갑 중심이라 **협업 또는 동조 매매**를 잡기 어렵다.
2. 단순 입출금 트래킹에 그쳐, **매도 준비용 자금 분산** 같은 사전 신호를 놓친다.
3. 특정 토큰을 누가 샀는지는 보여주지만, **서로 무관해 보이던 고래들이 같은 신규 자산으로 모이는 현상**을 구조적으로 탐지하지 못한다.
4. 실시간 알림이 있어도 “무슨 일이 일어났는지”는 알려주지만, **왜 중요한지**를 설명하지 못한다.

### 2.3 제품 가설

아래 세 가지를 동시에 제공하면, 트레이더·리서처·프로토콜 팀에게 높은 가치를 준다.

- **Whale Cluster Intelligence:** 같은 행위자 또는 동조 세력으로 추정되는 지갑 묶음 탐지
- **Shadow Exit Detection:** 물량 분산 및 거래소 인접 이동을 통한 매도 준비 신호 탐지
- **First-Connection Discovery:** 여러 고래가 처음으로 같은 신규 토큰/프로토콜에 연결되는 순간 탐지

### 2.4 핵심 가치 제안

WhaleGraph는 단순한 지갑 뷰어가 아니라 다음 질문에 답하는 제품이다.

- 이 지갑은 누구와 함께 움직이는가?
- 이 자금은 어디로 흘러가고 있는가?
- 이 움직임은 단순 이체인가, 포지션 정리의 전조인가?
- 서로 관련 없어 보이던 고래들이 왜 같은 토큰에 동시에 들어왔는가?
- 지금 어떤 행동 패턴이 다음 시장 이벤트로 이어질 가능성이 높은가?

---

## 3. 목표 사용자

### 3.1 주요 사용자

**1) 온체인 트레이더 / 디젠 리서처**  
빠른 알파 발굴, 고래 선행 매매 감지, 위험 회피가 목적.

**2) 크립토 리서치 팀 / 애널리스트**  
토큰 분배, 고래 집중도, 세력 네트워크 분석이 목적.

**3) 프로젝트 팀 / BD / Growth 팀**  
누가 우리 토큰을 사고 있는지, 어떤 그룹이 유입되는지 파악이 목적.

**4) 소규모 펀드 / 서치펀드 / 온체인 데스크**  
고래 포지셔닝 추적, 청산/분산 신호 조기 인지가 목적.

### 3.2 비목표 사용자

- 일반 소비자용 초보자 교육 툴
- 지갑 생성/서명/송금 기능이 필요한 월렛 제품
- 법적 포렌식 수준의 실명 확정 도구

---

## 4. 제품 원칙

### 4.1 Evidence-first
모든 탐지 결과는 반드시 **증거 목록**과 함께 제공한다.

### 4.2 Probabilistic, not absolute
WhaleGraph는 “확정 판정”보다 **점수 기반 추정**을 기본으로 한다.

### 4.3 Multichain by design
EVM과 Solana를 동등한 1급 체인으로 취급한다.

### 4.4 Budget-aware architecture
무료 티어 운영 가정 하에서 동작 가능해야 하며, 기능 설계에도 호출 예산 제한이 반영되어야 한다.

### 4.5 Operator-friendly
내부 운영자가 수동 라벨링, watchlist 편집, false positive 교정, alert mute를 쉽게 할 수 있어야 한다.

---

## 5. 제품 목표와 비목표

### 5.1 제품 목표

1. 사용자가 특정 주소를 입력하면 **해당 주소의 관계망, 자금 흐름, 라벨, 클러스터 소속 가능성**을 볼 수 있다.
2. seed whale 집합으로부터 **동조 그룹(Cluster)** 을 계산하고 지속적으로 업데이트한다.
3. **Shadow Exit Risk** 를 실시간에 가깝게 계산하고 알림으로 발송한다.
4. 여러 고래가 **동일 신규 자산에 처음 연결되는 이벤트**를 탐지하고 우선순위를 매긴다.
5. public web app + 유료 구독 플랜 + 내부 운영 콘솔까지 포함한 제품 구조를 만든다.

### 5.2 비목표

1. 실명 KYC 수준의 신원 확정
2. 체인 전체를 무제한 풀 인덱싱하는 범용 노드/데이터 웨어하우스 구축
3. 모든 체인 지원
4. 자동 매매 봇 / 주문 실행 기능

---

## 6. 핵심 사용자 시나리오

### 6.1 지갑 검색
사용자는 특정 EVM 또는 Solana 주소를 검색하고, 90일 기준 관계망 요약·입출금 상대·클러스터 가능성·최근 위험 신호를 확인한다.

### 6.2 그룹 탐지
사용자는 “이 고래가 누구와 함께 움직이는가?”를 보고, 같은 그룹의 다른 지갑과 최근 공동 진입 토큰을 확인한다.

### 6.3 매도 전조 확인
사용자는 특정 고래 혹은 그룹에서 발생한 대규모 fan-out, 거래소 근접 이동, 브릿지 이동을 보고 shadow exit risk를 확인한다.

### 6.4 신규 알파 탐지
사용자는 지난 24시간 동안 2명 이상의 고래가 처음 진입한 토큰 리스트를 보고, 관련 지갑·거래 규모·시점 차이를 확인한다.

### 6.5 알림 구독
사용자는 관심 고래, 토큰, 클러스터, 위험 유형에 대해 실시간 또는 지연 알림을 설정한다.

---

## 7. 기능 요구사항

## 7.1 Wallet Intelligence Profile

### 목적
개별 주소를 중심으로 기본적인 온체인 프로필과 행동 요약을 제공한다.

### 제공 정보
- 체인, 주소, 라벨, 라벨 confidence
- 최근 24시간 / 7일 / 30일 activity
- 주요 counterparties 상위 N개
- 유입/유출 비율
- 토큰 보유 구성
- 관련 클러스터 후보
- 최근 알림 및 위험 점수

### 표시 조건
- 주소 검색 즉시 기본 프로필은 3초 이내 반환
- 상세 관계망과 심화 분석은 비동기 계산 후 단계적으로 표시
- 관련 주소 그래프는 검색한 주소를 중심으로 시각적으로 연결되어 보여야 한다.
- 그래프 레이아웃은 `원형 네트워크 고정`이 아니라 `분석형 hub-and-spoke + partial flow hints`를 기본으로 한다.
- 특정 화면은 유입/유출 방향성을 약하게 암시할 수 있지만, 좌우 고정형 플로우를 강제하지 않고 signal-first investigation view를 우선한다.

### 예외 처리
- 신규 주소, 휴면 주소, 거래량 없는 주소도 결과를 반환해야 함
- 미라벨 주소는 “Unknown”으로 두되, 추정 엔티티가 있으면 evidence와 함께 표시

---

## 7.2 Coordinated Whale Cluster Engine

### 목적
서로 다른 지갑들이 같은 행위자 또는 강한 동조 그룹일 가능성을 점수화하고 시각화한다.

### 입력
- Seed Whale 집합
- 지갑별 historical transfers / swaps / funding source / counterparties
- 거래소 및 프로토콜 라벨
- 시간축 기반 행동 시퀀스

### 핵심 정의
**Cluster** 는 확정 실체가 아니라, 아래 증거가 일정 점수 이상 누적된 **행동적 그룹**이다.

### 클러스터 점수
`cluster_score = 0.35*same_funder + 0.25*co_trading + 0.15*shared_counterparties + 0.10*cex_pattern + 0.10*temporal_sync + 0.05*bridge_similarity`

각 항목은 0~1 범위로 정규화한다.

### 판정 규칙
- `cluster_score >= 0.70` 이면 동일 클러스터 후보
- `0.55 <= cluster_score < 0.70` 이면 weak cluster 후보
- `same_funder` 단독으로는 클러스터 확정 불가
- 최근 90일 기준 데이터가 우선이며, 장기 기록은 보조 증거로 사용

### 세부 신호 정의
- **same_funder:** 최초 확인된 자금 공급자가 동일하거나, 동일 엔티티에서 14일 내 자금 유입
- **co_trading:** 동일 토큰을 같은 방향으로 일정 시간창 안에서 반복적으로 매수/매도
- **shared_counterparties:** 공통 상대 주소 비율과 공통 volume 비율
- **cex_pattern:** 동일 거래소 핫월렛/입금 패턴 공유 여부
- **temporal_sync:** 특정 이벤트 발생 간 시간차의 일관성
- **bridge_similarity:** 동일 브릿지·동일 목적 체인·동일 시점 이동 유사성

### 출력
- cluster_id
- cluster_score
- member wallets
- 핵심 증거 목록
- 최근 30일 공통 행동
- 합산 순유입/순유출
- 공통 신규 진입 자산

### UI 요구사항
- 클러스터는 단일 색상으로 표시
- 각 멤버 간 엣지는 증거 유형별 토글 가능
- “왜 묶였는지”를 자연어 설명 카드로 제공

---

## 7.3 Shadow Exit Detection Engine

### 목적
고래가 직접 대량 매도하지 않고, 하위 지갑으로 분산하거나 거래소 근처로 이동시키는 과정을 조기 탐지한다.

### 입력
- 지갑별 outgoing transfer stream
- 하위 fan-out 패턴
- 거래소 라벨 / 브릿지 라벨
- 최근 유동성 및 가격 이벤트

### 핵심 정의
**Shadow Exit** 는 “매도 확정”이 아니라, **매도 준비 혹은 익명화된 유출 가능성**을 나타내는 점수다.

### shadow_exit_risk 점수
`shadow_exit_risk = 0.30*fanout_score + 0.25*cex_proximity + 0.15*bridge_escape + 0.15*outflow_ratio + 0.10*fresh_wallet_usage + 0.05*timing_intensity`

### 판정 규칙
- 동일 소스 지갑에서 24시간 내 5개 이상 신규 하위 주소로 fan-out 발생 시 후보 생성
- fan-out 대상 중 일정 비율 이상이 72시간 내 거래소/브릿지/고빈도 라벨 주소와 연결되면 가중치 상승
- 단순 treasury rebalancing 또는 내부 운영 지갑 이동으로 추정되면 risk 감점

### 출력
- risk score (0~1)
- root wallet / cluster
- 분산된 하위 지갑 수
- 총 유출 규모
- 추정 목적지 엔티티
- 증거 이벤트 타임라인

### 알림 문구 원칙
알림은 “매도 중”이라고 단정하지 않고 다음 형식을 따른다.

> Whale Cluster A에서 12개 신규 지갑으로 6시간 내 분산 유출 발생.  
> 4개 주소가 24시간 내 CEX 인접 주소와 연결됨.  
> Shadow Exit Risk: 0.81

---

## 7.4 First-Connection Discovery

### 목적
서로 직접 연관이 약하던 고래들이 특정 신규 자산/프로토콜에 동시에 처음 진입하는 순간을 포착한다.

### 핵심 정의
**First-Connection Event** 는 아래 조건을 만족하는 신규 공통 진입 이벤트다.

- 동일 토큰 혹은 동일 프로토콜에 대해
- 최근 90일간 해당 지갑의 보유/거래 이력이 없고
- 24시간 이내 2개 이상 seed whale 또는 cluster member가 처음 진입

### alpha_score 점수
`alpha_score = 0.35*whale_quality + 0.25*novelty + 0.20*time_convergence + 0.10*size_signal + 0.10*cross_cluster_diversity`

### 출력
- token / protocol
- 최초 진입한 고래 목록
- 각 진입 시각과 규모
- 과거 유사 사례
- alpha_score
- 관련 클러스터 수

### UI 요구사항
- 별도 “Hot” 피드에서 최신순 및 점수순 제공
- 토큰 클릭 시 진입 고래와 이후 price/volume 반응 비교 제공

---

## 7.5 Alerting

### 지원 대상
- Wallet
- Cluster
- Token
- Shadow Exit
- First-Connection
- Funding Source

### 알림 채널
- In-app
- Email
- Telegram
- Discord Webhook

### 알림 설정 예시
- 특정 고래가 거래소 라벨 주소에 50만 달러 이상 보낼 때
- 내 watchlist의 어떤 고래든 신규 토큰에 첫 진입할 때
- 특정 클러스터의 shadow_exit_risk가 0.75 이상일 때

### 시스템 요구사항
- 중복 알림 방지
- alert cooldown 지원
- 동일 이벤트의 severity 상승 시 재알림 가능

---

## 7.6 Watchlists & Discovery

### 기능
- 사용자는 주소, 엔티티, 클러스터, 토큰을 watchlist에 저장할 수 있다.
- watchlist별로 태그와 설명을 붙일 수 있다.
- public curated lists 제공: Smart Money, Early Rotation, Exit Risk, Solana Whales, EVM Funds

### 운영 기능
- 내부 운영자가 curated list를 수동 편집 가능해야 한다.
- false positive가 많은 주소는 suppress 가능해야 한다.

---

## 7.7 Public API

### 목적
유료 고객과 내부 파트너가 WhaleGraph의 결과를 서비스에 재사용할 수 있게 한다.

### 제공 엔드포인트
- `/wallet/:chain/:address/summary`
- `/wallet/:chain/:address/graph`
- `/cluster/:id`
- `/signals/first-connections`
- `/signals/shadow-exits`
- `/alerts`

### API 정책
- Free 플랜은 지연 데이터와 낮은 rate limit 제공
- Pro/Team 플랜은 실시간성, 더 깊은 hop, 더 많은 결과 수를 제공

---

## 8. 데이터 전략 및 외부 데이터 공급자

## 8.1 운영 원칙
WhaleGraph는 현재 단계에서 외부 데이터 공급자의 **무료 티어만 사용**한다.  
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
6. Scoring Jobs
7. Alert Delivery

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

## 9.5 Enrichment
- entity labeling
- USD pricing normalization
- token metadata 정규화
- protocol / exchange category 매핑

## 9.6 Materialization
- 정규화 트랜잭션을 기반으로 graph edge 생성
- cluster / shadow exit / first-connection 점수는 별도 계산 후 캐시

---

## 10. 데이터 모델

WhaleGraph는 **하이브리드 저장 구조**를 사용한다.

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
- shortest path / neighborhood / cluster graph 조회

### Scoring Service
- cluster_score
- shadow_exit_risk
- alpha_score 계산

### Alert Service
- dedup
- severity evaluation
- multi-channel delivery

### API Service
- public web app용 API
- internal ops admin API
- partner API

### Frontend
- Search
- Wallet page
- Cluster page
- Hot signals feed
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
7. alert evaluation
8. delivery

---

## 13. API 요구사항

## 13.1 응답 원칙
- summary 응답은 빠르게, graph/detail 응답은 비동기로 확장
- score는 항상 evidence와 함께 반환
- chain field는 모든 응답에 포함

## 13.2 예시 응답

### Wallet Summary
- address
- chain
- label
- cluster_candidates
- netflow_7d
- top_counterparties
- latest_signals

### Cluster Detail
- cluster_id
- score
- members
- common_tokens
- common_funders
- recent_actions
- related_alerts

### Signal Feed Item
- signal_type
- severity
- title
- evidence
- wallets
- token
- timestamp

---

## 14. UX 요구사항

## 14.1 정보 설계

### Global Search
- address / ENS / Solana address / token / entity 검색

### Wallet Page
- 요약 카드
- 관계 그래프
- counterparties 테이블
- transaction timeline
- cluster section
- signal history

### Cluster Page
- 멤버 리스트
- 공통 행동
- funding source 비교
- shared token activity
- exit risk history

### Hot Signals Feed
- first-connection
- emerging cluster
- shadow exit
- unusual funding

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
- wallet summary p95 3초 이내
- 캐시된 cluster detail p95 5초 이내
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
- 단일 지갑 검색
- 1-hop 관계망
- 지연 데이터
- 제한된 alert 수
- public hot signals 일부 열람

## 16.2 Pro
- 실시간 alert
- cluster detail 전체 열람
- 2-hop graph
- shadow exit history
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
- cluster explanation click-through

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

정식 베타 오픈 전 아래 조건을 만족해야 한다.

1. EVM/Solana 모두 wallet summary 동작
2. seed whale 자동 갱신 배치 동작
3. cluster_score / shadow_exit_risk / alpha_score 계산 가능
4. alert dedup 및 cooldown 동작
5. admin label editor와 suppression 기능 동작
6. provider 호출량 대시보드 구축
7. 최소 한 개 유료 플랜 결제 및 구독 흐름 완성

---

## 20. 향후 확장

- 추가 체인 지원(Base, BNB Chain, Sui 등)
- 유료 라벨 데이터셋 도입
- fund / MM / DAO / treasury 타입별 모델 분화
- 팀 협업 기능 및 조사 리포트 생성
- AI 설명 레이어: “왜 이 신호가 중요한가” 자동 요약

---

## 21. 한 줄 정의

**WhaleGraph는 고래 지갑을 보여주는 서비스가 아니라, 고래의 관계와 행동을 점수화해 세력 네트워크와 자금 이동의 의미를 해석하는 멀티체인 인텔리전스 제품이다.**
