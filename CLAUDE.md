# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build

# Build and run
go build && ./fina --input ./input/Primljeni --input ./input/Poslani --ura ./input/Obrazac_URA.csv

# Run directly without building
go run main.go --input ./input/Primljeni --input ./input/Poslani --ura ./input/Obrazac_URA.csv

# Run with auto-discovered URA CSV (looks for CSV in input folders)
go run main.go --input ./input/Primljeni --input ./input/Poslani
```

## Project Overview

Single-file Go tool (`main.go`) that processes Croatian e-invoices (UBL 2.0 XML format) from the Croatian Financial Agency (FINA). It cross-references invoices against a URA reference CSV and outputs normalized data for accounting.

**Company context:** Processes invoices for Zebra (OIB: 37617049457). Invoices not matching this OIB are filtered out.

## Architecture

**Single file:** All logic is in `main.go`. No packages or submodules.

**Data flow:**
1. Load `Obrazac_URA.csv` → builds map of `(InvoiceID-SupplierOIB)` → invoice number (`Broj`)
2. Walk input directories recursively for `.xml` files
3. Parse each XML into `Invoice` struct, deduplicate by `(ID + SupplierID)`
4. Skip invoices not found in URA reference
5. Write three output files to `./output/` (default)

**Output files** (semicolon-delimited, Windows-1250 encoded, CRLF line endings):
- `invoices.txt` — invoice headers
- `customer.txt` — unique supplier/customer records
- `lines.txt` — line-item details

**Key details:**
- `readUra()` auto-detects CSV delimiter (comma vs semicolon) by sampling first lines
- `fixRune()` corrects mis-encoded Croatian characters (Č, č, Ć, ć, Đ, đ, Š, š, Ž, ž)
- Unit codes are translated: H87/PCE→kom, KGM→kg, MTR→m, LTR→l, HUR→sat, DAY→dan
- Dates formatted as `DD.MM.YYYY HH:MM:SS`, decimals use comma not period

**CLI flags:**
- `--input` (repeatable) — folders containing XML invoices
- `--ura` — path to URA CSV reference file (auto-discovered from input folders if omitted)
- `--output` — output folder (default: `./output`)
