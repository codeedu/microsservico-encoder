package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type VideoUpload struct {
	Paths        []string
	VideoPath    string
	OutputBucket string
	Errors       []string
}

func NewVideoUpload() *VideoUpload {
	return &VideoUpload{}
}

func (vu *VideoUpload) UploadObject(objectPath string, s3Client *s3.S3) error {

	// caminho/x/b/arquivo.mp4
	// split: caminho/x/b/
	// [0] caminho/x/b/
	// [1] arquivo.mp4
	path := strings.Split(objectPath, os.Getenv("localStoragePath")+"/")

	f, err := os.Open(objectPath)
	if err != nil {
		return err
	}

	defer f.Close()

	object := s3.PutObjectInput{
		Bucket: aws.String(vu.OutputBucket),
		Key:    aws.String(path[1]),
		Body:   f,
		ACL:    aws.String("private"),
		Metadata: map[string]*string{
			"x-amz-meta-my-key": aws.String("test"),
		},
	}

	_, err = s3Client.PutObject(&object)

	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("File " + path[1] + " sended")
	return nil
}

func (vu *VideoUpload) loadPaths() error {

	err := filepath.Walk(vu.VideoPath, func(path string, info os.FileInfo, err error) error {

		if !info.IsDir() {
			fmt.Println(path)
			vu.Paths = append(vu.Paths, path)
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (vu *VideoUpload) ProcessUpload(concurrency int, doneUpload chan string) error {

	in := make(chan int, runtime.NumCPU()) // qual o arquivo baseado na posicao do slice Paths
	returnChannel := make(chan string)

	err := vu.loadPaths()
	if err != nil {
		return err
	}

	s3Client, err := getClientUpload()
	if err != nil {
		return err
	}

	for process := 0; process < concurrency; process++ {
		fmt.Println("Worker: ", process)
		go vu.uploadWorker(in, returnChannel, s3Client)
	}

	go func() {
		for x := 0; x < len(vu.Paths); x++ {
			in <- x
		}
		// close(in) // COMENTADO
	}()

	count := 0 // NOVO
	for r := range returnChannel {
		count++ // NOVO

		if r != "" {
			doneUpload <- r
			break
		}

		if count == len(vu.Paths) { // NOVO
			close(in) // NOVO
		} // NOVO
	}

	return nil
}

func (vu *VideoUpload) uploadWorker(in chan int, returnChan chan string, s3Client *s3.S3) {

	for x := range in {
		err := vu.UploadObject(vu.Paths[x], s3Client)

		if err != nil {
			vu.Errors = append(vu.Errors, vu.Paths[x])
			log.Printf("error during the upload: %v. Error: %v", vu.Paths[x], err)
			returnChan <- err.Error()
		}

		returnChan <- ""
	}

	returnChan <- "upload completed"
}

func getClientUpload() (*s3.S3, error) {
	key := os.Getenv("SPACES_KEY")
	secret := os.Getenv("SPACES_SECRET")

	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:    aws.String("https://nyc3.digitaloceanspaces.com"),
		Region:      aws.String("us-east-1"),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return nil, err
	}
	s3Client := s3.New(newSession)

	return s3Client, nil
}
