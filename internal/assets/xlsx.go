package assets

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xint-tech/xint-lanpulse/internal/config"
	"github.com/xuri/excelize/v2"
)

var headers = []string{"设备IP", "设备名称", "设备区域"}

func ExportTemplate(path string) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, value)
	}
	_ = f.SetCellValue(sheet, "A2", "192.168.20.15")
	_ = f.SetCellValue(sheet, "B2", "示例设备")
	_ = f.SetCellValue(sheet, "C2", "示例区域")
	_ = f.SetColWidth(sheet, "A", "A", 20)
	_ = f.SetColWidth(sheet, "B", "C", 24)
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	return f.SaveAs(path)
}

func Import(path string) ([]config.Asset, error) {
	if strings.ToLower(filepath.Ext(path)) != ".xlsx" {
		return nil, fmt.Errorf("请选择 .xlsx Excel 文件")
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取 Excel：%w", err)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel 至少需要一行表头和一行设备数据")
	}
	indexes := map[string]int{}
	for i, value := range rows[0] {
		indexes[strings.TrimSpace(value)] = i
	}
	for _, required := range headers {
		if _, ok := indexes[required]; !ok {
			return nil, fmt.Errorf("Excel 缺少列：%s", required)
		}
	}
	result := make([]config.Asset, 0, len(rows)-1)
	seen := map[int]bool{}
	for rowNo, row := range rows[1:] {
		if empty(row) {
			continue
		}
		value := func(column string) string {
			index := indexes[column]
			if index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}
		suffix, err := suffixFromIP(value("设备IP"))
		if err != nil {
			return nil, fmt.Errorf("第 %d 行：%w", rowNo+2, err)
		}
		if seen[suffix] {
			return nil, fmt.Errorf("第 %d 行：重复设备 IP 192.168.20.%d", rowNo+2, suffix)
		}
		seen[suffix] = true
		result = append(result, config.Asset{Suffix: suffix, Name: value("设备名称"), Area: value("设备区域")})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("Excel 中没有可导入的设备")
	}
	return result, nil
}
func suffixFromIP(ip string) (int, error) {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 || parts[0] != "192" || parts[1] != "168" || parts[2] != "20" {
		return 0, fmt.Errorf("设备IP 必须是 192.168.20.x")
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil || n < 1 || n > 254 {
		return 0, fmt.Errorf("设备IP 后缀必须在 1～254 之间")
	}
	return n, nil
}
func empty(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}
