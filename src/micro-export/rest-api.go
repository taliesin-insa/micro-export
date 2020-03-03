package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Meta struct {
	Type string
	URL  string
}

type Location struct {
	Type    string
	Polygon [][2]int
	Id      string
}

type Data struct {
	Type       string
	LocationId string
	Value      string
	Id         string
}

type PiFFStruct struct {
	Meta     Meta
	Location []Location
	Data     []Data
	Children []int
	Parent   int
}

type Picture struct {
	_id        []byte
	PiFF       PiFFStruct
	Url        string
	Annotated  bool
	Corrected  bool
	SentToReco bool
	SentToUser bool
	Unreadable bool
}

type PictureArray struct {
	Pictures []Picture
}

func exportPiFF(w http.ResponseWriter, r *http.Request) {
	// get all PiFF from database
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, "http://database-api.gitlab-managed-apps.svc.cluster.local:8080/db/retrieve/all", nil)
	if err != nil {
		log.Printf("[ERROR] Get request: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Get request: " + err.Error()))
		return
	}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("[ERROR] Do request: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Do request: " + err.Error()))
		return
	}

	// get body of returned data
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("[ERROR] Read data: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Read data: " + err.Error()))
		return
	}

	// checks whether there was an error during request
	if response.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Do request: %v", err.Error())
		w.WriteHeader(response.StatusCode)
		w.Write(body)
		return
	}

	// transform json into struct
	var piFFData PictureArray
	err = json.Unmarshal(body, &piFFData)
	if err != nil {
		log.Printf("[ERROR] Unmarshal data: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Unmarshal data: " + err.Error()))
		return
	}

	// create zip file and zip writer
	outFile := new(bytes.Buffer)
	writer := zip.NewWriter(outFile)

	// add files to zip
	for _, picture := range piFFData.Pictures {
		// get image variables
		imageURL := picture.Url
		imagePath := ""

		segments := strings.Split(imageURL, "/")
		imageNameWithExt := segments[len(segments)-1] // image name with extension
		segments = strings.Split(imageNameWithExt, ".")
		imageName := segments[len(segments)-2] // image name without extension

		if picture.Unreadable { // if unreadable, we store the image and the file in a different folder
			imagePath = "Unreadable/"
		}

		// add file to zip
		file, err := writer.Create(imagePath + imageName + ".piff")
		if err != nil {
			log.Printf("[ERROR] Create piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Create piFF: " + err.Error()))
			return
		}

		piFF, err := json.MarshalIndent(picture.PiFF, "", "    ")
		if err != nil {
			log.Printf("[ERROR] Marshal piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Marshal piFF: " + err.Error()))
			return
		}

		_, err = file.Write(piFF)
		if err != nil {
			log.Printf("[ERROR] Write piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Write piFF: " + err.Error()))
			return
		}

		// add image to zip
		image, err := ioutil.ReadFile(imageURL)
		if err != nil {
			log.Printf("[ERROR] Read image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Read image: " + err.Error()))
			return
		}

		file, err = writer.Create(imagePath + imageNameWithExt)
		if err != nil {
			log.Printf("[ERROR] Create image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Create image: " + err.Error()))
			return
		}

		_, err = file.Write(image)
		if err != nil {
			log.Printf("[ERROR] Write image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Write image: " + err.Error()))
			return
		}

	}

	// close writer
	err = writer.Close()
	if err != nil {
		log.Printf("[ERROR] Close writer: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Close writer: " + err.Error()))
		return
	}

	// send data
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(outFile.Len()), 10))
	io.Copy(w, outFile)
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
