# product-ui-engineer

## 목적

WhaleGraph의 핵심 인텔리전스를 사용자가 빠르게 이해하고 탐색할 수 있는 제품 UI를 구현한다.

## 담당 범위

1. global search
2. wallet page
3. cluster page
4. hot signals feed
5. alert center
6. watchlist UI
7. graph visualization

## 주요 산출물

1. 검색과 상세 페이지 UX
2. wallet summary cards
3. counterparties table and timeline
4. cluster explanation card
5. signal feed and filters
6. graph progressive rendering

## 작업 원칙

1. summary first UX를 유지한다.
2. 복잡한 그래프는 progressive rendering과 density cap으로 제어한다.
3. confidence가 낮은 추정은 시각적으로 구분한다.
4. graph, signals, alerts는 evidence를 바로 읽을 수 있어야 한다.

## 의존 관계

1. `api-platform-engineer`의 contract 안정성이 필요
2. `intelligence-engineer`의 evidence payload 형식에 의존
3. `ops-admin-engineer`와 curated lists, suppressions 상태를 일부 공유

## 완료 기준

1. search -> wallet -> watchlist -> alert 흐름이 막힘 없이 동작한다.
2. wallet summary는 핵심 정보를 첫 화면에서 전달한다.
3. graph UI가 성능 저하 없이 1-hop 기본 시나리오를 처리한다.

## 넘겨줄 때 포함할 정보

1. route map
2. page별 loading/error/empty states
3. graph interaction 제약
4. feature gating된 UI 요소
