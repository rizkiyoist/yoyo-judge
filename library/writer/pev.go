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

const pevFile = "IYYF-SCORE-CALC-FINAL-2017.xlsx"

type Pev struct {
	PlayerName string
	JB1Exe     string
	JB1Ctl     string
	JB1Tdv     string
	JB1Sem     string
	JB1Mu1     string
	JB1Mu2     string
	JB1Bdy     string
	JB1Shw     string
	JB2Exe     string
	JB2Ctl     string
	JB2Tdv     string
	JB2Sem     string
	JB2Mu1     string
	JB2Mu2     string
	JB2Bdy     string
	JB2Shw     string
	JB3Exe     string
	JB3Ctl     string
	JB3Tdv     string
	JB3Sem     string
	JB3Mu1     string
	JB3Mu2     string
	JB3Bdy     string
	JB3Shw     string
	JB4Exe     string
	JB4Ctl     string
	JB4Tdv     string
	JB4Sem     string
	JB4Mu1     string
	JB4Mu2     string
	JB4Bdy     string
	JB4Shw     string
	JB5Exe     string
	JB5Ctl     string
	JB5Tdv     string
	JB5Sem     string
	JB5Mu1     string
	JB5Mu2     string
	JB5Bdy     string
	JB5Shw     string
	JB6Exe     string
	JB6Ctl     string
	JB6Tdv     string
	JB6Sem     string
	JB6Mu1     string
	JB6Mu2     string
	JB6Bdy     string
	JB6Shw     string
}

func WritePev(p *Pev) error {
	ef, err := excelize.OpenFile(pevFile)
	if err != nil {
		return fmt.Errorf("failed opening file, %v", err)
	}
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Exe, p.JB1Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Ctl, p.JB1Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Tdv, p.JB1Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Sem, p.JB1Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Mu1, p.JB1Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Mu2, p.JB1Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Bdy, p.JB1Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB1Shw, p.JB1Shw)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Exe, p.JB2Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Ctl, p.JB2Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Tdv, p.JB2Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Sem, p.JB2Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Mu1, p.JB2Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Mu2, p.JB2Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Bdy, p.JB2Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB2Shw, p.JB2Shw)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Exe, p.JB3Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Ctl, p.JB3Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Tdv, p.JB3Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Sem, p.JB3Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Mu1, p.JB3Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Mu2, p.JB3Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Bdy, p.JB3Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB3Shw, p.JB3Shw)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Exe, p.JB4Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Ctl, p.JB4Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Tdv, p.JB4Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Sem, p.JB4Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Mu1, p.JB4Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Mu2, p.JB4Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Bdy, p.JB4Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB4Shw, p.JB4Shw)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Exe, p.JB5Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Ctl, p.JB5Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Tdv, p.JB5Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Sem, p.JB5Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Mu1, p.JB5Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Mu2, p.JB5Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Bdy, p.JB5Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB5Shw, p.JB5Shw)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Exe, p.JB6Exe)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Ctl, p.JB6Ctl)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Tdv, p.JB6Tdv)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Sem, p.JB6Sem)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Mu1, p.JB6Mu1)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Mu2, p.JB6Mu2)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Bdy, p.JB6Bdy)
	ef.SetCellValue(calc.RawEvSheet, calc.JB6Shw, p.JB6Shw)

	err = ef.SaveAs(pevFile)
	if err != nil {
		return fmt.Errorf("failed saving file, %v", err)
	}
	return nil
}
