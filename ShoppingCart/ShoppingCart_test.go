package ShoppingCart

import(
    "../VectorClock"
    "testing"
    "reflect"
)

func TestNewShoppingCart(t *testing.T){

    testItems1 := map[string]int{"Lakewood Guitar":1,"Elixir Phosphur Bronze Light Gauge":2}
    testVersion1 := VectorClock.VectorClock{map[string]int{"A":1,"B":1}}
    testOutput1 := ShoppingCart{testItems1,testVersion1}

    tt := []struct{
        Name string
        Items map[string]int
        VectorClock VectorClock.VectorClock
        ExpectedOutcome ShoppingCart
    }{
        {"Basic Test",testItems1,testVersion1,testOutput1},
    }

    for _,tc := range tt{
        outcome := NewShoppingCart(tc.Items,tc.VectorClock)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }

}

func TestUpdateCart(t *testing.T){

    testInputCart1 := ShoppingCart{
        map[string]int{"Lionel":1,"An Guo":1,"Yi Cheng":1},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputItems1 := map[string]int{"Lionel":1,"An Guo":1,"Yi Cheng":1,"Gabriel":1,"Cheow Fu":1}
    testNodeUpdater := "D"
    testOutput1 := ShoppingCart{
        testInputItems1,
        VectorClock.VectorClock{map[string]int{"A":1,"B":1,"D":1}},
    }

    tt := []struct{
        Name string
        InputCart ShoppingCart
        InputItem map[string]int
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
    testInputCart1_1 := ShoppingCart{
        map[string]int{"Lionel":1,"An Guo":1},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputCart1_2 := ShoppingCart{
        map[string]int{"Lionel":1,"Cheow Fu":1},
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
    testInputCart1_1 := ShoppingCart{
        map[string]int{"Lionel":1,"An Guo":1,"Gabriel":1},
        VectorClock.VectorClock{map[string]int{"A":1,"B":1}},
    }
    testInputCart1_2 := ShoppingCart{
        map[string]int{"Lionel":1,"Cheow Fu":1,"Yi Cheng":1},
        VectorClock.VectorClock{map[string]int{"A":2,"C":1}},
    }
    testInputConflictingShoppingCarts1 := []ShoppingCart{testInputCart1_2,testInputCart1_1}
    MergingNodeID1 := "A"
    expectedOutcome1 := ShoppingCart{
        map[string]int{"Lionel":1,"Cheow Fu":1,"An Guo":1,"Gabriel":1,"Yi Cheng":1},
        VectorClock.VectorClock{map[string]int{"A":3,"B":1,"C":1}},
    }

    tt := []struct{
        Name string
        InputConflictingCarts []ShoppingCart
        MergingNodeID string
        ExpectedOutcome ShoppingCart
    }{
        {"Basic Test",testInputConflictingShoppingCarts1,MergingNodeID1,expectedOutcome1},
    }

    for _,tc := range tt{
        outcome := MergeShoppingCarts(tc.InputConflictingCarts,tc.MergingNodeID)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}
