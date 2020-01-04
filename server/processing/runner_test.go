package processing

import (
	"log"
	"testing"
)

func TestPrep(t *testing.T) {
	Prep("")
}

func TestOmg(t *testing.T) {
	err := Normalise("/Users/carl/Movies/run.mp4", "/Users/carl/Movies/nomral.mp4")
	log.Println(err)
}
