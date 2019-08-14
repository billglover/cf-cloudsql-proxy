package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type server struct {
	db *sql.DB
}

func main() {

	user := os.Getenv("CLOUDSQL_USER")
	if user == "" {
		panic("CLOUDSQL_USER variable not set")
	}

	pass := os.Getenv("CLOUDSQL_PASS")
	if pass == "" {
		panic("CLOUDSQL_PASS variable not set")
	}

	project := os.Getenv("CLOUDSQL_PROJECT")
	if project == "" {
		panic("CLOUDSQL_PROJECT variable not set")
	}

	socketDir := os.Getenv("CLOUDSQL_SOCKET_DIR")
	if socketDir == "" {
		panic("CLOUDSQL_SOCKET_DIR variable not set")
	}

	socket := os.Getenv("CLOUDSQL_INSTANCE")
	if socket == "" {
		panic("CLOUDSQL_INSTANCE variable not set")
	}

	db, err := sql.Open("mysql", user+":"+pass+"@unix("+socketDir+"/"+socket+")/")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// We need to retry here as we can't guarantee that the Cloud SQL proxy
	// will be up and running when we try to connect.
	err = retry(5, 2*time.Second, func() (err error) {
		return db.Ping()
	})

	if err != nil {
		panic(err.Error())
	}

	srv := server{
		db: db,
	}

	http.HandleFunc("/", srv.handler)
	http.ListenAndServe(":8080", nil)
}

// Retry takes a function and retries a number of times before declaring an error
// Source: https://stackoverflow.com/a/47606858
func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Println("retrying after error:", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (srv server) handler(w http.ResponseWriter, r *http.Request) {

	rows, err := srv.db.Query("SHOW databases")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Fprintln(w, "Available databases:")
	fmt.Fprintln(w, "--------------------")

	var table string
	if rows != nil {
		for rows.Next() {
			rows.Scan(&table)
			fmt.Fprintln(w, table)
		}
	}
}
