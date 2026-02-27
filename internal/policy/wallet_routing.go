package policy

// WalletRoutingPolicy defines which payment sources are allowed/blocked.
type WalletRoutingPolicy struct {
	AllowedSources []string `json:"allowed_sources,omitempty"` // empty = all allowed
	BlockedSources []string `json:"blocked_sources,omitempty"`
	AllowedTypes   []string `json:"allowed_types,omitempty"`   // empty = all allowed
}

// DefaultWalletRoutingPolicy returns a policy that allows all sources.
func DefaultWalletRoutingPolicy() WalletRoutingPolicy {
	return WalletRoutingPolicy{}
}

// WalletRouteEvaluation holds the result of a routing check.
type WalletRouteEvaluation struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// EvaluateWalletRoute checks if a payment source and type are allowed.
func EvaluateWalletRoute(policy WalletRoutingPolicy, source, txType string) WalletRouteEvaluation {
	// Check blocked sources
	for _, blocked := range policy.BlockedSources {
		if blocked == source {
			return WalletRouteEvaluation{Allowed: false, Reason: "source blocked: " + source}
		}
	}

	// Check allowed sources (empty = all allowed)
	if len(policy.AllowedSources) > 0 {
		found := false
		for _, allowed := range policy.AllowedSources {
			if allowed == source {
				found = true
				break
			}
		}
		if !found {
			return WalletRouteEvaluation{Allowed: false, Reason: "source not in allowed list: " + source}
		}
	}

	// Check allowed types (empty = all allowed)
	if len(policy.AllowedTypes) > 0 {
		found := false
		for _, allowed := range policy.AllowedTypes {
			if allowed == txType {
				found = true
				break
			}
		}
		if !found {
			return WalletRouteEvaluation{Allowed: false, Reason: "transaction type not allowed: " + txType}
		}
	}

	return WalletRouteEvaluation{Allowed: true}
}
