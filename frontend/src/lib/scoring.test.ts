// Ported from library/calc/rules_test.go — same inputs, same expected
// numbers, so the TS engine stays provably in sync with the Go one.
import { describe, expect, it } from 'vitest'
import { calculate, newContest, type PlayerInput } from './scoring'

function evalScores(
  exe: number,
  ctl: number,
  tdv: number,
  sem: number,
  mu1: number,
  mu2: number,
  bdy: number,
  shw: number,
): Record<string, number>[] {
  const scores = { EXE: exe, CTL: ctl, TDV: tdv, SEM: sem, MU1: mu1, MU2: mu2, BDY: bdy, SHW: shw }
  return Array.from({ length: 6 }, () => scores)
}

function clickers(...nets: number[]) {
  return nets.map((n) => ({ plus: n, minus: 0 }))
}

describe('calculate (final stage)', () => {
  const players: PlayerInput[] = [
    {
      number: 1,
      name: 'A',
      clickers: clickers(10, 12, 11, 9, 10, 11),
      evalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4),
      deductions: { Stop: 1 },
    },
    {
      number: 2,
      name: 'B',
      clickers: clickers(12, 14, 13, 10, 12, 13),
      evalScores: evalScores(4.5, 4.5, 4.5, 4.5, 4.5, 4.5, 4.5, 4.5),
      deductions: {},
    },
  ]

  const results = calculate(newContest('final', players))
  const [a, b] = results

  it('scales T.Ex against the field-best net click count per judge', () => {
    expect(a.technicalExecution).toBeCloseTo(51.16117216117215, 9)
    expect(b.technicalExecution).toBeCloseTo(60.0, 9)
  })

  it('halves and sums evaluation category group totals', () => {
    expect(a.groupTotals.TEv).toBeCloseTo(8.0, 9)
    expect(a.groupTotals.PEv).toBeCloseTo(8.0, 9)
    expect(b.groupTotals.TEv).toBeCloseTo(9.0, 9)
  })

  it('sums T.Ex + T.Ev + P.Ev into the evaluation total', () => {
    expect(a.evaluationTotal).toBeCloseTo(67.16117216117215, 9)
    expect(b.evaluationTotal).toBeCloseTo(78.0, 9)
  })

  it('deducts major-deduction points from the evaluation total', () => {
    expect(a.deductionTotal).toBeCloseTo(1.0, 9)
    expect(a.finalScore).toBeCloseTo(66.16117216117215, 9)
    expect(b.finalScore).toBeCloseTo(78.0, 9)
  })

  it('ranks by final score, descending', () => {
    expect(a.place).toBe(2)
    expect(b.place).toBe(1)
  })
})

describe('calculate (tied places skip rank)', () => {
  const players: PlayerInput[] = [
    {
      number: 1,
      name: '',
      clickers: clickers(10, 10, 10, 10, 10, 10),
      evalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4),
      deductions: {},
    },
    {
      number: 2,
      name: '',
      clickers: clickers(10, 10, 10, 10, 10, 10),
      evalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4),
      deductions: {},
    },
    {
      number: 3,
      name: '',
      clickers: clickers(5, 5, 5, 5, 5, 5),
      evalScores: evalScores(2, 2, 2, 2, 2, 2, 2, 2),
      deductions: {},
    },
  ]

  it('shares a place on ties and skips the next place', () => {
    const results = calculate(newContest('final', players))
    expect(results[0].place).toBe(1)
    expect(results[1].place).toBe(1)
    expect(results[2].place).toBe(3)
  })
})

describe('calculate (prelim stage, no halving)', () => {
  const players: PlayerInput[] = [
    {
      number: 1,
      name: '',
      clickers: clickers(10, 10, 10, 10, 10, 10),
      evalScores: evalScores(8, 8, 0, 0, 8, 0, 8, 0),
      deductions: {},
    },
  ]

  const [r] = calculate(newContest('prelim', players))

  it('does not halve category scores', () => {
    expect(r.categoryScores.EXE).toBeCloseTo(8.0, 9)
  })

  it('sums group totals without halving', () => {
    expect(r.groupTotals.TEv).toBeCloseTo(16.0, 9)
    expect(r.groupTotals.PEv).toBeCloseTo(16.0, 9)
  })
})
