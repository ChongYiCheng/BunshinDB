package ShoppingCart

import(
    "../VectorClock"
    "../Item"
    "testing"
    "reflect"
)

func TestNewShoppingCart(t *testing.T){

    shopperID := "noblekid96"
    lakeWoodGuitar := Item.Item{"Lakewood Guitar","Fucking expensive guitar",1,3500}
    strings := Item.Item{"Elixir Phosphur Bronze Light Gauge","Long lasting strings",2,20}
    testItems1 := map[string]Item.Item{lakeWoodGuitar.Name:lakeWoodGuitar,strings.Name:strings}
    testVersion1 := VectorClock.VectorClock{map[string]int{"A":1,"B":1}}
    testOutput1 := ShoppingCart{shopperID,testItems1,testVersion1}

    tt := []struct{
        Name string
        ShopperID string
        Items map[string]Item.Item
        VectorClock VectorClock.VectorClock
        ExpectedOutcome ShoppingCart
    }{
        {"Basic Test",shopperID,testItems1,testVersion1,testOutput1},
    }

    for _,tc := range tt{
        outcome := NewShoppingCart(tc.ShopperID,tc.Items,tc.VectorClock)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}

func TestUpdateCart(t *testing.T){

    shopperID := "noblekid96"
    lionel := Item.Item{"Lionel","Zai",1,100}
    anguo := Item.Item{"An Guo","Zai",1,90}
    yicheng := Item.Item{"Yi Cheng","Zai",1,80}
    gabriel := Item.Item{"Gabriel","Zai",1,80}
    cheowfu := Item.Item{"Cheow Fu","noob",1,70}

    testInputCart1 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,anguo.Name:anguo,yicheng.Name:yicheng},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputItems1 := map[string]Item.Item{lionel.Name:lionel,anguo.Name:anguo,yicheng.Name:yicheng,gabriel.Name:gabriel,cheowfu.Name:cheowfu}
    //testInputItems1 := map[string]int{"Lionel":1,"An Guo":1,"Yi Cheng":1,"Gabriel":1,"Cheow Fu":1}
    testNodeUpdater := "D"
    testOutput1 := ShoppingCart{
        shopperID,
        testInputItems1,
        VectorClock.VectorClock{map[string]int{"A":1,"B":1,"D":1}},
    }

    tt := []struct{
        Name string
        InputCart ShoppingCart
        InputItem map[string]Item.Item
        InputUpdaterID string
        ExpectedOutcome ShoppingCart
    }{
        {"Basic Test",testInputCart1,testInputItems1,testNodeUpdater,testOutput1},
    }

    for _,tc := range tt{
        outcome := UpdateCart(tc.InputCart,tc.InputItem,tc.InputUpdaterID)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}

func TestCompareShoppingCarts (t *testing.T) {

    shopperID := "noblekid96"
    lionel := Item.Item{"Lionel","Zai",1,100}
    anguo := Item.Item{"An Guo","Zai",1,90}
    //yicheng := Item.Item{"Yi Cheng","Zai",1,80}
    //gabriel := Item.Item{"Gabriel","Zai",1,80}
    cheowfu := Item.Item{"Cheow Fu","noob",1,70}

    testInputCart1_1 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,anguo.Name:anguo},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputCart1_2 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,cheowfu.Name:cheowfu},
        //map[string]int{"Lionel":1,"Cheow Fu":1},
        VectorClock.VectorClock{map[string]int{"A":2}},
    }
    testInputConflictingShoppingCarts1 := []ShoppingCart{testInputCart1_2,testInputCart1_1}
    expectedOutcome1 := []ShoppingCart{testInputCart1_2,testInputCart1_1}

    tt := []struct{
        Name string
        InputConflictingCarts []ShoppingCart
        ExpectedOutcome []ShoppingCart
    }{
        {"Basic Test",testInputConflictingShoppingCarts1,expectedOutcome1},
    }

    for _,tc := range tt{
        outcome := CompareShoppingCarts(tc.InputConflictingCarts)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}

func TestMergeShoppingCarts(t *testing.T){
    shopperID := "noblekid96"
    lionel := Item.Item{"Lionel","Zai",1,100}
    anguo := Item.Item{"An Guo","Zai",1,90}
    yicheng := Item.Item{"Yi Cheng","Zai",1,80}
    gabriel := Item.Item{"Gabriel","Zai",1,80}
    cheowfu := Item.Item{"Cheow Fu","noob",1,70}

    testInputCart1_1 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,anguo.Name:anguo,gabriel.Name:gabriel},
        //map[string]int{"Lionel":1,"An Guo":1,"Gabriel":1},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputCart1_2 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,cheowfu.Name:cheowfu,yicheng.Name:yicheng},
        //map[string]int{"Lionel":1,"Cheow Fu":1,"Yi Cheng":1},
        VectorClock.VectorClock{map[string]int{"A":2,"C":1}},
    }
    testInputConflictingShoppingCarts1 := []ShoppingCart{testInputCart1_2,testInputCart1_1}
    //MergingNodeID1 := "A"
    expectedOutcome1 := ShoppingCart{
        shopperID,
        map[string]Item.Item{lionel.Name:lionel,anguo.Name:anguo,cheowfu.Name:cheowfu,gabriel.Name:gabriel,yicheng.Name:yicheng},
        //map[string]int{"Lionel":1,"Cheow Fu":1,"An Guo":1,"Gabriel":1,"Yi Cheng":1},
        VectorClock.VectorClock{map[string]int{"A":2,"B":1,"C":1}},
    }

    tt := []struct{
        Name string
        InputConflictingCarts []ShoppingCart
        //MergingNodeID string
        ExpectedOutcome ShoppingCart
    }{
        {"Basic Test",testInputConflictingShoppingCarts1,expectedOutcome1},
    }

    for _,tc := range tt{
        outcome := MergeShoppingCarts(tc.InputConflictingCarts)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}
