package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// XML structures
type Invoice struct {
	XMLName       xml.Name `xml:"Invoice"`
	ID            string   `xml:"ID"`
	IssueDate     string   `xml:"IssueDate"`
	Reference     string   `xml:"DespatchDocumentReference>ID"`
	ReferenceName string   `xml:"AdditionalDocumentReference>ID"`

	Suplier  Customer `xml:"AccountingSupplierParty>Party"`
	Customer Customer `xml:"AccountingCustomerParty>Party"`

	LineExtension string `xml:"LegalMonetaryTotal>LineExtensionAmount"`
	TaxExclusive  string `xml:"LegalMonetaryTotal>TaxExclusiveAmount"`
	TaxInclusive  string `xml:"LegalMonetaryTotal>TaxInclusiveAmount"`
	Payable       string `xml:"LegalMonetaryTotal>PayableAmount"`
	Tax           string `xml:"TaxTotal>TaxAmount"`

	Lines []InvoiceLine `xml:"InvoiceLine"`
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

	Quantity  string `xml:"InvoicedQuantity"`    // kolicina
	UnitPrice string `xml:"Price>PriceAmount"`   // neto cijena
	Amount    string `xml:"LineExtensionAmount"` // neto iznos

	TaxPercent string `xml:"Item>ClassifiedTaxCategory>Percent"`      // pdv
	TaxID      string `xml:"Item>ClassifiedTaxCategory>ID"`           // sifra kategorije pdv-a
	TaxScheme  string `xml:"Item>ClassifiedTaxCategory>TaxScheme>ID"` // sifra kategorije pdv-a
}

func main() {
	//parse("48-prociscen.xml")

	files := []string{"invoices.txt", "customers.txt", "lines.txt"}
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			fmt.Printf("Error removing %s: %v\n", file, err)
		}
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if file and has .xml extension
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".xml" {
			fmt.Printf("processing: %s\n", path)
			parse(path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("filepath: %v", err)
	}
}

func parse(xmlFile string) {
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

	err = writeInvoiceCSV("invoices.txt", &invoice)
	if err != nil {
		log.Fatalf("Error writing invoice CSV: %v", err)
	}

	err = writeCustomer("customers.txt", &invoice)
	if err != nil {
		log.Fatalf("Error writing customer CSV: %v", err)
	}

	err = writeInvoiceLinesCSV("lines.txt", &invoice)
	if err != nil {
		log.Fatalf("Error writing lines CSV: %v", err)
	}
}

func writeInvoiceCSV(filename string, invoice *Invoice) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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
	header := []string{"ID", "Reference", "ReferenceName", "IssueDate", "LineExtension", "TaxExclusive", "TaxInclusive", "Tax", "Payable"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Invoice row
	row := []string{
		invoice.ID,
		invoice.Reference,
		invoice.ReferenceName,
		invoice.IssueDate,

		invoice.LineExtension,
		invoice.TaxExclusive,
		invoice.TaxInclusive,
		invoice.Tax,
		invoice.Payable,
	}
	return writer.Write(row)
}

func writeInvoiceLinesCSV(filename string, invoice *Invoice) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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
	header := []string{"InvoiceID", "ID", "ItemName", "ItemID", "Quantity", "UnitPrice", "Amount", "TaxPercent", "TaxScheme"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Invoice lines
	for _, line := range invoice.Lines {
		row := []string{
			invoice.ID,
			line.ID,

			line.ItemName,
			line.ItemID,

			formatNumber(line.Quantity),
			formatNumber(line.UnitPrice),
			formatNumber(line.Amount),

			formatNumber(line.TaxPercent),
			line.TaxID + "-" + line.TaxScheme,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func writeCustomer(filename string, invoice *Invoice) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

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

	for _, customer := range []Customer{invoice.Suplier, invoice.Customer} {
		row := []string{
			customer.ID,
			customer.Name,
			customer.OIB,

			customer.Street,
			customer.City,
			customer.PostalZone,
			customer.Country,

			customer.Contact,
			customer.Email,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func formatNumber(num string) string {
	return strings.ReplaceAll(num, ".", ",")
}
