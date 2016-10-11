#include "sync_array.h"

#define ID(a) a.key
#define LESS_OR_EQUAL(a, b) a.key <= b.key
#define GREATER(a, b) a.key > b.key
#define FUNC(a) node##a
#define STRUCT(a) node##a
#define EMPTY() nil

package radix

MAKE_ARRAY(*Node, uint)
