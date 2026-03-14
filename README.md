# fina

Alat za obradu e-računa primljenih i poslanih putem sustava FINA e-Račun.

Čita XML datoteke (UBL 2.0 format) iz zadanih mapa, uspoređuje ih s Obrascem URA te generira tri izlazne datoteke za uvoz u računovodstveni sustav.

## Priprema

Preuzmi ZIP s FINA portala koji sadrži mape `Primljeni/` i `Poslani/` te datoteku `Obrazac_URA.xls`.

## Pokretanje

```
./fina --input <mapa> [--input <mapa>] [--ura <datoteka>] [--output <mapa>]
```

### Zastavice

| Zastavica | Opis | Zadano |
|-----------|------|--------|
| `--input` | Mapa s XML računima (može se ponoviti više puta) | — |
| `--ura` | Putanja do `Obrazac_URA.xls` ili `.csv` | auto-otkrije u `--input` mapama |
| `--output` | Mapa za izlazne datoteke | `./output` |

### Primjer

```
./fina --input ~/Downloads/fina/Primljeni --input ~/Downloads/fina/Poslani --ura ~/Downloads/fina/Obrazac_URA.xls
```

Ako je `Obrazac_URA.xls` u jednoj od `--input` mapa, zastavica `--ura` nije potrebna:

```
./fina --input ~/Downloads/fina
```

## Izlaz

Generira tri datoteke u `./output/` (Windows-1250, točka-zarez delimiter, CRLF):

| Datoteka | Sadržaj |
|----------|---------|
| `invoices.txt` | Zaglavlja računa |
| `customer.txt` | Dobavljači i kupci (bez duplikata) |
| `lines.txt` | Stavke računa |

Računi koji se ne nalaze u Obrascu URA se preskaču.
