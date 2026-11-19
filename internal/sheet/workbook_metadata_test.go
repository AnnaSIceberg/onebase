package sheet

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestWorkbookReplaceKeepsExtensionsAndCalculationProperties(t *testing.T) {
	original := excelize.NewFile()
	refMode, calcMode := "R1C1", "manual"
	no := false
	require.NoError(t, original.SetCalcProps(&excelize.CalcPropsOptions{
		RefMode: &refMode, CalcMode: &calcMode, FullCalcOnLoad: &no,
	}))
	require.NoError(t, original.SetDefinedName(&excelize.DefinedName{
		Name: "KeepExistingRange", RefersTo: "Sheet1!$A$1", Scope: "Workbook",
	}))
	require.NoError(t, original.SetDefinedName(&excelize.DefinedName{
		Name: "_xlnm.Print_Area", RefersTo: "Sheet1!$Z$10", Scope: "Sheet1",
	}))
	bytesBefore, err := original.WriteToBuffer()
	require.NoError(t, err)
	require.NoError(t, original.Close())

	const extension = `<extLst><ext xmlns:x15="http://schemas.microsoft.com/office/spreadsheetml/2010/11/main" uri="{TEST}"><x15:workbookPr chartTrackingRefBase="1"/></ext></extLst>`
	originalXML := workbookPartXML(t, bytesBefore.Bytes())
	withExtension := bytes.Replace(originalXML, []byte("</workbook>"), []byte(extension+"</workbook>"), 1)
	require.NotEqual(t, originalXML, withExtension)
	fixture, err := restoreWorkbookPart(bytesBefore.Bytes(), withExtension)
	require.NoError(t, err)

	replacement := NewWorkbook()
	for _, name := range []string{"Sheet1", "New & tab"} {
		doc := NewDocument()
		doc.SetCell(0, 0, "new value")
		require.NoError(t, replacement.AddSheet(name, doc))
	}
	updated, err := replacement.ReplaceXLSX(bytes.NewReader(fixture))
	require.NoError(t, err)
	updatedXML := string(workbookPartXML(t, updated))
	require.Contains(t, updatedXML, extension)
	require.Contains(t, updatedXML, `refMode="R1C1"`)
	require.Contains(t, updatedXML, `calcMode="manual"`)
	require.Contains(t, updatedXML, `fullCalcOnLoad="1"`)
	require.Contains(t, updatedXML, `name="KeepExistingRange"`)
	require.NotContains(t, updatedXML, `name="_xlnm.Print_Area"`)
	restored, err := excelize.OpenReader(bytes.NewReader(updated))
	require.NoError(t, err)
	defer func() { require.NoError(t, restored.Close()) }()
	require.Equal(t, []string{"Sheet1", "New & tab"}, restored.GetSheetList())
	got, err := restored.GetCellValue("New & tab", "A1")
	require.NoError(t, err)
	require.Equal(t, "new value", got)
}

func workbookPartXML(t *testing.T, data []byte) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	entry, err := archive.Open(workbookPart)
	require.NoError(t, err)
	defer func() { require.NoError(t, entry.Close()) }()
	result, err := io.ReadAll(entry)
	require.NoError(t, err)
	return result
}
