package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"back/domain/outputs"
)

//go:embed branding/logo.png
var logoPNG []byte

const (
	colorDark   = 30
	colorSlate  = 41
	colorBlue   = 59
	colorBlue2  = 130
	colorBlue3  = 246
	colorGreen  = 22
	colorGreen2 = 163
	colorRed    = 220
	colorRed2   = 38
	colorRed3   = 38
	colorAmber  = 245
	colorAmber2 = 158
	colorAmber3 = 11
)

func addFonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes("go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("go", "B", gobold.TTF)
	pdf.SetFont("go", "", 10)
}

func registerLogo(pdf *fpdf.Fpdf) {
	pdf.RegisterImageOptionsReader("brand", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(logoPNG))
}

func pdfHeader(pdf *fpdf.Fpdf, title, subtitle string) {
	width, _ := pdf.GetPageSize()
	pdf.SetFillColor(30, 41, 59)
	pdf.Rect(0, 0, width, 26, "F")
	pdf.ImageOptions("brand", 10, 5, 28, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("go", "B", 13)
	pdf.SetXY(44, 7)
	pdf.CellFormat(0, 6, title, "", 0, "L", false, 0, "")
	pdf.SetFont("go", "", 8)
	pdf.SetXY(44, 14)
	pdf.CellFormat(0, 5, subtitle, "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(34)
}

func pdfFooter(pdf *fpdf.Fpdf) {
	width, _ := pdf.GetPageSize()
	pdf.SetY(-16)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY()-2, width-10, pdf.GetY()-2)
	pdf.SetFont("go", "", 7)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 8, fmt.Sprintf("Generado %s  ·  Página %d de {nb}", time.Now().Format("2006-01-02 15:04"), pdf.PageNo()), "", 0, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func summaryBox(pdf *fpdf.Fpdf, x, y, w, h float64, label, value string, r, g, b int) {
	pdf.SetFillColor(r, g, b)
	pdf.RoundedRect(x, y, w, h, 2, "1234", "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("go", "", 7)
	pdf.SetXY(x+4, y+3)
	pdf.CellFormat(w-8, 4, strings.ToUpper(label), "", 0, "L", false, 0, "")
	pdf.SetFont("go", "B", 11)
	pdf.SetXY(x+4, y+9)
	pdf.CellFormat(w-8, 6, value, "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func drawBusesSummary(pdf *fpdf.Fpdf, report outputs.ReporteBuses) {
	width, _ := pdf.GetPageSize()
	left := 10.0
	available := width - 20
	boxW := (available - 18) / 4
	y := pdf.GetY() + 2

	summaryBox(pdf, left, y, boxW, 18, "Total buses", fmt.Sprintf("%d", report.Resumen.Total), 30, 41, 59)
	summaryBox(pdf, left+boxW+6, y, boxW, 18, "Activos", fmt.Sprintf("%d", report.Resumen.Activos), 22, 163, 74)
	summaryBox(pdf, left+2*(boxW+6), y, boxW, 18, "Inactivos", fmt.Sprintf("%d", report.Resumen.Inactivos), 220, 38, 38)
	summaryBox(pdf, left+3*(boxW+6), y, boxW, 18, "Precio total", fmt.Sprintf("%.2f", report.Resumen.PrecioTotal), 37, 99, 235)
	pdf.SetY(y + 24)
}

func drawBusesFilters(pdf *fpdf.Fpdf, report outputs.ReporteBuses) {
	pdf.SetFont("go", "", 8)
	pdf.SetFillColor(245, 247, 250)
	pdf.SetDrawColor(220, 225, 230)
	text := fmt.Sprintf("Filtros — q: %v  |  tipo: %v  |  activo: %v  |  desde: %v  |  hasta: %v",
		report.Filtros["q"], report.Filtros["tipo"], report.Filtros["activo"], report.Filtros["desde"], report.Filtros["hasta"])
	pdf.MultiCell(0, 5, text, "1", "L", true)
	pdf.Ln(3)
}

func drawBusesTable(pdf *fpdf.Fpdf, report outputs.ReporteBuses) {
	width, height := pdf.GetPageSize()
	available := width - 20
	cols := []struct {
		label string
		pct   float64
		align string
	}{
		{"Placa", 13, "L"}, {"Nombre", 20, "L"}, {"Marca", 13, "L"}, {"Tipo", 10, "L"},
		{"Año", 7, "C"}, {"Cap.", 7, "C"}, {"Precio", 12, "R"}, {"Estado", 10, "C"},
	}
	widths := make([]float64, len(cols))
	var totalPct float64
	for _, c := range cols {
		totalPct += c.pct
	}
	for i, c := range cols {
		widths[i] = available * c.pct / totalPct
	}

	pdf.SetFont("go", "B", 8)
	pdf.SetFillColor(30, 41, 59)
	pdf.SetTextColor(255, 255, 255)
	for i, c := range cols {
		pdf.CellFormat(widths[i], 7, c.label, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)

	for idx, b := range report.Data {
		if pdf.GetY() > height-24 {
			pdf.AddPageFormat(pageOrientation(pdf), pageSize(pdf))
			pdf.SetFont("go", "B", 8)
			pdf.SetFillColor(30, 41, 59)
			pdf.SetTextColor(255, 255, 255)
			for i, c := range cols {
				pdf.CellFormat(widths[i], 7, c.label, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetTextColor(0, 0, 0)
		}

		fill := idx%2 == 0
		if fill {
			pdf.SetFillColor(245, 247, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		values := []string{
			b.Placa, b.Nombre, b.Marca, b.Tipo,
			fmt.Sprintf("%d", b.Anio), fmt.Sprintf("%d", b.Capacidad),
			fmt.Sprintf("%.2f", b.Precio), estadoBus(b.Activo),
		}
		for i, v := range values {
			align := cols[i].align
			if i == 7 {
				if b.Activo {
					pdf.SetTextColor(22, 101, 52)
				} else {
					pdf.SetTextColor(185, 28, 28)
				}
			}
			pdf.CellFormat(widths[i], 6, v, "1", 0, align, true, 0, "")
			if i == 7 {
				pdf.SetTextColor(0, 0, 0)
			}
		}
		pdf.Ln(-1)
	}
}

func estadoBus(activo bool) string {
	if activo {
		return "Activo"
	}
	return "Inactivo"
}

func pageOrientation(pdf *fpdf.Fpdf) string {
	width, height := pdf.GetPageSize()
	if width > height {
		return "L"
	}
	return "P"
}

func pageSize(pdf *fpdf.Fpdf) fpdf.SizeType {
	width, height := pdf.GetPageSize()
	return fpdf.SizeType{Wd: width, Ht: height}
}

func GenerateBusesPDF(report outputs.ReporteBuses, template string) ([]byte, error) {
	var pdf *fpdf.Fpdf
	switch template {
	case "buses-a4-l":
		pdf = fpdf.New("L", "mm", "A4", "")
	case "buses-dinamico":
		height := 120.0 + float64(len(report.Data))*6.2
		if height < 180 {
			height = 180
		}
		pdf = fpdf.NewCustom(&fpdf.InitType{UnitStr: "mm", Size: fpdf.SizeType{Wd: 210, Ht: height}})
	default:
		pdf = fpdf.New("P", "mm", "A4", "")
	}
	addFonts(pdf)
	registerLogo(pdf)
	pdf.AliasNbPages("{nb}")
	pdf.SetMargins(10, 34, 10)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetHeaderFunc(func() { pdfHeader(pdf, "Reporte de Buses", "Flota registrada") })
	pdf.SetFooterFunc(func() { pdfFooter(pdf) })

	if template == "buses-mixto" {
		pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
		drawBusesSummary(pdf, report)
		drawBusesFilters(pdf, report)
		pdf.SetFont("go", "B", 12)
		pdf.CellFormat(0, 8, "El detalle de la flota se presenta en la página horizontal.", "", 1, "L", false, 0, "")

		pdf.AddPageFormat("L", fpdf.SizeType{Wd: 297, Ht: 210})
		drawBusesTable(pdf, report)

		pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
		pdf.SetFont("go", "B", 12)
		pdf.CellFormat(0, 8, "Resumen por tipo", "", 1, "L", false, 0, "")
		drawBusesTypeSummary(pdf, report)
	} else {
		switch template {
		case "buses-a4-l":
			pdf.AddPageFormat("L", fpdf.SizeType{Wd: 297, Ht: 210})
		case "buses-dinamico":
			pdf.AddPage()
		default:
			pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
		}
		drawBusesSummary(pdf, report)
		drawBusesFilters(pdf, report)
		drawBusesTable(pdf, report)
		if template != "buses-dinamico" {
			pdf.Ln(4)
			pdf.SetFont("go", "B", 10)
			pdf.CellFormat(0, 6, "Resumen por tipo", "", 1, "L", false, 0, "")
			drawBusesTypeSummary(pdf, report)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawBusesTypeSummary(pdf *fpdf.Fpdf, report outputs.ReporteBuses) {
	pdf.SetFont("go", "", 8)
	pdf.SetFillColor(245, 247, 250)
	for _, t := range report.PorTipo {
		text := fmt.Sprintf("%-12s  cantidad: %-4d  precio: %.2f  capacidad prom.: %.1f",
			t.Tipo, t.Cantidad, t.PrecioTotal, t.CapacidadPromedio)
		pdf.CellFormat(0, 5, text, "1", 1, "L", true, 0, "")
	}
}

func GenerateVentaPDF(venta outputs.VentaResponse, template string) ([]byte, error) {
	var pdf *fpdf.Fpdf
	ticket := template == "venta-ticket"
	if ticket {
		height := 70.0 + float64(len(venta.Items))*12 + 34
		pdf = fpdf.NewCustom(&fpdf.InitType{UnitStr: "mm", Size: fpdf.SizeType{Wd: 80, Ht: height}})
	} else {
		pdf = fpdf.New("P", "mm", "A4", "")
	}
	addFonts(pdf)
	registerLogo(pdf)
	pdf.AliasNbPages("{nb}")
	pdf.SetAutoPageBreak(true, 18)
	if !ticket {
		pdf.SetMargins(10, 34, 10)
		title := "Factura"
		if template == "venta-nota" {
			title = "Nota de venta"
		}
		pdf.SetHeaderFunc(func() { pdfHeader(pdf, title, venta.Codigo) })
		pdf.SetFooterFunc(func() { pdfFooter(pdf) })
		pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
		drawVentaHeader(pdf, venta)
		drawVentaItems(pdf, venta)
		drawVentaTotals(pdf, venta)
	} else {
		pdf.SetMargins(5, 6, 5)
		pdf.AddPage()
		drawTicket(pdf, venta)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawVentaHeader(pdf *fpdf.Fpdf, venta outputs.VentaResponse) {
	pdf.SetFont("go", "B", 10)
	pdf.CellFormat(0, 6, "Datos del cliente", "", 1, "L", false, 0, "")
	pdf.SetFont("go", "", 9)
	pdf.CellFormat(0, 5, "Cliente: "+venta.ClienteNombre, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Documento: "+venta.ClienteDocumento, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Dirección: "+venta.ClienteDireccion, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Fecha: "+venta.Fecha+"   Moneda: "+venta.Moneda, "", 1, "L", false, 0, "")
	pdf.Ln(3)
}

func drawVentaItems(pdf *fpdf.Fpdf, venta outputs.VentaResponse) {
	width, _ := pdf.GetPageSize()
	available := width - 20
	cols := []struct {
		label string
		pct   float64
		align string
	}{
		{"Descripción", 42, "L"}, {"Cant.", 9, "C"}, {"P. Unit.", 14, "R"},
		{"Dscto.", 11, "R"}, {"Imp.", 11, "R"}, {"Total", 13, "R"},
	}
	var sum float64
	for _, c := range cols {
		sum += c.pct
	}
	widths := make([]float64, len(cols))
	for i, c := range cols {
		widths[i] = available * c.pct / sum
	}

	pdf.SetFont("go", "B", 8)
	pdf.SetFillColor(30, 41, 59)
	pdf.SetTextColor(255, 255, 255)
	for i, c := range cols {
		pdf.CellFormat(widths[i], 7, c.label, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)

	for idx, it := range venta.Items {
		if idx%2 == 0 {
			pdf.SetFillColor(245, 247, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		values := []string{
			it.Descripcion, fmt.Sprintf("%.2f", it.Cantidad), fmt.Sprintf("%.2f", it.PrecioUnitario),
			fmt.Sprintf("%.2f", it.Descuento), fmt.Sprintf("%.2f", it.Impuesto), fmt.Sprintf("%.2f", it.Total),
		}
		for i, v := range values {
			pdf.CellFormat(widths[i], 6, v, "1", 0, cols[i].align, true, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(4)
}

func drawVentaTotals(pdf *fpdf.Fpdf, venta outputs.VentaResponse) {
	width, _ := pdf.GetPageSize()
	x := width - 85
	pdf.SetXY(x, pdf.GetY())
	pdf.SetFont("go", "", 9)
	rows := [][2]string{
		{"Subtotal", fmt.Sprintf("%.2f", venta.Subtotal)},
		{"Descuento", fmt.Sprintf("%.2f", venta.Descuento)},
		{"Impuesto", fmt.Sprintf("%.2f", venta.Impuesto)},
		{"Total", fmt.Sprintf("%.2f %s", venta.Total, venta.Moneda)},
	}
	for i, row := range rows {
		if i == len(rows)-1 {
			pdf.SetFont("go", "B", 11)
			pdf.SetFillColor(240, 245, 255)
		}
		pdf.CellFormat(35, 6, row[0], "1", 0, "L", i == len(rows)-1, 0, "")
		pdf.CellFormat(35, 6, row[1], "1", 1, "R", i == len(rows)-1, 0, "")
	}
	if venta.Observaciones != "" {
		pdf.Ln(3)
		pdf.SetFont("go", "", 8)
		pdf.MultiCell(0, 5, "Observaciones: "+venta.Observaciones, "1", "L", false)
	}
}

func drawTicket(pdf *fpdf.Fpdf, venta outputs.VentaResponse) {
	pdf.SetFont("go", "B", 10)
	pdf.CellFormat(0, 6, "TEMPLATE", "", 1, "C", false, 0, "")
	pdf.SetFont("go", "", 7)
	pdf.CellFormat(0, 4, "Ticket de venta", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, venta.Codigo, "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, venta.Fecha+"  ·  "+venta.Moneda, "", 1, "C", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("go", "", 7)
	pdf.MultiCell(0, 4, "Cliente: "+venta.ClienteNombre+"\nDoc: "+venta.ClienteDocumento, "", "L", false)
	pdf.Ln(1)
	for _, it := range venta.Items {
		pdf.SetFont("go", "", 7)
		pdf.MultiCell(0, 4, fmt.Sprintf("%s\n%.2f x %.2f", it.Descripcion, it.Cantidad, it.PrecioUnitario), "", "L", false)
		pdf.SetFont("go", "B", 7)
		pdf.CellFormat(0, 4, fmt.Sprintf("%.2f", it.Total), "", 1, "R", false, 0, "")
	}
	pdf.Ln(1)
	pdf.SetFont("go", "", 7)
	pdf.CellFormat(0, 4, fmt.Sprintf("Subtotal: %.2f", venta.Subtotal), "", 1, "R", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("Impuesto: %.2f", venta.Impuesto), "", 1, "R", false, 0, "")
	pdf.SetFont("go", "B", 9)
	pdf.CellFormat(0, 5, fmt.Sprintf("TOTAL: %.2f", venta.Total), "", 1, "R", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("go", "", 7)
	pdf.CellFormat(0, 4, "Gracias por su compra", "", 1, "C", false, 0, "")
}
