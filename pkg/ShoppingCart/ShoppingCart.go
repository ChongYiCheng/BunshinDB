package ShoppingCart

import(
    "50.041-DistSysProject-BunshinDB/pkg/VectorClock"
    "50.041-DistSysProject-BunshinDB/pkg/Item"
)

type ShoppingCart struct{
    ShopperID string
    Items map[string]Item.Item // Map the item names or IDs to Item Details {Name,Desc,Qty,Price}
    Version VectorClock.VectorClock
}

func NewShoppingCart(shopperID string,items map[string]Item.Item, version VectorClock.VectorClock) ShoppingCart{
    //items to be determined by client, vectorClock determined by Node
    shoppingCart := ShoppingCart{ShopperID:shopperID,Items: items,Version: version}

    return shoppingCart
}

func UpdateVersion(version VectorClock.VectorClock, nodeID string) VectorClock.VectorClock{
    if _,exists := version.Vector[nodeID]; exists{
        version.Vector[nodeID]++
    } else{
        version.Vector[nodeID] = 1
    }
    return version
}

func UpdateCart(shoppingCart ShoppingCart, items map[string]Item.Item, WriterNodeCanonName string) ShoppingCart{
    shoppingCart.Items = items
    //The Node needs to update the vector clock outside of this function
    shoppingCart.Version = UpdateVersion(shoppingCart.Version,WriterNodeCanonName)
    return shoppingCart
}

func CompareShoppingCarts (shoppingCarts []ShoppingCart) []ShoppingCart{
    //Take in an arbritrary number of Shopping Carts

    listOfConflictingShoppingCarts := []ShoppingCart{}

    for i := 0; i < len(shoppingCarts); i++{
        conflictWithEveryVector := true
        for j := 0; j < len(shoppingCarts); j++{
            if j != i{
                _,Err := VectorClock.CompareVectors(shoppingCarts[i].Version,shoppingCarts[j].Version)

                if Err == nil{
                    conflictWithEveryVector = false
                }
            }
        }
        if conflictWithEveryVector == true{
            listOfConflictingShoppingCarts = append(listOfConflictingShoppingCarts,shoppingCarts[i])
        }
    }
    return listOfConflictingShoppingCarts
}

func MergeShoppingCarts (conflictingShoppingCarts []ShoppingCart) ShoppingCart{
    //This assumes that syntactic reconciliation was performed which leaves us with only
    //the shopping carts that have conflicting versions

    items := map[string]Item.Item{}
    conflictingVectorClocks := []VectorClock.VectorClock{}
    shopperID := conflictingShoppingCarts[0].ShopperID

    for _,shoppingCart := range conflictingShoppingCarts{
        for k,v := range shoppingCart.Items{
            if currentV,exists := items[k]; exists{
                if v.Quantity > currentV.Quantity{
                    items[k] = v
                }
            } else{
                items[k] = v
            }
        }
        conflictingVectorClocks = append(conflictingVectorClocks,shoppingCart.Version)
    }

    version := VectorClock.MergeVectorClocks(conflictingVectorClocks)
    //version = UpdateVersion(version,mergerNodeID)

    newShoppingCart := ShoppingCart{shopperID,items,version}
    return newShoppingCart
}
