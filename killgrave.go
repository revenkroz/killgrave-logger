package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

type KillgraveRequest struct {
	Method   string            `json:"method,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

func (r *KillgraveRequest) Hash() string {
	bytes, err := json.Marshal(r)
	if err != nil {
		return ""
	}

	return string(bytes)
}

type KillgraveResponse struct {
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type KillgraveImposter struct {
	Request  KillgraveRequest  `json:"request,omitempty"`
	Response KillgraveResponse `json:"response,omitempty"`
}

func createImposters(
	impostersDir string,
	logChan chan Log,
) {
	for {
		select {
		case l := <-logChan:
			dir, err := createDir(impostersDir, l.URL)
			if err != nil {
				continue
			}

			err = saveLog(l, dir)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

// createDir creates directory by host and url path of request and returns path
func createDir(
	baseDir string,
	url url.URL,
) (string, error) {
	host := strings.ReplaceAll(url.Host, ":", "_")

	dir := baseDir + "/" + host + url.Path

	// if dir exists, return
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return dir, nil
}

func createFile(
	filepath string,
) error {
	if _, err := os.Stat(filepath); err != nil {
		file, err := os.Create(filepath)
		if err != nil {
			fmt.Println(err)
			return err
		}

		_, err = file.Write([]byte("[]"))
		if err != nil {
			return err
		}

		return file.Close()
	}

	return nil
}

func saveLog(
	l Log,
	dir string,
) error {
	imposters := make([]KillgraveImposter, 0)

	filepath := dir + "/imposters.json"
	err := createFile(filepath)
	if err != nil {
		return err
	}

	// read file
	data, err := fileGetContents(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	err = json.Unmarshal([]byte(data), &imposters)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}

	req := KillgraveRequest{
		Method:   l.Request.Method,
		Endpoint: l.URL.Path,
	}
	if l.URL.RawQuery != "" {
		req.Params = map[string]string{}
		for k, v := range l.URL.Query() {
			req.Params[k] = strings.Join(v, ", ")
		}
	}

	// delete if already exists
	for imposter := range imposters {
		if imposters[imposter].Request.Hash() == req.Hash() {
			imposters = append(imposters[:imposter], imposters[imposter+1:]...)
		}
	}

	res := KillgraveResponse{
		Status:  l.Response.StatusCode,
		Headers: map[string]string{},
	}
	for k, v := range l.Response.Header {
		if k == "Content-Encoding" || k == "Date" {
			continue
		}

		res.Headers[k] = strings.Join(v, ", ")
	}
	body, err := io.ReadAll(l.Response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	res.Body = string(body)

	// append new imposter
	imposters = append(imposters, KillgraveImposter{
		Request:  req,
		Response: res,
	})

	// write to file
	impostersJson, err := json.Marshal(imposters)
	if err != nil {
		return fmt.Errorf("failed to marshal file: %w", err)
	}

	err = filePutContents(filepath, impostersJson)
	if err != nil {
		return fmt.Errorf("failed to encode file: %w", err)
	}

	return nil
}
