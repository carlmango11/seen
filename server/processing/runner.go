package processing

import (
	"bufio"
	"encoding/base64"
	"errors"
	"log"
	"os/exec"
)

const (
	ScriptDir  = "processing/scripts/"
	PrepScript = ScriptDir + "prep.py"
	BlurScript = ScriptDir + "blur.py"
)

func Normalise(inPath, outPath string) error {
	// TODO: atm it's assuming anything that completes is a success, output is v complicated
	_, err := run("ffmpeg", "-i", inPath, outPath, "-hide_banner", "-y")
	return err
}

func CreateFrames(inPath, framesDir string) error {
	return runReadFirstLine("./"+PrepScript, inPath, framesDir, "3")
}

func Blur(inPath, outPath, guideJson string) error {
	encJson := base64.StdEncoding.EncodeToString([]byte(guideJson))
	return runReadFirstLine("./"+BlurScript, inPath, outPath, encJson)
}

func runReadFirstLine(name string, args ...string) error {
	res, err := run(name, args...)
	log.Println("OUT", res)
	if err != nil {
		if len(res) == 0 {
			return err
		}

		// script can sometimes spit out an exception from Python
		return errors.New(res[0])
	}

	return nil
}

func run(name string, args ...string) ([]string, error) {
	// TODO: timeout
	cmd := exec.Command(name, args...)

	errStream, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	errScan := bufio.NewScanner(errStream)

	if err := cmd.Start(); err != nil {
		log.Printf("error starting cmd: %v", err)
		return nil, err
	}

	var waitErr error
	go func() {
		waitErr = cmd.Wait()
	}()

	var output []string
	for errScan.Scan() {
		output = append(output, errScan.Text())
	}

	return output, waitErr
}
