//go:build windows

package filepicker

import (
	"errors"
	"github.com/sqweek/dialog"
)

// OpenXLSX, OpenJSON and SaveJSON use Windows' native file dialog (Explorer),
// not an in-app custom picker.
func OpenXLSX() (string, bool, error) {
	return open("选择设备列表 Excel", "Excel 工作簿", "xlsx")
}
func SaveXLSX() (string, bool, error) {
	path, err := dialog.File().Title("选择 Excel 模板保存位置").Filter("Excel 工作簿", "xlsx").SetStartFile("devices-template.xlsx").Save()
	if errors.Is(err, dialog.ErrCancelled) {
		return "", false, nil
	}
	return path, err == nil, err
}
func OpenJSON() (string, bool, error) { return open("选择配置备份", "JSON 备份", "json") }
func OpenComponentPackage() (string, bool, error) {
	return open("选择组件包", "组件清单", "xlpkg", "json")
}
func SaveJSON() (string, bool, error) {
	path, err := dialog.File().Title("选择配置备份位置").Filter("JSON 备份", "json").SetStartFile("xint-lanpulse-backup.json").Save()
	if errors.Is(err, dialog.ErrCancelled) {
		return "", false, nil
	}
	return path, err == nil, err
}
func open(title, description string, extensions ...string) (string, bool, error) {
	path, err := dialog.File().Title(title).Filter(description, extensions...).Load()
	if errors.Is(err, dialog.ErrCancelled) {
		return "", false, nil
	}
	return path, err == nil, err
}
