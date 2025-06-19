/*
 * Created on Thu Jun 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package writer

import (
	"fmt"
	"yoyo-judge/library/calc"

	"github.com/xuri/excelize/v2"
)

const texFile = "IYYF-SCORE-CALC-FINAL-2017.xlsx"

type Tex struct {
	PlayerName         string
	Stop               string
	Discard            string
	Cut                string
	ClickerJudge1Plus  string
	ClickerJudge1Minus string
	ClickerJudge2Plus  string
	ClickerJudge2Minus string
	ClickerJudge3Plus  string
	ClickerJudge3Minus string
	ClickerJudge4Plus  string
	ClickerJudge4Minus string
	ClickerJudge5Plus  string
	ClickerJudge5Minus string
	ClickerJudge6Plus  string
	ClickerJudge6Minus string
}

func WriteTex(t *Tex) error {
	ef, err := excelize.OpenFile(texFile)
	if err != nil {
		return fmt.Errorf("failed opening file, %v", err)
	}
	ef.SetCellValue(calc.RawTexSheet, calc.Name, t.PlayerName)
	ef.SetCellValue(calc.RawTexSheet, calc.Stop, t.Stop)
	ef.SetCellValue(calc.RawTexSheet, calc.Discard, t.Discard)
	ef.SetCellValue(calc.RawTexSheet, calc.Cut, t.Cut)

	ef.SetCellValue(calc.RawTexSheet, calc.JA1Plus, t.ClickerJudge1Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA1Minus, t.ClickerJudge1Minus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA2Plus, t.ClickerJudge2Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA2Minus, t.ClickerJudge2Minus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA3Plus, t.ClickerJudge3Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA3Minus, t.ClickerJudge3Minus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA4Plus, t.ClickerJudge4Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA4Minus, t.ClickerJudge4Minus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA5Plus, t.ClickerJudge5Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA5Minus, t.ClickerJudge5Minus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA6Plus, t.ClickerJudge6Plus)
	ef.SetCellValue(calc.RawTexSheet, calc.JA6Minus, t.ClickerJudge6Minus)

	err = ef.SaveAs(texFile)
	if err != nil {
		return fmt.Errorf("failed saving file, %v", err)
	}
	return nil
}
