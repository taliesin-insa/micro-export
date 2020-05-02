package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var DatabaseAPI string

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
	Filename   string
	Annotated  bool
	Corrected  bool
	SentToReco bool
	SentToUser bool
	Unreadable bool
	Annotator  string
}

func exportPiFF(w http.ResponseWriter, r *http.Request) {
	// get all PiFF from database
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, DatabaseAPI+"/db/retrieve/all", nil)
	request.Header.Set("Authorization", r.Header.Get("Authorization"))
	if err != nil {
		log.Printf("[ERROR] Get request: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't get request"))
		return
	}

	response, err := client.Do(request)
	if err != nil {
		log.Printf("[ERROR] Do request: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't do request"))
		return
	}

	// get body of returned data
	if response.Body == nil {
		log.Printf("[ERROR] Returned body is null")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't read returned data (body is null)"))
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("[ERROR] Read data: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't read data"))
		return
	}

	// checks whether there was an error during request
	if response.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Do request: %v", body)
		w.WriteHeader(response.StatusCode)
		w.Write(body)
		return
	}

	// transform json into struct
	var piFFData []Picture
	err = json.Unmarshal(body, &piFFData)
	if err != nil {
		log.Printf("[ERROR] Unmarshal data: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't unmarshal data"))
		return
	}

	// create zip file and zip writer
	outFile := new(bytes.Buffer)
	writer := zip.NewWriter(outFile)

	namesMap := make(map[string]int) // to check names which already exist

	// add files to zip
	for _, picture := range piFFData {
		// get image variables
		imagePath := ""

		imageName := strings.TrimSuffix(picture.Filename, filepath.Ext(picture.Filename)) // image name without extension, filepath.Ext returns the extension of a path (returns ".png" for "image.png")

		if picture.Unreadable { // was marked as unreadable
			imagePath = "Unreadable/"
		} else if picture.Annotator == "$taliesin_recognizer" { // was annotated by the recognizer but not corrected
			imagePath = "Uncorrected/"
		} else if picture.Annotator == "" { // wasn't annotated
			imagePath = "Unannotated/"
		}

		// change name if file already exist
		namesMap[imagePath+imageName] = namesMap[imagePath+imageName] + 1
		occurrence := namesMap[imagePath+imageName]
		if occurrence > 1 { // file already exist
			imageName = imageName + "_" + strconv.Itoa(occurrence)
		}

		// add file to zip

		file, err := writer.Create(imagePath + imageName + ".piff")
		if err != nil {
			log.Printf("[ERROR] Create piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't create piFF"))
			return
		}

		piFF, err := json.MarshalIndent(picture.PiFF, "", "    ")
		if err != nil {
			log.Printf("[ERROR] Marshal piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't marshal piFF"))
			return
		}

		_, err = file.Write(piFF)
		if err != nil {
			log.Printf("[ERROR] Write piFF: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't write piFF"))
			return
		}

		// add image to zip

		// open image in file server
		imageFile, err := os.Open(picture.Url)
		if err != nil {
			log.Printf("[ERROR] Open image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't open image"))
			return
		}

		// read image
		imageData, imageExt, err := image.Decode(imageFile)
		if err != nil {
			log.Printf("[ERROR] Decode image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't decode image"))
			return
		}

		// copy image data into image file according to its extension
		switch imageExt {
		case "jpeg":
			file, err = writer.Create(imagePath + imageName + ".jpg")
			if err != nil {
				log.Printf("[ERROR] Create image: %v", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("[MICRO-EXPORT] Couldn't create image"))
				return
			}

			jpeg.Encode(file, imageData, nil)
			break
		case "png":
			file, err = writer.Create(imagePath + imageName + ".png")
			if err != nil {
				log.Printf("[ERROR] Create image: %v", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("[MICRO-EXPORT] Couldn't create image"))
				return
			}

			png.Encode(file, imageData)
			break
		default:
			log.Printf("[ERROR] Switch image type: unhandled format (" + imageExt + ")")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't handle format " + imageExt))
			return
		}

		err = imageFile.Close()
		if err != nil {
			log.Printf("[ERROR] Close image: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("[MICRO-EXPORT] Couldn't close image"))
			return
		}
	}

	// close writer
	err = writer.Close()
	if err != nil {
		log.Printf("[ERROR] Close writer: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("[MICRO-EXPORT] Couldn't close writer"))
		return
	}

	// send data
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(outFile.Len()), 10))
	io.Copy(w, outFile)
}

// function to test whether docker file is correctly built
func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "[MICRO-EXPORT] Welcome home!")
}

func main() {
	dbEnvVal, dbEnvExists := os.LookupEnv("DATABASE_API_URL")

	if dbEnvExists {
		DatabaseAPI = dbEnvVal
	} else {
		DatabaseAPI = "http://database-api.gitlab-managed-apps.svc.cluster.local:8080"
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/export/piff", exportPiFF).Methods("GET")
	router.HandleFunc("/export", homeLink).Methods("GET")
	log.Fatal(http.ListenAndServe(":22022", router))
}
