package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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

// Returns a Buffer (representing the zip archive) containing all PiFF files and images
func getPiFFArchive() *bytes.Buffer {
	// create zip file and zip writer
	outFile := new(bytes.Buffer)
	w := zip.NewWriter(outFile)

	piFFData := getData()

	// add files to zip
	for _, picture := range piFFData.Pictures {
		// get image name
		imagePath := picture.Url

		// add image to zip
		image, err := ioutil.ReadFile(imagePath)
		checkError(err)
		file, err := w.Create(imagePath)
		checkError(err)
		_, err = file.Write(image)

		// add file to zip
		file, err = w.Create(getImageName(imagePath) + ".piff")
		checkError(err)
		piFF, err := json.MarshalIndent(picture, "", "    ")
		_, err = file.Write(piFF)

	}

	// close writer
	checkError(w.Close())

	return outFile
}

// Get all piFF from database
func getData() PictureArray {
	resp, err := http.Get("url ici")
	checkError(err)

	// get body of returned data
	body, err := ioutil.ReadAll(resp.Body)
	checkError(err)

	// transform json into struct
	var PiFFData PictureArray
	err = json.Unmarshal(body, &PiFFData)
	checkError(err)

	return PiFFData
}

// From "/example/of/path/image.png" to "image"
func getImageName(imagePath string) string {
	segments := strings.Split(imagePath, "/")
	nameWithExt := segments[len(segments)-1] // image name with extension
	segments = strings.Split(nameWithExt, ".")
	name := segments[len(segments)-2] // image name without extension
	return name
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
