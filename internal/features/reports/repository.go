package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository считает отчёты прямо в PostgreSQL
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создаёт репозиторий отчётов
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// funnelSummarySQL — итоги по всем неудалённым лидам
const funnelSummarySQL = `
SELECT
	count(*) 									   AS total,
	count(*) FILTER (WHERE current_stage = 'won')  AS won,
	count(*) FILTER (WHERE current_stage = 'lost') AS lost,
	round(100.0 * count(*) FILTER (WHERE current_stage = 'won')
		/ nullif(count(*) FILTER (WHERE current_stage IN ('won', 'lost')), 0), 1) AS win_rate_pct
FROM crm.leads
WHERE deleted_at IS NULL`

// funnelStagesSQL — показатели каждой стадии по истории переходов
const funnelStagesSQL = `
WITH stages (stage, ord) AS (
	VALUES ('new', 1), ('qualified', 2), ('proposal', 3), ('won', 4), ('lost', 5)
),
periods AS (
	SELECT ls.stage, ls.lead_id, ls.entered_at, ls.left_at
	FROM crm.lead_stages ls
	JOIN crm.leads l ON l.id = ls.lead_id
	WHERE l.deleted_at IS NULL
),
stats AS (
	SELECT
		s.stage,
		s.ord,
		count(DISTINCT p.lead_id) 						  AS reached,
		count(p.lead_id) FILTER (WHERE p.left_at IS NULL) AS current,
		avg(extract(epoch FROM p.left_at - p.entered_at) / 86400)
			FILTER (WHERE p.left_at IS NOT NULL)              AS avg_days
	FROM stages s
	LEFT JOIN periods p ON p.stage = s.stage
	GROUP BY s.stage, s.ord
)
SELECT
	stage,
	reached,
	current,
	round(avg_days, 1) AS avg_days,
	CASE WHEN stage IN ('qualified', 'proposal', 'won')
		THEN round(100.0 * reached / nullif (lag(reached) OVER (ORDER BY ord), 0), 1)
	END AS conversion_pct
FROM stats
ORDER BY ord`

// Funnel считает отчёт «Воронка продаж»
func (r *Repository) Funnel(ctx context.Context) (Funnel, error) {
	var f Funnel
	err := r.pool.QueryRow(ctx, funnelSummarySQL).Scan(&f.Total, &f.Won, &f.Lost, &f.WinRatePct)
	if err != nil {
		return Funnel{}, fmt.Errorf("funnel summary: %w", err)
	}

	rows, err := r.pool.Query(ctx, funnelStagesSQL)
	if err != nil {
		return Funnel{}, fmt.Errorf("funnel stage: %w", err)
	}
	defer rows.Close()

	f.Stages = []StageStats{}
	for rows.Next() {
		var s StageStats
		if err := rows.Scan(&s.Stage, &s.Reached, &s.Current, &s.AvgDays, &s.ConversionPct); err != nil {
			return Funnel{}, fmt.Errorf("scan funnel stage: %w", err)
		}
		f.Stages = append(f.Stages, s)
	}

	if err := rows.Err(); err != nil {
		return Funnel{}, fmt.Errorf("read funnel stages: %w", err)
	}

	return f, nil
}
