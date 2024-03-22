/*
 * Created on Thu Jul 25 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package handler

// this is for version 2.3 (2017-8-3 Hironori Mii)

// SetUp is mandatory
// First sheet
type SetUp struct {
	ContestName   string
	DivisionName  string
	Stage         string // preliminary, final
	ClickerJudge1 int
	ClickerJudge2 int
	ClickerJudge3 int
	ClickerJudge4 int
	ClickerJudge5 int
	ClickerJudge6 int
	EvalJudge1    int
	EvalJudge2    int
	EvalJudge3    int
	EvalJudge4    int
	EvalJudge5    int
	EvalJudge6    int
}

// OptionalSetUpFinal is not mandatory
// First sheet
type OptionalSetUpFinal struct {
	ClickerValue           int // default 60
	Exe                    int // default 5
	Ctl                    int // default 5
	Tdv                    int // default 5
	Sem                    int // default 5
	Mu1                    int // default 5
	Mu2                    int // default 5
	Bdy                    int // default 5
	Shw                    int // default 5
	DeductionStopValue     int // default 1
	DeductionDiscardValue  int // default 3
	DeductionCutValueValue int // default 5
}

type OptionalSetUpPrelim struct {
	// TODO: define prelim structure
}

// Second sheet
type Player struct {
	Number        int
	Name          string
	OptionalData1 string
	OptionalData2 string
	OptionalData3 string
	OptionalData4 string
	OptionalData5 string
}

// Third sheet
type RawTex struct {
	Number             int
	Stop               int
	Discard            int
	Cut                int
	ClickerJudge1Plus  int
	ClickerJudge1Minus int
	ClickerJudge2Plus  int
	ClickerJudge2Minus int
	ClickerJudge3Plus  int
	ClickerJudge3Minus int
	ClickerJudge4Plus  int
	ClickerJudge4Minus int
	ClickerJudge5Plus  int
	ClickerJudge5Minus int
	ClickerJudge6Plus  int
	ClickerJudge6Minus int
}

// Fourth sheet
type RawPev struct {
	Number        int
	EvalJudge1Exe int
	EvalJudge1Ctl int
	EvalJudge1Tdv int
	EvalJudge1Sem int
	EvalJudge1Mu1 int
	EvalJudge1Mu2 int
	EvalJudge1Bdy int
	EvalJudge1Shw int
	EvalJudge2Exe int
	EvalJudge2Ctl int
	EvalJudge2Tdv int
	EvalJudge2Sem int
	EvalJudge2Mu1 int
	EvalJudge2Mu2 int
	EvalJudge2Bdy int
	EvalJudge2Shw int
	EvalJudge3Exe int
	EvalJudge3Ctl int
	EvalJudge3Tdv int
	EvalJudge3Sem int
	EvalJudge3Mu1 int
	EvalJudge3Mu2 int
	EvalJudge3Bdy int
	EvalJudge3Shw int
	EvalJudge4Exe int
	EvalJudge4Ctl int
	EvalJudge4Tdv int
	EvalJudge4Sem int
	EvalJudge4Mu1 int
	EvalJudge4Mu2 int
	EvalJudge4Bdy int
	EvalJudge4Shw int
	EvalJudge5Exe int
	EvalJudge5Ctl int
	EvalJudge5Tdv int
	EvalJudge5Sem int
	EvalJudge5Mu1 int
	EvalJudge5Mu2 int
	EvalJudge5Bdy int
	EvalJudge5Shw int
	EvalJudge6Exe int
	EvalJudge6Ctl int
	EvalJudge6Tdv int
	EvalJudge6Sem int
	EvalJudge6Mu1 int
	EvalJudge6Mu2 int
	EvalJudge6Bdy int
	EvalJudge6Shw int
}
