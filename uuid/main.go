package main

import (
 "bytes"
 "fmt"
 "github.com/google/uuid"
 "slices"
)

func main() {
 id := []byte("cE8Jiq!9zCnN0qWYa1&$(zdF")
 putDateTime := []byte("2024122412345612")
 finalBase := slices.Concat(id, putDateTime)

 res, err := uuid.NewRandomFromReader(bytes.NewReader(finalBase))
 if err != nil {
  fmt.Printf("Error: %v\n", err)
  return
 }
 fmt.Println(res.String())
}