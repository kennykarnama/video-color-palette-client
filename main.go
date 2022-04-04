package main

import (
	"github.com/alexflint/go-arg"
	"github.com/dghubble/sling"
	"github.com/gocarina/gocsv"

	"log"
	"os"
	"io"
	"fmt"
	"context"
	"path/filepath"

	"gitlab.com/ruangguru/kennykarnama/video-color-palette-client/client/colorpalette"
	sharedS3Lib "gitlab.com/ruangguru/kennykarnama/video-color-palette-client/shared/s3"
)

var args struct {
	InputFile    string `arg:"required,--input-file,-i" help:"input file in csv format"`
	BaseURL      string `arg:"required,--base-url" help:"base URL for colorPalette Service"`
	InputBucket  string `arg:"required,--input-bucket" help:"input bucket"`
	OutputBucket string `arg:"required,--output-bucket" help:"output bucket for results"`
	OutputPrefix string `arg:"--output-prefix" help:"output prefix" default:"video-color-palette-extraction"`
	ResultCmd    *ResultCmd `arg:"subcommand:save-result" help:"save result"`
}

type ResultCmd struct {
	SuccessFile string `arg:"--success-file" help:"success file in csv"`
	SkippedFile string `arg:"--skipped-file" help:"skip file in csv"`
	ErrorFile string `arg:"--error-file" help:"error file in csv"`
}

type VideoVersion struct {
	Serial           string `csv:"serial"`
	OriginalFilePath string `csv:"original_file_path"`
}

func (vv *VideoVersion) GetS3URL() string {
	// Format URL S3:
	// https://[bucket].s3.ap-southeast-1.amazonaws.com/[key]	
	return fmt.Sprintf("https://%s.s3.ap-southeast-1.amazonaws.com/%s", args.InputBucket, sharedS3Lib.KeyEscape(vv.OriginalFilePath))
}

func GetDestinationURI(vv *VideoVersion, csvOutfile string) string {
	// Format URL S3:
	// https://[bucket].s3.ap-southeast-1.amazonaws.com/[key]
	prefix := args.OutputPrefix
	resultPath := filepath.Join(prefix, vv.OriginalFilePath)
	return fmt.Sprintf("https://%s.s3.ap-southeast-1.amazonaws.com/%s", args.OutputBucket, resultPath)
}


func main() {
	arg.MustParse(&args)
	slingLib := sling.New()
	colorPaletteClient := colorpalette.NewHttpClient(slingLib, args.BaseURL)
	f, err := os.Open(args.InputFile)
	if err != nil {
		log.Fatalf("openFile err=%v", err)
	}
	defer f.Close()
	var videoVersions []*VideoVersion
	err = parseCsv(f, &videoVersions)
	if err != nil {
		log.Fatalf("err=%v", err)
	}
	for _, vv := range videoVersions  {
		log.Printf("Processing vv.serial=%v vv.original_file_path=%v", vv.Serial, vv.OriginalFilePath)
		paletteReq := colorpalette.ColorPaletteGenerationRequest{
			SourceURL: vv.GetS3URL(),
			SourceSerial: vv.Serial,
			PeriodSeconds: 60,
			PaletteSize: 5,
			FunctionType: 1,
			DestinationURI: "",
		}
		err = colorPaletteClient.GenerateColorPalette(context.Background(), paletteReq)
		if err != nil {
			log.Fatalf("err=%v", err)
		}
	}
}

func parseCsv(reader io.Reader, out interface{}) error {
	err := gocsv.Unmarshal(reader, out)
	if err != nil {
		return fmt.Errorf("parseCsv err=%v", err)
	}
	return nil
}
