#include "sort.h"

#define ID(a) a.Key
#define LESS_OR_EQUAL(a, b) a.Key <= b.Key
#define GREATER(a, b) a.Key > b.Key
#define FUNC(a) pair##a

package radix

GEN_SORT(Pair, uint)
