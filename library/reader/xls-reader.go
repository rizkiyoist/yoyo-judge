/*
 * Created on Fri Mar 22 2024
 *
 * Copyright (c) 2024 Rizki Hadiaturrasyid
 */

package reader

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

const file = "IYYF-SCORE-CALC-FINAL-2017.xlsx"

func ReadXls() {
	ef, err := excelize.OpenFile(file)
	if err != nil {
		fmt.Printf("failed opening file, %v", err)
	}
	for i := 1; i <= 10; i++ {
		val, err := ef.GetCellValue("SET-UP", "A"+fmt.Sprintf("%d", i))
		if err != nil {
			fmt.Printf("failed reading file, %v", err)
		}
		fmt.Println(val)
	}
}
