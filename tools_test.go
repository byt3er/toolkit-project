package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTool_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandonString(10)

	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed image type", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// setup a pipe to avoid buffering
		pr, pw := io.Pipe()
		// create a actual Multipart writer
		// this gives me something to simulate a multipart file upload
		writer := multipart.NewWriter(pw)

		// I'm going to be doing this by executing a go routine in the
		// background more than once. And for I need a waitGroup
		// I need to make sure things occur in a particular sequence
		wg := sync.WaitGroup{}
		wg.Add(1)
		// now I'll fire off a go routine in the background

		go func() {
			// go func(), simple inline function that's being sent off
			// to run concurrenly with the current program
			defer writer.Close()
			// decrement the wairGroup as soon as this function finish
			// executing
			defer wg.Done()

			// create the form data field "file"
			//
			// now, I need to get some data into that.
			// I have to have some kind of image to try uploading and it
			// has to be a jpeg or png because in my test, those are
			// the only types I'm allowing
			// Create the form data field and it needs to be populated
			// with some data.
			// so this create a part for the multipart file upload
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			// I don't want a resource lead so defer
			defer f.Close()

			// decode the image
			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image ", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}() /// This run in the background and when it finished
		// I've to simulate creating a multipart request with the <img.png>
		// file in it

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		// so this creates a request that will use the reader we want
		// and we'll add to that request a header
		// writer.FormDataContentType() sets the correct content type
		// for whatever the payload is.
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			// if no expecting error and got an error
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				// This means the file didn't got uploaded
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			// Clean the file from the ./testdata/uploads/
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		// Expecting error but no error found
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but not received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// setup a pipe to avoid buffering
	pr, pw := io.Pipe()
	// create a actual Multipart writer
	// this gives me something to simulate a multipart file upload
	writer := multipart.NewWriter(pw)

	// we don't need a waitGroup here because we're running this once

	go func() {
		// go func(), simple inline function that's being sent off
		// to run concurrenly with the current program
		defer writer.Close()

		// create the form data field "file"
		//
		// now, I need to get some data into that.
		// I have to have some kind of image to try uploading and it
		// has to be a jpeg or png because in my test, those are
		// the only types I'm allowing
		// Create the form data field and it needs to be populated
		// with some data.
		// so this create a part for the multipart file upload
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		// I don't want a resource lead so defer
		defer f.Close()

		// decode the image
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image ", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}() /// This run in the background and when it finished
	// I've to simulate creating a multipart request with the <img.png>
	// file in it

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	// so this creates a request that will use the reader we want
	// and we'll add to that request a header
	// writer.FormDataContentType() sets the correct content type
	// for whatever the payload is.
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		// This means the file didn't got uploaded
		t.Errorf(" expected file to exist: %s", err.Error())
	}

	// Clean the file from the ./testdata/uploads/
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))

}

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExists("./testdata/mydir")
	if err != nil {
		t.Error(err)
	}

	// try to create the same directory that already exists
	err = testTool.CreateDirIfNotExists("./testdata/mydir")
	if err != nil {
		t.Error(err)
	}

	// clean up
	_ = os.Remove("./testdata/mydir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for all GOOD man ! + fish & such &^123", expected: "now-is-the-time-for-all-good-man-fish-such-123", errorExpected: false},
	{name: "Hindi string", s: "हैलो वर्ल्ड", expected: "", errorExpected: true},
	{name: "Hindi string and roman characters", s: "hello world हैलो वर्ल्ड", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", e.name, e.expected, slug)
		}

	}
}
func TestTools_DownloadStaticFile(t *testing.T) {
	// http.NewRecorder(response recorder) === http.ResponseWriter
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata", "img.png", "screenshort.png")

	//result
	res := rr.Result()
	defer res.Body.Close() // avlid resource leak

	// test that the entire file is downloaded.
	// That can be check using the content-length
	// inside the header.
	if res.Header["Content-Length"][0] != "157004" {
		t.Error("wrong content lenght of ", res.Header["Content-Length"])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"screenshort.png\"" {
		t.Error("wrong content disposition")
	}

	// check for an error when try to read the response body.
	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}
