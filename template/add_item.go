package template

var indexAddItem = `
	{{if $.ArgsInfo.OptimizeFlagUsePoolAndDisableDeleteUnload}}
			{{if $indexItem.Unique}}    
				{{if $indexItem.Pk}}    
					actual, loaded := m.hash{{$indexItem.Keys}}.LoadOrStore({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, -1)
					if !loaded {
						idx := m.objectPool.Put(cls)
						actual = idx
						m.hash{{$indexItem.Keys}}.Store({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, idx)
				{{else}}
					m.hash{{$indexItem.Keys}}.Store({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, idx)
				{{end}}
			{{else}}
				if v, ok := m.hash{{$indexItem.Keys}}.LoadOrStore({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, -1); !ok {
					idxMap := m.objectMapPool.New(idx)
					m.hash{{$indexItem.Keys}}.Store({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, idxMap)
				} else {
					m.objectMapPool.Put(v, idx)
				}
				
			{{end}}
	{{else}}
			{{if $indexItem.Unique}}    
				{{if $indexItem.Pk}}    
					actual, loaded := m.hash{{$indexItem.Keys}}.LoadOrStore({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, cls)
					if !loaded {
						actual = cls
				{{else}}
					m.hash{{$indexItem.Keys}}.Store({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, cls)
				{{end}}
			{{else}}
				if v, ok := m.hash{{$indexItem.Keys}}.LoadOrStore({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }, &{{$.DataName}}Set{}); !ok {
					v.Store(cls, true)
				} else {
					v.Store(cls, true)
				}
			{{end}}
	{{end}}
`
