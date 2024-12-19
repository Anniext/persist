package template

var indexRemoveItem = `
	{{if $.ArgsInfo.OptimizeFlagUsePoolAndDisableDeleteUnload}}
			{{if $indexItem.Unique}}    
			//	m.hash{{$indexItem.Keys}}.Delete({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} })
			{{else}}
			//	if v, ok := m.hash{{$indexItem.Keys}}.Load({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }); ok {
			//		v.Delete(cls)
			//	}
			//	if v, ok := m.hash{{$indexItem.Keys}}.Load({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }); ok {
			//		has := false
			//		v.Range(func(key *{{$.ArgsInfo.PersistPkgName}}.{{$.DataName}}, value bool) bool {
			//			has = true
			//			return false
			//		})
			//		if !has {
			//			m.hash{{$indexItem.Keys}}.Delete({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} })
			//		}
			//	}
			{{end}}
	{{else}}
			{{if $indexItem.Unique}}    
				m.hash{{$indexItem.Keys}}.Delete({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} })
			{{else}}
				if v, ok := m.hash{{$indexItem.Keys}}.Load({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }); ok {
					v.Delete(cls)
				}
				if v, ok := m.hash{{$indexItem.Keys}}.Load({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} }); ok {
					has := false
					v.Range(func(key *{{$.ArgsInfo.PersistPkgName}}.{{$.DataName}}, value bool) bool {
						has = true
						return false
					})
					if !has {
						m.hash{{$indexItem.Keys}}.Delete({{$.DataName}}KeyTypeHash{{$indexItem.Keys}}{ {{$indexItem.ClsPointKeys}} })
					}
				}
			{{end}}
	{{end}}
`
