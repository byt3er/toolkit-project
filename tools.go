package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOP0123456789-+"

// Tools is the type used to instantiate this moudule. Any
// Any variable of this type will have access to all the methods
// with ther reciver *Tools
type Tools struct {
	MaxFileSize      int64
	AllowedFileTypes []string
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

	// check and validate the uploaded file size
	if err := r.ParseMultipartForm(int64(t.MaxFileSize)); err != nil {
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
