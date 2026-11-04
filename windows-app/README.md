# Carpets Ameer — Windows Safe Save

A bilingual Arabic / English offline carpet-sales program for Windows 10/11 (64-bit).

**Start:** Double-click `Carpets_Ameer_Windows_SafeSave.exe`. The app runs as a Windows executable and opens its interface in your default browser. **No Java, installation, cloud account, or internet is required.** Keep the executable running while editing sales; the page talks only to a local service on your own PC.

## Features

- Price-per-square-metre calculator, carpet dimensions, quantity, purchase cost, selling price, profit and margin.
- Arabic and English interface; Arabic headings in the exported Excel sales ledger.
- Customer names, descriptions, payment methods, notes, reference numbers, monthly summaries.
- Edit and delete past sales.
- One remembered Excel workbook location: choose once, then every successful save updates the exact same `.xlsx` path rather than downloading another copy.
- Automatic save after edits, plus manual **Save to Excel**.
- **Verified, timestamped backups on every successful save**—both the new saved version and the previous version (when one exists).
- Keeps a separate on-disk sales draft *before* confirming an Add/Edit/Delete. A failed Excel save **does not erase that draft**.

## First use

1. Double-click `Carpets_Ameer_Windows_SafeSave.exe`.
2. Record a sale or import an existing sales workbook with **Open existing / فتح ملف موجود**.
3. Click **Save to Excel / حفظ في Excel**. For a new workbook, select a location such as `Documents\\Carpets_Ameer_Sales.xlsx`.
4. The app remembers that path between launches. Subsequent saves update that **same** file.
5. To update an existing Excel workbook, use **Open existing** first. The Save dialog refuses to overwrite an existing unimported workbook.

**Important:** Close the linked `.xlsx` in Microsoft Excel before saving from the app. If Excel locks the workbook, the app shows an error and keeps your draft and backup files. Close Excel and click Save again.

## Data and backups

- **Linked workbook:** wherever you choose, e.g. `Documents\\Carpets_Ameer_Sales.xlsx`.
- **Backup folder:** next to the workbook, named `Carpets_Ameer_Backups`.
- **PC draft + remembered workbook location:** `%APPDATA%\\CarpetsAmeer\\sales-draft.json` and `workbook-link.json`. The app also retains a previous draft snapshot.
- **Backups:** never automatically deleted by this program. Check available disk space occasionally.

The program writes a temporary Excel file, checks its integrity and SHA-256 checksum, writes and verifies backups, atomically replaces the linked Excel file, reads it back to verify success, and only then confirms the save. If the workbook was changed in Excel or another program since the last save, the app rejects the update to avoid overwriting someone else's changes. **Open existing** to import the revised workbook, then edit and save it.

The app generates its own standard Arabic Excel workbook with a sales ledger and monthly summary. Opening an unrelated `.xlsx` may not be supported; always review the imported sales before saving, and keep the original/backup.

## Uninstall

Delete the `.exe`. Your Excel workbook, backups, and draft stay on your PC until you delete them manually.

## Developer build

Go 1.23+ (standard library only):

```sh
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -H windowsgui' -o Carpets_Ameer_Windows_SafeSave.exe .
```

The executable is currently **unsigned**. Windows SmartScreen might show a warning on first launch. We have tested saving via automated Go and local API tests; this build has **not** been tested on a physical Windows PC.
