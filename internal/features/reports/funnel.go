// Package reports — отчёты по лидам
package reports

// StageStats - показатели одной стадии воронки
type StageStats struct {
	Stage         string   `json:"stage"`
	Reached       int      `json:"reached"`
	Current       int      `json:"current"`
	AvgDays       *float64 `json:"avg_days"`
	ConversionPct *float64 `json:"conversion_pct"`
}

// Funnel - отчет "Воронка продаж"
type Funnel struct {
	Total      int          `json:"total"`
	Won        int          `json:"won"`
	Lost       int          `json:"lost"`
	WinRatePct *float64     `json:"win_rate_pct"`
	Stages     []StageStats `json:"stages"`
}
