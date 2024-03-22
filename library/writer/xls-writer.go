/*
 * Created on Thu Jul 25 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package writer

import (
	"fmt"
	"yoyo-judge/library/calcmap"

	"github.com/xuri/excelize/v2"
)

const file = "IYYF-SCORE-CALC-FINAL-2017.xlsx"

func WriteXls() {
	ef, err := excelize.OpenFile(file)
	if err != nil {
		fmt.Printf("failed opening file, %v", err)
	}
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.ContestName, "Kontes Orang Main Yoyo")
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.DivisionName, "Divisi Apa Aja Boleh")
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.Stage, "Asal Podium")
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.JA1, "Vaketu")
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.JA2, "Bukan Palit")

	ef.SetCellValue(calcmap.SetUpSheet, calcmap.JB1, "Bukan Juri Kliker")
	ef.SetCellValue(calcmap.SetUpSheet, calcmap.JB2, "Saya Belajar")

	ef.SaveAs(file)
}
