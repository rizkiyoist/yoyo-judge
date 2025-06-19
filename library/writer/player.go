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

type Player struct {
	PlayerSheet string
	Name        string
}

func WritePlayer(p *Player) error {
	ef, err := excelize.OpenFile(p.PlayerSheet)
	if err != nil {
		return fmt.Errorf("failed opening file, %v", err)
	}
	ef.SetCellValue(calc.PlayerSheet, calc.Name, p.Name)

	err = ef.SaveAs(p.PlayerSheet)
	if err != nil {
		return fmt.Errorf("failed saving file, %v", err)
	}
	return nil

}
