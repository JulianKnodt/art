/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2020 Tailscale Inc. All Rights Reserved.
 */

package art

import (
	"reflect"
	"testing"
)

func TestBaseIndex(t *testing.T) {
	tests := []struct {
		w    int
		a    uint64
		l    int
		want uint64
	}{
		{4, 0, 0, 1},
		{4, 0, 1, 2},
		{4, 8, 1, 3},
		{4, 0, 2, 4},
		{4, 4, 2, 5},
		{4, 8, 2, 6},
		{4, 12, 2, 7},
		{4, 0, 3, 8},
		{4, 2, 3, 9},
		{4, 4, 3, 10},
		// ...
		{4, 14, 4, 30},
		{4, 15, 4, 31},
	}
	for _, tt := range tests {
		if got := baseIndex(tt.w, tt.a, tt.l); got != tt.want {
			t.Errorf("baseIndex(%v, %v, %v) = %v; want %v", tt.w, tt.a, tt.l, got, tt.want)
		}
	}
}

// route4b is a 4-bit route as used in the paper examples.
type route4b struct {
	a uint8 // addr
	l uint8 // prefix len
}

func (r route4b) RouteParams() RouteParams {
	return RouteParams{
		Width: 4,
		Addr:  uint64(r.a),
		Len:   int(r.l),
	}
}

func newSingleLevelTestTable() *Table {
	return &Table{r: make([]Route, 32)}
}

var _ Route = route4b{}

func TestInsertSingleLevel(t *testing.T) {
	x := newSingleLevelTestTable()

	// Figure 3-1.
	r1 := route4b{12, 2}
	if !x.InsertSingleLevel(r1) {
		t.Errorf("insert %v failed", r1)
	}
	want := newSingleLevelTestTable()
	for _, i := range []int{7, 14, 15, 28, 29, 30, 31} {
		want.r[i] = r1
	}
	if !reflect.DeepEqual(x, want) {
		t.Errorf("wrong after 1st step\n got: %v\nwant: %v\n", x, want)
	}

	// Figure 3-2. ("Now assume we insert a route to prefix 14/3")
	r2 := route4b{14, 3}
	if !x.InsertSingleLevel(r2) {
		t.Errorf("insert %v failed", r2)
	}
	for _, i := range []int{15, 30, 31} {
		want.r[i] = r2
	}
	if !reflect.DeepEqual(x, want) {
		t.Errorf("wrong after 2nd step\n got: %v\nwant: %v\n", x, want)
	}

	// Figure 3-3. ("Now assume we insert a route to prefix 8/1")
	r3 := route4b{8, 1}
	if !x.InsertSingleLevel(r3) {
		t.Errorf("insert %v failed", r3)
	}
	for _, i := range []int{3, 6, 12, 13, 24, 25, 26, 27} {
		want.r[i] = r3
	}
	if !reflect.DeepEqual(x, want) {
		t.Errorf("wrong after 3rd step\n got: %v\nwant: %v\n", x, want)
	}
}

// testTable returns the example table set up before section 2.1.2 of the paper.
func testTable() *Table {
	x := newSingleLevelTestTable()
	x.InsertSingleLevel(route4b{12, 2})
	x.InsertSingleLevel(route4b{14, 3})
	x.InsertSingleLevel(route4b{8, 1})
	return x
}

func TestLookup(t *testing.T) {
	x := testTable()
	for _, tt := range []struct {
		addr uint64
		want Route
	}{
		{0, nil},
		{1, nil},
		// ...
		{6, nil},
		{7, nil},
		{8, route4b{8, 1}},
		{9, route4b{8, 1}},
		{10, route4b{8, 1}},
		{11, route4b{8, 1}},
		{12, route4b{12, 2}},
		{13, route4b{12, 2}},
		{14, route4b{14, 3}},
		{15, route4b{14, 3}},
	} {
		got, _ := x.LookupSingleLevel(4, tt.addr)
		if got != tt.want {
			t.Errorf("lookup(addr=%v) = %v; want %v", tt.addr, got, tt.want)
		}
	}
}

func TestDelete(t *testing.T) {
	x := testTable()
	old, ok := x.DeleteSingleLevel(RouteParams{Width: 4, Addr: 12, Len: 2})
	if !ok {
		t.Fatal("didn't delete")
	}
	if want := (route4b{12, 2}); old != want {
		t.Fatalf("deleted %v; want %v", old, want)
	}

	// Note: the paper seems to have a mistake. 2.1.3. ends with
	// "After the route to 12/2 is deleted, the ART returns to
	// Figure 3-2", but none of Figures 3-1, 3-2, 3-3 have just
	// 8/1 and 14/3 in them. Instead, do what the paper probably
	// meant to get back to figure 3-2:
	x = testTable()
	old, ok = x.DeleteSingleLevel(RouteParams{Width: 4, Addr: 8, Len: 1})
	if !ok {
		t.Fatal("didn't delete")
	}
	if want := (route4b{8, 1}); old != want {
		t.Fatalf("deleted %v; want %v", old, want)
	}
	want := &Table{
		r: []Route{
			7:  route4b{12, 2},
			14: route4b{12, 2},
			28: route4b{12, 2},
			29: route4b{12, 2},
			15: route4b{14, 3},
			30: route4b{14, 3},
			31: route4b{14, 3},
		},
	}
	if !reflect.DeepEqual(x, want) {
		t.Errorf("not like Figure 3-2:\n got: %v\nwant: %v\n", x, want)
	}
}
