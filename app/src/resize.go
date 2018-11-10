package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type ExecInfo struct {
	Elapsed  time.Duration
	MaxRSSKB int
}

type ImageInfo struct {
	ExecInfo ExecInfo
	Width    int
	Height   int
}

type ResizeInput struct {
	TargetHeightWidth int
	InFile            string
	OutFile           string
	Quality           int
}

type ResizeOutput struct {
	ExecInfo ExecInfo
	OutFile  string
}

type Identifier func(fname string) (ImageInfo, error)

type Resizer func(input ResizeInput) (ResizeOutput, error)

func parseElapsedKB(s string) int {
	pat := regexp.MustCompile("Maximum resident set[^:]+: (\\d+)")
	matches := pat.FindStringSubmatch(s)
	if len(matches) > 1 {
		v, _ := strconv.Atoi(matches[1])
		return v
	}
	return -1
}

func parseWidthHeight(out string) (int, int) {
	pat := regexp.MustCompile(" \\S+ (\\d+)x(\\d+)[ +]")
	matches := pat.FindStringSubmatch(out)
	if len(matches) > 2 {
		w, _ := strconv.Atoi(matches[1])
		h, _ := strconv.Atoi(matches[2])
		return w, h
	}
	return -1, -1
}

func ImageMagickResizer(input ResizeInput) (ResizeOutput, error) {
	size := fmt.Sprintf("%dx%d", input.TargetHeightWidth, input.TargetHeightWidth)
	cmd := exec.Command("/usr/bin/time", "-v", "convert", input.InFile, "-resize", size,
		"-quality", strconv.Itoa(input.Quality), input.OutFile)
	return resize(cmd, input.OutFile)
}

func GraphicsMagickResizer(input ResizeInput) (ResizeOutput, error) {
	size := fmt.Sprintf("%dx%d", input.TargetHeightWidth, input.TargetHeightWidth)
	cmd := exec.Command("/usr/bin/time", "-v", "gm", "convert", input.InFile, "-resize", size,
		"-quality", strconv.Itoa(input.Quality), input.OutFile)
	return resize(cmd, input.OutFile)
}

func VipsResizer(input ResizeInput) (ResizeOutput, error) {
	size := fmt.Sprintf("%dx%d", input.TargetHeightWidth, input.TargetHeightWidth)
	opts := fmt.Sprintf("[Q=%d,optimize_coding]", input.Quality)
	cmd := exec.Command("/usr/bin/time", "-v", "vipsthumbnail", input.InFile, "--size", size, "-o",
		input.OutFile+opts)
	return resize(cmd, input.OutFile)
}

func resize(cmd *exec.Cmd, outFile string) (ResizeOutput, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("resize: err running: %v - %v", cmd, err)
		fmt.Println(err)
		return ResizeOutput{}, err
	}
	//log.Printf("OUTPUT: %s", out.String())
	elapsed := time.Now().Sub(start)
	//log.Printf("%v %v", elapsed, cmd)
	execInfo := ExecInfo{
		Elapsed:  elapsed,
		MaxRSSKB: parseElapsedKB(stderr.String()),
	}
	return ResizeOutput{
		ExecInfo: execInfo,
		OutFile:  outFile,
	}, nil
}

func identifier(cmd *exec.Cmd) (ImageInfo, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	if err != nil {
		return ImageInfo{}, fmt.Errorf("identifier: err running: %v - %v", cmd, err)
	}
	//log.Printf("OUTPUT: %s", out.String())
	width, height := parseWidthHeight(out.String())
	execInfo := ExecInfo{
		Elapsed:  time.Now().Sub(start),
		MaxRSSKB: parseElapsedKB(stderr.String()),
	}
	return ImageInfo{
		ExecInfo: execInfo,
		Width:    width,
		Height:   height,
	}, nil
}

func ImageMagickIdentifier(fname string) (ImageInfo, error) {
	cmd := exec.Command("/usr/bin/time", "-v", "identify", fname)
	return identifier(cmd)
}

func GraphicsMagickIdentifier(fname string) (ImageInfo, error) {
	cmd := exec.Command("/usr/bin/time", "-v", "gm", "identify", fname)
	return identifier(cmd)
}

func runIdentify() {
	fname := "/home/james/src/talk-image-resize/imprev_images/large/7b68bf29d27ba6f884265466d2d46451.jpg"
	info, err := ImageMagickIdentifier(fname)
	if err == nil {
		log.Printf("im identify: %v", info)
	} else {
		log.Printf("ERROR: %v", err)
	}
	info, err = GraphicsMagickIdentifier(fname)
	if err == nil {
		log.Printf("gm identify: %v", info)
	} else {
		log.Printf("ERROR: %v", err)
	}
}

type ResizeJob struct {
	InFile            string
	ImageMagickOut    ResizeJobResult
	GraphicsMagickOut ResizeJobResult
	VipsOut           ResizeJobResult
}

type ResizeJobResult struct {
	Error  error
	Output ResizeOutput
}

func (r ResizeJobResult) String() string {
	if r.Error == nil {
		return fmt.Sprintf("%10d %10d", r.Output.ExecInfo.Elapsed/1e6, r.Output.ExecInfo.MaxRSSKB)
	}
	return "error"
}

type SvcResizeInput struct {
	Url    string `json:url`
	Height int    `json:height`
	Width  int    `json:width`

	// valid values: im, gm, vips
	Impl string `json:impl`
}

type SvcResizeOutput struct {
	Error         string `json:error`
	ElapsedMillis int    `json:elapsedMillis`
	MaxRSSKB      int    `json:maxRSSKB`
	Impl          string `json:impl`
	OrigUrl       string `json:origUrl`
	OrigHeight    int    `json:origHeight`
	OrigWidth     int    `json:origWidth`
	ResizeUrl     string `json:resizeUrl`
	ResizeExif    string `json:resizeExif`
}

func resizePost(c echo.Context) error {
	outdir := "output"
	cwd, _ := os.Getwd()
	quality := 80

	var input SvcResizeInput
	dec := json.NewDecoder(c.Request().Body)
	defer c.Request().Body.Close()
	err := dec.Decode(&input)
	if err != nil {
		return fmt.Errorf("resizePost: JSON decode failed: %v", err)
	}

	fileExt := ".jpg"
	pos := strings.LastIndex(input.Url, ".")
	if pos > -1 {
		fileExt = input.Url[pos:]
	}
	tmpFile := randSeq(16) + fileExt
	defer os.Remove(tmpFile)

	w, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("resizePost: temp file create %s failed: %v", tmpFile, err)
	}
	defer w.Close()

	resp, err := http.Get(input.Url)
	if err != nil {
		return fmt.Errorf("resizePost: http Get failed for url %s - %v", input.Url, err)
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("resizePost: temp file copy failed: %v", err)
	}
	w.Close()

	imgInfo, err := GraphicsMagickIdentifier(tmpFile)
	if err != nil {
		return fmt.Errorf("gm identify failed for file: %s - err: %v", tmpFile, err)
	}

	outFileBase := fmt.Sprintf("%s.jpg", randSeq(16))
	resInput := ResizeInput{
		TargetHeightWidth: input.Height,
		Quality:           quality,
		InFile:            tmpFile,
		OutFile:           path.Join(path.Join(cwd, outdir), outFileBase),
	}

	output := SvcResizeOutput{
		OrigUrl:    input.Url,
		OrigHeight: imgInfo.Height,
		OrigWidth:  imgInfo.Width,
		Impl:       input.Impl,
	}

	var resout ResizeOutput
	switch input.Impl {
	case "im":
		resout, err = ImageMagickResizer(resInput)
	case "gm":
		resout, err = GraphicsMagickResizer(resInput)
	case "vips":
		resout, err = VipsResizer(resInput)
	default:
		output.Error = fmt.Sprintf("Unknown impl: %s", input.Impl)
	}

	if err != nil {
		output.Error = fmt.Sprintf("%s resize failed: %v", input.Impl, resInput.OutFile, err)
	} else {
		output.ResizeUrl = fmt.Sprintf("http://127.0.0.1:1333/%s", outFileBase)
		output.ElapsedMillis = int(resout.ExecInfo.Elapsed / 1e6)
		output.MaxRSSKB = resout.ExecInfo.MaxRSSKB
	}

	return c.JSON(http.StatusOK, output)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.POST("/resize", resizePost)
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "output",
		Browse: true,
	}))
	e.Logger.Fatal(e.Start(":1333"))
}
