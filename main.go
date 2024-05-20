package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/lipgloss"
)

type Chapter struct {
	name string
	url  string
}

var _ = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	PaddingTop(2).
	PaddingLeft(4).
	Width(22)

var errorStyle = lipgloss.NewStyle().Bold(true).
	Foreground(lipgloss.Color("red")).
	Background(lipgloss.Color("white")).
	Width(22)

func getChapters(url string) (string, []Chapter) {
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln("error getting the webpage : ", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	// load html
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalln("error loading html : ", err)
	}
	// get chapters
	title := doc.Find(".entry-title").Text()
	var chapters []Chapter
	doc.Find("#chapterlist > ul > li").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a")
		href, ok := link.Attr("href")
		if !ok {
			fmt.Printf("cant get href of chapter N° %v", i)
		}
		chapterTitle := link.Find(".chapternum").Text()
		chapters = append(chapters, Chapter{
			url:  href,
			name: chapterTitle,
		})
	})
	return title, chapters
}

func createFolder(path string) {
	_, err := os.Stat(path)
	if err != nil {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Fatalf("cant create folder at path: %s error: %v", path, err)
		}
	}
}

func getChapterImages(url string) []string {
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln("error getting the chapter webpage : ", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	// load html
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalln("error loading html : ", err)
	}
	// get images
	var images []string
	doc.Find("#readerarea > p > img").Each(func(i int, image *goquery.Selection) {
		src, ok := image.Attr("src")
		if !ok {
			fmt.Printf("cant get src of image N° %v", i)
		}
		images = append(images, src)
	})
	return images
}

func downloadImages(images []string, chapterPath string) {
	var wg sync.WaitGroup
	for i, imageUrl := range images {
		wg.Add(1)
		go downloadFile(imageUrl, chapterPath+"/"+fmt.Sprintf("%v.webp", i), &wg)
	}
	wg.Wait()
}

func downloadFile(URL, fileName string, wg *sync.WaitGroup) error {
	defer wg.Done()
	//Get the response bytes from the URL
	response, err := http.Get(URL)
	// check for errors
	if err != nil {
		return err
	}
	// close
	defer response.Body.Close()
	// check if res is valid
	if response.StatusCode != 200 {
		return errors.New("received non 200 response code")
	}
	//Create a empty file
	file, err := os.Create(fileName)
	// check for errors
	if err != nil {
		panic(err)
	}
	// close file after
	defer file.Close()
	//Write the bytes to the fiel
	_, err = io.Copy(file, response.Body)
	// check for errors
	if err != nil {
		return err
	}
	//
	return nil
}

func main() {
	args := os.Args
	if len(args) <= 1 {
		log.Fatalln(errorStyle.Render("No arguments provided."))
	}
	manhwaUrl := args[1]
	title, chapters := getChapters(manhwaUrl)
	createFolder("./assets")
	createFolder("./assets/" + title)
	for _, chapter := range chapters {
		chapterPath := "./assets/" + title + "/" + chapter.name
		createFolder(chapterPath)
		images := getChapterImages(chapter.url)
		downloadImages(images, chapterPath)
	}
}
