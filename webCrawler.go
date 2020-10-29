package main

/*

	The idea is that there will be a max limit of go-routines that can be created.
	First of all the program will create a channel for which the URLs found in each
	page will be sent back to the main thread
	The program will then loop through each of the seed URLs and create a respective go-routine.


*/

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func downloadImage(imageURL string, exitimgch chan bool) {
	defer func() {
		exitimgch <- true
	}()

	resp, err := http.Get(imageURL)
	if err != nil {
		return
	}

	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	imageName := strings.Replace(imageURL, "/", "", -1)
	fileName := "Images/" + imageName
	err = ioutil.WriteFile(fileName, imageData, 0644)
	if err != nil {
		return
	}

	//fmt.Println("Succesfully saved image to a file:", imageURL)
}

func downloadFile(fileName string, data []byte) {
	err := ioutil.WriteFile(fileName, data, 0644)
	if err != nil {
		return
	}
}

func extractURLs(n *html.Node, resp *http.Response, urlch chan string, imgch chan string) {
	if n.Type == html.ElementNode {
		//fmt.Println(n.Data)
		if n.Data == "a" {
			for _, attribute := range n.Attr {
				if attribute.Key == "href" {
					urlch <- attribute.Val
				}
			}
		} else if n.Data == "img" {
			for _, attribute := range n.Attr {
				if attribute.Key == "src" {
					link, err := resp.Request.URL.Parse(attribute.Val)
					if err == nil {
						imgch <- link.String()
						fmt.Println(link.String())
					}
				}
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractURLs(c, resp, urlch, imgch)
	}
}

func crawl(urlName string, urlch chan string, imgch chan string, exiturlch chan bool) {
	defer func() {
		exiturlch <- true
	}()

	resp, err := http.Get(urlName)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return
	}

	extractURLs(doc, resp, urlch, imgch)
	//fmt.Println("Successfully visited site:", urlName)

	//hostName := strings.Replace(urlName, "/", "", -1)
	//fileName := "Sites/" + hostName + ".html"
	//downloadFile(fileName, data)
}

func main() {
	currentURLGoRoutines, currentImgGoRoutines := 0, 0
	const maxURLGoRoutines, maxImgGoRoutines int = 5000, 100
	urlch, exiturlch := make(chan string), make(chan bool)
	imgch, exitimgch := make(chan string), make(chan bool)

	urlMap := make(map[string]bool)
	hostMap := make(map[string]bool)
	links := os.Args[1:]
	images := make([]string, 0)

	for i := range links { // prepend http protocol
		if links[i][:8] != "https://" && links[i][:7] != "http://" {
			links[i] = "https://" + links[i]
		}
		urlMap[links[i]] = false
	}

	for len(links) > 0 || currentURLGoRoutines > 0 || currentImgGoRoutines > 0 {
		for currentURLGoRoutines < maxURLGoRoutines && len(links) > 0 {
			go crawl(links[0], urlch, imgch, exiturlch)
			if len(links) > 0 {
				links = links[1:]
			}
			currentURLGoRoutines++
		}

		for currentImgGoRoutines < maxImgGoRoutines && len(images) > 0 {
			go downloadImage(images[0], exitimgch)
			if len(images) > 0 {
				images = images[1:]
			}
			currentImgGoRoutines++

		}

		select {
		case urlName := <-urlch:
			{
				if !urlMap[urlName] {
					u, err := url.Parse(urlName)
					if err != nil {
						continue
					}

					links = append(links, urlName)
					urlMap[urlName] = true

					if hostName := u.Host; !hostMap[hostName] {
						//fmt.Println(hostName)
						hostMap[hostName] = true
					}

				}

			}
		case imgURL := <-imgch:
			{
				images = append(images, imgURL)
			}
		case <-exiturlch:
			{
				currentURLGoRoutines--
			}
		case <-exitimgch:
			{
				currentImgGoRoutines--
			}
		}

	}

}
