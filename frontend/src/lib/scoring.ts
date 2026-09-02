// TS port of yoyo-judge's Go scoring engine (library/calc/stage.go + rules.go).
// Keep this in lockstep with the Go source - it's the same IYYF formula chain,
// just running client-side against the mock API's stored raw scores.

export type ScoringStage = 'final' | 'prelim'

export type EvalGroup = 'TEv' | 'PEv'

export interface EvalCategory {
  name: string
  group: EvalGroup
  maxValue: number
  halve: boolean
}

export interface Deduction {
  name: string
  value: number
}

export interface ClickerScore {
  plus: number
  minus: number
}

export function clickerNet(c: ClickerScore): number {
  return c.plus - c.minus
}

export interface PlayerInput {
  number: number
  name: string
  clickers: ClickerScore[] // 6 clicker judges, in slot order
  evalScores: Record<string, number>[] // 6 eval judges, each a category -> raw score map
  deductions: Record<string, number> // deduction name -> occurrence count
}

export interface Contest {
  stage: ScoringStage
  clickerValue: number
  categories: EvalCategory[]
  deductions: Deduction[]
  players: PlayerInput[]
}

export interface PlayerResult {
  number: number
  name: string
  technicalExecution: number
  categoryScores: Record<string, number>
  groupTotals: Partial<Record<EvalGroup, number>>
  evaluationTotal: number
  deductionTotals: Record<string, number>
  deductionTotal: number
  finalScore: number
  place: number
}

export const DEFAULT_CLICKER_VALUE = 60

// Judges score each FINAL category 0-10, same input scale as PRELIM;
// halving (dividing the averaged score by 2, in calculate() below) is what
// differentiates FINAL from PRELIM, not the input's own range - maxValue
// here is just the input widget's hint, not read by calculate().
export function finalCategories(): EvalCategory[] {
  return [
    { name: 'EXE', group: 'TEv', maxValue: 10, halve: true },
    { name: 'CTL', group: 'TEv', maxValue: 10, halve: true },
    { name: 'TDV', group: 'TEv', maxValue: 10, halve: true },
    { name: 'SEM', group: 'TEv', maxValue: 10, halve: true },
    { name: 'MU1', group: 'PEv', maxValue: 10, halve: true },
    { name: 'MU2', group: 'PEv', maxValue: 10, halve: true },
    { name: 'BDY', group: 'PEv', maxValue: 10, halve: true },
    { name: 'SHW', group: 'PEv', maxValue: 10, halve: true },
  ]
}

export function finalDeductions(): Deduction[] {
  return [
    { name: 'Stop', value: 1 },
    { name: 'Discard', value: 3 },
    { name: 'Cut', value: 5 },
  ]
}

export function prelimCategories(): EvalCategory[] {
  return [
    { name: 'EXE', group: 'TEv', maxValue: 10, halve: false },
    { name: 'CTL', group: 'TEv', maxValue: 10, halve: false },
    { name: 'MU1', group: 'PEv', maxValue: 10, halve: false },
    { name: 'BDY', group: 'PEv', maxValue: 10, halve: false },
  ]
}

export function prelimDeductions(): Deduction[] {
  return [
    { name: 'Stop', value: 1 },
    { name: 'Discard', value: 3 },
    { name: 'Detach', value: 5 },
  ]
}

export function newContest(stage: ScoringStage, players: PlayerInput[]): Contest {
  return {
    stage,
    clickerValue: DEFAULT_CLICKER_VALUE,
    categories: stage === 'prelim' ? prelimCategories() : finalCategories(),
    deductions: stage === 'prelim' ? prelimDeductions() : finalDeductions(),
    players,
  }
}

function scaleClicker(net: number, maxNet: number, clickerValue: number): number {
  if (maxNet <= 0) return 0
  return (net / maxNet) * clickerValue
}

// Reproduces the workbook's ADJ-CLICK -> ADJ-GIVEN -> FINAL-SCORE formula
// chain: scale each clicker judge's net click count against the best net
// count anyone in the field scored for that same judge, average and (for
// FINAL) halve each evaluation category's raw judge scores, sum into an
// evaluation total, deduct major-deduction points, and rank players by final
// score (descending, ties sharing a rank).
export function calculate(contest: Contest): PlayerResult[] {
  // FINAL-SCORE scales each judge's net click count against that judge's own
  // best net count across the whole field, so the max must be found per
  // judge column before any player's score can be computed.
  const maxNetByJudge = [0, 0, 0, 0, 0, 0]
  const netByPlayerJudge: number[][] = contest.players.map((p) => {
    const nets = [0, 1, 2, 3, 4, 5].map((j) => clickerNet(p.clickers[j] ?? { plus: 0, minus: 0 }))
    nets.forEach((net, j) => {
      if (net > maxNetByJudge[j]) maxNetByJudge[j] = net
    })
    return nets
  })

  const results: PlayerResult[] = contest.players.map((p, i) => {
    const categoryScores: Record<string, number> = {}
    const groupTotals: Partial<Record<EvalGroup, number>> = {}
    const deductionTotals: Record<string, number> = {}

    let clickerSum = 0
    for (let j = 0; j < 6; j++) {
      clickerSum += scaleClicker(netByPlayerJudge[i][j], maxNetByJudge[j], contest.clickerValue)
    }
    const technicalExecution = clickerSum / 6

    for (const cat of contest.categories) {
      let sum = 0
      for (let j = 0; j < 6; j++) {
        sum += p.evalScores[j]?.[cat.name] ?? 0
      }
      let score = sum / 6
      if (cat.halve) score /= 2
      categoryScores[cat.name] = score
      groupTotals[cat.group] = (groupTotals[cat.group] ?? 0) + score
    }

    let evaluationTotal = technicalExecution
    for (const total of Object.values(groupTotals)) {
      evaluationTotal += total ?? 0
    }

    let deductionTotal = 0
    for (const d of contest.deductions) {
      const pts = (p.deductions[d.name] ?? 0) * d.value
      deductionTotals[d.name] = pts
      deductionTotal += pts
    }

    const finalScore = evaluationTotal - deductionTotal

    return {
      number: p.number,
      name: p.name,
      technicalExecution,
      categoryScores,
      groupTotals,
      evaluationTotal,
      deductionTotals,
      deductionTotal,
      finalScore,
      place: 0,
    }
  })

  assignPlaces(results)
  return results
}

// Reproduces Excel's RANK(final_score, all_final_scores, 0): descending
// rank, with tied scores sharing a rank and the next rank skipping
// accordingly.
function assignPlaces(results: PlayerResult[]): void {
  for (let i = 0; i < results.length; i++) {
    let place = 1
    for (let j = 0; j < results.length; j++) {
      if (results[j].finalScore > results[i].finalScore) place++
    }
    results[i].place = place
  }
}
