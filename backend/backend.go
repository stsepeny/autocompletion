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
	"strconv"

	"google.golang.org/appengine/memcache"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

var ChunkWords []string
var ChunkData []interface{}
var ExampleWords []string
var ExampleData []interface{}

var ExampleConverter = func(s string) []byte { return []byte(s) }
var ExampleSearchEngine *ferret.InvertedSuffix

const chunkSize = 10000
const listStatement = `select name, length(name) as nml from Products`

func Min(x, y int) int {
    if x < y {
        return x
    }
    return y
}

func Max(x, y int) int {
    if x > y {
        return x
    }
    return y
}

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
	
	//Connect to Cloud SQL DB
	var err error
	db, err = sql.Open("mysql", datastoreName)
	if err != nil {
		log.Errorf(ctx, "mysql open error: %v", err)
	}

	//Query DB
	rows, err := db.Query(listStatement)
	if err != nil {
		log.Errorf(ctx, "Could not get the list of products: %v", err)
	}
	defer rows.Close()

	// Iterate through rows and append data to ExampleWords and ExampleData slices
	for rows.Next() {
		var vname string
		var vlen interface{}
		if err := rows.Scan(&vname, &vlen); err != nil {
			log.Errorf(ctx, "Could not get products name and char length: %v", err)
		}
		ExampleWords = append(ExampleWords, vname)
		ExampleData = append(ExampleData, vlen)
	}

	// Figure out number of chunks to store in Memcache
	chunksNum := len(ExampleWords) / chunkSize + 1

	// Store chunksNum in Memcache
	itemchunk := &memcache.Item{
		Key: "chunksNum-autocompletion-17230",
		Object: chunksNum,
	}
	if err := memcache.Gob.Add(ctx, itemchunk); err == memcache.ErrNotStored {
		log.Infof(ctx, "item with key %q already exists in memcache", itemchunk.Key)
	} else if err != nil {
		log.Errorf(ctx, "error adding chunksNum memcache item: %v", err)
	}

	// Store ExampleData in Memcache
	itemd := &memcache.Item{
		Key: "ExampleData-autocompletion-17230",
		Object: ExampleData,
	}
	if err := memcache.Gob.Add(ctx, itemd); err == memcache.ErrNotStored {
		log.Infof(ctx, "item with key %q already exists in memcache", itemd.Key)
	} else if err != nil {
		log.Errorf(ctx, "error adding ExampleData memcache item: %v", err)
	}

	// Loop to store ExampleWords in Memcache chunk by chunk
	for i := 0; i < chunksNum; i++ {
		ew := ExampleWords[i*chunkSize:Min((i+1)*chunkSize, len(ExampleWords))]

		itemname := "ExampleWords-autocompletion-17230-" + strconv.Itoa(i)
		itemw := &memcache.Item{
			Key: itemname,
			Object: ew,
		}
		if err := memcache.Gob.Add(ctx, itemw); err == memcache.ErrNotStored {
			log.Infof(ctx, "item with key %q already exists in memcache", itemw.Key)
		} else if err != nil {
			log.Errorf(ctx, "error adding words memcache item: %v", err)
		}
	}

	ExampleSearchEngine = ferret.New(ExampleWords, ExampleWords, ExampleData, ExampleConverter)
}

func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	// Get the value of parameter
	vars := mux.Vars(r)
	inputword := vars["word"]	

	ctx := appengine.NewContext(r)
	log.Infof(ctx, "the word is: %v", inputword)

	// Only if we do not have any data then we load them from Memcache
	if ExampleWords == nil {
		// Get a number of chunks from Memcache
		var chunksNum int
		_, err := memcache.Gob.Get(ctx, "chunksNum-autocompletion-17230", &chunksNum)
		if err != nil {
			log.Errorf(ctx, "error getting chunksNum memcache item: %v", err)
		}

		// Get ExampleData of chunks from Memcache
		_, err = memcache.Gob.Get(ctx, "ExampleData-autocompletion-17230", &ExampleData)
		if err != nil {
			log.Errorf(ctx, "error getting ExampleData memcache item: %v", err)
		}

		// Get all data from Memcache
		for i := 0; i < chunksNum; i++ {
			_, err := memcache.Gob.Get(ctx, "ExampleWords-autocompletion-17230-" + strconv.Itoa(i), &ChunkWords)
			if err != nil {
				log.Errorf(ctx, "error getting words memcache item: %v", err)
			}
			ExampleWords = append(ExampleWords, ChunkWords...)
		}

		ExampleSearchEngine = ferret.New(ExampleWords, ExampleWords, ExampleData, ExampleConverter)

		log.Infof(ctx, "loaded suffix array")
	} else {
		log.Infof(ctx, "ExampleWords is not empty, data is preloaded")
	}

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
