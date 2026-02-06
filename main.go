package main

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"flag"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// XML structures
type Invoice struct {
	XMLName       xml.Name `xml:"Invoice"`
	ID            string   `xml:"ID"`
	IssueDate     string   `xml:"IssueDate"`
	IssueTime     string   `xml:"IssueTime"`
	DueDate       string   `xml:"DueDate"`
	Reference     string   `xml:"DespatchDocumentReference>ID"`
	ReferenceName string   `xml:"AdditionalDocumentReference>ID"`

	Supplier Customer `xml:"AccountingSupplierParty>Party"`
	Customer Customer `xml:"AccountingCustomerParty>Party"`

	LineExtension string `xml:"LegalMonetaryTotal>LineExtensionAmount"`
	TaxExclusive  string `xml:"LegalMonetaryTotal>TaxExclusiveAmount"`
	TaxInclusive  string `xml:"LegalMonetaryTotal>TaxInclusiveAmount"`
	Payable       string `xml:"LegalMonetaryTotal>PayableAmount"`
	Tax           string `xml:"TaxTotal>TaxAmount"`

	Lines []InvoiceLine `xml:"InvoiceLine"`

	Broj string
}

type Customer struct {
	ID   string `xml:"EndpointID"`
	Name string `xml:"PartyName>Name"`
	OIB  string `xml:"PartyTaxScheme>CompanyID"`

	Address    string `xml:"PostalAddress>AddressLine>Line"`
	Street     string `xml:"PostalAddress>StreetName"`
	City       string `xml:"PostalAddress>CityName"`
	PostalZone string `xml:"PostalAddress>PostalZone"`
	Country    string `xml:"PostalAddress>Country>IdentificationCode"`

	Contact string `xml:"Contact>Name"`
	Email   string `xml:"Contact>ElectronicMail"`
}

type InvoiceLine struct {
	ID string `xml:"ID"` // redni broj

	ItemName string `xml:"Item>Name"` // naziv artikla
	ItemID   string `xml:"Item>SellersItemIdentification>ID"`

	Quantity  Quantity `xml:"InvoicedQuantity"`    // kolicina
	UnitPrice string   `xml:"Price>PriceAmount"`   // neto cijena
	Amount    string   `xml:"LineExtensionAmount"` // neto iznos

	TaxPercent string `xml:"Item>ClassifiedTaxCategory>Percent"`      // pdv
	TaxID      string `xml:"Item>ClassifiedTaxCategory>ID"`           // sifra kategorije pdv-a
	TaxScheme  string `xml:"Item>ClassifiedTaxCategory>TaxScheme>ID"` // sifra kategorije pdv-a
}

type Quantity struct {
	Value    string `xml:",chardata"`     // The number (5)
	UnitCode string `xml:"unitCode,attr"` // The attribute
}

// var xml_paths = [2]string{
// 	"/home/ianic/Downloads/wetransfer_poslani_2026-02-06_1605/Primljeni/",
// 	"/home/ianic/Downloads/wetransfer_poslani_2026-02-06_1605/Poslani/",
// }

var output = "./output"

// const ura_path = "/home/ianic/Downloads/wetransfer_poslani_2026-02-06_1605/Obrazac_URA-2.csv"
const zebra_oib = "37617049457"

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s) // For -h usage
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var input []string
	flag.Var((*stringSlice)(&input), "input", "input folder")
	var ura_path = flag.String("ura", "Obrazac_URA-2.csv", "obrazac ura path")
	var outputFlag = flag.String("output", output, "output folder")
	flag.Parse()
	output = *outputFlag

	err := os.MkdirAll(output, 0755)
	if err != nil {
		log.Fatal(err)
	}

	files := []string{"invoices.txt", "customer.txt", "lines.txt"}
	for _, file := range files {
		if err := os.Remove(filepath.Join(output, file)); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error removing %s: %v\n", file, err)
		}
	}

	var invoices []Invoice
	de_dup_id := make(map[string]struct{})

	ura, err := readUra(*ura_path)
	if err != nil {
		log.Fatalf("read ura error: %v", err)
	}

	for _, path := range input {
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Check if file and has .xml extension
			if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".xml" {
				invoice := parse(path)
				fmt.Printf("%-20s ", path)
				defer fmt.Printf("\n")
				key := invoice.ID + "-" + invoice.Supplier.ID

				if _, ok := de_dup_id[key]; ok {
					fmt.Printf("preskacem duplikat ID: %s OIB: %s", invoice.ID, invoice.Supplier.ID)
					return nil
				} else {

					if invoice.Customer.ID == zebra_oib {
						broj, broj_ok := ura[key]
						if !broj_ok {
							fmt.Printf("preskacem nema broja racuna %s !!!", key)
							return nil
						}
						invoice.Broj = broj
					}
					fmt.Printf("OK")
					invoices = append(invoices, invoice)
					de_dup_id[key] = struct{}{}
				}

			}
			return nil
		})
		if err != nil {
			log.Fatalf("filepath: %v", err)
		}
	}

	writeFiles(invoices)
}

func parse(xmlFile string) Invoice {
	// Read XML file
	data, err := os.ReadFile(xmlFile)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Parse XML
	var invoice Invoice
	err = xml.Unmarshal(data, &invoice)
	if err != nil {
		log.Fatalf("Error parsing XML: %v", err)
	}

	return invoice
}

func writeFiles(invoices []Invoice) {
	err := writeInvoice("invoices.txt", invoices)
	if err != nil {
		log.Fatalf("Error writing invoice CSV: %v", err)
	}

	err = writeCustomer("customer.txt", invoices)
	if err != nil {
		log.Fatalf("Error writing customer CSV: %v", err)
	}

	err = writeInvoiceLines("lines.txt", invoices)
	if err != nil {
		log.Fatalf("Error writing lines CSV: %v", err)
	}
}

func writeInvoice(filename string, invoices []Invoice) error {
	file, err := os.OpenFile(filepath.Join(output, filename), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := charmap.Windows1250.NewEncoder()
	transformedWriter := transform.NewWriter(file, encoder)
	defer transformedWriter.Close()

	writer := csv.NewWriter(transformedWriter)
	writer.Comma = ';'
	writer.UseCRLF = true

	defer writer.Flush()

	// Header
	header := []string{"ID", "IssueDate", "DueDate", "Supplier", "Customer",
		"Reference", "ReferenceName",
		"LineExtension", "TaxExclusive", "TaxInclusive", "Tax", "Payable",
		"Broj",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, invoice := range invoices {
		// Invoice row
		row := []string{
			invoice.ID,
			formatDate(invoice.IssueDate + " " + invoice.IssueTime),
			formatDate(invoice.DueDate),
			invoice.Supplier.ID,
			invoice.Customer.ID,

			invoice.Reference,
			invoice.ReferenceName,

			invoice.LineExtension,
			invoice.TaxExclusive,
			invoice.TaxInclusive,
			invoice.Tax,
			invoice.Payable,

			invoice.Broj,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func writeInvoiceLines(filename string, invoices []Invoice) error {
	file, err := os.OpenFile(filepath.Join(output, filename), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := charmap.Windows1250.NewEncoder()
	transformedWriter := transform.NewWriter(file, encoder)
	defer transformedWriter.Close()

	writer := csv.NewWriter(transformedWriter)
	writer.Comma = ';'
	writer.UseCRLF = true
	defer writer.Flush()

	// Header
	header := []string{"InvoiceID", "ID",
		"ItemName", "ItemID",
		"Quantity", "UnitPrice", "Amount", "Unit",
		"TaxPercent", "TaxScheme"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, invoice := range invoices {
		// Invoice lines

		for _, line := range invoice.Lines {
			unit, ok := unitCodes[line.Quantity.UnitCode]
			if !ok {
				unit = line.Quantity.UnitCode
			}
			row := []string{
				invoice.ID,
				line.ID,

				line.ItemName,
				line.ItemID,

				formatNumber(line.Quantity.Value),
				formatNumber(line.UnitPrice),
				formatNumber(line.Amount),
				unit,

				formatNumber(line.TaxPercent),
				line.TaxID + "-" + line.TaxScheme,
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

var unitCodes = map[string]string{
	"H87": "kom",
	"PCE": "kom",
	"KGM": "kg",
	"MTR": "m",
	"LTR": "l",
	"HUR": "sat",
	"DAY": "dan",
}

func writeCustomer(filename string, invoices []Invoice) error {
	file, err := os.OpenFile(filepath.Join(output, filename), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := charmap.Windows1250.NewEncoder()
	transformedWriter := transform.NewWriter(file, encoder)
	defer transformedWriter.Close()

	writer := csv.NewWriter(transformedWriter)
	writer.Comma = ';'
	writer.UseCRLF = true
	defer writer.Flush()

	// Header
	header := []string{"ID", "Name", "OIB", "Street", "City", "PostalZone", "Country", "Contact", "Email"}
	if err := writer.Write(header); err != nil {
		return err
	}

	de_dup_id := make(map[string]struct{})

	for _, invoice := range invoices {
		for _, customer := range []Customer{invoice.Supplier, invoice.Customer} {
			if _, ok := de_dup_id[customer.ID]; ok {
				continue
			} else {
				de_dup_id[customer.ID] = struct{}{}
			}
			row := []string{
				customer.ID,
				customer.Name,
				customer.OIB,

				fixRune(customer.Street),
				fixRune(customer.City),
				customer.PostalZone,
				customer.Country,

				customer.Contact,
				customer.Email,
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

func fixRune(s string) string {
	var buf bytes.Buffer
	last_unknown := false
	for _, r := range s {
		if r <= unicode.MaxASCII { //|| utf8.ValidRune(r) {
			buf.WriteRune(r)
			last_unknown = false
		} else {
			if last_unknown {
				buf.WriteRune('?')
			}
			last_unknown = true
		}
	}
	return buf.String()
}

// func fixRune(field string) string {
// 	return string([]byte(field))
// }

func formatDate(input string) string {
	// Format: YYYY-MM-DD HH:MM:SS
	t, err := time.Parse("2006-01-02 15:04:05", input)
	if err != nil {
		t, err = time.Parse("2006-01-02", input)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			return ""
		}
	}
	return t.Format("02.01.2006 15:04:05")
}

func formatNumber(num string) string {
	return strings.ReplaceAll(num, ".", ",")
}

func readUra(filename string) (map[string]string, error) {
	// 1. Open the file
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Detect delimiter from first lines
	delim := detectDelimiter(f)
	fmt.Printf("Detected delimiter: '%c'\n", delim)

	// Rewind file
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	// 2. Initialize the CSV reader
	reader := csv.NewReader(f)
	reader.Comma = delim

	// 3. Read all records
	// records is a [][]string (2D slice of strings)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	ura := make(map[string]string)
	// 4. Map records to structs
	for i, row := range records {
		if i == 0 { // Skip header row if present
			continue
		}
		id := row[2]
		supplierID := row[7]
		broj := row[1]
		key := id + "-" + supplierID
		ura[key] = broj

		fmt.Printf("ura: %-7s %-25s %s\n", broj, id, supplierID)
	}

	return ura, nil
}

func detectDelimiter(r io.Reader) rune {
	// Read first few lines for analysis
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	data := string(buf[:n])

	lines := strings.SplitN(data, "\n", 3) // First 3 lines max

	commaCount, semicolonCount := 0, 0

	for _, line := range lines {
		if line == "" {
			continue
		}
		commaCount += strings.Count(line, ",")
		semicolonCount += strings.Count(line, ";")
	}

	if semicolonCount > commaCount {
		return ';'
	}
	return ','
}
