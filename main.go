package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Image struct {
	Id   int64
	Data []byte
}

type ImageData struct {
	SampleHz int
	Height   int
	Width    int
	Images   []*Image
}

func main() {
	http.HandleFunc("/image", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")

		resp := &ImageData{
			SampleHz: 1,
			Height:   1000,
			Width:    1400,
			Images:   getImageData(),
		}

		bs, _ := json.Marshal(resp)
		writer.Write(bs)
	})

	panic(http.ListenAndServe(":9999", nil))
}

func getImageData() []*Image {
	imgs := []*Image{}

	files, err := ioutil.ReadDir("/Users/carl/IdeaProjects/Seen/out")
	if err != nil {
		panic(err)
	}

	for _, thisFile := range files {
		name := thisFile.Name()
		fullPath := "/Users/carl/IdeaProjects/Seen/out/" + name

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
