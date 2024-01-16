package command

import (
	"log"
	"net/http"

	cnst "github.com/danishprakash/kizai/constants"
	"github.com/fsnotify/fsnotify"
)

func Serve() {
	go watch()
	http.Handle("/", http.FileServer(http.Dir(cnst.BUILD_DIR)))
	http.ListenAndServe(":8000", nil)
}

func watch() {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					log.Println("modified file:", event.Name)
					Build()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Add a path.
	// TODO: fsnotify doesn't support
	// recursive watch, walk over the dirs
	// and watch them
	err = watcher.Add(cnst.DIR)
	err = watcher.Add(cnst.DIR + "/posts")
	err = watcher.Add(cnst.DIR + "/reading")
	err = watcher.Add(cnst.STATIC)
	err = watcher.Add(cnst.STATIC + "/css")
	err = watcher.Add(cnst.TEMPLATES)
	if err != nil {
		log.Fatal(err)
	}

	// Block main goroutine forever.
	<-make(chan struct{})
}
