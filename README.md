# Carpet Sales Website | موقع مبيعات السجاد

Bilingual (Arabic / English) offline-first carpet sales ledger and pricing calculator.

## Features

- Carpet dimensions and price per square meter calculator
- Sales entries with date, customer, item description, purchase cost, cash sale price, payment method, and notes
- Profit, profit margin, daily totals, and monthly summaries
- Arabic/English interface switch with RTL layout
- Local browser storage for sales
- Excel (.xlsx) create/update on supported browsers (Chrome / Edge)

## Run

Open `index.html` in Chrome or Edge. No server installation is necessary.

For hosting, deploy this repository as a static website. **Data is saved locally in each browser**, not in GitHub or on a shared server. Back up your Excel workbook regularly.

To update the same Excel workbook, use Chrome/Edge and grant access to the selected file when prompted. Other browsers may download a new file instead.

## Security

This repository contains application source code only. Do not commit customer data or Excel files containing sales to a public repository.

## License

Personal project; all rights reserved.


## Windows desktop app

The separate [Windows Safe Save application](windows-app/) lets you choose an Excel workbook once and update that same file, with local drafts and timestamped verified backups. It includes an Arabic/English interface. The app is offline-first and runs as a Windows executable that opens a local browser window; it does not send customer data to GitHub.

To get the executable, open **Actions → Build Windows Safe Save** and download the build artifact. Or build from source with Go 1.23 from the `windows-app` folder.

**Privacy:** This public source code repository must not contain live sales workbooks, customer records, or backup files. Data remains on the user's PC.
