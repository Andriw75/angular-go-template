import { Injectable } from '@angular/core';
import type { ReporteBuses } from '../models/reporte';
import type { Venta } from '../models/venta';

type AnyRecord = Record<string, any>;

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

async function loadLogo(): Promise<string | null> {
  try {
    const res = await fetch(new URL('branding/logo.png', document.baseURI).toString());
    if (!res.ok) return null;
    const blob = await res.blob();
    return await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(blob);
    });
  } catch {
    return null;
  }
}

function money(v: number, moneda = 'PEN'): string {
  return `${moneda} ${v.toFixed(2)}`;
}

@Injectable({ providedIn: 'root' })
export class ReportExportService {
  async exportBusesPdf(report: ReporteBuses, template: string): Promise<void> {
    const { jsPDF } = await import('jspdf');
    const autoTableModule: any = await import('jspdf-autotable');
    const autoTable = autoTableModule.default ?? autoTableModule;
    const logo = await loadLogo();

    const portrait = template !== 'buses-a4-l';
    const dynamic = template === 'buses-dinamico';
    const format: any = dynamic ? [210, Math.max(180, 130 + report.data.length * 8)] : 'a4';
    const doc: any = new jsPDF({ unit: 'mm', format, orientation: portrait ? 'portrait' : 'landscape' });

    const pageWidth = () => doc.internal.pageSize.getWidth();
    const pageHeight = () => doc.internal.pageSize.getHeight();

    const drawHeader = () => {
      const w = pageWidth();
      doc.setFillColor(30, 41, 59);
      doc.rect(0, 0, w, 24, 'F');
      if (logo) doc.addImage(logo, 'PNG', 10, 4, 28, 7.5);
      doc.setTextColor(255, 255, 255);
      doc.setFontSize(13);
      doc.text('Reporte de Buses', 42, 10);
      doc.setFontSize(8);
      doc.text('Flota registrada', 42, 15);
      doc.setTextColor(0, 0, 0);
    };

    const drawFooter = () => {
      const total = doc.getNumberOfPages();
      for (let i = 1; i <= total; i++) {
        doc.setPage(i);
        const w = pageWidth();
        const h = pageHeight();
        doc.setDrawColor(200, 200, 200);
        doc.line(10, h - 14, w - 10, h - 14);
        doc.setFontSize(7);
        doc.setTextColor(100, 100, 100);
        doc.text(
          `Generado ${new Date().toLocaleString()}   ·   Página ${i} de ${total}`,
          w / 2,
          h - 8,
          { align: 'center' },
        );
        doc.setTextColor(0, 0, 0);
      }
    };

    const drawSummary = (y: number) => {
      const w = pageWidth();
      const available = w - 20;
      const boxW = (available - 18) / 4;
      const boxes = [
        ['Total buses', String(report.resumen.total), [30, 41, 59]],
        ['Activos', String(report.resumen.activos), [22, 163, 74]],
        ['Inactivos', String(report.resumen.inactivos), [220, 38, 38]],
        ['Precio total', money(report.resumen.precio_total), [37, 99, 235]],
      ] as const;
      boxes.forEach((b, idx) => {
        const x = 10 + idx * (boxW + 6);
        doc.setFillColor(b[2][0], b[2][1], b[2][2]);
        doc.roundedRect(x, y, boxW, 18, 2, 2, 'F');
        doc.setTextColor(255, 255, 255);
        doc.setFontSize(7);
        doc.text(b[0].toUpperCase(), x + 4, y + 5);
        doc.setFontSize(11);
        doc.text(b[1], x + 4, y + 13);
      });
      doc.setTextColor(0, 0, 0);
      return y + 24;
    };

    const drawFilters = (y: number) => {
      doc.setFontSize(8);
      doc.setFillColor(245, 247, 250);
      doc.setDrawColor(220, 225, 230);
      const text = `Filtros — q: ${report.filtros['q'] ?? ''}  |  tipo: ${report.filtros['tipo'] ?? ''}  |  activo: ${report.filtros['activo'] ?? ''}  |  desde: ${report.filtros['desde'] ?? ''}  |  hasta: ${report.filtros['hasta'] ?? ''}`;
      const lines = doc.splitTextToSize(text, pageWidth() - 24);
      const h = Math.max(8, lines.length * 4 + 4);
      doc.roundedRect(10, y, pageWidth() - 20, h, 1, 1, 'FD');
      doc.text(lines, 12, y + 5);
      return y + h + 4;
    };

    const drawTable = (y: number) => {
      autoTable(doc, {
        startY: y,
        head: [['Placa', 'Nombre', 'Marca', 'Tipo', 'Año', 'Cap.', 'Precio', 'Estado']],
        body: report.data.map((b) => [
          b.placa, b.nombre, b.marca, b.tipo, String(b.anio), String(b.capacidad),
          b.precio.toFixed(2), b.activo ? 'Activo' : 'Inactivo',
        ]),
        theme: 'grid',
        styles: { fontSize: 7, cellPadding: 1.4 },
        headStyles: { fillColor: [30, 41, 59], textColor: 255, halign: 'center' },
        alternateRowStyles: { fillColor: [245, 247, 250] },
        columnStyles: { 6: { halign: 'right' } },
        margin: { top: 30, left: 10, right: 10, bottom: 18 },
        didDrawPage: (data: any) => {
          if (data.pageNumber > 1) drawHeader();
        },
      });
      return (doc as any).lastAutoTable.finalY + 5;
    };

    const drawTypeSummary = (y: number) => {
      doc.setFontSize(10);
      doc.text('Resumen por tipo', 10, y);
      autoTable(doc, {
        startY: y + 3,
        head: [['Tipo', 'Cantidad', 'Precio total', 'Capacidad prom.']],
        body: report.por_tipo.map((t) => [t.tipo, String(t.cantidad), t.precio_total.toFixed(2), t.capacidad_promedio.toFixed(1)]),
        theme: 'grid',
        styles: { fontSize: 7 },
        headStyles: { fillColor: [30, 41, 59], textColor: 255 },
        margin: { left: 10, right: 10, bottom: 18 },
      });
    };

    drawHeader();
    let y = 32;
    if (template === 'buses-mixto') {
      y = drawSummary(y);
      y = drawFilters(y);
      doc.setFontSize(11);
      doc.text('El detalle de la flota se presenta en la página horizontal.', 10, y);
      doc.addPage('a4', 'landscape');
      drawHeader();
      drawTable(30);
      doc.addPage('a4', 'portrait');
      drawHeader();
      drawTypeSummary(32);
    } else {
      y = drawSummary(y);
      y = drawFilters(y);
      y = drawTable(y);
      drawTypeSummary(y);
    }

    drawFooter();
    const blob = doc.output('blob');
    downloadBlob(blob, `reporte-buses-${Date.now()}.pdf`);
  }

  async exportBusesExcel(report: ReporteBuses): Promise<void> {
    const ExcelJS: any = await import('exceljs');
    const Workbook = ExcelJS.Workbook ?? ExcelJS.default?.Workbook;
    const wb = new Workbook();

    const titleStyle = { font: { bold: true, size: 16, color: { argb: 'FF1E293B' } } };
    const headerStyle = {
      font: { bold: true, color: { argb: 'FFFFFFFF' }, size: 10 },
      fill: { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF1E293B' } },
      alignment: { horizontal: 'center', vertical: 'middle' },
      border: { top: { style: 'thin' }, bottom: { style: 'thin' }, left: { style: 'thin' }, right: { style: 'thin' } },
    };

    const resumen = wb.addWorksheet('Resumen');
    resumen.getCell('A1').value = 'Reporte de Buses';
    resumen.getCell('A1').style = titleStyle;
    resumen.getCell('A3').value = 'Total';
    resumen.getCell('B3').value = report.resumen.total;
    resumen.getCell('A4').value = 'Activos';
    resumen.getCell('B4').value = report.resumen.activos;
    resumen.getCell('A5').value = 'Inactivos';
    resumen.getCell('B5').value = report.resumen.inactivos;
    resumen.getCell('A6').value = 'Precio total';
    resumen.getCell('B6').value = report.resumen.precio_total;
    resumen.getCell('A7').value = 'Peso total';
    resumen.getCell('B7').value = report.resumen.peso_total;
    resumen.getCell('A8').value = 'Capacidad promedio';
    resumen.getCell('B8').value = report.resumen.capacidad_promedio;
    resumen.getColumn('A').width = 24;
    resumen.getColumn('B').width = 20;

    const sheet = wb.addWorksheet('Buses');
    const headers = ['Placa', 'Nombre', 'Marca', 'Modelo', 'Año', 'Capacidad', 'Tipo', 'Precio', 'Peso', 'Color', 'Activo', 'Fecha compra'];
    sheet.addRow(headers);
    sheet.getRow(1).eachCell((cell: any) => { cell.style = headerStyle; });
    report.data.forEach((b) => {
      sheet.addRow([b.placa, b.nombre, b.marca, b.modelo, b.anio, b.capacidad, b.tipo, b.precio, b.peso, b.color, b.activo ? 'Activo' : 'Inactivo', b.fecha_compra]);
    });
    sheet.columns.forEach((col: any, i: number) => { col.width = i === 1 ? 24 : 14; });
    sheet.autoFilter = { from: { row: 1, column: 1 }, to: { row: Math.max(1, report.data.length + 1), column: headers.length } };
    sheet.views = [{ state: 'frozen', ySplit: 1 }];

    const tipoSheet = wb.addWorksheet('Por tipo');
    tipoSheet.addRow(['Tipo', 'Cantidad', 'Precio total', 'Capacidad prom.']);
    tipoSheet.getRow(1).eachCell((cell: any) => { cell.style = headerStyle; });
    report.por_tipo.forEach((t) => tipoSheet.addRow([t.tipo, t.cantidad, t.precio_total, t.capacidad_promedio]));
    tipoSheet.columns.forEach((col: any) => { col.width = 22; });

    const buffer = await wb.xlsx.writeBuffer();
    downloadBlob(new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), `reporte-buses-${Date.now()}.xlsx`);
  }

  async exportVentaPdf(venta: Venta, template: string): Promise<void> {
    const { jsPDF } = await import('jspdf');
    const autoTableModule: any = await import('jspdf-autotable');
    const autoTable = autoTableModule.default ?? autoTableModule;
    const logo = await loadLogo();
    const ticket = template === 'venta-ticket';
    const ticketHeight = ticket ? Math.max(110, 70 + venta.items.length * 12) : 0;
    const docFormat: any = ticket ? [80, ticketHeight] : 'a4';
    const doc: any = new jsPDF({ unit: 'mm', format: docFormat, orientation: 'portrait' });

    const drawHeader = () => {
      if (ticket) return;
      const w = doc.internal.pageSize.getWidth();
      doc.setFillColor(30, 41, 59);
      doc.rect(0, 0, w, 24, 'F');
      if (logo) doc.addImage(logo, 'PNG', 10, 4, 28, 7.5);
      doc.setTextColor(255, 255, 255);
      doc.setFontSize(13);
      doc.text(template === 'venta-nota' ? 'Nota de venta' : 'Factura', 42, 10);
      doc.setFontSize(8);
      doc.text(venta.codigo, 42, 15);
      doc.setTextColor(0, 0, 0);
    };

    drawHeader();
    if (ticket) {
      doc.setFontSize(10);
      doc.text('TEMPLATE', 40, 8, { align: 'center' });
      doc.setFontSize(7);
      doc.text('Ticket de venta', 40, 13, { align: 'center' });
      doc.text(venta.codigo, 40, 18, { align: 'center' });
      doc.text(`${venta.fecha} · ${venta.moneda}`, 40, 23, { align: 'center' });
      let y = 30;
      doc.text(`Cliente: ${venta.cliente_nombre}`, 5, y);
      y += 4;
      doc.text(`Doc: ${venta.cliente_documento}`, 5, y);
      y += 6;
      venta.items.forEach((it) => {
        doc.text(it.descripcion, 5, y);
        y += 4;
        doc.text(`${it.cantidad.toFixed(2)} x ${it.precio_unitario.toFixed(2)}`, 5, y);
        doc.text(it.total.toFixed(2), 75, y, { align: 'right' });
        y += 5;
      });
      y += 2;
      doc.text(`Subtotal: ${venta.subtotal.toFixed(2)}`, 75, y, { align: 'right' });
      y += 4;
      doc.text(`Impuesto: ${venta.impuesto.toFixed(2)}`, 75, y, { align: 'right' });
      y += 5;
      doc.setFontSize(9);
      doc.text(`TOTAL: ${venta.total.toFixed(2)}`, 75, y, { align: 'right' });
      y += 6;
      doc.setFontSize(7);
      doc.text('Gracias por su compra', 40, y, { align: 'center' });
    } else {
      let y = 32;
      doc.setFontSize(9);
      doc.text(`Cliente: ${venta.cliente_nombre}`, 10, y); y += 5;
      doc.text(`Documento: ${venta.cliente_documento}`, 10, y); y += 5;
      doc.text(`Dirección: ${venta.cliente_direccion}`, 10, y); y += 5;
      doc.text(`Fecha: ${venta.fecha}   Moneda: ${venta.moneda}`, 10, y); y += 8;

      autoTable(doc, {
        startY: y,
        head: [['Descripción', 'Cant.', 'P. Unit.', 'Dscto.', 'Imp.', 'Total']],
        body: venta.items.map((it) => [it.descripcion, it.cantidad.toFixed(2), it.precio_unitario.toFixed(2), it.descuento.toFixed(2), it.impuesto.toFixed(2), it.total.toFixed(2)]),
        theme: 'grid',
        styles: { fontSize: 8 },
        headStyles: { fillColor: [30, 41, 59], textColor: 255 },
        margin: { left: 10, right: 10, bottom: 18 },
      });
      y = (doc as any).lastAutoTable.finalY + 6;
      const x = doc.internal.pageSize.getWidth() - 70;
      doc.setFontSize(9);
      doc.text('Subtotal', x, y); doc.text(venta.subtotal.toFixed(2), doc.internal.pageSize.getWidth() - 10, y, { align: 'right' }); y += 5;
      doc.text('Descuento', x, y); doc.text(venta.descuento.toFixed(2), doc.internal.pageSize.getWidth() - 10, y, { align: 'right' }); y += 5;
      doc.text('Impuesto', x, y); doc.text(venta.impuesto.toFixed(2), doc.internal.pageSize.getWidth() - 10, y, { align: 'right' }); y += 5;
      doc.setFontSize(11);
      doc.text('TOTAL', x, y); doc.text(`${venta.total.toFixed(2)} ${venta.moneda}`, doc.internal.pageSize.getWidth() - 10, y, { align: 'right' });

      const totalPages = doc.getNumberOfPages();
      for (let i = 1; i <= totalPages; i++) {
        doc.setPage(i);
        const w = doc.internal.pageSize.getWidth();
        const h = doc.internal.pageSize.getHeight();
        doc.setDrawColor(200, 200, 200);
        doc.line(10, h - 14, w - 10, h - 14);
        doc.setFontSize(7);
        doc.setTextColor(100, 100, 100);
        doc.text(`Página ${i} de ${totalPages}`, w / 2, h - 8, { align: 'center' });
        doc.setTextColor(0, 0, 0);
      }
    }

    downloadBlob(doc.output('blob'), `${venta.codigo || 'venta'}-${template}.pdf`);
  }

  async exportVentaExcel(venta: Venta): Promise<void> {
    const ExcelJS: any = await import('exceljs');
    const Workbook = ExcelJS.Workbook ?? ExcelJS.default?.Workbook;
    const wb = new Workbook();
    const sheet = wb.addWorksheet('Factura');
    sheet.getCell('A1').value = 'Factura / Nota de venta';
    sheet.getCell('A1').font = { bold: true, size: 16 };
    sheet.getCell('A3').value = 'Código';
    sheet.getCell('B3').value = venta.codigo;
    sheet.getCell('A4').value = 'Fecha';
    sheet.getCell('B4').value = venta.fecha;
    sheet.getCell('A5').value = 'Cliente';
    sheet.getCell('B5').value = venta.cliente_nombre;
    sheet.getCell('A6').value = 'Documento';
    sheet.getCell('B6').value = venta.cliente_documento;

    const headerRow = sheet.getRow(8);
    ['Descripción', 'Cantidad', 'P. Unit.', 'Descuento', 'Impuesto', 'Total'].forEach((h, i) => {
      const cell = headerRow.getCell(i + 1);
      cell.value = h;
      cell.font = { bold: true, color: { argb: 'FFFFFFFF' } };
      cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF1E293B' } };
    });
    venta.items.forEach((it) => {
      sheet.addRow([it.descripcion, it.cantidad, it.precio_unitario, it.descuento, it.impuesto, it.total]);
    });
    sheet.columns.forEach((col: any, i: number) => { col.width = i === 0 ? 32 : 14; });

    const start = 10 + venta.items.length;
    sheet.getCell(`E${start}`).value = 'Subtotal';
    sheet.getCell(`F${start}`).value = venta.subtotal;
    sheet.getCell(`E${start + 1}`).value = 'Impuesto';
    sheet.getCell(`F${start + 1}`).value = venta.impuesto;
    sheet.getCell(`E${start + 2}`).value = 'Total';
    sheet.getCell(`F${start + 2}`).value = venta.total;
    sheet.getCell(`E${start + 2}`).font = { bold: true };
    sheet.getCell(`F${start + 2}`).font = { bold: true };

    const buffer = await wb.xlsx.writeBuffer();
    downloadBlob(new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), `${venta.codigo || 'venta'}.xlsx`);
  }
}
