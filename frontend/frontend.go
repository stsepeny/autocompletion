package frontend

import (
	"html/template"
	"net/http"
	"path/filepath"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var templates = make(map[string]*template.Template)

func init() {
	initializeTemplates()
	defineRoutes()
}

func defineRoutes() {
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/autocomplete", autocompleteHandler)
}

// Base template is 'theme.html'  Can add any variety of content fillers in /layouts directory
func initializeTemplates() {
	layouts, err := filepath.Glob("templates/*.html")
	if err != nil {
		fmt.Print(err.Error())
	}

	for _, layout := range layouts {
		templates[filepath.Base(layout)] = template.Must(template.ParseFiles(layout, "templates/layouts/theme.html"))
	}
}

type Welcome struct {
	Title   string
	Message string
}

type Matches struct {
	Matches []string
}

func AsString(val Matches) []string {
	out := make([]string, len(val.Matches))
	for i, v := range val.Matches {
		out[i] = string(v)
	}
	return out
}

func AsMatches(val []string) Matches {
	out := Matches {}
	for _, v := range val {
		out.Matches = append(out.Matches, v)
	}
	return out
}

// A template taking a struct pointer (&message) containing data to render
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	message := Welcome{Title: "Autocomplete Demo", Message: "Start typing text in the box below to see autocomplete functionality"}

	// outerTheme refernces the template defined within theme.html
	templates["welcome.html"].ExecuteTemplate(w, "outerTheme", &message)
}

func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	// instead of this line, do something smarter: get `q` from request, do a database lookup for records containing `q`,
	// and populate matches
	myval := r.FormValue("q")
	fmt.Println(myval)

	ctx := appengine.NewContext(r)
	hostname, err := appengine.ModuleHostname(ctx, "autocomplete", "", "")
	if err != nil {
		fmt.Print(err.Error())
	}
//	log.Infof(ctx, "My hostname is: %v", hostname)

	// Calling backend service
	url := "http://" + hostname + "/ferret/" + myval
//	log.Infof(ctx, "My URL is: %v", url)
	client := urlfetch.Client(ctx)

	response, err := client.Get(url)
	if err != nil {
		log.Errorf(ctx, "The error is: %v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
//    log.Infof(ctx, "And the response is: %v", response)

    responseData, err := ioutil.ReadAll(response.Body)
    if err != nil {
    	log.Errorf(ctx, "The error getting responseData is: %v", err.Error())
	}
//	log.Infof(ctx, "But responseData is: %v", responseData)

	var mymatches []string
	json.Unmarshal(responseData, &mymatches)
//	log.Infof(ctx, "So far, mymatches are: %v", mymatches)
	
	js, err := json.Marshal(AsMatches(mymatches))

//	matches := Matches{ []string{"foo", "bar"}}
//	js, err := json.Marshal(matches)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Println("What is in js?")
	fmt.Println(js)
	w.Write(js)
}
