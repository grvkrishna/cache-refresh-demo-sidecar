// fakerepo pretends to be the repo's refresh API for the demo.
//
//	go run ./tools/fakerepo                 # always succeeds (200)
//	FAIL_TIMES=2 go run ./tools/fakerepo    # first 2 calls fail (500), then 200 -> shows retries
//	FAIL_TIMES=99 go run ./tools/fakerepo   # always fails -> shows the final FAILED log
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
)

func main() {
	failTimes, _ := strconv.Atoi(os.Getenv("FAIL_TIMES"))
	var calls atomic.Int64

	http.HandleFunc("POST /cache/refresh", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n := calls.Add(1)

		status := http.StatusOK
		if int(n) <= failTimes {
			status = http.StatusInternalServerError
		}
		log.Printf("[repo] call #%d body=%s -> %d", n, body, status)
		w.WriteHeader(status)
	})

	log.Println("fake repo listening on :8080 (POST /cache/refresh)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
