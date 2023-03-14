package main

import (
	"C"
	"fmt"
)
import (
	"encoding/base64"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"os"
)

type Person struct {
	UUID    string
	AUTHOR  string
	MESSAGE string
	LIKES   int
	IMG     string
}

type Image struct {
	Img string `json:"img"`
}

type ApiRes struct {
	U   string `json:"u"`
	A   string `json:"a"`
	M   string `json:"m"`
	L   int    `json:"l"`
	I   uint8  `json:"i"`
	DEL uint8  `json:"D"`
}

type PostBody struct {
	A string `json:"a"`
	M string `json:"m"`
	L int    `json:"l"`
	G int
}

//export helloWorld
func sync() {

	serve_url := "127.0.0.1"

	content, err := ioutil.ReadFile("last_sync_time.txt")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	url := fmt.Sprintf("http://%s:3000/api/messages/", serve_url) + string(content)

	gob.Register(PostBody{})
	resp, err := http.Get(url)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	apiResList := []ApiRes{}

	err = json.Unmarshal(body, &apiResList)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	var post_map map[string]PostBody
	var runner map[string]bool
	binary_file, err := os.Open("data.gob")
	if err == nil {
		defer binary_file.Close()

		decoder := gob.NewDecoder(binary_file)
		err = decoder.Decode(&post_map)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			// panic(err)
			return
		}

	} else {

		post_map = make(map[string]PostBody)
	}

	runner_binary, err := os.Open("runner.gob")
	if err == nil {
		defer runner_binary.Close()

		decoder := gob.NewDecoder(runner_binary)
		err = decoder.Decode(&runner)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			// panic(err)
			return
		}
	} else {
		runner = make(map[string]bool)

	}

	updated_list := make(map[string]bool)

	for _, post := range apiResList {
		if post.DEL == 1 {
			delete(post_map, post.U)
			delete(runner, post.U)
			continue
		}

		img := 0
		if val, ok := post_map[post.U]; ok && post.I == 0 {
			img = val.G
		}
		if post.I == 2 {
			//delete existing image
			updated_list[post.U] = false
			img = 0
		} else if post.I == 1 {
			// get new Image ==> get later
			updated_list[post.U] = true
			img = 1
		}

		post_map[post.U] = PostBody{
			post.A,
			post.M,
			post.L,
			img,
		}

		runner[post.U] = true
	}

	keys := make([]string, 0, len(runner))
	for key := range runner {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	client := http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: true,
			DisableKeepAlives:  false,
		},
	}

	//==================================csv===============================//
	file, err := os.Create("data.csv")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	//=================================read, write=========================//

	for _, key := range keys {

		obj := post_map[key]
		imageStr := ""
		if _, ok := updated_list[key]; !ok && obj.G == 1 {
			// not have in the update list and have image
			binaryData, err := ioutil.ReadFile(fmt.Sprintf("dump/%s.jpg", key))
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			imageStr = string(binaryData)
		}
		if updated_list[key] {
			// get image from api
			img_url := fmt.Sprintf("http://%s:3000/api/fetch/", serve_url) + key
			resp, err := client.Get(img_url)

			if err != nil || resp.StatusCode != 200 {
				fmt.Printf("Error: %s\n", err)
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return
			}

			base64Data := base64.StdEncoding.EncodeToString(body)
			imageStr = base64Data

			file, err := os.OpenFile(fmt.Sprintf("dump/%s.jpg", key), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				// handle error
				fmt.Println(err)
				return
			}

			_, err = file.Write([]byte(base64Data))
			if err != nil {
				// handle error
				log.Fatal(err)
			}

			file.Close()
			resp.Body.Close()

		}

		row := []string{key, obj.A, obj.M, strconv.Itoa(obj.L), imageStr}
		err = writer.Write(row)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	f, err := os.Create("data.gob")
	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(post_map)
	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}

	runfile, err := os.Create("runner.gob")
	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}
	defer runfile.Close()
	run_enc := gob.NewEncoder(runfile)
	err = run_enc.Encode(runner)
	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}

	last_api := fmt.Sprintf("http://%s:3000/api/last-sync", serve_url)
	last_resp, err := http.Get(last_api)

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	defer last_resp.Body.Close()
	last_body, err := ioutil.ReadAll(last_resp.Body)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	last_file, err := os.OpenFile("last_sync_time.txt", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		// handle error
		fmt.Println(err)
	}
	defer last_file.Close()
	_, err = last_file.Write(last_body)
	if err != nil {
		// handle error
		log.Fatal(err)
	}

}
