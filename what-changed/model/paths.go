// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"reflect"
	"sync"
)

type PathsChanges struct {
	PropertyChanges
	PathItemsChanges map[string]*PathItemChanges
	ExtensionChanges *ExtensionChanges
}

func (p *PathsChanges) TotalChanges() int {
	c := p.PropertyChanges.TotalChanges()
	for k := range p.PathItemsChanges {
		c += p.PathItemsChanges[k].TotalChanges()
	}
	if p.ExtensionChanges != nil {
		c += p.ExtensionChanges.TotalChanges()
	}
	return c
}

func (p *PathsChanges) TotalBreakingChanges() int {
	c := p.PropertyChanges.TotalBreakingChanges()
	for k := range p.PathItemsChanges {
		c += p.PathItemsChanges[k].TotalBreakingChanges()
	}
	return c
}

func ComparePaths(l, r any) *PathsChanges {

	var changes []*Change

	pc := new(PathsChanges)
	pathChanges := make(map[string]*PathItemChanges)

	// Swagger
	if reflect.TypeOf(&v2.Paths{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v2.Paths{}) == reflect.TypeOf(r) {

		lPath := l.(*v2.Paths)
		rPath := r.(*v2.Paths)

		// perform hash check to avoid further processing
		if low.AreEqual(lPath, rPath) {
			return nil
		}

		lKeys := make(map[string]low.ValueReference[*v2.PathItem])
		rKeys := make(map[string]low.ValueReference[*v2.PathItem])
		for k := range lPath.PathItems {
			lKeys[k.Value] = lPath.PathItems[k]
		}
		for k := range rPath.PathItems {
			rKeys[k.Value] = rPath.PathItems[k]
		}

		// run every comparison in a thread.
		var mLock sync.Mutex
		compare := func(path string, pChanges map[string]*PathItemChanges, l, r *v2.PathItem, doneChan chan bool) {
			if !low.AreEqual(l, r) {
				mLock.Lock()
				pathChanges[path] = ComparePathItems(l, r)
				mLock.Unlock()
			}
			doneChan <- true
		}

		doneChan := make(chan bool)
		pathsChecked := 0

		for k := range lKeys {
			if _, ok := rKeys[k]; ok {
				go compare(k, pathChanges, lKeys[k].Value, rKeys[k].Value, doneChan)
				pathsChecked++
				continue
			}
			g, p := lPath.FindPathAndKey(k)
			CreateChange(&changes, ObjectRemoved, v3.PathLabel,
				g.KeyNode, nil, true,
				p.Value, nil)
		}

		for k := range rKeys {
			if _, ok := lKeys[k]; !ok {
				g, p := rPath.FindPathAndKey(k)
				CreateChange(&changes, ObjectAdded, v3.PathLabel,
					nil, g.KeyNode, false,
					nil, p.Value)
			}
		}

		// wait for the things to be done.
		completedChecks := 0
		for completedChecks < pathsChecked {
			select {
			case <-doneChan:
				completedChecks++
			}
		}
		if len(pathChanges) > 0 {
			pc.PathItemsChanges = pathChanges
		}

		pc.ExtensionChanges = CompareExtensions(lPath.Extensions, rPath.Extensions)
	}

	// OpenAPI
	if reflect.TypeOf(&v3.Paths{}) == reflect.TypeOf(l) &&
		reflect.TypeOf(&v3.Paths{}) == reflect.TypeOf(r) {

		lPath := l.(*v3.Paths)
		rPath := r.(*v3.Paths)

		// perform hash check to avoid further processing
		if low.AreEqual(lPath, rPath) {
			return nil
		}

		lKeys := make(map[string]low.ValueReference[*v3.PathItem])
		rKeys := make(map[string]low.ValueReference[*v3.PathItem])
		for k := range lPath.PathItems {
			lKeys[k.Value] = lPath.PathItems[k]
		}
		for k := range rPath.PathItems {
			rKeys[k.Value] = rPath.PathItems[k]
		}

		// run every comparison in a thread.
		var mLock sync.Mutex
		compare := func(path string, pChanges map[string]*PathItemChanges, l, r *v3.PathItem, doneChan chan bool) {
			if !low.AreEqual(l, r) {
				mLock.Lock()
				pathChanges[path] = ComparePathItems(l, r)
				mLock.Unlock()
			}
			doneChan <- true
		}

		doneChan := make(chan bool)
		pathsChecked := 0

		for k := range lKeys {
			if _, ok := rKeys[k]; ok {
				go compare(k, pathChanges, lKeys[k].Value, rKeys[k].Value, doneChan)
				pathsChecked++
				continue
			}
			g, p := lPath.FindPathAndKey(k)
			CreateChange(&changes, ObjectRemoved, v3.PathLabel,
				g.KeyNode, nil, true,
				p.Value, nil)
		}

		for k := range rKeys {
			if _, ok := lKeys[k]; !ok {
				g, p := rPath.FindPathAndKey(k)
				CreateChange(&changes, ObjectAdded, v3.PathLabel,
					nil, g.KeyNode, false,
					nil, p.Value)
			}
		}
		// wait for the things to be done.
		completedChecks := 0
		for completedChecks < pathsChecked {
			select {
			case <-doneChan:
				completedChecks++
			}
		}
		if len(pathChanges) > 0 {
			pc.PathItemsChanges = pathChanges
		}

		pc.ExtensionChanges = CompareExtensions(lPath.Extensions, rPath.Extensions)
	}
	pc.Changes = changes
	return pc
}
