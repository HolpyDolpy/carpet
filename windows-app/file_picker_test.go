//go:build windows

package main

import (
  "encoding/base64"
  "path/filepath"
  "strings"
  "testing"
)

func TestDecodeFilePickerResult(t *testing.T) {
 cases:=[]struct{name, mode, output, wanted string; hasError bool}{
    {"Arabic workbook","open", "notice\r\nCARPETS_FILE_PATH_B64:"+base64.StdEncoding.EncodeToString([]byte("C:\\\\مبيعات\\\\سجاد نوفمبر.xlsx")),"C:\\\\مبيعات\\\\سجاد نوفمبر.xlsx",false},
    {"Extension added","save","CARPETS_FILE_PATH_B64:"+base64.StdEncoding.EncodeToString([]byte("C:\\\\Users\\\\Test\\\\Documents\\\\Sales")),"C:\\\\Users\\\\Test\\\\Documents\\\\Sales.xlsx",false},
    {"Cancelled","save","CARPETS_FILE_DIALOG_CANCELLED","",false},
    {"Unexpected output","save","Windows Forms failed","",true},
    {"Bad encoding","save","CARPETS_FILE_PATH_B64:***","",true},
  }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
    got,err:=decodeFilePickerResult(tc.output,tc.mode)
    if (err!=nil)!=tc.hasError { t.Fatalf("error=%v",err) }
    if got!=tc.wanted {t.Fatalf("got %q want %q",got,tc.wanted)}
  })
 }
}
func TestSavePathExtension(t *testing.T) {
 for _,name:=range []string{"sales.xlsx","سجاد نوفمبر.xlsx"} {
   if !strings.EqualFold(filepath.Ext(name),".xlsx") {t.Errorf("extension broken: %s",name)}
 }
}
