package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func downloadZIP(flavor string) {

	// URL of the zip file to download
	// Check if it's Windows or Linux
	url := ""
	if flavor == "windows" {
		url = "https://github.com/blurbdust/rainbowcrackalack/releases/download/untagged/rainbowcrackalack-win-x64.zip"
	} else if flavor == "linux" {
		url = "https://github.com/blurbdust/rainbowcrackalack/releases/download/untagged/rainbowcrackalack-linux-x64.zip"
	} else {
		fmt.Println("Unsupported OS!")
	}

	// Destination directory to extract the contents to
	destDir := "crackalack"

	// Create the destination directory if it doesn't already exist
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		os.MkdirAll(destDir, 0755)
	} else {
		println("Folder already exists, skipping download")
		return
	}

	// Create a temporary file to save the downloaded zip file to
	tmpFile, err := os.CreateTemp("", "rainbow*.zip")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name()) // Remove the temporary file when we're done with it

	// Download the zip file and save it to the temporary file
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		panic(err)
	}

	// Open the zip file
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		panic(err)
	}
	defer zipReader.Close()

	// Extract the contents of the zip file to the destination directory
	for _, file := range zipReader.File {
		filePath := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}
		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			panic(err)
		}
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			panic(err)
		}
		defer outFile.Close()
		inFile, err := file.Open()
		if err != nil {
			panic(err)
		}
		defer inFile.Close()
		_, err = io.Copy(outFile, inFile)
		if err != nil {
			panic(err)
		}
	}

	// Done!
	println("Archive downloaded and extracted successfully!")
}

func helpCheck(dir string, prog string, filePath string, cdir string, computeUnitsInt int) int {

	// Check if the file exists
	if _, err := os.Stat(filePath); err == nil {
		// Delete the file
		err = os.Remove(filePath)
		if err != nil {
			panic(err)
		}
		fmt.Printf("File '%s' deleted\n", filePath)
	} else if os.IsNotExist(err) {
		fmt.Printf("File '%s' does not exist\n", filePath)
	} else {
		panic(err)
	}

	// double check execute bit is set on Linux
	if runtime.GOOS == "linux" {
		err := os.Chmod(prog, 0775)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start the thing.exe process
	cmd := exec.Command(prog, "netntlmv1", "byte", "7", "7", "0", "2", "1", "0")

	// Create a pipe to capture the process output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	// Start the process
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	// Done!
	fmt.Printf("Started process in directory %s\n", cdir)

	// Read the output from the process
	outputBytes, err := io.ReadAll(stdout)
	if err != nil {
		panic(err)
	}

	// Convert the output to a string and print it
	output := strings.TrimSpace(string(outputBytes))

	temp := strings.Split(output, "\n")
	for _, line := range temp {
		if strings.Contains(line, "Max compute units:") {
			computeUnits := strings.Replace(line, "Max compute units: ", "", 1)
			computeUnits = strings.TrimSpace(computeUnits)
			computeUnitsInt, err = strconv.Atoi(computeUnits)
			if err != nil {
				panic(err)
			}
		}
	}

	// Wait for the process to finish
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
	return computeUnitsInt
}

func runCheck(dir string, prog string) int {
	computeUnitsInt := -1

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Create the path to the subdirectory
	cdir := filepath.Join(cwd, dir)

	// Change into the subdirectory
	err = os.Chdir(cdir)
	if err != nil {
		panic(err)
	}

	filePath := "netntlmv1_byte#7-7_0_2x1_0.rt"

	computeUnitsInt = helpCheck(dir, prog, filePath, cdir, computeUnitsInt)

	// Read the contents of the file into a byte slice
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	// Define the desired byte sequence
	desiredBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xcd, 0x72, 0xdf, 0xc6, 0xe6, 0xd0, 0x40, 0x00}

	// Compare the byte slices
	if bytesEqual(data, desiredBytes) {
		fmt.Println("The byte sequences match!")
		return computeUnitsInt
	} else {
		fmt.Println("The byte sequences do not match. Checking possibility of patch needed")
		// TODO: If Linux, patch to old.netntlmv1.cl and rt.cl

		oPath := filepath.Join("CL", "old.netntlmv1.cl")
		nPath := filepath.Join("CL", "netntlmv1.cl")
		bPath := filepath.Join("CL", "bad.netntlmv1.cl")
		rPath := filepath.Join("CL", "rt.cl")

		err := os.Rename(nPath, bPath)
		if err != nil {
			log.Fatal(err)
		}
		err = os.Rename(oPath, nPath)
		if err != nil {
			log.Fatal(err)
		}

		patchURL := "https://gist.githubusercontent.com/blurbdust/77bddb721489fa4359b7af17f68321a0/raw/ffd0b23ed0972dd609d816ee9f3e6a064a59504a/rt.patch"

		// Download the patch file and save it to the temporary file
		resp, err := http.Get(patchURL)
		if err != nil {
			panic(err)
		}

		// files is a slice of *gitdiff.File describing the files changed in the patch
		// preamble is a string of the content of the patch before the first file
		files, _, err := gitdiff.Parse(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		code, err := os.Open(rPath)
		if err != nil {
			log.Fatal(err)
		}

		// apply the changes in the patch to a source file
		var output bytes.Buffer
		if err := gitdiff.Apply(&output, code, files[0]); err != nil {
			log.Fatal(err)
		}
		code.Close()

		err = os.WriteFile(rPath, output.Bytes(), 0644)
		if err != nil {
			panic(err)
		}

		computeUnitsInt = helpCheck(dir, prog, filePath, cdir, computeUnitsInt)

		// Read the contents of the file into a byte slice
		data, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		// Compare the byte slices
		if bytesEqual(data, desiredBytes) {
			fmt.Println("The byte sequences match after patch!")
			return computeUnitsInt
		} else {
			panic(nil)
		}
	}

	return -1
}

func checkOutNum() int {
	url := "http://genrt.blurbdust.pw:65080/"

	// Send GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return -1
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return -1
	}
	bodyStr := string(body)
	if strings.Contains(bodyStr, "Error!") {
		println("Got error but trying to see if we already checked out")
		// Create new request to see if we already checked out this number
		req, err := http.NewRequest("OPTIONS", url, nil)
		if err != nil {
			fmt.Println("Error:", err)
			return -1
		}
		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error:", err)
			return -1
		}
		defer resp.Body.Close()
		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error:", err)
			return -1
		}
		println("Already checked out ", strings.TrimSpace(string(body)))
		bodyNum, _ := strconv.Atoi(strings.TrimSpace(string(body)))
		if bodyNum == -1 {
			return -1
		} else {
			return bodyNum
		}
	}
	// Print response body as string
	fmt.Println("Got assigned number ", bodyStr)
	ret, err := strconv.Atoi(strings.TrimSpace(bodyStr))
	println(err)
	return ret
}

func doWork(dir string, prog string, index int, gws int) {
	println("Starting on ", index)

	// Start the thing.exe process
	cmd := exec.Command(prog, "netntlmv1", "byte", "7", "7", "0", "300000", "134217727", strconv.Itoa(index), "-gws", strconv.Itoa(gws))

	// Create a pipe to capture the process output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	// Start the process
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	// Read the output from the process
	_, err = io.ReadAll(stdout)
	if err != nil {
		panic(err)
	}

	// Wait for the process to finish
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}

	file, err := os.Open(fmt.Sprintf("netntlmv1_byte#7-7_0_300000x134217727_%d.rt", index))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	url := "http://genrt.blurbdust.pw:65080/" + strconv.Itoa(index)
	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

}

func main() {

	flavor := runtime.GOOS
	dir := ""
	prog := ""
	num := -1

	// Check if it's Windows or Linux
	if flavor == "windows" {
		dir = "crackalack\\rainbowcrackalack-win-x64"
		prog = ".\\crackalack_gen.exe"
	} else if flavor == "linux" {
		dir = "crackalack/rainbowcrackalack-linux-x64"
		prog = "./crackalack_gen"
	} else {
		fmt.Println("Unsupported OS!")
	}
	downloadZIP(flavor)
	computeUnits := runCheck(dir, prog)
	if computeUnits == -1 {
		panic(nil)
	}
	println("Guessing work size of ", computeUnits*512)
	num = checkOutNum()
	if num != -1 {
		doWork(dir, prog, num, computeUnits*512)
	} else {
		panic(nil)
	}

}
