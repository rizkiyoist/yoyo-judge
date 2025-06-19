/*
 * Created on Thu Jul 25 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package writer

import (
	"fmt"
	"yoyo-judge/library/calc"

	"github.com/xuri/excelize/v2"
)

const file = "IYYF-SCORE-CALC-FINAL-2017.xlsx"

type Setup struct {
	ContestName  string
	DivisionName string
	Stage        string
	JA1          string // judge A 1
	JA2          string // judge A 2
	JA3          string // judge A 3
	JA4          string // judge A 4
	JA5          string // judge A 5
	JA6          string // judge A 6
	JB1          string // judge B 1
	JB2          string // judge B 2
	JB3          string // judge B 3
	JB4          string // judge B 4
	JB5          string // judge B 5
	JB6          string // judge B 6
}

func WriteSetup(w *Setup) error {
	ef, err := excelize.OpenFile(file)
	if err != nil {
		return fmt.Errorf("failed opening file, %v", err)
	}
	ef.SetCellValue(calc.SetUpSheet, calc.ContestName, w.ContestName)
	ef.SetCellValue(calc.SetUpSheet, calc.DivisionName, w.DivisionName)
	ef.SetCellValue(calc.SetUpSheet, calc.Stage, w.Stage)

	ef.SetCellValue(calc.SetUpSheet, calc.JA1, w.JA1)
	ef.SetCellValue(calc.SetUpSheet, calc.JA2, w.JA2)
	ef.SetCellValue(calc.SetUpSheet, calc.JA3, w.JA3)
	ef.SetCellValue(calc.SetUpSheet, calc.JA4, w.JA4)
	ef.SetCellValue(calc.SetUpSheet, calc.JA5, w.JA5)
	ef.SetCellValue(calc.SetUpSheet, calc.JA6, w.JA6)

	ef.SetCellValue(calc.SetUpSheet, calc.JB1, w.JB1)
	ef.SetCellValue(calc.SetUpSheet, calc.JB2, w.JB2)
	ef.SetCellValue(calc.SetUpSheet, calc.JB3, w.JB3)
	ef.SetCellValue(calc.SetUpSheet, calc.JB4, w.JB4)
	ef.SetCellValue(calc.SetUpSheet, calc.JB5, w.JB5)
	ef.SetCellValue(calc.SetUpSheet, calc.JB6, w.JB6)

	err = ef.SaveAs(file)
	if err != nil {
		return fmt.Errorf("failed saving file, %v", err)
	}
	return nil
}
