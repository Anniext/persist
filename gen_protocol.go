package main

//go:generate persist -src=protocol  -dst=data -pkgName=data -unload=true -fileName=bag.go  ItemLocal

//go:generate persist -src=protocol  -dst=data -pkgName=data -unload=true -fileName=bag.go  GoodsLocal
