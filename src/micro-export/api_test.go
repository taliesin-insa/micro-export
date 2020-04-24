package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	lib_auth "github.com/taliesin-insa/lib-auth"
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

// for mockedAuthServer
type VerifyRequest struct {
	Token string
}

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

	// fake server to replace the authentication call (in lib_auth)
	mockedAuthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/auth/verifyToken" {
				reqBody, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Printf("[TEST_ERROR] Create authentication mocked server (read body): %v", err.Error())
					panic(m)
				}

				var reqData VerifyRequest
				err = json.Unmarshal(reqBody, &reqData)
				if err != nil {
					log.Printf("[TEST_ERROR] Create authentication mocked server (unmarsal body): %v", err.Error())
					panic(m)
				}

				var result []byte
				if reqData.Token == "admin_token" {
					result, err = json.Marshal(lib_auth.UserData{Username: "morpheus", Role: lib_auth.RoleAdmin})
				} else {
					result, err = json.Marshal(lib_auth.UserData{Username: "morpheus", Role: lib_auth.RoleAnnotator})
				}

				if err != nil {
					log.Printf("[TEST_ERROR] Create authentication mocked server (marshal result): %v", err.Error())
					panic(m)
				}

				w.WriteHeader(http.StatusOK)
				w.Write(result)
			}
		}))

	// replace the redirect to authentication microservice
	previousAuthUrl := os.Getenv("AUTH_API_URL")
	os.Setenv("AUTH_API_URL", mockedAuthServer.URL)

	request := &http.Request{
		Method: http.MethodGet,
		Header: map[string][]string{
			"Authorization": {"admin_token"},
		},
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

	// replace the real authentication url
	os.Setenv("AUTH_API_URL", previousAuthUrl)
	os.Exit(code)
}

func TestExportPiFFBadAuth(t *testing.T) {
	wrongAuthRequest := &http.Request{
		Method: http.MethodGet,
		Header: map[string][]string{
			"Authorization": {"annotator_token"},
		},
	}

	// make http request
	wrongAuthRecorder := httptest.NewRecorder()
	exportPiFF(wrongAuthRecorder, wrongAuthRequest)

	// status test
	if status := wrongAuthRecorder.Code; status != http.StatusUnauthorized {
		t.Errorf("[TEST_ERROR] Handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
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
		return
	}

	files, err := getZipFiles(body)
	if err != nil {
		t.Errorf("[TEST_ERROR] Handler returned a body which isn't a correct zip: %v", err.Error())
		return
	}

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
		if names[i] != files[i].Name {
			t.Errorf("[TEST_ERROR] Handler returned wrong file name: got %v want %v",
				files[i].Name, names[i])
			return
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
		if err != nil {
			t.Errorf("[TEST_ERROR] Handler returned an file %v which wasn't a piFF: %v", files[i].Name, err.Error())
			return
		}

		if piFFContent.Data[0].Value != "TEST MICRO EXPORT" {
			t.Errorf("[TEST_ERROR] Handler returned wrong value for file %v, got %v want %v",
				files[i].Name, piFFContent.Data[0].Value, EmptyPiFF.Data[0].Value)
			return
		}
	}

}

func TestExportPiFFImageContent(t *testing.T) {
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

		if !reflect.DeepEqual(imageContent, originalContent) {
			t.Errorf("[TEST_ERROR] Handler returned wrong image %v: %v", files[i].Name, err.Error())
			return
		}
	}

	// close image
	if originalImage.Close() != nil {
		t.Errorf("[TEST_ERROR] Close the original image during test: %v", err.Error())
		return
	}
}
