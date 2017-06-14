#include "ppgo/struct/sync_slice.h"

#define ID(a) a.key
#define LESS_OR_EQUAL(a, b) a.key <= b.key
#define FUNC(a) node##a
#define STRUCT() nodeSyncSlice
#define CTOR() newNodeSyncSlice
#define EMPTY() nil

package radix

MAKE_SYNC_SORTED_SLICE(*Node, uint)
