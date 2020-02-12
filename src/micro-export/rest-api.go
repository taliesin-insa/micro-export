package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
)

func exportPiFF(w http.ResponseWriter, r *http.Request) {
	file, err := getPiFFArchive()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// send data
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(file.Len()), 10))
	io.Copy(w, file)
}

// function to test whether docker file is correctly built
func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/export/piff", exportPiFF).Methods("GET")
	router.HandleFunc("/", homeLink).Methods("GET")
	log.Fatal(http.ListenAndServe(":22022", router))
}
