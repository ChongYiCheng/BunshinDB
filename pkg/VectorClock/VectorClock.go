package VectorClock

import(
    //"fmt"
    //"sort"
    "errors"
)

type VectorClock struct{
    Vector map[string]int
}

func CompareVectors (vector1 VectorClock, vector2 VectorClock) (VectorClock,error) {

    var Conflict = errors.New("Conflict between both vectors")

    if EqualKeys(vector1,vector2) != true{
        if len(vector1.Vector) == len(vector2.Vector){
            //fmt.Println("Not equivalent tho equal length")
            return VectorClock{},Conflict
        }
        //There's a conflict and the versions diverged need to return false
        if len(vector1.Vector) < len(vector2.Vector){
            for k,_ := range vector1.Vector{
                _, exists := vector2.Vector[k]
                if exists == false{
                    //fmt.Println("Vector 2 doesnt even consist of vector 1")
                    return VectorClock{},Conflict
                }
            }
        } else if len(vector1.Vector) > len(vector2.Vector){
            //fmt.Println("Vector 1 is bigger than vector2")
            return VectorClock{},Conflict
        }
    }

    //If the nodes that have written to the vector clocks are the same, let's see which one is stale
    //fmt.Println("Checkpoint cleared")
    outputBoolean := true

    //Go through every Node's version
	for NodeID,_ := range vector1.Vector{
            outputBoolean = outputBoolean && (vector1.Vector[NodeID] <= vector2.Vector[NodeID]) // Compare element wise if timestamp is stale
    }

    //return true if vector 1 is indeed stale, else false
    if outputBoolean{
        return vector1,nil
    } else{
        return VectorClock{},Conflict
    }
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func EqualKeys(a, b VectorClock) bool {
    if len(a.Vector) != len(b.Vector) {
        return false
    }
    for k, _:= range a.Vector{
        _, exists := b.Vector[k]
        if exists == false{
            return false
        }
    }
    return true
}

func CompareVectorClocks (vectorClocks []VectorClock) []VectorClock{

    listOfConflictingVectorClocks := []VectorClock{}

    for i := 0; i < len(vectorClocks); i++{
        conflictWithEveryVector := true
        for j := 0; j < len(vectorClocks); j++{
            if j != i{
                _,Err := CompareVectors(vectorClocks[i],vectorClocks[j])

                if Err == nil{
                    conflictWithEveryVector = false
                }
            }
        }
        if conflictWithEveryVector == true{
            listOfConflictingVectorClocks = append(listOfConflictingVectorClocks,vectorClocks[i])
        }
    }

    return listOfConflictingVectorClocks
}

func MergeVectorClocks (vectorClocks []VectorClock) VectorClock{

    vector := map[string]int{}

    for _,vectorClock := range vectorClocks{
        for k,v := range vectorClock.Vector{
            if currentV,exists := vector[k]; exists{
                if v > currentV{
                    vector[k] = v
                }
            } else{
                vector[k] = v
            }
        }
    }
    mergedVectorClock := VectorClock{vector}
    return mergedVectorClock
}


