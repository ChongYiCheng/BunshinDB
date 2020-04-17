package VectorClock

import (
    "testing"
    "reflect"
)


func TestCompareVectorClocks(t *testing.T){

    Vector1 := VectorClock{Vector:map[string]int{"A":1}}
    Vector2 := VectorClock{Vector:map[string]int{"B":1}}
    Vector3 := VectorClock{Vector:map[string]int{"C":1}}
    Vector4 := VectorClock{Vector:map[string]int{"A":1,"B":1}}
    Vector5 := VectorClock{Vector:map[string]int{"A":1,"C":1}}
    Vector6 := VectorClock{Vector:map[string]int{"A":1,"D":1}}
    Vector7 := VectorClock{Vector:map[string]int{"A":1,"C":1,"B":1}}
    Vector8 := VectorClock{Vector:map[string]int{"A":1,"B":1,"D":1}}
    Vector9 := VectorClock{Vector:map[string]int{"A":1,"C":1,"D":1}}
    Vector10 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1,"D":1}}
    Vector11 := VectorClock{Vector:map[string]int{"A":2,"B":1,"C":1,"D":1}}
    Vector12 := VectorClock{Vector:map[string]int{"A":3}}
    Vector13 := VectorClock{Vector:map[string]int{"A":2,"B":2}}

    //Slice of test cases
    tt := []struct{
        name string
        VectorClocks []VectorClock
        ExpectedOutcome []VectorClock
    }{
        {"Only 1 Vector Clock",[]VectorClock{Vector9},[]VectorClock{Vector9}},
        {"Three Different Single Vector",[]VectorClock{Vector1,Vector2,Vector3},[]VectorClock{Vector1,Vector2,Vector3}},
        {"Vector 1 Stale",[]VectorClock{Vector1,Vector4},[]VectorClock{Vector4}},
        {"Vector 1 ancestor of Vector4, Vector3 different origin",[]VectorClock{Vector1,Vector3,Vector4},[]VectorClock{Vector3,Vector4}},
        {"Divergence and Convergence",[]VectorClock{Vector6,Vector5,Vector4,Vector3,Vector2,Vector1,Vector7,Vector8,Vector9},[]VectorClock{Vector7,Vector8,Vector9}},
        {"All Roads leads to Rome",[]VectorClock{Vector1,Vector2,Vector3,Vector4,Vector5,Vector7,Vector8,Vector9,Vector10},[]VectorClock{Vector10}},
        {"Rome wasn't built in a day",[]VectorClock{Vector1,Vector2,Vector3,Vector4,Vector5,Vector6,Vector7,Vector8,Vector9,Vector10,Vector11},[]VectorClock{Vector11}},
        {"Deep tech vs Full stack",[]VectorClock{Vector11,Vector12},[]VectorClock{Vector11,Vector12}},
        {"Deep tech vs Dual boot",[]VectorClock{Vector1,Vector12,Vector13},[]VectorClock{Vector12,Vector13}},
    }

    for _,tc := range tt{
        outcome := CompareVectorClocks(tc.VectorClocks)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }
}

func TestMergeVectorClocks (t *testing.T){

    Vector1 := VectorClock{Vector:map[string]int{"A":1}}
    Vector2 := VectorClock{Vector:map[string]int{"B":1}}
    Vector3 := VectorClock{Vector:map[string]int{"C":1}}
    Vector4 := VectorClock{Vector:map[string]int{"A":1,"B":1}}
    Vector5 := VectorClock{Vector:map[string]int{"A":1,"C":1}}
    Vector6 := VectorClock{Vector:map[string]int{"A":1,"D":1}}
    Vector7 := VectorClock{Vector:map[string]int{"A":1,"C":1,"B":1}}
    Vector8 := VectorClock{Vector:map[string]int{"A":1,"B":1,"D":1}}
    Vector9 := VectorClock{Vector:map[string]int{"A":1,"C":1,"D":1}}
    Vector10 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1,"D":1}}
    Vector11 := VectorClock{Vector:map[string]int{"A":2,"B":1,"C":1,"D":1}}
    Vector12 := VectorClock{Vector:map[string]int{"A":3}}
    Vector13 := VectorClock{Vector:map[string]int{"A":2,"B":2}}

    ExpectedOut1 := VectorClock{Vector:map[string]int{"A":1,"C":1,"D":1}}
    ExpectedOut2 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1}}
    ExpectedOut3 := VectorClock{Vector:map[string]int{"A":1,"B":1}}
    ExpectedOut4 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1}}
    ExpectedOut5 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1,"D":1}}
    ExpectedOut6 := VectorClock{Vector:map[string]int{"A":1,"B":1,"C":1,"D":1}}
    ExpectedOut7 := VectorClock{Vector:map[string]int{"A":2,"B":1,"C":1,"D":1}}
    ExpectedOut8 := VectorClock{Vector:map[string]int{"A":3,"B":1,"C":1,"D":1}}
    ExpectedOut9 := VectorClock{Vector:map[string]int{"A":3,"B":2}}


    //Slice of test cases
    tt := []struct{
        name string
        VectorClocks []VectorClock
        ExpectedOutcome VectorClock
    }{
        {"Only 1 Vector Clock",[]VectorClock{Vector9},ExpectedOut1},
        {"Three Different Single Vector",[]VectorClock{Vector1,Vector2,Vector3},ExpectedOut2},
        {"Vector 1 Stale",[]VectorClock{Vector1,Vector4},ExpectedOut3},
        {"Vector 1 ancestor of Vector4, Vector3 different origin",[]VectorClock{Vector1,Vector3,Vector4},ExpectedOut4},
        {"Divergence and Convergence",[]VectorClock{Vector6,Vector5,Vector4,Vector3,Vector2,Vector1,Vector7,Vector8,Vector9},ExpectedOut5},
        {"All Roads leads to Rome",[]VectorClock{Vector1,Vector2,Vector3,Vector4,Vector5,Vector7,Vector8,Vector9,Vector10},ExpectedOut6},
        {"Rome wasn't built in a day",[]VectorClock{Vector1,Vector2,Vector3,Vector4,Vector5,Vector6,Vector7,Vector8,Vector9,Vector10,Vector11},ExpectedOut7},
        {"Deep tech vs Full stack",[]VectorClock{Vector11,Vector12},ExpectedOut8},
        {"Deep tech vs Dual boot",[]VectorClock{Vector1,Vector12,Vector13},ExpectedOut9},
    }

    for _,tc := range tt{
        outcome := MergeVectorClocks(tc.VectorClocks)
        if reflect.DeepEqual(outcome,tc.ExpectedOutcome) != true{
            t.Errorf("Expected %v;\n Got %v\n",tc.ExpectedOutcome,outcome)
        }
    }

}




