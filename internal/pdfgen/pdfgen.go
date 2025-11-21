package pdfgen

import (
	"bytes"
	"fmt"

	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/models"

	"github.com/jung-kurt/gofpdf"
)

func GeneratePDF(sets []*models.LinkSet) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "") //портрет, ед. изм., формат страницы, путь к шрифтам

	pdf.AddUTF8Font("DejaVu", "", "./internal/pdfgen/fonts/DejaVuSans.ttf") //для unicode символов
	pdf.SetFont("DejaVu", "", 12)

	pdf.AddPage() //добавляет новую страницу
	pdf.Ln(12)    //отступ 12мм

	for _, s := range sets {
		//создается текстовая ячейка на ширину строки
		pdf.Cell(0, 8, fmt.Sprintf("Set %d - created: %s", s.ID, s.CreatedAt.Format("2006-01-02 15:04:05")))
		pdf.Ln(8)
		for _, url := range s.Links {
			res := s.Results[url]
			st := "unknown"
			if res != nil {
				switch res.State {
				case "available":
					st = "available"
				case "not_available":
					st = "not available"
				default:
					st = string(res.State)
				}
			}
			pdf.CellFormat(0, 7, fmt.Sprintf("%s - %s", url, st), "", 1, "", false, 0, "")
		}
		pdf.Ln(4)
	}

	buf := &bytes.Buffer{} //создаем буфер для pdf
	err := pdf.Output(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
