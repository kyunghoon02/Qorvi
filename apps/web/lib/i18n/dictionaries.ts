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
  },
};

export function getDictionary(locale: Locale): Dictionary {
  return dictionaries[locale] ?? dictionaries[defaultLocale];
}
