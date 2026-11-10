package tests

import (
	"path/filepath"
	"testing"

	"github.com/xint-tech/xint-lanpulse/internal/assets"
	"github.com/xuri/excelize/v2"
)

func TestExcelTemplateAndImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.xlsx")
	if err := assets.ExportTemplate(path); err != nil {
		t.Fatal(err)
	}
	items, err := assets.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Suffix != 15 || items[0].Name != "示例设备" {
		t.Fatalf("unexpected import: %+v", items)
	}
}

func TestExcelImportRejectsNon20Network(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]string{"设备IP", "设备名称", "设备区域"})
	_ = f.SetSheetRow(sheet, "A2", &[]string{"192.168.30.15", "屏幕", "大厅"})
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := assets.Import(path); err == nil {
		t.Fatal("expected non-20 network rejection")
	}
}
