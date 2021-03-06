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
	"github.com/gammazero/workerpool"
)

var args struct {
	InputFile    string     `arg:"required,--input-file,-i" help:"input file in csv format"`
	BaseURL      string     `arg:"required,--base-url" help:"base URL for colorPalette Service"`
	ApiKey string `arg:"required,--api-key" help:"apiKey"`
	InputBucket  string     `arg:"required,--input-bucket" help:"input bucket"`
	OutputBucket string     `arg:"required,--output-bucket" help:"output bucket for results"`
	OutputPrefix string     `arg:"--output-prefix" help:"output prefix" default:"video-color-palette-extraction"`
	ResultCmd    *ResultCmd `arg:"subcommand:save-result" help:"save result"`
	SkipIfExist  bool       `arg:"--skip-if-exist" help:"skip if exist"`
	NumWorker    int         `arg:"--num-worker,-n" help:"num of worker" default:"20"`
	DryRun       bool       `arg:"--dry-run" help:"dry run"`
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

func GetDestinationURI(vv VideoVersion, csvOutfile string) string {
	// Format URL S3:
	// https://[bucket].s3.ap-southeast-1.amazonaws.com/[key]
	prefix := args.OutputPrefix
	resultFile := vv.Serial + ".csv"
	resultPath := filepath.Join(prefix, resultFile)
	return fmt.Sprintf("https://%s.s3.ap-southeast-1.amazonaws.com/%s", args.OutputBucket, resultPath)
}

type SuccessResult struct {
	colorpalette.ColorPaletteGenerationRequest
}

type SkippedResult struct {
	VideoVersion
	Reason string `csv:"reason"`
}

type ErrorResult struct {
	colorpalette.ColorPaletteGenerationRequest
	ErrorMessage string `csv:"error_message"`
}


func main() {
	arg.MustParse(&args)
	slingLib := sling.New()
	colorPaletteClient := colorpalette.NewHttpClient(slingLib, args.BaseURL, args.ApiKey)
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
	var errorResults []*ErrorResult
	var skippedResults []*SkippedResult
	var successResults []*SuccessResult


	// open file

	successFile, err := os.OpenFile(args.ResultCmd.SuccessFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("action=run.openSuccessCsv err=%v", err)
	}
	defer successFile.Close()

	skippedFile, err := os.OpenFile(args.ResultCmd.SkippedFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("action=run.openSkippedCsv err=%v", err)
	}
	defer skippedFile.Close()

	errorFile, err := os.OpenFile(args.ResultCmd.ErrorFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("action=run.openErrorCsv err=%v", err)
	}
	defer errorFile.Close()

	if args.ResultCmd != nil {
		writeReportToFile(args.ResultCmd.SuccessFile, args.ResultCmd.SkippedFile, args.ResultCmd.ErrorFile, successResults, skippedResults, errorResults)
	}

	wp := workerpool.New(args.NumWorker)

	for _, videoVersion := range videoVersions  {

		vv := *videoVersion

		wp.Submit(func(){
		paletteReq := colorpalette.ColorPaletteGenerationRequest{
			SourceURL: vv.GetS3URL(),
			SourceSerial: vv.Serial,
			PeriodSeconds: 60,
			PaletteSize: 5,
			FunctionType: 1,
			DestinationURI: GetDestinationURI(vv, fmt.Sprintf("%v.csv", vv.Serial)),
		}
			log.Printf("Processing vv.serial=%v vv.original_file_path=%v", vv.Serial, vv.OriginalFilePath)
			
			if args.SkipIfExist {
			// check if csv is already exist in s3
				destinationObj, err := sharedS3Lib.ParseURL(paletteReq.DestinationURI)
				if err != nil {
					log.Printf("error: %v", err)
					errorResult := &ErrorResult{
						ColorPaletteGenerationRequest: paletteReq,
						ErrorMessage: err.Error(),
					}
					err = gocsv.MarshalWithoutHeaders([]*ErrorResult{errorResult}, errorFile)
					if err != nil {
						log.Fatalf("write.errorFileCsv err=%v", err)
					}
					return
				}
				csvExist, err := sharedS3Lib.CheckKeyExist(context.Background(), destinationObj.Bucket, destinationObj.Key)
				if err != nil {
					log.Printf("error: %v", err)
					errorResult := &ErrorResult{
						ColorPaletteGenerationRequest: paletteReq,
						ErrorMessage: err.Error(),
					}
					err = gocsv.MarshalWithoutHeaders([]*ErrorResult{errorResult}, errorFile)
					if err != nil {
						log.Fatalf("write.errorFileCsv err=%v", err)
					}
					return
				}
				if csvExist {
					log.Printf("Skipped cause csvFile already exist exist")
					skippedResult := &SkippedResult{
						VideoVersion: vv,
						Reason: "result already exist in destinationURI",
					}
					err = gocsv.MarshalWithoutHeaders([]*SkippedResult{skippedResult}, skippedFile)
					if err != nil {
						log.Fatalf("write.skippedFileFileCsv err=%v", err)
					}
					return
				}
			}
			exist, err := sharedS3Lib.CheckKeyExist(context.Background(), args.InputBucket, vv.OriginalFilePath)
			if err != nil {
				log.Printf("error: %v", err)
				errorResult := &ErrorResult{
					ColorPaletteGenerationRequest: paletteReq,
					ErrorMessage: err.Error(),
				}
				err = gocsv.MarshalWithoutHeaders([]*ErrorResult{errorResult}, errorFile)
				if err != nil {
					log.Fatalf("write.errorFileCsv err=%v", err)
				}
				return
			}
			if !exist {
				log.Printf("Skipped cause file not exist")
				skippedResult := &SkippedResult{
					VideoVersion: vv,
					Reason: "InputVideo doesn't exist in sourceURL",
				}
				err = gocsv.MarshalWithoutHeaders([]*SkippedResult{skippedResult}, skippedFile)
				if err != nil {
					log.Fatalf("write.skippedFileFileCsv err=%v", err)
				}
				return
			}
			if !args.DryRun {
				err = colorPaletteClient.GenerateColorPalette(context.Background(), paletteReq)
				if err != nil {
					log.Printf("error: %v", err)
					errorResult := &ErrorResult{
						ColorPaletteGenerationRequest: paletteReq,
						ErrorMessage: err.Error(),
					}
					err = gocsv.MarshalWithoutHeaders([]*ErrorResult{errorResult}, errorFile)
					if err != nil {
						log.Fatalf("write.errorFileCsv err=%v", err)
					}
					return
				}
			}

			successResult := &SuccessResult{
				paletteReq,
			}
			err = gocsv.MarshalWithoutHeaders([]*SuccessResult{successResult}, successFile)
			if err != nil {
				log.Fatalf("write.successFileCsv err=%v", err)
			}

		})
	}
	wp.StopWait()
}



func parseCsv(reader io.Reader, out interface{}) error {
	err := gocsv.Unmarshal(reader, out)
	if err != nil {
		return fmt.Errorf("parseCsv err=%v", err)
	}
	return nil
}

func writeReportToFile(success string, skippedPath string, errpath string, successData []*SuccessResult, skippedData []*SkippedResult, errData []*ErrorResult) error {
	successFile, err := os.OpenFile(success, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer successFile.Close()

	errFile, err := os.OpenFile(errpath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer errFile.Close()


	skippedFile, err := os.OpenFile(skippedPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer skippedFile.Close()

	err = gocsv.MarshalFile(successData, successFile)
	if err != nil {
		return fmt.Errorf("action=writeToFile path=%v err=%v", success, err)
	}

	err = gocsv.MarshalFile(errData, errFile)
	if err != nil {
		return fmt.Errorf("action=writeToFile path=%v err=%v", errpath, err)
	}


	err = gocsv.MarshalFile(skippedData, skippedFile)
	if err != nil {
		return fmt.Errorf("action=writeToFile path=%v err=%v", skippedPath, err)
	}

	return nil
}
