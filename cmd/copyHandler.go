// Copyright © 2017 Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"github.com/Azure/azure-storage-azcopy/common"

	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

// handles the copy command
// dispatches the job order (in parts) to the storage engine
func HandleCopyCommand(commandLineInput common.CopyCmdArgsAndFlags) {
	jobPartOrder := common.CopyJobPartOrderRequest{}
	copyHandlerUtil{}.applyFlags(&commandLineInput, &jobPartOrder)

	// generate job id
	jobPartOrder.JobID = common.NewJobID()
	jobStarted := true

	// depending on the source and destination type, we process the cp command differently
	if commandLineInput.SourceType == common.ELocation.Local() && commandLineInput.DestinationType == common.ELocation.Blob() {
		jobStarted = handleUploadFromLocalToBlobStorage(&commandLineInput, &jobPartOrder)
	} else if commandLineInput.SourceType == common.ELocation.Blob() && commandLineInput.DestinationType == common.ELocation.Local() {
		jobStarted = handleDownloadFromBlobStorageToLocal(&commandLineInput, &jobPartOrder)
	}

	// unexpected errors can happen while communicating with the transfer engine
	if !jobStarted {
		fmt.Println("Job with id", jobPartOrder.JobID, "was not abe to start. Please try again")
		return
	}

	// in background mode we would spit out the job id and quit
	// in foreground mode we would continuously print out status updates for the job, so the job id is not important
	fmt.Println("Job with id", jobPartOrder.JobID, "has started.")
	if commandLineInput.IsaBackgroundOp {
		return
	}

	// created a signal channel to receive the Interrupt and Kill signal send to OS
	cancelChannel := make(chan os.Signal, 1)
	// cancelChannel will be notified when os receives os.Interrupt and os.Kill signals
	signal.Notify(cancelChannel, os.Interrupt, os.Kill)

	// waiting for signals from either cancelChannel or timeOut Channel.
	// if no signal received, will fetch/display a job status update then sleep for a bit
	for {
		select {
		case <-cancelChannel:
			fmt.Println("Cancelling Job")
			HandleCancelCommand(jobPartOrder.JobID.String())
			os.Exit(1)
		default:
			jobStatus := copyHandlerUtil{}.fetchJobStatus(jobPartOrder.JobID)

			// happy ending to the front end
			if jobStatus == "JobCompleted" {
				os.Exit(0)
			}

			// wait a bit before fetching job status again, as fetching has costs associated with it on the backend
			time.Sleep(500 * time.Millisecond)
		}
	}
	return
}

func handleUploadFromLocalToBlobStorage(commandLineInput *common.CopyCmdArgsAndFlags,
	jobPartOrderToFill *common.CopyJobPartOrderRequest) bool {

	// set the source and destination type
	jobPartOrderToFill.SrcLocation = common.ELocation.Local()
	jobPartOrderToFill.DstLocation = common.ELocation.Blob()

	// attempt to parse the destination url
	destinationUrl, err := url.Parse(commandLineInput.Destination)
	if err != nil {
		// the destination should have already been validated, it would be surprising if it cannot be parsed at this point
		panic(err)
	}

	// list the source files and directories
	matches, err := filepath.Glob(commandLineInput.Source)
	if err != nil || len(matches) == 0 {
		fmt.Println("Cannot find source to upload.")
		return false
	}

	enumerator := newUploadTaskEnumerator(jobPartOrderToFill)
	err = enumerator.enumerate(matches, commandLineInput.Recursive, destinationUrl)

	if err != nil {
		fmt.Printf("Cannot start job due to error: %s.\n", err)
		return false
	} else {
		return true
	}
}

func handleDownloadFromBlobStorageToLocal(commandLineInput *common.CopyCmdArgsAndFlags,
	jobPartOrderToFill *common.CopyJobPartOrderRequest) bool {

	// set the source and destination type
	jobPartOrderToFill.SrcLocation = common.ELocation.Blob()
	jobPartOrderToFill.DstLocation = common.ELocation.Local()

	enumerator := newDownloadTaskEnumerator(jobPartOrderToFill)
	err := enumerator.enumerate(commandLineInput.Source, commandLineInput.Recursive, commandLineInput.Destination)

	if err != nil {
		fmt.Printf("Cannot start job due to error: %s.\n", err)
		return false
	} else {
		return true
	}
}