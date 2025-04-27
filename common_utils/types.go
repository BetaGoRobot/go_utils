package commonutils

type (
	transFunc[T, K any]         func(source T) (target K)
	transFuncWithErr[T, K any]  func(source T) (target K, err error)
	transFuncWithSkip[T, K any] func(source T) (target K, skip bool)

	kvTransFunc[K, RK comparable, V, RV any]         func(key K, value V) (newKey RK, newValue RV)
	kvTransFuncWithErr[K, RK comparable, V, RV any]  func(key K, value V) (newKey RK, newValue RV, err error)
	kvTransFuncWithSkip[K, RK comparable, V, RV any] func(key K, value V) (newKey RK, newValue RV, skip bool)
)
