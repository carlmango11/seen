package serve

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

const (
	DirIncoming   = "incoming/"
	DirNormalised = "normalised/"
	DirFrames     = "frames/"
	DirComplete   = "complete/"
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

type AnnotateReq struct {
	Id         uuid.UUID
	GuidesJson string
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

	http.HandleFunc("/workbenchData", s.handleGetImageData)
	http.HandleFunc("/autoblur", s.autoBlur)
	http.HandleFunc("/uploadVideo", s.handleFileUpload)
	http.HandleFunc("/status", s.handleStatusCheck)
	http.HandleFunc("/annotate", s.handleAnnotation)
	http.HandleFunc("/download", s.handleDownload)

	for i := 0; i < JobWorkers; i++ {
		go s.doJobs()
	}

	panic(http.ListenAndServe(":9999", nil))
}

func (s *Server) autoBlur(writer http.ResponseWriter, req *http.Request) {
	idStr := req.FormValue("id")

	id, err := uuid.FromString(idStr)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		log.Printf("%s is not a valid uuid: %v", idStr, err)
		return
	}

	log.Printf("AutoBlurring %v", id)

	inputPath := s.storageDir + DirNormalised + idStr + ".mp4"
	outputPath := s.storageDir + DirComplete + idStr + ".mp4"

	err = processing.AutoBlur(inputPath, outputPath)
	if err != nil {
		s.db.errStatus(id, err)
		return
	}

	writer.Write([]byte("Ok"))
}

func (s *Server) doJobs() {
	for job := range s.jobC {
		log.Printf("processing %v %v", job.Id, job.Status)

		switch job.Status {
		case StatusIncoming:
			s.doNormalise(job)
		case StatusNormalised:
			s.createFrames(job)
		case StatusAnnotated:
			s.doBlurring(job)
		}
	}
}

func (s *Server) doNormalise(job *Job) {
	inPath := s.storageDir + DirIncoming + job.Id.String() + ".mp4" // TODO: won't always be mp4
	outPath := s.storageDir + DirNormalised + job.Id.String() + ".mp4"

	err := processing.Normalise(inPath, outPath)
	if err != nil {
		s.db.errStatus(job.Id, err)
		return
	}

	s.db.setStatus(job.Id, StatusNormalised)

	// requeue for next stage
	job.Status = StatusNormalised
	s.jobC <- job
}

func (s *Server) createFrames(job *Job) {
	log.Printf("creating frames for %v", job.Id)

	inputPath := s.storageDir + DirNormalised + job.Id.String() + ".mp4"
	outputPath := s.storageDir + DirFrames + job.Id.String() + "/"

	err := processing.CreateFrames(inputPath, outputPath)
	if err != nil {
		s.db.errStatus(job.Id, err)
		return
	}

	s.db.setStatus(job.Id, StatusPrepped)

	log.Println("converted frames successfully")
}

func (s *Server) handleGetImageData(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	idStr := request.FormValue("id")
	if idStr == "" {
		log.Println("missing id")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		log.Printf("invalid ref (%s): %v", idStr, err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	status, err := s.db.getStatus(id)
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
			Images:   getImageData(s.storageDir, id),
		}
	}

	bs, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error marshalling: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write(bs)
}

func (s *Server) handleStatusCheck(writer http.ResponseWriter, req *http.Request) {
	idStr := req.FormValue("id")

	id, err := uuid.FromString(idStr)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		log.Printf("%s is not a valid uuid: %v", idStr, err)
		return
	}

	status, err := s.db.getStatus(id)
	if err != nil {
		log.Printf("error getting status for %v: %v", id, err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write([]byte(status))
}

func (s *Server) handleDownload(writer http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")

	if _, err := uuid.FromString(id); err != nil {
		// not a valid ID. Could be dodge
		log.Printf("received invalid complete id: %s %s", id, req.RemoteAddr)
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	if !s.db.matchesIp(id, req.RemoteAddr) {
		log.Printf("received unmatching ips: %v %v", id, req.RemoteAddr)
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	filePath := s.storageDir + DirComplete + id + ".mp4"

	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("error opening complete file: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	writer.Header().Set("Content-Type", "video/mp4")
	_, err = io.Copy(writer, f)
	if err != nil {
		log.Printf("error writing complete file to client %v: %v", id, err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleAnnotation(writer http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("error reading annotation body: %v", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	var annotateReq *AnnotateReq
	err = json.Unmarshal(body, &annotateReq)
	if err != nil {
		log.Printf("error unmarshalling annotation body: %v: %v", err, string(body))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.db.writeGuideJson(annotateReq.Id, annotateReq.GuidesJson)
	if err != nil {
		log.Printf("error writing json: %v %v: %v", err, annotateReq.Id, string(body))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	// queue up the blur job
	job := &Job{
		Id:     annotateReq.Id,
		Status: StatusAnnotated,
	}
	s.jobC <- job

	writer.WriteHeader(http.StatusOK)
}

func (s *Server) doBlurring(job *Job) {
	log.Printf("performing blurring for %v", job.Id)

	guideJson, err := s.db.getGuides(job.Id)
	if err != nil {
		s.db.errStatus(job.Id, err)
		return
	}

	inputPath := s.storageDir + DirNormalised + job.Id.String() + ".mp4"
	outputPath := s.storageDir + DirComplete + job.Id.String() + ".mp4"

	err = processing.Blur(inputPath, outputPath, guideJson)
	if err != nil {
		s.db.errStatus(job.Id, err)
		return
	}

	s.db.setComplete(job.Id)
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

	err = s.db.addReq(id, req.RemoteAddr)
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

func getImageData(storageDir string, id uuid.UUID) []*Image {
	log.Println("get images", id)
	imgs := []*Image{}

	dir := storageDir + DirFrames + id.String() + "/"

	files, err := ioutil.ReadDir(dir)
	log.Printf("reading images from: %s", dir)
	if err != nil {
		panic(err)
	}

	for _, thisFile := range files {
		name := thisFile.Name()
		fullPath := dir + name

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
