package main

import (
	"net/http"
	"net/http/httptest"
)

// RegistrationFMTEnH shows a registraton form for the FMT
func RegistrationFMTEnH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if false {
		ExampleRaw()
	}
	ExampleUsingLib()
}

func main() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	RegistrationFMTEnH(w, req)
}
