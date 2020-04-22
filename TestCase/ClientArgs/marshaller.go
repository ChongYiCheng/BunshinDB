package main

import(
    "fmt"
    "encoding/json"
    "../../pkg/ShoppingCart"
    "../../pkg/Item"
    "../../pkg/VectorClock"
    "io/ioutil"

)

func main(){

    lakeWoodGuitar := Item.Item{"Lakewood Guitar","Expensive AAA guitar",1,3500}
    version1 := VectorClock.VectorClock{map[string]int{"A":1,"B":1}}
    shopperID := "noblekid96"
    shoppingCart1 := ShoppingCart.ShoppingCart{shopperID,map[string]Item.Item{lakeWoodGuitar.Name:lakeWoodGuitar},version1}

    marshalTest,_ := json.Marshal(shoppingCart1)
    fmt.Println(marshalTest)
    var unMarshalledCart ShoppingCart.ShoppingCart
    _ = json.Unmarshal(marshalTest,&unMarshalledCart)
    fmt.Println(unMarshalledCart)
	file, _ := json.MarshalIndent(shoppingCart1, "", " ")

	_ = ioutil.WriteFile("test.json", file, 0644)
}
