//go:build go1.24
// +build go1.24

package commonutils

import "iter"

// maps:

// TransMapByValueSeq 将Map的Value通过TransFunc转换为目标类型
// for example:
//
//	m := map[string]int{"a": 1, "b": 2}
//	tM := TransMapByValueSeq(maps.All(m), func(v int) string { return strings.Itoa(v) })
//	// maps.Collect(tM): map[string]string{"a": "1", "b": "2"}
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapByValueSeq[K comparable, V any, R any](m iter.Seq2[K, V], fun transFunc[V, R]) iter.Seq2[K, R] {
	return func(yield func(K, R) bool) {
		for k, v := range m {
			if !yield(k, fun(v)) {
				return
			}
		}
	}
}

// TransMapSeq 灵活的Map转换，TransFunc每次需要处理K和V，返回目标的KV
//
// for example:
//
//	m := map[string]int{"2": 1, "1": 2}
//	tM := TransMapSeq(maps.All(m), func(k string, v int) (newK int, newV string) {
//		newK, _ = strconv.Atoi(k)
//		return newK, strconv.Itoa(v)
//	})
//
// // maps.Collect(tM): map[string]string{2: "1", 1: "2"}
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapSeq[K, RK comparable, V, RV any](m iter.Seq2[K, V], f kvTransFunc[K, RK, V, RV]) iter.Seq2[RK, RV] {
	return func(yield func(RK, RV) bool) {
		for k, v := range m {
			tgtKey, tgtValue := f(k, v)
			if !yield(tgtKey, tgtValue) {
				return
			}
		}
	}
}

// TransMapWithSkipSeq 灵活的Map转换，TransFunc每次需要处理K和V，返回目标的KV
//
// for example:
//
//	m := map[string]int{"2": 1, "1": 2}
//	tM := TransMapSeq(maps.All(m), func(k string, v int) (newK int, newV string, skip bool) {
//		newK, _ = strconv.Atoi(k)
//		if newK == 2 {
//			return "", "", true
//		}
//		return newK, strconv.Itoa(v), false
//	})
//	// maps.Collect(tM): map[int]string{1: "2"}
//
//	@param m map[K]V
//	@param fun mapValuesTransFun[V, T]
//	@return map
//	@update 2025-03-06 11:40:05
func TransMapWithSkipSeq[K, RK comparable, V, RV any](m iter.Seq2[K, V], f kvTransFuncWithSkip[K, RK, V, RV]) iter.Seq2[RK, RV] {
	return func(yield func(RK, RV) bool) {
		for k, v := range m {
			tgtKey, tgtValue, skip := f(k, v)
			if skip {
				continue
			}
			if !yield(tgtKey, tgtValue) {
				return
			}
		}
	}
}

// slices

// TransSliceSeq 将Slice的Value通过TransFunc转换为目标类型
//
// for example:
//
//	s := []int{1, 2, 3}
//	tS := TransSlice(slices.Value(s), func(v int) string { return strconv.Itoa(v) })
//	// slices.Collect(tS): []string{"1", "2", "3"}
//	@param s []T
//	@param extractFun transFunc[T, K]
//	@return []K
//	@update 2025-03-16 14:13:09
func TransSliceSeq[T, K any](s iter.Seq[T], f transFunc[T, K]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for item := range s {
			if !yield(f(item)) {
				return
			}
		}
	}
}

// TransSliceWithSkipSeq 将Slice的Value通过TransFunc转换为目标类型,允许通过skip=True来跳过当前元素
//
// for example:
//
//	s := []int{1, 2, 3}
//	tS := TransSliceWithSkipSeq(slices.Value(s), func(v int) (string, bool) { if v == 2 { return "", true } return strconv.Itoa(v), false })
//	// slices.Collect(tS): []string{"1", "3"}
//
//	@param s []T
//	@param extractFun transFuncWithSkipErr[T, K]
//	@return []K
//	@update 2025-03-16 14:15:11
func TransSliceWithSkipSeq[T, K any](s iter.Seq[T], extractFun transFuncWithSkip[T, K]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for item := range s {
			tgt, skip := extractFun(item)
			if skip {
				continue
			}
			if !yield(tgt) {
				return
			}
		}
	}
}
