package Item

type Item struct{
    Name string
    Description string
    Quantity int
    Price float32
}

func NewItem(name string, desc string, qty int, price float32) Item{
    return Item{
        Name:name,
        Description: desc,
        Quantity: qty,
        Price: price,
    }
}
