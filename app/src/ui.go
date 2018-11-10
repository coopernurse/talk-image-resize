package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	errorPage := fmt.Sprintf("%d.html", code)
	if err := c.File(errorPage); err != nil {
		c.Logger().Error(err)
	}
	c.Logger().Error(err)
}

func imageFolders() []string {
	folders := make([]string, 0)
	parts := strings.Split(os.Getenv("IMG_FOLDERS"), ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			folders = append(folders, p)
		}
	}
	return folders
}

//////////////////////////////////////////////////////////////////

type ResizeInput struct {
	Url    string `json:url`
	Height int    `json:height`
	Width  int    `json:width`

	// valid values: im, gm, vips
	Impl string `json:impl`
}

type ResizeOutput struct {
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

type GroupedResizeOutput struct {
	OrigUrl    string
	OrigHeight int
	OrigWidth  int
	Outputs    []ResizeOutput
}

func toResizeInputs(p url.Values) ([]ResizeInput, error) {
	height, err := strconv.Atoi(p.Get("height"))
	if err != nil || height < 1 {
		return nil, fmt.Errorf("Invalid height value")
	}
	width, err := strconv.Atoi(p.Get("width"))
	if err != nil || width < 1 {
		return nil, fmt.Errorf("Invalid width value")
	}

	urls := strings.TrimSpace(p.Get("urls"))
	if urls == "" {
		return nil, fmt.Errorf("Provide at least one URL")
	}

	allImpls := []string{"im", "gm", "vips"}
	inputs := make([]ResizeInput, 0)
	for _, impl := range allImpls {
		for _, url := range strings.Split(p.Get("urls"), "\n") {
			url = strings.TrimSpace(url)
			if url != "" {
				inputs = append(inputs, ResizeInput{
					Url:    strings.TrimSpace(url),
					Height: height,
					Width:  width,
					Impl:   impl,
				})
			}
		}
	}

	return inputs, nil
}

func resizeForm(c echo.Context) error {
	formParams, _ := c.FormParams()
	params := map[string]interface{}{
		"form":  formParams,
		"error": c.Get("error"),
	}
	return c.Render(http.StatusOK, "resize_form", params)
}

func invokeHttp(url string, input ResizeInput) (ResizeOutput, error) {
	var output ResizeOutput
	data, err := json.Marshal(input)
	if err != nil {
		return output, fmt.Errorf("invokeHttp: error marshaling input: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return output, fmt.Errorf("invokeHttp: error posting to url %s - %v", url, err)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&output)
	if err != nil {
		return output, fmt.Errorf("invokeHttp: error decoding response JSON - %v", err)
	}
	return output, nil
}

func invokeResize(formParams url.Values, inputs []ResizeInput) ([]ResizeOutput, error) {
	url := os.Getenv("RESIZE_URL")
	if url == "" {
		url = "http://127.0.0.1:1333/resize"
	}
	outputs := make([]ResizeOutput, 0)
	for _, input := range inputs {
		output, err := invokeHttp(url, input)
		if err != nil {
			return nil, fmt.Errorf("invokeResize: error invoking %s - %v", url, err)
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func groupOutputs(outputs []ResizeOutput) []GroupedResizeOutput {
	byOrigUrl := make(map[string][]ResizeOutput)
	for _, out := range outputs {
		origUrl := strings.Replace(out.OrigUrl, "http://static:", "http://127.0.0.1:", 1)
		arr := byOrigUrl[origUrl]
		if arr == nil {
			arr = make([]ResizeOutput, 0)
		}
		arr = append(arr, out)
		byOrigUrl[origUrl] = arr
	}

	grouped := make([]GroupedResizeOutput, len(byOrigUrl))
	i := 0
	for k, v := range byOrigUrl {
		grouped[i] = GroupedResizeOutput{
			OrigUrl:    k,
			OrigHeight: v[0].OrigHeight,
			OrigWidth:  v[0].OrigWidth,
			Outputs:    v,
		}
		i++
	}
	return grouped
}

func resizePost(c echo.Context) error {
	formParams, _ := c.FormParams()
	inputs, err := toResizeInputs(formParams)
	if err == nil {
		outputs, err := invokeResize(formParams, inputs)
		if err == nil {
			params := map[string]interface{}{
				"outputs": groupOutputs(outputs),
			}
			return c.Render(http.StatusOK, "resize_out", params)
		}
	}
	c.Set("error", err)
	return resizeForm(c)
}

//////////////////////////////////////////////////////////////////

func main() {
	t := &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}
	e := echo.New()

	e.GET("/", resizeForm)
	e.POST("/resize", resizePost)

	e.Renderer = t
	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Logger.Fatal(e.Start(":1323"))
}
