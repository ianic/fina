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
	"time"
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

const path = "./txt/"

func main() {
	files := []string{"invoices.txt", "customer.txt", "lines.txt"}
	for _, file := range files {
		if err := os.Remove(path + file); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error removing %s: %v\n", file, err)
		}
	}

	var invoices []Invoice
	de_dup_id := make(map[string]struct{})

	err := filepath.Walk("./xml/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if file and has .xml extension
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".xml" {
			invoice := parse(path)
			fmt.Printf("processing: %s ", path)

			if _, ok := de_dup_id[invoice.ID]; ok {
				fmt.Printf("skipping duplicate invoice id: %s", invoice.ID)
			} else {
				invoices = append(invoices, invoice)
				de_dup_id[invoice.ID] = struct{}{}
			}
			fmt.Printf("\n")
		}
		return nil
	})
	if err != nil {
		log.Fatalf("filepath: %v", err)
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

	var err = writeInvoice("invoices.txt", invoices)
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
	file, err := os.OpenFile(path+filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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
		"LineExtension", "TaxExclusive", "TaxInclusive", "Tax", "Payable"}
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
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func writeInvoiceLines(filename string, invoices []Invoice) error {
	file, err := os.OpenFile(path+filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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

	for _, invoice := range invoices {
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
	}

	return nil
}

func writeCustomer(filename string, invoices []Invoice) error {
	file, err := os.OpenFile(path+filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
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
	}

	return nil
}

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
