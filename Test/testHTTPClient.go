package main



import (
    "fmt"
    "net/http"
    "encoding/json"
    "bytes"
    //"io/ioutil"
)

type User struct{
    Id      string
    Balance uint64
}


func main(){

	client := &http.Client{
	}

	url := "http://localhost:1337/"

	testUser := User{"test",1337}
	//jsonBuffer := []bytes

    jsonBuffer, _ := json.Marshal(testUser)
	//err := json.NewEncoder(jsonBuffer).Encode(testUser)

  	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    res, err := client.Do(req)
    if err != nil {
         fmt.Println("Unable to reach the server.")
    } else {
        var u User
		json.NewDecoder(res.Body).Decode(&u)
		fmt.Println(u)
    }
}


