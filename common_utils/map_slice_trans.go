// Package commonutils 存放一些通用的工具函数
//
//	@update 2025-03-16 14:11:25
package commonutils

// maps:

// TransMapByValue 将Map的Value通过TransFunc转换为目标类型
// for example:
//
//	m := map[string]int{"a": 1, "b": 2}
//	tM := TransMapByValue(m, func(v int) string { return strings.Itoa(v) })
//	// tM: map[string]string{"a": "1", "b": "2"}
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapByValue[K comparable, V any, T any](m map[K]V, fun transFunc[V, T]) map[K]T {
	res := map[K]T{}
	for k, v := range m {
		res[k] = transFunc[V, T](fun)(v)
	}
	return res
}

// TransMapByValueWithErr 将Map的Value通过TransFunc转换为目标类型,允许通过error来终止转换并返回error
// for example:
//
//	m := map[string]int{"a": 1, "b": 2}
//	tM, err := TransMapByValueWithErr(m, func(v int) (string, error) { return strings.Itoa(v), errors.New("error") })
//	// tM: map[string]string{"a": "1", "b": "2"}, err: error
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapByValueWithErr[K comparable, V, R any](m map[K]V, f transFuncWithErr[V, R]) (map[K]R, error) {
	res := make(map[K]R)
	for k, v := range m {
		tgt, err := f(v)
		if err != nil {
			return nil, err
		}
		res[k] = tgt
	}
	return res, nil
}

// TransMap 灵活的Map转换，TransFunc每次需要处理K和V，返回目标的KV
//
// for example:
//
//	m := map[string]int{"2": 1, "1": 2}
//	tM := TransMap(m, func(k string, v int) (newK int, newV string) {
//		newK, _ = strconv.Atoi(k)
//		return newK, strconv.Itoa(v)
//	})
//
// // tM: map[string]string{2: "1", 1: "2"}
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMap[K, RK comparable, V, RV any](m map[K]V, f kvTransFunc[K, RK, V, RV]) map[RK]RV {
	res := make(map[RK]RV)
	for k, v := range m {
		tgtKey, tgtValue := f(k, v)
		res[tgtKey] = tgtValue
	}
	return res
}

// TransMapWithErr 灵活的Map转换，TransFunc每次需要处理K和V，返回目标的KV;
//
//	允许通过error来终止转换并返回error
//
// for example:
//
//	m := map[string]int{"2": 1, "1": 2}
//	tM, err := TransMapWithErr(m, func(k string, v int) (newK int, newV string, err error) {
//		newK, _ = strconv.Atoi(k)
//		return newK, strconv.Itoa(v), nil
//	})
//
// // tM: map[string]string{2: "1", 1: "2"}, err: nil
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapWithErr[K, RK comparable, V, RV any](m map[K]V, f kvTransFuncWithErr[K, RK, V, RV]) (map[RK]RV, error) {
	res := make(map[RK]RV)
	for k, v := range m {
		tgtKey, tgtValue, err := f(k, v)
		if err != nil {
			return nil, err
		}
		res[tgtKey] = tgtValue
	}
	return res, nil
}

// TransMapWithSkip to be filled
//
// for example:
//
//	m := map[string]int{"2": 1, "1": 2}
//	tM := TransMapSeq(m, func(k string, v int) (newK int, newV string, skip bool) {
//		newK, _ = strconv.Atoi(k)
//		if newK == 2 {
//			return "", "", true
//		}
//		return newK, strconv.Itoa(v), false
//	})
//	//tM: map[int]string{1: "2"}
//
// @param m map[K]V
// @param f kvTransFuncWithSkip[K
// @param RK
// @param V
// @param RV]
// @return map[RK]RV
// @return error
// @update 2025-03-27 21:13:01
func TransMapWithSkip[K, RK comparable, V, RV any](m map[K]V, f kvTransFuncWithSkip[K, RK, V, RV]) (map[RK]RV, error) {
	res := make(map[RK]RV)
	for k, v := range m {
		tgtKey, tgtValue, skip := f(k, v)
		if skip {
			continue
		}
		res[tgtKey] = tgtValue
	}
	return res, nil
}

// slices

// TransSlice 将Slice的Value通过TransFunc转换为目标类型
//
// for example:
//
//	s := []int{1, 2, 3}
//	tS := TransSlice(s, func(v int) string { return strconv.Itoa(v) })
//	// tS: []string{"1", "2", "3"}
//	@param s []T
//	@param extractFun transFunc[T, K]
//	@return []K
//	@update 2025-03-16 14:13:09
func TransSlice[T, K any](s []T, f transFunc[T, K]) []K {
	res := []K{}
	for _, item := range s {
		res = append(res, f(item))
	}
	return res
}

// TransSliceWithErr 将Slice的Value通过TransFunc转换为目标类型,允许通过error来终止转换并返回error
//
// for example:
//
//	s := []int{1, 2, 3}
//	tS, err := TransSliceWithErr(s, func(v int) (string, error) { return strconv.Itoa(v), nil })
//	// tS: []string{"1", "2", "3"}, err: nil
//
// @param s []T
// @param f transFuncWithErr[T, K]
// @return []K
// @return error
// @update 2025-03-16 14:13:29
func TransSliceWithErr[T, K any](s []T, f transFuncWithErr[T, K]) ([]K, error) {
	res := []K{}
	for _, item := range s {
		ni, err := f(item)
		if err != nil {
			return nil, err
		}
		res = append(res, ni)
	}
	return res, nil
}

// TransSliceWithSkip 将Slice的Value通过TransFunc转换为目标类型,允许通过skip=True来跳过当前元素
//
// for example:
//
//	s := []int{1, 2, 3}
//	tS := TransSliceWithSkip(s, func(v int) (string, bool) { if v == 2 { return "", true } return strconv.Itoa(v), false })
//	// tS: []string{"1", "3"}
//
//	@param s []T
//	@param extractFun transFuncWithSkipErr[T, K]
//	@return []K
//	@update 2025-03-16 14:15:11
func TransSliceWithSkip[T, K any](s []T, extractFun transFuncWithSkip[T, K]) []K {
	res := []K{}
	for _, item := range s {
		tgt, skip := extractFun(item)
		if skip {
			continue
		}
		res = append(res, tgt)
	}
	return res
}
