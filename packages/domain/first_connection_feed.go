package domain

type FirstConnectionFeedItem struct {
	WalletID       string `json:"wallet_id"`
	Chain          Chain  `json:"chain"`
	Address        string `json:"address"`
	Label          string `json:"label"`
	WalletRoute    string `json:"wallet_route"`
	Recommendation string `json:"recommendation"`
	ObservedAt     string `json:"observed_at"`
	Score          Score  `json:"score"`
}

type FirstConnectionFeedPage struct {
	Items      []FirstConnectionFeedItem `json:"items"`
	NextCursor *string                   `json:"next_cursor,omitempty"`
	HasMore    bool                      `json:"has_more"`
}
