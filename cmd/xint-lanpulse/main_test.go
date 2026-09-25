package main

import (
	"bytes"
	"encoding/base64"
	"github.com/xuri/excelize/v2"
	"strings"
	"testing"
)

func TestPingRejectsInvalidAddresses(t *testing.T) {
	for _, address := range []string{"", "example.com", "::1", "999.1.1.1"} {
		if got := (&App{}).Ping(address, 1000, 1); !strings.Contains(got, "有效") {
			t.Fatalf("%q: %q", address, got)
		}
	}
}

func TestPingRejectsInvalidOptions(t *testing.T) {
	for _, options := range [][2]int{{99, 1}, {60001, 1}, {1000, 0}, {1000, 11}} {
		if got := (&App{}).Ping("127.0.0.1", options[0], options[1]); !strings.Contains(got, "超时") {
			t.Fatalf("%v: %q", options, got)
		}
	}
}

func TestMigrationWorkbook(t *testing.T) {
	app := &App{}
	data, err := app.ExportMigration([]string{"192.168.20.15"})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(decoded))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(rows[0]) != 3 || rows[0][0] != "设备IP" || rows[0][1] != "设备名称" || rows[0][2] != "设备区域" || rows[1][0] != "192.168.20.15" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	for _, addresses := range [][]string{nil, {"10.0.0.1"}, {"192.168.20.0"}, {"192.168.20.1", "192.168.20.1"}} {
		if _, err := app.ExportMigration(addresses); err == nil {
			t.Fatalf("expected rejection: %#v", addresses)
		}
	}
}
