#define SLICE(a) []a
#define VAR(a) i##a

#define GEN_SORT(T, K);;\
	func FUNC(Partition)(data SLICE(T), l, r int) int {;;\
		DO_PARTITION(data, l, r, j);;\
		return j;;\
	};;;;\
	func FUNC(QuickSort)(data SLICE(T), lo, hi int) {;;\
		if lo >= hi {;;\
			return;;\
		};;\
		DO_PARTITION(data, lo, hi, p);;\
		FUNC(QuickSort)(data, lo, p);;\
		FUNC(QuickSort)(data, p+1, hi);;\
	};;;;\
	func FUNC(InsertionSort)(data SLICE(T), l, r int) {;;\
		DO_INSERTION_SORT(data, l, r)\
	};;;;\
	func FUNC(Sort)(data SLICE(T), l, r int) {;;\
		if r-l > 12 {;;\
			FUNC(QuickSort)(data, l, r);;\
			return;;\
		};;\
		DO_INSERTION_SORT(data, l, r);;\
	};;;;\
	func FUNC(Search)(data SLICE(T), key K) (int, bool) {;;\
		DO_SEARCH(data, key, i, ok);;\
		return i, ok;;\
	};;;;\

#define DO_SWAP(DATA, A, B)\
	DATA[A], DATA[B] = DATA[B], DATA[A]\

#define DO_PARTITION(DATA, L, R, PIVOT)\
	;Inlined partition algorithm;;\
	var PIVOT int;;\
	{;;\
		;Let x be a pivot;;\
		x := DATA[L];;\
		PIVOT = L;;\
		for i := L + 1; i < R; i++ {;;\
			if LESS_OR_EQUAL(DATA[i], x) {;;\
				PIVOT++;;\
				DO_SWAP(DATA, PIVOT, i);;\
			};;\
		};;\
		DO_SWAP(DATA, PIVOT, L);;\
	}\

#define DO_INSERTION_SORT(DATA, L, R)\
	;Inlined insertion sort;;\
	for i := L + 1;; i < R;; i++ {;;\
		for j := i;; j > L && GREATER(DATA[j-1], DATA[j]);; j-- {;;\
			data[j], data[j-1] = data[j-1], data[j];;\
		};;\
	}\

#define DO_SEARCH(DATA, KEY, RIGHT, OK)\
	;Inlined binary search;;\
	var OK bool;;\
	RIGHT := len(DATA);;\
	{;;\
		l := 0;;\
		for !OK && l < RIGHT {;;\
			m := l + (RIGHT-l)/2;;\
			switch {;;\
			case ID(DATA[m]) == KEY:;;\
				OK = true;;\
				RIGHT = m;;\
			case ID(DATA[m]) < KEY:;;\
				l = m + 1;;\
			case ID(DATA[m]) > KEY:;;\
				RIGHT = m;;\
			};;\
		};;\
	}\


#define _CONCAT(a, b) a ## b
#define CONCAT(a, b) _CONCAT(a, b)

#define DO_SEARCH_SHORT(DATA, KEY, RIGHT)\
	DO_SEARCH(DATA, KEY, RIGHT, CONCAT(ok, __COUNTER__))\

