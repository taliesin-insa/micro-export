package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/ioutil"
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
func getPiFFArchive() (*bytes.Buffer, error) {
	// create zip file and zip writer
	outFile := new(bytes.Buffer)
	w := zip.NewWriter(outFile)

	piFFData, err := getData()
	if err != nil {
		return nil, err
	}

	// add files to zip
	for _, picture := range piFFData.Pictures {
		// get image name
		imagePath := picture.Url

		// add image to zip
		image, err := ioutil.ReadFile(imagePath)
		if err != nil {
			return nil, err
		}

		file, err := w.Create(imagePath)
		if err != nil {
			return nil, err
		}

		_, err = file.Write(image)
		if err != nil {
			return nil, err
		}

		// add file to zip
		file, err = w.Create(getImageName(imagePath) + ".piff")
		if err != nil {
			return nil, err
		}

		piFF, err := json.MarshalIndent(picture.PiFF, "", "    ")
		_, err = file.Write(piFF)
		if err != nil {
			return nil, err
		}

	}

	// close writer
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return outFile, nil
}

// Get all piFF from database
func getData() (PictureArray, error) {
	client := &http.Client{}

	request, err := http.NewRequest(http.MethodPut, "http://database-api.gitlab-managed-apps.svc.cluster.local:8080/db/retrieve/all", nil)
	if err != nil {
		return PictureArray{}, err
	}

	response, err := client.Do(request)
	if err != nil {
		return PictureArray{}, err
	}

	// get body of returned data
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return PictureArray{}, err
	}

	// transform json into struct
	var PiFFData PictureArray
	err = json.Unmarshal(body, &PiFFData)
	if err != nil {
		return PictureArray{}, err
	}

	return PiFFData, nil
}

// From "/example/of/path/image.png" to "image"
func getImageName(imagePath string) string {
	segments := strings.Split(imagePath, "/")
	nameWithExt := segments[len(segments)-1] // image name with extension
	segments = strings.Split(nameWithExt, ".")
	name := segments[len(segments)-2] // image name without extension
	return name
}
