package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

var EmptyPiFF = PiFFStruct{
	Meta: Meta{
		Type: "line",
		URL:  "",
	},
	Location: []Location{
		{Type: "line",
			Polygon: [][2]int{
				{0, 0},
				{0, 0},
				{0, 0},
				{0, 0},
			},
			Id: "loc_0",
		},
	},
	Data: []Data{
		{
			Type:       "line",
			LocationId: "loc_0",
			Value:      "TEST MICRO EXPORT",
			Id:         "0",
		},
	},
	Children: nil,
	Parent:   0,
}

var imagePath string
var imageName string

var recorder *httptest.ResponseRecorder

func TestMain(m *testing.M) { // executed before all tests
	// create an image for the test
	width := 200
	height := 100

	upLeft := image.Point{0, 0}
	lowRight := image.Point{X: width, Y: height}

	image := image.NewRGBA(image.Rectangle{Min: upLeft, Max: lowRight})

	// set color for each pixel
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			image.Set(x, y, color.White)
		}
	}

	// save temporary the images
	imageFile, err := ioutil.TempFile("", "MICRO_EXPORT_TEST_*.png") // name must have an extension for the export, see doc for the '*' explanation
	if err != nil {
		log.Printf("[TEST_ERROR] Create the original image: %v", err.Error())
		panic(m)
	}
	imagePath = imageFile.Name()
	segments := strings.Split(imagePath, "/")
	imageName = segments[len(segments)-1] // image name with extension
	segments = strings.Split(imageName, ".")
	imageName = segments[len(segments)-2] // image name without extension

	err = png.Encode(imageFile, image)
	if err != nil {
		imageFile.Close()
		log.Printf("[TEST_ERROR] Encode the original image: %v", err.Error())
		panic(m)
	}

	if imageFile.Close() != nil {
		log.Printf("[TEST_ERROR] Close the original image: %v", err.Error())
		panic(m)
	}

	// create fake database data (pictures)
	readablePicture := Picture{
		PiFF:       EmptyPiFF,
		Url:        imagePath,
		Unreadable: false,
	}

	unreadablePicture := Picture{
		PiFF:       EmptyPiFF,
		Url:        imagePath,
		Unreadable: true,
	}

	// fake server to replace the database call
	mockedServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/db/retrieve/all" {

				piFFArray := []Picture{readablePicture, unreadablePicture}
				piFFJSON, err := json.Marshal(piFFArray)
				if err != nil {
					log.Printf("[TEST_ERROR] Create mocked server: %v", err.Error())
					panic(m)
				}

				w.WriteHeader(http.StatusOK)
				w.Write(piFFJSON)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	// replace the redirect to database microservice
	DatabaseAPI = mockedServer.URL

	request := &http.Request{
		Method: http.MethodGet,
	}

	// make http request
	recorder = httptest.NewRecorder()
	exportPiFF(recorder, request)

	os.Exit(m.Run())
}

func TestExportPiFFStatus(t *testing.T) {
	// status test
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("[TEST_ERROR] Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

// returns the list of files in the zip
func getZipFiles(body *bytes.Buffer) ([]*zip.File, error) {
	// read body as a zip
	bodyBytes := body.Bytes()
	reader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))

	if err != nil {
		return nil, err
	}

	return reader.File, nil
}

func TestExportPiFFFormat(t *testing.T) {
	// content tests
	body := recorder.Body
	if body == nil {
		t.Errorf("[TEST_ERROR] Handler returned nil body")
	}

	files, err := getZipFiles(body)
	if err != nil {
		t.Errorf("[TEST_ERROR] Handler returned a body which isn't a correct zip: %v", err.Error())
	}

	// test file names
	names := []string{imageName + ".piff", imageName + ".png", "Unreadable/" + imageName + ".piff", "Unreadable/" + imageName + ".png"}

	for i := 0; i < len(files); i++ {
		if names[i] != files[i].Name {
			t.Errorf("[TEST_ERROR] Handler returned wrong file name: got %v want %v",
				files[i].Name, names[i])
		}
	}
}

func TestExportPiFFFileContent(t *testing.T) {
	files, _ := getZipFiles(recorder.Body) // test over error already done in previous test

	// test file content
	for _, i := range []int{0, 2} { // test only for piFF files
		file, err := files[i].Open()
		if err != nil {
			t.Errorf("[TEST_ERROR] Open file %v: .%v", files[i].Name, err.Error())
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("[TEST_ERROR] Read file %v: %v", files[i].Name, err.Error())
		}

		if file.Close() != nil {
			t.Errorf("[TEST_ERROR] Close %v: %v", files[i].Name, err.Error())
		}

		var piFFContent PiFFStruct
		err = json.Unmarshal(content, &piFFContent)
		if err != nil {
			t.Errorf("[TEST_ERROR] Handler returned an file %v which wasn't a piFF: %v", files[i].Name, err.Error())
		}

		if piFFContent.Data[0].Value != "TEST MICRO EXPORT" {
			t.Errorf("[TEST_ERROR] Handler returned wrong value for file %v, got %v want %v",
				files[i].Name, piFFContent.Data[0].Value, EmptyPiFF.Data[0].Value)
		}
	}

}

func TestExportPiFFImageContent(t *testing.T) {
	files, _ := getZipFiles(recorder.Body) // test over error already done in previous test

	// get info on the original image
	originalImage, err := os.Open(imagePath)
	if err != nil {
		t.Errorf("[TEST_ERROR] Open the original image: %v", err.Error())
	}

	originalContent, err := png.Decode(originalImage)
	if err != nil {
		t.Errorf("[TEST_ERROR] Decode the original image: %v", err.Error())
	}

	// test image content
	for _, i := range []int{1, 3} { // test only for images
		image, err := files[i].Open()
		if err != nil {
			t.Errorf("[TEST_ERROR] Open image %v: %v", files[i].Name, err.Error())
		}

		imageContent, err := png.Decode(image)
		if err != nil {
			t.Errorf("[TEST_ERROR] Read image %v: %v", files[i].Name, err.Error())
		}

		if !reflect.DeepEqual(imageContent, originalContent) {
			t.Errorf("[TEST_ERROR] Handler returned wrong image %v: %v", files[i].Name, err.Error())
		}
	}

	// close and delete image because tests are finished
	if originalImage.Close() != nil {
		t.Errorf("[TEST_ERROR] Close the original image during test: %v", err.Error())
	}

	if os.Remove(imagePath) != nil {
		t.Errorf("[TEST_ERROR] Delete the original image: %v", err.Error())
	}
}
