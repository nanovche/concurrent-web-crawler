package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"
	"os"
	"regexp"
	"sync"
)

const maxNumOfGoRoutines int = 20

var (
	fl *bool
 	fileName = "technologies.json"
	inputTechStackRegexMap map[string]interface{}
	outputTechStack map[string]map[string][]string
	maxDepth int = 3
	tokens = make(chan struct{}, maxNumOfGoRoutines)
)

type Work struct {
	url   string
	depth int
}

func main() {
	//debug.SetGCPercent(-1)

	//reads jsonfile, extracts chosen technologies and creates regexpressions for scripts html meta
	jsonData := readFile(fileName)
	inputTechStackRegexMap = extractTechStackWithRegex(jsonData)

	workList := make(chan []Work)
	var n int // number of pending sends to workList
	//var baseURL string = os.Args[1]

	n++
	go func() {
		var works []Work
		for _, URL := range os.Args[1:] {
			works = append(works, Work{URL, 1})
		}
		workList <- works
	}()

	visited := make(map[string]bool)
	for ; n > 0; n-- {
		list := <-workList
		for _, work := range list {
			//work.url = fixURL(work.url, baseURL)
			if !visited[work.url] {
				visited[work.url] = true
				n++
				go func(work Work) {
					workList <- crawl(work)
				}(work)
			}
		}
	}

	data := marshalIntoJson(outputTechStack)
	writeToFile(data)

	for wbr, vals := range outputTechStack {
		fmt.Printf("Webroot: %s:\n", wbr)
		for tech, fingerprints := range vals {
			fmt.Printf("Technology: %s:\n", tech)
			fmt.Println("Fingerprints:")
			for _, fingerprint := range fingerprints {
				fmt.Printf("%s:\n", fingerprint)
			}
		}
	}
}

func writeToFile(data []byte) {

	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)

}
func crawl(work Work) []Work {
	fmt.Printf("%d\t%s\n", work.depth, work.url)
	if work.depth >= maxDepth {
		return nil
	}


	tokens <- struct{}{} // acquire a token
	links, byteData, err := ExtractLinks(work.url)
	<-tokens // release the token
	searchForPatterns(work.url, byteData)

	if err != nil {
		log.Print(err)
	}

	var works []Work
	for _, link := range links {
		works = append(works, Work{link, work.depth + 1})
	}
	return works
}
func ExtractLinks(URL string) ([]string, []byte, error)   {

	resp, err := http.Get(URL)
	if err != nil {
		fmt.Println(err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("getting %s: %s", URL, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing %s as HTML: %v", URL, err)
	}

	byteData, err := ioutil.ReadAll(resp.Body)
	//analyzeHeader(URL, resp)
	//analyzeCookies(URL, resp)
	var links []string
	node := func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						link, err := resp.Request.URL.Parse(a.Val)
						if err != nil {
							continue // ignore bad URLs
						}
						links = append(links, link.String())
					}
				}
			}
		}
	}
	forEachNode(doc, node, nil)
	return links, byteData, nil

}

func forEachNode(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}
	if post != nil {
		post(n)
	}
}

//var mh sync.Mutex
//func analyzeHeader(URL string, resp *http.Response) {
//
//	for tech, properties := range inputTechStackRegexMap {
//		props := properties.(map[string]interface{})
//
//		for key, value := range props {
//			 if key == "headers" {
//
//				headerEntries := value.(map[string]interface{})
//
//				for field, val := range headerEntries {
//					fingerprint := val.(string)
//					re, _ := regexp.Compile(fingerprint)
//
//					if field == "Server" {
//						valOfResponseHeader := resp.Header.Get("Server")
//						if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//							mh.Lock()
//							if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//								outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//							}
//							mh.Unlock()
//						}
//					}
//					if field == "X-Powered-By" {
//						valOfResponseHeader := resp.Header.Get("X-Powered-By")
//						if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//							mh.Lock()
//							if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//								outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//							}
//							mh.Unlock()
//						}
//					}
//					if field == "Content-Type" {
//						valOfResponseHeader := resp.Header.Get("Content-Type")
//						if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//							mh.Lock()
//							if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//								outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//							}
//							mh.Unlock()
//						}
//					}
//					if field == "X-Content-Encoded-By" {
//						valOfResponseHeader := resp.Header.Get("X-Content-Encoded-By")
//						if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//							mh.Lock()
//							if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//								outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//							}
//							mh.Unlock()
//						}
//					}
//					if field == "X-Jenkins" {
//						if _, ok := resp.Header["X-Jenkins"]; ok {
//							valOfResponseHeader := resp.Header.Get("X-Jenkins")
//							if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//								mh.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mh.Unlock()
//							}
//						}
//					}
//					if field == "X-AspNet-Version" {
//						if _, ok := resp.Header["X-AspNet-Version"]; ok {
//							valOfResponseHeader := resp.Header.Get("X-AspNet-Version")
//							if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//								mh.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mh.Unlock()
//							}
//						}
//					}
//					if field == "X-Fastcgi-Cache" {
//						if _, ok := resp.Header["X-Fastcgi-Cache"]; ok {
//							valOfResponseHeader := resp.Header.Get("X-Fastcgi-Cache")
//							if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//								mh.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mh.Unlock()
//							}
//						}
//					}
//					if field == "X-Pingback" {
//						if _, ok := resp.Header["X-Pingback"]; ok {
//							valOfResponseHeader := resp.Header.Get("X-Pingback")
//							if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//								mh.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mh.Unlock()
//							}
//						}
//					}
//					if field == "X-Application-Context" {
//						if _, ok := resp.Header["X-Application-Context"]; ok {
//							valOfResponseHeader := resp.Header.Get("X-Pingback")
//							if len(re.FindAllString(valOfResponseHeader, - 1)) != 0 {
//								mh.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mh.Unlock()
//							}
//						}
//					}
//				}
//			}
//		}
//	}
//}
//var mc sync.Mutex
//func analyzeCookies(URL string, resp *http.Response){
//
//	for tech, properties := range inputTechStackRegexMap {
//		props := properties.(map[string]interface{})
//		for key, value := range props {
//			if key == "cookies" {
//				cookiesEntries := value.(map[string]interface{})
//				for field, value := range cookiesEntries {
//					fingerprint := value.(string)
//					re, _ := regexp.Compile(fingerprint)
//
//					for _, v := range resp.Cookies() {
//
//						if v.Name == field {
//							valOfCookie := v.Value
//							if len(re.FindAllString(valOfCookie, - 1)) != 0 {
//								mc.Lock()
//								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
//									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
//								}
//								mc.Unlock()
//							}
//						}
//					}
//				}
//			}
//		}
//	}
//}
var mu sync.Mutex
func searchForPatterns(URL string, webResourceInBytes []byte) {

	for tech, properties := range inputTechStackRegexMap {
		if props, ok  := properties.(map[string]interface{}); ok {
			for prop, vals := range props {

				switch val := vals.(type) {

					case []string: {
						for _, v := range val {
							re, _ := regexp.Compile(v)
							matches := re.FindSubmatch(webResourceInBytes)
							if matches != nil {
								var fingerprint string
								for _, sl := range matches[1:] {
									fingerprint = fingerprint + string(sl)
								}
								mu.Lock()
								if !sliceContains(outputTechStack[URL][tech], fingerprint) {
									outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
								}
								mu.Unlock()
							}
						}
					}

					case map[string]string:{
						for k, v := range val {
							if prop == "meta" {
								//key and value take part in regex for meta tag
								re, _ := regexp.Compile(k+v)
								matches := re.FindSubmatch(webResourceInBytes)
								if matches != nil {
									var fingerprint string
									for _, sl := range matches[1:] {
										fingerprint = fingerprint + string(sl)
									}
									mu.Lock()
									if !sliceContains(outputTechStack[URL][tech], fingerprint) {
										outputTechStack[URL][tech] = append(outputTechStack[URL][tech],fingerprint)
									}
									mu.Unlock()
								}
							}
						}
					}
				}
			}
		}
	}
}
//func fixURL(href, baseURL string) string {
//	base, err := url.Parse(baseURL)
//	if err != nil {
//		return ""
//	}
//	path, err := url.Parse(href)
//	if err != nil {
//		return ""
//	}
//	fixedURL := base.ResolveReference(path)
//	return fixedURL.String()
//}
func marshalIntoJson(v interface{}) (dataJSON []byte) {

	dataJSON, err := json.MarshalIndent(v, "","\t")
	if err != nil {
		fmt.Println(err)
	}
	return
}
func readFile (fileName string) (data []byte) {

	file, err := os.Open(fileName)
	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "Error: %v", err); err != nil {
			fmt.Println(err)
		}
	}
	defer file.Close()

	data, _ = ioutil.ReadAll(file)

	return
}
func extractTechStackWithRegex(jsonData []byte) map[string]interface{} {

	var v interface{}
	err := json.Unmarshal(jsonData, &v)
	if err != nil {
		log.Fatalf("%s", err)
	}

	data := v.(map[string]interface{})

	//initialize inputTechStack
	techStack := make(map[string]interface{})
	for k, v := range data {

		if k == "technologies" {
			if technologies, ok := v.(map[string]interface{}); ok {

				for kt, vt := range technologies {
					switch kt {
					case "Node.js", "Java", "Python", "PHP", "MySQL", "Nginx", "Apache Tomcat", "Laravel", "Django", "Flask",
						"Scala", "Go", "Haskell", "Google Analytics", "Raspbian", "React", "Red Hat", "Reddit", "Java Servlet",
						"Jenkins", "Joomla", "Docker", "Angular", "AngularJS", "Microsoft ASP.NET", "Ruby on Rails", "Spring", "Symfony",
						"ThinkPHP", "gRPC": {

						if values, ok := vt.(map[string]interface{}); ok {
							for prop, v := range values {
								if prop == "scripts" || prop == "html" || prop == "meta" || prop == "headers" || prop == "cookies"{
									techStack[kt] = v
								}
							}
						}
					}
					}
				}
			} else {
				log.Fatalf("Unexpected formatting of json")
			}
		}
	}
	return techStack
}
func sliceContains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

