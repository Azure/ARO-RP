package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestLowercase(t *testing.T) {
	w := httptest.NewRecorder()
	r := mux.NewRouter()
	r.Use(Lowercase)

	r.NewRoute().Path("/test").Methods(http.MethodGet).
		HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Write([]byte("ack"))
		})
	h := Lowercase(r)

	req := httptest.NewRequest(http.MethodGet, "/TeST", bytes.NewBuffer([]byte("")))
	h.ServeHTTP(w, req)

	if w.Body.String() != "ack" {
		t.Errorf("path lowercase sensitivity test failed")
	}
}
