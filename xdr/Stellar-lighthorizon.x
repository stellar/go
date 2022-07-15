// Copyright 2022 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

%#include "xdr/Stellar-ledger.h"
%#include "xdr/Stellar-types.h"

namespace stellar
{

struct CheckpointIndex {
    uint32 firstCheckpoint;
    uint32 lastCheckpoint;
    Value bitmap;
};

struct TrieIndex {
    uint32 version_; // goxdr gives an error if we simply use "version" as an identifier
    TrieNode root;
};

struct TrieNodeChild {
    opaque key[1];
    TrieNode node;
};

struct TrieNode {
    Value prefix;
    Value value;
    TrieNodeChild children<>;
};

union SerializedLedgerCloseMeta switch (int v)
{
case 0:
    LedgerCloseMeta v0;
};

}
