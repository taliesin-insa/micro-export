package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
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
	imageName = "original_image_name"

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
		Filename:   imageName + ".png", // add fake extension
		Annotator:  "someone",
	}

	unreadablePicture := Picture{
		PiFF:       EmptyPiFF,
		Url:        imagePath,
		Unreadable: true,
		Filename:   imageName + ".png", // add fake extension
	}

	uncorrectedPicture := Picture{
		PiFF:       EmptyPiFF,
		Url:        imagePath,
		Unreadable: false,
		Filename:   imageName + ".png", // add fake extension
		Annotator:  "$taliesin_recognizer",
	}

	unannotatedPicture := Picture{
		PiFF:       EmptyPiFF,
		Url:        imagePath,
		Unreadable: false,
		Filename:   imageName + ".png", // add fake extension
		Annotator:  "",                 // no annotation
	}

	// fake server to replace the database call
	mockedDatabaseServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/db/retrieve/all" {

				piFFArray := []Picture{readablePicture, readablePicture, unreadablePicture, uncorrectedPicture, unannotatedPicture}
				piFFJSON, err := json.Marshal(piFFArray)
				if err != nil {
					log.Printf("[TEST_ERROR] Create database mocked server: %v", err.Error())
					panic(m)
				}

				w.WriteHeader(http.StatusOK)
				w.Write(piFFJSON)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	// replace the redirect to database microservice
	DatabaseAPI = mockedDatabaseServer.URL

	request := &http.Request{
		Method: http.MethodGet,
	}

	// make http request
	recorder = httptest.NewRecorder()
	exportPiFF(recorder, request)

	code := m.Run()

	// delete image because tests are finished
	if os.Remove(imagePath) != nil {
		log.Printf("[TEST_ERROR] Delete the original image: %v", err.Error())
		panic(m)
	}

	os.Exit(code)
}

func TestExportPiFFStatus(t *testing.T) {
	// status test
	assert.Equal(t, http.StatusOK, recorder.Code)
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
	assert := assert.New(t)

	// content tests
	body := recorder.Body
	assert.NotNil(body)

	files, err := getZipFiles(body)
	assert.Nil(err, "Handler returned a body which isn't a correct zip")

	// test file names
	names := []string{
		imageName + ".piff",
		imageName + ".png",
		imageName + "_2.piff",
		imageName + "_2.png",
		"Unreadable/" + imageName + ".piff",
		"Unreadable/" + imageName + ".png",
		"Uncorrected/" + imageName + ".piff",
		"Uncorrected/" + imageName + ".png",
		"Unannotated/" + imageName + ".piff",
		"Unannotated/" + imageName + ".png"}

	for i := 0; i < len(files); i++ {
		assert.Equal(files[i].Name, names[i])
	}
}

func TestExportPiFFFileContent(t *testing.T) {
	assert := assert.New(t)

	files, _ := getZipFiles(recorder.Body) // test over error already done in previous test

	// test file content
	for _, i := range []int{0, 2} { // test only for piFF files
		file, err := files[i].Open()
		if err != nil {
			t.Errorf("[TEST_ERROR] Open file %v: .%v", files[i].Name, err.Error())
			return
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("[TEST_ERROR] Read file %v: %v", files[i].Name, err.Error())
			return
		}

		if file.Close() != nil {
			t.Errorf("[TEST_ERROR] Close %v: %v", files[i].Name, err.Error())
			return
		}

		var piFFContent PiFFStruct
		err = json.Unmarshal(content, &piFFContent)
		assert.Nil(err, "Handler returned a file "+files[i].Name+" which wasn't a piFF")

		assert.Equal(EmptyPiFF.Data[0].Value, piFFContent.Data[0].Value, "Handler returned wrong value for file "+files[i].Name)
	}

}

func TestExportPiFFImageContent(t *testing.T) {
	assert := assert.New(t)

	files, _ := getZipFiles(recorder.Body) // test over error already done in previous test

	// get info on the original image
	originalImage, err := os.Open(imagePath)
	if err != nil {
		t.Errorf("[TEST_ERROR] Open the original image: %v", err.Error())
		return
	}

	originalContent, err := png.Decode(originalImage)
	if err != nil {
		t.Errorf("[TEST_ERROR] Decode the original image: %v", err.Error())
		return
	}

	// test image content
	for _, i := range []int{1, 3} { // test only for images
		image, err := files[i].Open()
		if err != nil {
			t.Errorf("[TEST_ERROR] Open image %v: %v", files[i].Name, err.Error())
			return
		}

		imageContent, err := png.Decode(image)
		if err != nil {
			t.Errorf("[TEST_ERROR] Read image %v: %v", files[i].Name, err.Error())
			return
		}

		assert.True(reflect.DeepEqual(imageContent, originalContent), "Handler returned wrong image "+files[i].Name)
	}

	// close image
	if originalImage.Close() != nil {
		t.Errorf("[TEST_ERROR] Close the original image during test: %v", err.Error())
		return
	}
}
