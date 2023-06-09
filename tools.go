package toolkit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOP0123456789-+"

// Tools is the type used to instantiate this moudule. Any
// Any variable of this type will have access to all the methods
// with ther reciver *Tools
type Tools struct {
	MaxFileSize      int64
	AllowedFileTypes []string
	MaxJSONSize      int
	//AllowUnknownFields is true if we're going to permit JSON
	// that includes unknown fields
	AllowUnknownFields bool
}

// UploadFiles is the type returned to the user
// holding uploaded file information like: filename, size etc.
type UploadFile struct {
	OriginalFileName string
	NewFileName      string
	FileSize         int64
}

func (t *Tools) RandonString(n int) string {

	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)

}

func (t *Tools) UploadOneFile(r *http.Request, dirName string, rename ...bool) (*UploadFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	uploadedFile, err := t.UploadFiles(r, dirName, renameFile)
	if err != nil {
		return nil, err
	}
	return uploadedFile[0], nil
}

func (t *Tools) UploadFiles(r *http.Request, dirName string, rename ...bool) ([]*UploadFile, error) {

	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	// check if the directory exists
	err := t.CreateDirIfNotExists(dirName)
	if err != nil {
		return nil, err
	}

	// check and validate the uploaded file size
	if err = r.ParseMultipartForm(int64(t.MaxFileSize)); err != nil {
		return nil, err
	}
	//
	//MultipartForm is the parsed multipart form, including file uploads.
	// This field is only available after ParseMultipartForm is called.
	// A FileHeader describes a file part of a multipart request.
	// type FileHeader struct {
	// 	Filename string
	// 	Header   textproto.MIMEHeader
	// 	Size     int64

	// 	content   []byte
	// 	tmpfile   string
	// 	tmpoff    int64
	// 	tmpshared bool
	// }
	for _, fHeaders := range r.MultipartForm.File { //map[string][]*FileHeader
		for _, hdr := range fHeaders {
			uploadedFiles, err := func([]*UploadFile) ([]*UploadFile, error) {
				var uploadedFile UploadFile
				// hdr = *multipart.FileHeader
				infile, err := hdr.Open() //multipart.File
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				// check the actual file tpye
				// read the 512 of the file to get its actual file type
				buff := make([]byte, 512)
				_, err = infile.Read(buff) //func (io.Reader).Read(p []byte) (n int, err error)
				if err != nil {
					// fail to read the file
					return nil, err
				}
				// validate the file type
				allowed := false
				// func http.DetectContentType(data []byte) string
				// DetectContentType implements the algorithm described
				// at https://mimesniff.spec.whatwg.org/ to determine
				// the Content-Type of the given data.
				// It considers at most the first 512 bytes of data.
				// DetectContentType always returns a valid MIME type:
				// if it cannot determine a more specific one,
				// it returns "application/octet-stream"
				fileType := http.DetectContentType(buff)
				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(x, fileType) { //case-insensitivity compare
							// if exists in the allowed list
							allowed = true
						}
					}
				} else {
					allowed = true
				}
				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				//
				//
				// reset the file pointer to the begining
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}
				// check whether rename the file or not??
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandonString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}
				uploadedFile.OriginalFileName = hdr.Filename

				// try to write to the disk
				var outFile *os.File
				defer outFile.Close()
				//
				//
				outFile, err = os.Create(filepath.Join(dirName, uploadedFile.NewFileName))
				if err != nil {
					return nil, err
				}
				//
				//
				fileSize, err := io.Copy(outFile, infile)
				if err != nil {
					return nil, err
				}
				uploadedFile.FileSize = fileSize
				//
				//
				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, nil
			}(uploadedFiles)

			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	log.Println("upload sucessful")
	return uploadedFiles, nil

}

// CreateDirIfNotExists creates a directory, and all necessary parents,
//
//	if it does not exist
func (t *Tools) CreateDirIfNotExists(path string) error {
	// when you create a directory, you have some kind
	// of mode.
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// Slugify is a very simple means of creating a slug from string
func (t *Tools) Slugify(s string) (string, error) {
	// So we receive some kind of string, we perform an operations on it,
	// convert it into something that's safe for a URL
	if s == "" {
		return "", errors.New("empty string not permitted")
	}

	//accept any a-z letter, also accept digits, + means of any length
	var re = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")

	// string that consists of no letters and no numbers
	// if it's some other kind of charaters, may be Japanese characters,
	// whatever the case may be.
	if len(slug) == 0 {
		return "", errors.New("after removing characters, slug is zero length")
	}

	return slug, nil

}

// DownloadStaticFile downloads a file, and tries to force the browser
// to avoid display it in the browser window by setting content disposition.
// It also allows specification of the display name
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, filename, displayedFileName string) {
	filePath := path.Join(pathName, filename)
	// Content-Disposition header, allows the file to be actually
	// downloaded to the user's file system and not to be displayed
	// in the browser.
	// So, this header tell the browser to download the file instead of
	// trying to display it in the browser.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayedFileName))
	http.ServeFile(w, r, filePath)
}

// func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, p, file, displayName string) {
// 	fp := path.Join(p, file)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

// 	http.ServeFile(w, r, fp)
// }

//we receive a JSON payload, we want to read it, send back an error
// message if something went wrong. Otherwise we'll just decode the
// JSON into some king of go data structure

// JSONResponse is the type used for sending JSON around
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON tries to read the body of a request and converts from json
// to go data variable
func (t *Tools) ReadJson(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// limit the maximum size that a given JSON payload can be just to
	// avoid someone sending a gigabyte of data to me just in an effort
	// to bring the server down or something.
	// default is 1MiB
	maxBytes := 1024 * 1024 // 1MiB
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}

	// read the body from the request
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	// check, should we allow people to send JSON to whatever site is
	// using the service that include fields we don't know about?
	if !t.AllowUnknownFields {
		// we're not going to process JSON that has fields we don't
		// know about
		dec.DisallowUnknownFields()
	}
	// buf := make([]byte, 512)
	// _, err := r.Body.Read(buf)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(buf[:]))
	// decode the data
	err := dec.Decode(data)
	if err != nil {
		// return err

		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError): // JSON is badly formed
			log.Println(err.Error())
			// syntaxError.Offset tell exactly where the character takes place
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			log.Println(err.Error())
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			log.Println(err.Error())
			if unmarshalTypeError.Field != "" {
				// so you tried to send me JSON, that was supposed to be an int,
				// but it's actually a string, or something like that
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		// what if we have a empty file?
		// there's no body included
		case errors.Is(err, io.EOF):
			log.Println(err.Error())
			// you try to send me JSON, but there's none there.
			return errors.New("body must not be empty")

			//this error will never occur if the user actually included
			// that disallow unknown fields when they instantiated the
			// variable of the tyoe toolkil.Tools and set that to true
			// otherwise this error is possible
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			log.Println(err.Error())
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// maybe the request body is too large
		case err.Error() == "http: request body too large":
			log.Println(err.Error())
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		// what if there's an unmarshal error of some sort?
		case errors.As(err, &invalidUnmarshalError):
			log.Println(err.Error())
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())
		default:
			return err
		}
	}

	// check the r.Body contains more than one JSON file
	// &struct{}{} will try to decode more JSON from that file
	err = dec.Decode(&struct{}{})
	// if i get an error that is io.EOF that means there's more than
	// one JSON value in this body
	if err != io.EOF {
		return errors.New("body must contain only one JSON value")
	}
	return nil
}

// WriteJson takes a response status code and arbitrary data  and writes
// json to the client
func (t *Tools) WriteJson(w http.ResponseWriter, responseStatus int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Set custom headers, if any
	if len(headers) > 0 {
		// deal with one additional header
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-type", "application/json")
	// write the response header
	w.WriteHeader(responseStatus)

	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

// ErrorJSON takes an error and optionally a status code and generates
// and sends a JSON error message
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	// default status code if not provided
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()
	return t.WriteJson(w, statusCode, payload)

}

// push JSON to some remote APT or URL and get a response back.
