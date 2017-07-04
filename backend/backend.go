package backend

import (
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/argusdusty/Ferret"
	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"database/sql"
	"os"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

var ExampleWords []string
var ExampleData []interface{}

var ExampleConverter = func(s string) []byte { return []byte(s) }
var ExampleSearchEngine *ferret.InvertedSuffix

const listStatement = `select name, length(name) as nml from Products`

func init() {
	handleRequests()
}

func handleRequests() {
	myRouter := mux.NewRouter()
	myRouter.HandleFunc("/ferret/{word}", autocompleteHandler)
	myRouter.HandleFunc("/_ah/warmup", warmupHandler)
	// The path "/" matches everything not matched by some other path.
    http.Handle("/", myRouter)
}

func warmupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// Perform warmup tasks, including ones that require a context,
	// such as retrieving data from Datastore.
	datastoreName := os.Getenv("MYSQL_CONNECTION")
	
	var err error
	db, err = sql.Open("mysql", datastoreName)
	if err != nil {
		log.Errorf(ctx, "mysql open error: %v", err)
	}
	log.Infof(ctx, "opened db: %v", db)

	rows, err := db.Query(listStatement)
	if err != nil {
		log.Errorf(ctx, "Could not get the list of products: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var vname string
		var vlen interface{}
		if err := rows.Scan(&vname, &vlen); err != nil {
			log.Errorf(ctx, "Could not get products name and char length: %v", err)
		}
		ExampleWords = append(ExampleWords, vname)
		ExampleData = append(ExampleData, vlen)
	}

	ExampleSearchEngine = ferret.New(ExampleWords, ExampleWords, ExampleData, ExampleConverter)
	log.Infof(ctx, "warmup done")
}

func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	// instead of this line, do something smarter: get `q` from request, do a database lookup for records containing `q`,
	// and populate matches
	vars := mux.Vars(r)
	inputword := vars["word"]	

	ctx := appengine.NewContext(r)
	log.Infof(ctx, "the word is: %v", inputword)

	ss, _ := ExampleSearchEngine.Query(inputword, 8)

	js, err := json.Marshal(ss)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Println("What is in js?")
	fmt.Println(js)
	w.Write(js)
}
