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
	//request, err := http.NewRequest(http.MethodGet, "http://database-api.gitlab-managed-apps.svc.cluster.local:8080/db/retrieve/all", nil)
	request, err := http.NewRequest(http.MethodGet, "http://localhost:22022/essai/archive", nil)
	if err != nil {
		errorHeader := "[MICRO-EXPORT] Get request: "
		fmt.Println(errorHeader + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorHeader + err.Error()))
		return
	}

	response, err := client.Do(request)
	if err != nil {
		errorHeader := "[MICRO-EXPORT] Do request: "
		fmt.Println(errorHeader + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorHeader + err.Error()))
		return
	}

	// get body of returned data
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		errorHeader := "[MICRO-EXPORT] Read data: "
		fmt.Println(errorHeader + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorHeader + err.Error()))
		return
	}

	// transform json into struct
	var piFFData PictureArray
	err = json.Unmarshal(body, &piFFData)
	if err != nil {
		errorHeader := "[MICRO-EXPORT] Unmarshal data: "
		fmt.Println(errorHeader + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorHeader + err.Error()))
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
			errorHeader := "[MICRO-EXPORT] Create piFF: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

		piFF, err := json.MarshalIndent(picture.PiFF, "", "    ")
		if err != nil {
			errorHeader := "[MICRO-EXPORT] Marshal piFF: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

		_, err = file.Write(piFF)
		if err != nil {
			errorHeader := "[MICRO-EXPORT] Write piFF: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

		// add image to zip
		image, err := ioutil.ReadFile(imageURL)
		if err != nil {
			errorHeader := "[MICRO-EXPORT] Read image: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

		file, err = writer.Create(imagePath + imageNameWithExt)
		if err != nil {
			errorHeader := "[MICRO-EXPORT] Create image: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

		_, err = file.Write(image)
		if err != nil {
			errorHeader := "[MICRO-EXPORT] Write image: "
			fmt.Println(errorHeader + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorHeader + err.Error()))
			return
		}

	}

	// close writer
	err = writer.Close()
	if err != nil {
		errorHeader := "[MICRO-EXPORT] Close writer: "
		fmt.Println(errorHeader + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorHeader + err.Error()))
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
