package store

// ResolveTarget: Abo-Ziel → letzter Pfad der Sendung → zuletzt genutzt → <target_dir>/<Titel>.
func ResolveTarget(cfg Config, hist History, urn, title string) string {
	if sub := cfg.Subscription(urn); sub != nil {
		return cfg.SubscriptionTarget(*sub) // dasselbe Ziel wie sync, auch ohne target_dir
	}
	if path := hist.ShowPaths[urn]; path != "" {
		return path
	}
	if len(hist.RecentPaths) > 0 {
		return hist.RecentPaths[0]
	}
	return cfg.DefaultTarget(title)
}
