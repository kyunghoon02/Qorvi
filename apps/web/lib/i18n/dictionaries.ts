export type Locale = "en" | "ko";

export const defaultLocale: Locale = "en";

export type Dictionary = {
  hero: {
    title: string;
    subtitle: string;
    searchPlaceholder: string;
    searchButton: string;
  };
  home: {
    feedTitle: string;
    feedSubtitle: string;
    feedItem: {
      nextWatch: string;
      analyzeWallet: string;
      importance: string;
      confidence: string;
      subjectType: string;
    };
  };
  walletDetail: {
    headers: {
      aiBrief: string;
      graphInvestigation: string;
      relatedAddresses: string;
      recentFlow: string;
      coverage: string;
    };
    labels: {
      indexing: string;
      coverageReady: string;
      expandCoverage: string;
      continueIndexing: string;
    };
  };
  landing: {
    eyebrow: string;
    headlineLeft: string;
    headlineRight: string;
    sub: string;
    ctaPrimary: string;
    ctaSecondary: string;
  };
  limitations: {
    head: string;
    kicker: string;
    notFinancialAdvice: string;
    notFinancialAdviceDetail: string;
    notSecurityAudit: string;
    notSecurityAuditDetail: string;
    labelsHeuristic: string;
    labelsHeuristicDetail: string;
    cexHints: string;
    cexHintsDetail: string;
    etherscanCap: string;
    etherscanCapDetail: string;
  };
};

export const dictionaries: Record<Locale, Dictionary> = {
  en: {
    hero: {
      title: "Start with a wallet address",
      subtitle:
        "Search an EVM or Solana wallet to open the AI brief, key findings, and a full graph investigation canvas.",
      searchPlaceholder: "EVM or Solana address",
      searchButton: "Search",
    },
    home: {
      feedTitle: "Findings feed",
      feedSubtitle:
        "AI findings and signal interpretations from the current indexed coverage.",
      feedItem: {
        nextWatch: "Open wallet brief",
        analyzeWallet: "Analyze wallet",
        importance: "importance",
        confidence: "confidence",
        subjectType: "Wallet",
      },
    },
    walletDetail: {
      headers: {
        aiBrief: "AI brief",
        graphInvestigation: "Graph investigation canvas",
        relatedAddresses: "Related addresses",
        recentFlow: "Recent flow",
        coverage: "Coverage status",
      },
      labels: {
        indexing: "Background indexing",
        coverageReady: "Coverage ready",
        expandCoverage: "Expand coverage",
        continueIndexing: "Continue indexing",
      },
    },
    landing: {
      eyebrow: "QORVI · LIVE ETHEREUM DATA",
      headlineLeft: "Understand any wallet's on-chain story",
      headlineRight: "in thirty seconds.",
      sub: "Qorvi is an AI-native wallet copilot for Ethereum mainnet. Paste an address, pick a window, and get an evidence-first brief — token flows, DeFi interactions, CEX hints, approval risks, and a behavior profile, all sourced from live Etherscan data with every tool call traced.",
      ctaPrimary: "Open Copilot →",
      ctaSecondary: "Explore findings",
    },
    limitations: {
      head: "Limitations & safety",
      kicker: "All facts derive from provider / tool output.",
      notFinancialAdvice: "Not financial advice.",
      notFinancialAdviceDetail:
        "Interpret findings alongside your own due diligence.",
      notSecurityAudit: "Not a security audit.",
      notSecurityAuditDetail:
        "Qorvi summarizes on-chain behavior, not contract code.",
      labelsHeuristic: "Labels are heuristic.",
      labelsHeuristicDetail: "Counterparty entities and tags may misclassify.",
      cexHints: "CEX hints are possible, not definitive.",
      cexHintsDetail: "Deposit-pattern signals require manual confirmation.",
      etherscanCap: "Etherscan free plan may cap returned rows.",
      etherscanCapDetail:
        "Some long-tail transfers can be omitted from the window.",
    },
  },
  ko: {
    hero: {
      title: "지갑 주소로 시작하기",
      subtitle:
        "EVM 또는 Solana 지갑을 검색하여 AI 요약, 핵심 발견 사항 및 전체 그래프 탐색 캔버스를 엽니다.",
      searchPlaceholder: "EVM 또는 Solana 주소",
      searchButton: "검색",
    },
    home: {
      feedTitle: "핵심 발견 사항 피드",
      feedSubtitle:
        "현재 색인된 정보를 바탕으로 AI가 분석한 핵심 발견 사항 및 신호 해석을 제공합니다.",
      feedItem: {
        nextWatch: "지갑 요약본 열기",
        analyzeWallet: "지갑 분석하기",
        importance: "중요도",
        confidence: "신뢰도",
        subjectType: "지갑",
      },
    },
    walletDetail: {
      headers: {
        aiBrief: "AI 요약본",
        graphInvestigation: "그래프 탐색 캔버스",
        relatedAddresses: "관련 주소",
        recentFlow: "최근 흐름",
        coverage: "커버리지 현황",
      },
      labels: {
        indexing: "백그라운드 색인 중",
        coverageReady: "커버리지 준비 완료",
        expandCoverage: "커버리지 확장",
        continueIndexing: "색인 계속하기",
      },
    },
    landing: {
      eyebrow: "QORVI · LIVE ETHEREUM DATA",
      headlineLeft: "지갑의 온체인 스토리를",
      headlineRight: "30초 안에 파악하세요.",
      sub: "Qorvi는 이더리움 메인넷을 위한 AI-네이티브 지갑 코파일럿입니다. 주소를 붙여넣고 기간을 고르면 토큰 흐름, DeFi 상호작용, CEX 힌트, 승인 리스크, 행동 프로파일까지 — 모두 실시간 Etherscan 데이터에서 가져오고, 모든 도구 호출이 추적됩니다.",
      ctaPrimary: "Copilot 열기 →",
      ctaSecondary: "발견 사항 살펴보기",
    },
    limitations: {
      head: "제한 사항 및 안전 안내",
      kicker: "모든 정보는 제공자/도구 출력에서 유도됩니다.",
      notFinancialAdvice: "금융 자문이 아닙니다.",
      notFinancialAdviceDetail: "직접 실사를 병행하여 결과를 해석하세요.",
      notSecurityAudit: "보안 감사가 아닙니다.",
      notSecurityAuditDetail:
        "Qorvi는 온체인 활동을 요약할 뿐, 컨트랙트 코드를 감사하지 않습니다.",
      labelsHeuristic: "레이블은 휴리스틱입니다.",
      labelsHeuristicDetail: "상대방 엔티티 및 태그가 잘못 분류될 수 있습니다.",
      cexHints: "CEX 힌트는 가능성일 뿐, 확정이 아닙니다.",
      cexHintsDetail: "입금 패턴 신호는 수동 확인이 필요합니다.",
      etherscanCap: "Etherscan 무료 플랜은 반환 행을 제한할 수 있습니다.",
      etherscanCapDetail:
        "롱테일 트랜잭션 일부가 윈도우에서 누락될 수 있습니다.",
    },
  },
};

export function getDictionary(locale: Locale): Dictionary {
  return dictionaries[locale] ?? dictionaries[defaultLocale];
}
