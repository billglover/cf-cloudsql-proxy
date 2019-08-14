package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	_ "github.com/go-sql-driver/mysql"
)

type server struct {
	db *sql.DB
}

func main() {

	appEnv, _ := cfenv.Current()
	svc, err := appEnv.Services.WithName("cloudsql-proxy")
	if err != nil {
		panic(err.Error())
	}

	dbHost, ok1 := svc.CredentialString("db_host")
	dbPort, ok2 := svc.CredentialString("db_port")
	dbUser, ok3 := svc.CredentialString("db_user")
	dbPass, ok4 := svc.CredentialString("db_pass")

	if ok1 && ok2 && ok3 && ok4 == false {
		fmt.Fprintf(os.Stderr, "Service credentials are missing or unspecified: %+v\n", appEnv)
	}

	fmt.Println("opening DB")
	db, err := sql.Open("mysql", dbUser+":"+dbPass+"@tcp("+dbHost+":"+dbPort+")/")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// We need to retry here as we can't guarantee that the Cloud SQL proxy
	// will be up and running when we try to connect.
	fmt.Println("pinging DB")
	err = retry(5, 2*time.Second, func() (err error) {
		return db.Ping()
	})

	if err != nil {
		panic(err.Error())
	}

	srv := server{
		db: db,
	}

	fmt.Println("starting server")

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
