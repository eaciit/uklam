package test

import (
    "github.com/eaciit/uklam"
    "github.com/eaciit/toolkit"
    "testing"
)

var path string = "/users/ariefdarmawan/Temp/bhesada/original"

func TestWalk(t *testing.T){
    toolkit.Println("OK")
    
    w := uklam.NewFS(path)
    w.Start()
    
    w.Stop()
}