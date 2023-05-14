// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types2

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// This file contains a definition of the type-checking context; an opaque type
// that may be supplied by users during instantiation.
//
// Contexts serve two purposes:
//  - reduce the duplication of identical instances
//  - short-circuit instantiation cycles
//
// For the latter purpose, we must always have a context during instantiation,
// whether or not it is supplied by the user. For both purposes, it must be the
// case that hashing a pointer-identical type produces consistent results
// (somewhat obviously).
//
// However, neither of these purposes require that our hash is perfect, and so
// this was not an explicit design goal of the context type. In fact, due to
// concurrent use it is convenient not to guarantee de-duplication.
//
// Nevertheless, in the future it could be helpful to allow users to leverage
// contexts to canonicalize instances, and it would probably be possible to
// achieve such a guarantee.

// A Context is an opaque type checking context. It may be used to share
// identical type instances across type-checked packages or calls to
// Instantiate. Contexts are safe for concurrent use.
//
// The use of a shared context does not guarantee that identical instances are
// deduplicated in all cases.
type Context struct {
	mu        sync.Mutex
	typeMap   map[string][]ctxtEntry // type hash -> instances entries
	nextID    int                    // next unique ID
	originIDs map[Type]int           // origin type -> unique ID
}

type ctxtEntry struct {
	orig     Type
	targs    []Type
	instance Type // = orig[targs]
}

// NewContext creates a new Context.
func NewContext() *Context {
	return &Context{
		typeMap:   make(map[string][]ctxtEntry),
		originIDs: make(map[Type]int),
	}
}

// instanceHash returns a string representation of typ instantiated with targs.
// The hash should be a perfect hash, though out of caution the type checker
// does not assume this. The result is guaranteed to not contain blanks.
func (ctxt *Context) instanceHash(orig Type, targs []Type) string {
	assert(ctxt != nil)
	assert(orig != nil)
	var buf bytes.Buffer

	h := newTypeHasher(&buf, ctxt)
	h.string(strconv.Itoa(ctxt.getID(orig)))
	// Because we've already written the unique origin ID this call to h.typ is
	// unnecessary, but we leave it for hash readability. It can be removed later
	// if performance is an issue.
	h.typ(orig)
	if len(targs) > 0 {
		// TODO(rfindley): consider asserting on isGeneric(typ) here, if and when
		// isGeneric handles *Signature types.
		h.typeList(targs)
	}

	return strings.Replace(buf.String(), " ", "#", -1) // ReplaceAll is not available in Go1.4
}

// lookup returns an existing instantiation of orig with targs, if it exists.
// Otherwise, it returns nil.
func (ctxt *Context) lookup(h string, orig Type, targs []Type) Type {
	ctxt.mu.Lock()
	defer ctxt.mu.Unlock()

	for _, e := range ctxt.typeMap[h] {
		if identicalInstance(orig, targs, e.orig, e.targs) {
			return e.instance
		}
		if debug {
			// Panic during development to surface any imperfections in our hash.
			panic(fmt.Sprintf("non-identical instances: (orig: %s, targs: %v) and %s", orig, targs, e.instance))
		}
	}

	return nil
}

// update de-duplicates n against previously seen types with the hash h.  If an
// identical type is found with the type hash h, the previously seen type is
// returned. Otherwise, n is returned, and recorded in the Context for the hash
// h.
func (ctxt *Context) update(h string, orig Type, targs []Type, inst Type) Type {
	assert(inst != nil)

	ctxt.mu.Lock()
	defer ctxt.mu.Unlock()

	for _, e := range ctxt.typeMap[h] {
		if inst == nil || Identical(inst, e.instance) {
			return e.instance
		}
		if debug {
			// Panic during development to surface any imperfections in our hash.
			panic(fmt.Sprintf("%s and %s are not identical", inst, e.instance))
		}
	}

	ctxt.typeMap[h] = append(ctxt.typeMap[h], ctxtEntry{
		orig:     orig,
		targs:    targs,
		instance: inst,
	})

	return inst
}

// getID returns a unique ID for the type t.
func (ctxt *Context) getID(t Type) int {
	ctxt.mu.Lock()
	defer ctxt.mu.Unlock()
	id, ok := ctxt.originIDs[t]
	if !ok {
		id = ctxt.nextID
		ctxt.originIDs[t] = id
		ctxt.nextID++
	}
	return id
}
