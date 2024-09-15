package lib

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Max returns the max value in values
func Max[T Number](values ...T) (ret T) {
	if len(values) == 0 {
		return
	}
	ret = values[0]
	for _, v := range values[1:] {
		if v > ret {
			ret = v
		}
	}
	return
}

// Min returns the min value in values
func Min[T Number](values ...T) (ret T) {
	if len(values) == 0 {
		return
	}
	ret = values[0]
	for _, v := range values[1:] {
		if v < ret {
			ret = v
		}
	}
	return
}
