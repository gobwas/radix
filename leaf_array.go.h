#include "ppgo/struct/array.h"
#include "ppgo/util.h"

#define ID(a) a
#define LESS_OR_EQUAL(a, b) a <= b
#define FUNC(a) a
#define STRUCT() uintArray
#define CTOR() newUintArray
#define EMPTY() 0
#define VAR(a) CONCAT(UintArray, a)

package radix

MAKE_SORTED_ARRAY(15, uint, uint)
