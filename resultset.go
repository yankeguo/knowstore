package main

type NamespacedName struct {
	Namespace string
	Name      string
}

type ResultSet struct {
	m map[NamespacedName]map[string]int64
}

func NewResultSet() *ResultSet {
	return &ResultSet{
		m: make(map[NamespacedName]map[string]int64),
	}
}

func (rs *ResultSet) Len() int {
	return len(rs.m)
}

func (rs *ResultSet) AddCID(cid string, item NamespacedName) {
	result := rs.m[item]
	if result == nil {
		result = make(map[string]int64)
		rs.m[item] = result
	}
	result[cid] = -1
}

func (rs *ResultSet) HasCID(cid string) bool {
	for _, results := range rs.m {
		if _, ok := results[cid]; ok {
			return true
		}
	}
	return false
}

func (rs *ResultSet) SaveUsage(cid string, size int64) (item NamespacedName, ok bool) {
	var results map[string]int64
	for item, results = range rs.m {
		if _, ok = results[cid]; ok {
			results[cid] = size
			return
		}
	}
	return
}

func (rs *ResultSet) GetUsage(item NamespacedName) (total int64, complete bool) {
	results := rs.m[item]
	if results == nil {
		return
	}
	complete = true
	for _, v := range results {
		if v < 0 {
			complete = false
		} else {
			total += v
		}
	}
	return
}

func (rs *ResultSet) List() (items []NamespacedName) {
	for item := range rs.m {
		items = append(items, item)
	}
	return
}
