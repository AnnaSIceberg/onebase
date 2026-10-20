// Package xlsxadvisory provides small, synthetic regression workbooks for the
// publicly disclosed Excelize advisories. Never open these in the parent test
// process: Run confines crashes and excessive iteration to a timed subprocess.
package xlsxadvisory

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/xuri/excelize/v2"
)

var Cases = []string{"short-encryption-info", "oversized-cfb-stream", "excessive-row", "cyclic-array"}

func Run(t *testing.T, body func()) {
	t.Helper()
	if os.Getenv("ONEBASE_XLSX_ADVISORY_CHILD") == "1" {
		debug.SetMaxStack(8 << 20)
		body()
		return
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$", "-test.v")
	cmd.Env = append(os.Environ(), "ONEBASE_XLSX_ADVISORY_CHILD=1", "GOMEMLIMIT=256MiB", "TMPDIR="+dir, "TMP="+dir, "TEMP="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated XLSX path failed (deadline=%v): %v\n%s", ctx.Err(), err, out)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary files survived: %v (%v)", entries, err)
	}
	t.Logf("%s", out)
}

func Book(t *testing.T, scenario string) []byte {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetCellStr("Sheet1", "A1", "007"); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	source := buf.Bytes()
	switch scenario {
	case "short-encryption-info", "oversized-cfb-stream":
		data, err := excelize.Encrypt(source, &excelize.Options{Password: "fixture"})
		if err != nil {
			t.Fatal(err)
		}
		var name []byte
		for _, u := range utf16.Encode([]rune("EncryptionInfo")) {
			name = binary.LittleEndian.AppendUint16(name, u)
		}
		off := bytes.Index(data, name)
		if off < 512 || off%128 != 0 || off+128 > len(data) || data[off+66] != 2 {
			t.Fatal("missing CFB directory entry")
		}
		size := uint64(4)
		// Larger than the entire file, small enough that a dependency regression
		// cannot allocate gigabytes even inside the isolated worker.
		if scenario == "oversized-cfb-stream" {
			size = uint64(len(data) + 4096)
		}
		binary.LittleEndian.PutUint64(data[off+120:off+128], size)
		return data
	case "excessive-row", "cyclic-array":
		sheet := `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`
		if scenario == "excessive-row" {
			sheet += `<row r="1"><c r="A1" t="inlineStr"><is><t>007</t></is></c></row><row r="2147483647"><c r="A2147483647"><v>1</v></c></row>`
		} else {
			sheet += `<row r="1"><c r="A1"><f t="array" ref="B1">B1</f><v>7</v></c><c r="B1"><f t="array" ref="A1">A1</f><v>12</v></c></row>`
		}
		sheet += `</sheetData></worksheet>`
		zr, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		zw := zip.NewWriter(&out)
		for _, entry := range zr.File {
			w, err := zw.Create(entry.Name)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Name == "xl/worksheets/sheet1.xml" {
				_, err = io.WriteString(w, sheet)
			} else {
				r, e := entry.Open()
				if e != nil {
					t.Fatal(e)
				}
				_, err = io.Copy(w, r)
				if e = r.Close(); e != nil {
					t.Fatal(e)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	default:
		t.Fatalf("unknown scenario %s", scenario)
		return nil
	}
}

func NoRecoveredPanic(t *testing.T, err error) {
	t.Helper()
	if err != nil && (strings.Contains(err.Error(), "panic") || strings.Contains(err.Error(), "runtime error")) {
		t.Fatalf("dependency panicked: %v", err)
	}
}
