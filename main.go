package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func (d Download) Do() error {
	fmt.Println("Making connection")
	r, err := d.getNewRequest("HEAD")

	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if response.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't progress, response is %v", response.StatusCode))
	}

	size, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	fmt.Printf("Size is %v bytes\n", size)

	var sections = make([][2]int, d.TotalSections)

	eachSize := size / d.TotalSections

	fmt.Printf("Each size is %v bytes\n", eachSize)
	// Example: if file size is 100 bytes, our section should look like:
	// [[0 10] [11 21] ... [99 99]]

	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSize
		} else {
			sections[i][1] = size - 1
		}
	}

	fmt.Println(sections)

	var wg sync.WaitGroup

	for i, section := range sections {
		wg.Add(1)

		i := i
		section := section

		go func() {
			defer wg.Done()

			err = d.downloadSection(i, section)
			if err != nil {
				panic(err)
			}
		}()
	}

	wg.Wait()

	err = d.mergeFiles(sections)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	request, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "Silly Download Manager v001")
	return request, nil
}

func (d Download) downloadSection(i int, section [2]int) error {
	request, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}

	request.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", section[0], section[1]))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	fmt.Printf("Download %v bytes for section %v: %v\n", response.Header.Get("Content-Length"), i, section)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), body, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := range sections {
		body, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp", i))
		if err != nil {
			return err
		}

		n, err := f.Write(body)
		if err != nil {
			return err
		}

		fmt.Printf("%v bytes merged \n", n)
	}

	return nil
}

func main() {
	startTime := time.Now()
	d := Download{
		Url:           "https://www.dropbox.com/s/lgvhj57pw1gxzon/Check%20and%20Measure%20Any%20Website%27s%20Response%20Time.mp4?dl=1",
		TargetPath:    "final.mp4",
		TotalSections: 10,
	}

	err := d.Do()
	if err != nil {
		log.Fatalf("An error occurred while download the file: %s \n", err)
	}

	fmt.Printf("Download completed in %v seconds \n", time.Now().Sub(startTime).Seconds())
}
