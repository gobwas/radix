#include "sort.h"

#define ID(a) a
#define LESS_OR_EQUAL(a, b) a <= b
#define GREATER(a, b) a > b
#define FUNC(a) uint##a

package radix_test

MAKE_SORT(uint, uint)

