#include "sort.h"

#define ID(a) a.Key
#define LESS_OR_EQUAL(a, b) a.Key <= b.Key
#define GREATER(a, b) a.Key > b.Key
#define FUNC(a) pair##a

package radix

MAKE_SORT(Pair, uint)

func (p Path) has(k uint) (int, bool) {
	DO_SEARCH(p.pairs, k, i, ok)
	ok = ok && p.includes(i)
	return i, ok
}

