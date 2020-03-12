package main
 
import (
    "fmt"
    "log"
    "net/http"
    "encoding/json"
     )

type User struct{
    Id      string
    Balance uint64
}


func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var u User
        serverUser := User{Id:"Server_Res",Balance:8008135}
        w.Header().Set("Content-Type", "application/json") 
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
        fmt.Println(r.Body)
		err := json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		fmt.Println(u)
        json.NewEncoder(w).Encode(serverUser)
	})
	log.Fatal(http.ListenAndServe(":1337", nil))
}
