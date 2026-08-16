/*
 * Created on Thu Jun 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package calc

// PlayerResult is one player's fully computed score, matching the FINAL-SCORE
// sheet's per-player row.
type PlayerResult struct {
	Number             int
	Name               string
	TechnicalExecution float64
	CategoryScores     map[string]float64
	GroupTotals        map[EvalGroup]float64
	EvaluationTotal    float64
	DeductionTotals    map[string]float64
	DeductionTotal     float64
	FinalScore         float64
	Place              int
}

// Calculate reproduces the IYYF workbook's ADJ-CLICK -> ADJ-GIVEN ->
// FINAL-SCORE formula chain: it scales each clicker judge's net click count
// against the best net count in the field (RAW-TEx/ADJ-CLICK), averages and
// (for FINAL) halves each evaluation category's raw judge scores
// (RAW-TEvPEv/ADJ-GIVEN), sums everything into an evaluation total, deducts
// major-deduction points, and ranks players by final score
// (FINAL-SCORE, descending, ties sharing a rank).
func (c *Contest) Calculate() []PlayerResult {
	results := make([]PlayerResult, len(c.Players))

	// FINAL-SCORE scales each judge's net click count against that judge's
	// own best net count across the whole field, so the max must be found
	// per judge column before any player's score can be computed.
	var maxNetByJudge [6]int
	netByPlayerJudge := make([][6]int, len(c.Players))
	for i, p := range c.Players {
		for j := 0; j < 6; j++ {
			net := p.Clickers[j].Net()
			netByPlayerJudge[i][j] = net
			if net > maxNetByJudge[j] {
				maxNetByJudge[j] = net
			}
		}
	}

	for i, p := range c.Players {
		r := PlayerResult{
			Number:          p.Number,
			Name:            p.Name,
			CategoryScores:  make(map[string]float64, len(c.Categories)),
			GroupTotals:     make(map[EvalGroup]float64),
			DeductionTotals: make(map[string]float64, len(c.Deductions)),
		}

		var clickerSum float64
		for j := 0; j < 6; j++ {
			clickerSum += scaleClicker(netByPlayerJudge[i][j], maxNetByJudge[j], c.ClickerValue)
		}
		r.TechnicalExecution = clickerSum / 6

		for _, cat := range c.Categories {
			var sum float64
			for j := 0; j < 6; j++ {
				sum += p.EvalScores[j][cat.Name]
			}
			score := sum / 6
			if cat.Halve {
				score /= 2
			}
			r.CategoryScores[cat.Name] = score
			r.GroupTotals[cat.Group] += score
		}

		r.EvaluationTotal = r.TechnicalExecution
		for _, total := range r.GroupTotals {
			r.EvaluationTotal += total
		}

		for _, d := range c.Deductions {
			pts := float64(p.Deductions[d.Name]) * d.Value
			r.DeductionTotals[d.Name] = pts
			r.DeductionTotal += pts
		}

		r.FinalScore = r.EvaluationTotal - r.DeductionTotal
		results[i] = r
	}

	assignPlaces(results)
	return results
}

// scaleClicker reproduces ADJ-CLICK's H/J/L/N/P/R columns: a judge's net
// click count as a fraction of the best net count anyone in the field
// scored for that same judge, scaled up to the contest's clicker value.
func scaleClicker(net, maxNet int, clickerValue float64) float64 {
	if maxNet <= 0 {
		return 0
	}
	return float64(net) / float64(maxNet) * clickerValue
}

// assignPlaces reproduces Excel's RANK(final_score, all_final_scores, 0):
// descending rank, with tied scores sharing a rank and the next rank
// skipping accordingly.
func assignPlaces(results []PlayerResult) {
	for i := range results {
		place := 1
		for j := range results {
			if results[j].FinalScore > results[i].FinalScore {
				place++
			}
		}
		results[i].Place = place
	}
}
