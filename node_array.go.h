#include "array.h"

#define ID(a) a.key
#define LESS_OR_EQUAL(a, b) a.key <= b.key
#define GREATER(a, b) a.key > b.key
#define FUNC(a) node##a
#define STRUCT(a) node##a

package radix

MAKE_ARRAY(*node, uint)
