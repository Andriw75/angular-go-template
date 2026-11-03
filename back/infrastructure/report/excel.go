package report

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"back/domain/outputs"
)

func newWorkbook() *excelize.File {
	f := excelize.NewFile()
	_ = f.DeleteSheet("Sheet1")
	return f
}

func titleStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 18, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	return style
}

func headerStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1E293B"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "334155", Style: 1},
			{Type: "right", Color: "334155", Style: 1},
			{Type: "top", Color: "334155", Style: 1},
			{Type: "bottom", Color: "334155", Style: 1},
		},
	})
	return style
}

func cellStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1},
			{Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1},
			{Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})
	return style
}

func moneyStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1},
			{Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1},
			{Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})
	return style
}

func totalStyle(f *excelize.File) int {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"DBEAFE"}, Pattern: 1},
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	return style
}

func GenerateBusesExcel(report outputs.ReporteBuses) ([]byte, error) {
	f := newWorkbook()
	defer f.Close()

	// Resumen
	_, _ = f.NewSheet("Resumen")
	_ = f.SetColWidth("Resumen", "A", "D", 24)
	_ = f.SetCellValue("Resumen", "A1", "Reporte de Buses")
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 18, Color: "1E293B"}})
	_ = f.SetCellStyle("Resumen", "A1", "D1", title)
	_ = f.SetCellValue("Resumen", "A2", "Generado")
	_ = f.SetCellValue("Resumen", "B2", time.Now().Format("2006-01-02 15:04"))
	_ = f.SetCellValue("Resumen", "A3", "Filtros")
	_ = f.SetCellValue("Resumen", "B3", fmt.Sprintf("%v", report.Filtros))
	_ = f.MergeCell("Resumen", "B3", "D3")
	_ = f.SetCellValue("Resumen", "A5", "Indicador")
	_ = f.SetCellValue("Resumen", "B5", "Valor")
	_ = f.SetCellStyle("Resumen", "A5", "B5", headerStyle(f))

	rows := [][2]any{
		{"Total", report.Resumen.Total},
		{"Activos", report.Resumen.Activos},
		{"Inactivos", report.Resumen.Inactivos},
		{"Precio total", report.Resumen.PrecioTotal},
		{"Peso total", report.Resumen.PesoTotal},
		{"Capacidad promedio", report.Resumen.CapacidadPromedio},
	}
	for i, row := range rows {
		cell := fmt.Sprintf("A%d", 6+i)
		_ = f.SetCellValue("Resumen", cell, row[0])
		_ = f.SetCellValue("Resumen", fmt.Sprintf("B%d", 6+i), row[1])
	}

	// Buses
	_, _ = f.NewSheet("Buses")
	headers := []string{"Placa", "Nombre", "Marca", "Modelo", "Año", "Capacidad", "Tipo", "Precio", "Peso", "Color", "Activo", "Fecha compra"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Buses", cell, h)
	}
	_ = f.SetCellStyle("Buses", "A1", "L1", headerStyle(f))
	cell, _ := f.NewStyle(&excelize.Style{})
	_ = cell
	for r, b := range report.Data {
		row := r + 2
		values := []any{b.Placa, b.Nombre, b.Marca, b.Modelo, b.Anio, b.Capacidad, b.Tipo,
			b.Precio, b.Peso, b.Color, estadoBus(b.Activo), b.FechaCompra}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue("Buses", cell, v)
		}
	}
	lastRow := len(report.Data) + 1
	if lastRow < 2 {
		lastRow = 2
	}
	_ = f.SetCellStyle("Buses", "A2", fmt.Sprintf("L%d", lastRow), cellStyle(f))
	_ = f.SetColWidth("Buses", "A", "L", 14)
	_ = f.SetColWidth("Buses", "B", "B", 24)
	_ = f.AutoFilter("Buses", fmt.Sprintf("A1:L%d", lastRow), nil)
	_ = f.SetPanes("Buses", &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	// Conditional formatting for active column
	green, _ := f.NewConditionalStyle(&excelize.Style{Font: &excelize.Font{Color: "166534", Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"DCFCE7"}, Pattern: 1}})
	red, _ := f.NewConditionalStyle(&excelize.Style{Font: &excelize.Font{Color: "B91C1C", Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"FEE2E2"}, Pattern: 1}})
	_ = f.SetConditionalFormat("Buses", fmt.Sprintf("K2:K%d", lastRow), []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "==", Value: `"Activo"`, Format: &green},
		{Type: "cell", Criteria: "==", Value: `"Inactivo"`, Format: &red},
	})

	// Por tipo
	_, _ = f.NewSheet("Por tipo")
	_ = f.SetCellValue("Por tipo", "A1", "Tipo")
	_ = f.SetCellValue("Por tipo", "B1", "Cantidad")
	_ = f.SetCellValue("Por tipo", "C1", "Precio total")
	_ = f.SetCellValue("Por tipo", "D1", "Capacidad prom.")
	_ = f.SetCellStyle("Por tipo", "A1", "D1", headerStyle(f))
	for i, t := range report.PorTipo {
		row := i + 2
		_ = f.SetCellValue("Por tipo", fmt.Sprintf("A%d", row), t.Tipo)
		_ = f.SetCellValue("Por tipo", fmt.Sprintf("B%d", row), t.Cantidad)
		_ = f.SetCellValue("Por tipo", fmt.Sprintf("C%d", row), t.PrecioTotal)
		_ = f.SetCellValue("Por tipo", fmt.Sprintf("D%d", row), t.CapacidadPromedio)
	}
	_ = f.SetColWidth("Por tipo", "A", "D", 22)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateVentaExcel(venta outputs.VentaResponse) ([]byte, error) {
	f := newWorkbook()
	defer f.Close()

	_, _ = f.NewSheet("Factura")
	_ = f.SetColWidth("Factura", "A", "F", 18)
	_ = f.SetCellValue("Factura", "A1", "Factura / Nota de venta")
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 18, Color: "1E293B"}})
	_ = f.SetCellStyle("Factura", "A1", "F1", title)
	_ = f.SetCellValue("Factura", "A2", "Código")
	_ = f.SetCellValue("Factura", "B2", venta.Codigo)
	_ = f.SetCellValue("Factura", "A3", "Fecha")
	_ = f.SetCellValue("Factura", "B3", venta.Fecha)
	_ = f.SetCellValue("Factura", "A4", "Cliente")
	_ = f.SetCellValue("Factura", "B4", venta.ClienteNombre)
	_ = f.SetCellValue("Factura", "A5", "Documento")
	_ = f.SetCellValue("Factura", "B5", venta.ClienteDocumento)
	_ = f.SetCellValue("Factura", "A6", "Dirección")
	_ = f.SetCellValue("Factura", "B6", venta.ClienteDireccion)

	headers := []string{"Descripción", "Cantidad", "P. Unit.", "Descuento", "Impuesto", "Total"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 8)
		_ = f.SetCellValue("Factura", cell, h)
	}
	_ = f.SetCellStyle("Factura", "A8", "F8", headerStyle(f))
	for i, it := range venta.Items {
		row := 9 + i
		_ = f.SetCellValue("Factura", fmt.Sprintf("A%d", row), it.Descripcion)
		_ = f.SetCellValue("Factura", fmt.Sprintf("B%d", row), it.Cantidad)
		_ = f.SetCellValue("Factura", fmt.Sprintf("C%d", row), it.PrecioUnitario)
		_ = f.SetCellValue("Factura", fmt.Sprintf("D%d", row), it.Descuento)
		_ = f.SetCellValue("Factura", fmt.Sprintf("E%d", row), it.Impuesto)
		_ = f.SetCellValue("Factura", fmt.Sprintf("F%d", row), it.Total)
	}
	lastRow := 8 + len(venta.Items)
	if lastRow < 9 {
		lastRow = 9
	}
	_ = f.SetCellStyle("Factura", "A9", fmt.Sprintf("F%d", lastRow), moneyStyle(f))
	_ = f.SetCellStyle("Factura", "A9", "A"+fmt.Sprint(lastRow), cellStyle(f))

	totalRow := lastRow + 2
	_ = f.SetCellValue("Factura", fmt.Sprintf("E%d", totalRow), "Subtotal")
	_ = f.SetCellValue("Factura", fmt.Sprintf("F%d", totalRow), venta.Subtotal)
	_ = f.SetCellValue("Factura", fmt.Sprintf("E%d", totalRow+1), "Impuesto")
	_ = f.SetCellValue("Factura", fmt.Sprintf("F%d", totalRow+1), venta.Impuesto)
	_ = f.SetCellValue("Factura", fmt.Sprintf("E%d", totalRow+2), "Total")
	_ = f.SetCellValue("Factura", fmt.Sprintf("F%d", totalRow+2), venta.Total)
	_ = f.SetCellStyle("Factura", fmt.Sprintf("E%d", totalRow+2), fmt.Sprintf("F%d", totalRow+2), totalStyle(f))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
