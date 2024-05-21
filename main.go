package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/lipgloss"
)

type Chapter struct {
	name string
	url  string
}

type Manga struct {
	title    string
	author   string
	artist   string
	postedOn string
	genres   []string
}

var errorStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#dc2626")).
	PaddingLeft(2).
	PaddingRight(2)

var helpStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#65a30d")).
	PaddingLeft(2).
	PaddingRight(2)

var infoStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#0ea5e9")).
	PaddingLeft(2).
	PaddingRight(2)

var dialogBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#2563eb")).
	Padding(1, 0).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true)

func getChapters(url string) (Manga, []Chapter) {
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
	// get manga infos
	title := doc.Find(".entry-title").Text()
	artist := doc.Find(".fmed:has(b:contains('Artist')) > span").Text()
	author := doc.Find(".fmed:has(b:contains('Author')) > span").Text()
	postedOn := doc.Find(".fmed:has(b:contains('Posted On')) > span").Text()
	var genres []string
	doc.Find("div:has(b:contains('Genres')) > span.mgen > a").Each(func(i int, a *goquery.Selection) {
		genres = append(genres, a.Text())
	})
	manga := Manga{
		title:    strings.TrimSpace(title),
		artist:   strings.TrimSpace(artist),
		author:   strings.TrimSpace(author),
		postedOn: strings.TrimSpace(postedOn),
		genres:   genres,
	}
	return manga, chapters
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
		fmt.Println(errorStyle.Render("No arguments provided."))
		fmt.Println(helpStyle.Render("Help:"))
		fmt.Println(infoStyle.Render("Dev: go run main.go 'https://asuratoon.com/manga/<manga name>/'"))
		fmt.Println(infoStyle.Render("Prod: /manga-scrapper 'https://asuratoon.com/manga/<manga name>/'"))
		os.Exit(1)
	}
	manhwaUrl := args[1]
	mangaDetails, chapters := getChapters(manhwaUrl)
	createFolder("./assets")
	createFolder("./assets/" + mangaDetails.title)
	{

		genreStyle := lipgloss.NewStyle().
			Width(15).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#bfdbfe"}).
			Padding(0, 3)

		mangaTitle := lipgloss.NewStyle().Width(50).Italic(true).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.AdaptiveColor{Light: "#0284c7", Dark: "#0284c7"}).
			Bold(true).Render(mangaDetails.title)

		detailStyle := lipgloss.NewStyle().Align(lipgloss.Left)
		var genres string
		for i, genre := range mangaDetails.genres {
			if i%2 == 0 && i > 0 {
				genres = genres + "\n"
			}
			genres = genres + " " + genreStyle.Render(strings.TrimSpace(genre))
		}
		mangaInfos := lipgloss.JoinVertical(lipgloss.Left,
			detailStyle.Render(fmt.Sprintf("• Artist: %s", mangaDetails.artist)),
			detailStyle.Render(fmt.Sprintf("• Author: %s", mangaDetails.author)),
			detailStyle.Render(fmt.Sprintf("• Chapters: %d", len(chapters))),
			detailStyle.Render(fmt.Sprintf("• Posted On: %s", mangaDetails.postedOn)),
			detailStyle.Render("• Genres :"),
			detailStyle.Render(genres),
		)
		ui := lipgloss.JoinVertical(lipgloss.Center, mangaTitle, mangaInfos)

		dialog := lipgloss.Place(70, 9,
			lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(ui),
			lipgloss.WithWhitespaceChars("猫咪"),
			lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#bfdbfe"}),
		)

		fmt.Println(dialog + "\n")
	}
	for i := range chapters {
		chapter := chapters[len(chapters)-1-i]
		chapterPath := "./assets/" + mangaDetails.title + "/" + chapter.name
		createFolder(chapterPath)
		images := getChapterImages(chapter.url)
		fmt.Println(infoStyle.Render(fmt.Sprintf("getting %s (%d images)", chapter.name, len(images))))
		downloadImages(images, chapterPath)
	}
}
