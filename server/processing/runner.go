package processing

import (
	"bufio"
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

func Prep(inPath, framesDir string) error {
	res, err := run("./"+PrepScript, inPath, framesDir, "3")
	if err != nil {
		if len(res) == 0 {
			return err
		}

		// script can sometimes spit out an exception from Python
		return errors.New(res[0])
	}

	// TODO: bad idea to expose exec output to internet?
	return errors.New("")
}

func run(name string, args ...string) ([]string, error) {
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
