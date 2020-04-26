package main

import(
    "fmt"
    "encoding/json"
    // "../../pkg/ShoppingCart"
    // "../../pkg/Item"
    // "../../pkg/VectorClock"
    "50.041-DistSysProject-BunshinDB/pkg/ShoppingCart"
    "50.041-DistSysProject-BunshinDB/pkg/Item"
    "50.041-DistSysProject-BunshinDB/pkg/VectorClock"
    "io/ioutil"

)

func main(){

    //synwinViola := Item.Item{"Synwin Viola","Cheapgrade Beginner Viola",1,500}
    stradiViolin := Item.Item{"Stradivarius Violin","Only for the best of the best",1,10000000}
    version1 := VectorClock.VectorClock{map[string]int{"A":1,"B":1}}
    shopperID := "ChongYiCheng"
    shoppingCart1 := ShoppingCart.ShoppingCart{shopperID,map[string]Item.Item{stradiViolin.Name:stradiViolin},version1}
    marshalTest,_ := json.Marshal(shoppingCart1)
    fmt.Println(marshalTest)
    var unMarshalledCart ShoppingCart.ShoppingCart
    _ = json.Unmarshal(marshalTest,&unMarshalledCart)
    fmt.Println(unMarshalledCart)
	file, _ := json.MarshalIndent(shoppingCart1, "", " ")

	_ = ioutil.WriteFile("test_yc_2.json", file, 0644)
}
