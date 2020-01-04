package serve

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/carlmango11/seen/server/processing"
	uuid "github.com/satori/go.uuid"
)

const (
	JobWorkers = 5
)

type Status string

const (
	StatusUnknown    Status = ""
	StatusIncoming   Status = "incoming"   // uploaded and waiting conversion to standardised MP4
	StatusNormalised Status = "normalised" // converted to a standard format understood by OpenCV
	StatusPrepped    Status = "prepped"    // sampled into individual frames for use by workbench
	StatusAnnotated  Status = "annotated"  // guide keyframes received and ready for blurring
	StatusComplete   Status = "complete"   // blurring complete
	StatusError      Status = "error"
)

type Job struct {
	Id     uuid.UUID
	Status Status
}

type Server struct {
	jobC       chan *Job
	storageDir string
	db         *DB
}

type Image struct {
	Id   int64
	Data []byte
}

type Response struct {
	Status Status
	Error  string
	Data   *ImageData
}

type ImageData struct {
	SampleHz int
	Height   int
	Width    int
	Images   []*Image
}

func New(storageDir string, db *sql.DB) *Server {
	return &Server{
		jobC:       make(chan *Job, 100000),
		storageDir: storageDir,
		db:         NewDb(db),
	}
}

func (s *Server) Start() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.HandleFunc("/image", s.handleGetImageData)
	http.HandleFunc("/uploadVideo", s.handleFileUpload)

	for i := 0; i < JobWorkers; i++ {
		go s.doJobs()
	}

	panic(http.ListenAndServe(":9999", nil))
}

func (s *Server) doJobs() {
	for job := range s.jobC {
		switch job.Status {
		case StatusIncoming:
			s.processIncoming(job)
		}
	}
}

func (s *Server) processIncoming(job *Job) {
	err := processing.Normalise(fmt.Sprintf("%s/%s", s.storageDir, job.Id.String()))
	if err != nil {
		s.db.errStatus(job.Id, err)
		return
	}

	s.db.setStatus(job.Id, StatusNormalised)

	// requeue for next stage
	job.Status = StatusNormalised
	//s.jobC <- job
}

func (s *Server) createFrames(id uuid.UUID) {
	log.Printf("creating frames for %v", id)

	processing.Prep(fmt.Sprintf("%s/%s", s.storageDir, id.String()))

	log.Println("converted frames successfully")
}

func (s *Server) handleGetImageData(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	refStr := request.FormValue("ref")
	if refStr == "" {
		log.Println("missing ref")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := uuid.FromString(refStr)
	if err != nil {
		log.Printf("invalid ref (%s): %v", refStr, err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	status, err := s.db.status(id)
	if err != nil {
		log.Printf("error getting status for %v: %v", id, err)
	}

	resp := &Response{
		Status: status,
	}

	if status == StatusPrepped {
		resp.Data = &ImageData{
			SampleHz: 1,
			Height:   1000,
			Width:    1400,
			Images:   getImageData(s.storageDir),
		}
	}

	bs, _ := json.Marshal(resp)
	writer.Write(bs)
}

func (s *Server) handleFileUpload(writer http.ResponseWriter, req *http.Request) {
	req.ParseMultipartForm((10 << 20) * 50)

	file, handler, err := req.FormFile("file")
	if err != nil {
		log.Println("error reading form", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Uploaded File: %s, %d", handler.Filename, handler.Size)

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("read err", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, _ := uuid.NewV4()

	ext := handler.Filename[strings.LastIndex(handler.Filename, "."):]

	out, err := os.Create(fmt.Sprintf("%sincoming/%s%s", s.storageDir, id.String(), ext))
	if err != nil {
		log.Printf("error opening output file; %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	// write this byte array to our temporary file
	out.Write(fileBytes)

	err = s.db.addReq(id)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.jobC <- &Job{
		Id:     id,
		Status: StatusIncoming,
	}

	writer.Write([]byte(id.String()))
}

func getImageData(storageDir string) []*Image {
	log.Println("get images")
	imgs := []*Image{}

	files, err := ioutil.ReadDir(storageDir + "frames")
	if err != nil {
		panic(err)
	}

	for _, thisFile := range files {
		name := thisFile.Name()
		fullPath := storageDir + "frames/" + name

		log.Println(fullPath)
		if !strings.Contains(fullPath, ".jpg") {
			continue
		}

		f, err := os.Open(fullPath)
		if err != nil {
			panic(err)
		}

		bs, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		imgId, err := strconv.ParseInt(name[:len(name)-4], 10, 64)
		if err != nil {
			panic(err)
		}

		imgs = append(imgs, &Image{
			Data: bs,
			Id:   imgId,
		})
	}

	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].Id < imgs[j].Id
	})

	return imgs
}
