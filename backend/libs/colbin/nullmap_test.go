package colbin

import (
	"math/rand"
	"reflect"
	"testing"
)

func i32p(v int32) *int32   { return &v }
func strp(v string) *string { return &v }
func tagp(v Tag) *Tag       { return &v }

// PtrRecord exercises pointer fields (scalar, string, struct), a slice of pointers,
// and maps with scalar/string keys and both scalar and pointer values.
type PtrRecord struct {
	ID      int64            `cb:"id"`
	OptInt  *int32           `cb:"opt_int"` // nil / &0 / &value must all round-trip
	OptName *string          `cb:"opt_name"`
	OptTag  *Tag             `cb:"opt_tag"` // nullable nested struct
	List    []*int32         `cb:"list"`    // array with null slots
	Counts  map[string]int32 `cb:"counts"`
	Refs    map[int32]*Tag   `cb:"refs"` // map with pointer values
}

func ptrSample() []PtrRecord {
	return []PtrRecord{
		{
			ID:      1,
			OptInt:  i32p(0), // present, value zero (distinct from nil)
			OptName: strp("hello"),
			OptTag:  tagp(Tag{Key: "t", Weight: 5}),
			List:    []*int32{i32p(10), nil, i32p(30)},
			Counts:  map[string]int32{"a": 1, "b": 2},
			Refs:    map[int32]*Tag{7: tagp(Tag{Key: "r", Weight: 9}), 8: nil},
		},
		{
			ID:      2,
			OptInt:  nil, // absent
			OptName: nil,
			OptTag:  nil,
			List:    []*int32{nil, nil},
			Counts:  nil,
			Refs:    nil,
		},
		{
			ID:      3,
			OptInt:  i32p(-42),
			OptName: strp(""),
			OptTag:  tagp(Tag{Key: "z", Weight: 0}),
			List:    nil,
			Counts:  map[string]int32{"solo": 100},
			Refs:    map[int32]*Tag{1: tagp(Tag{Key: "x", Weight: -3})},
		},
	}
}

func TestPointerAndMapRoundTrip(t *testing.T) {
	in := ptrSample()
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []PtrRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("ptr/map round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// TestRandomPtrMapRoundTrip stresses the null bitmap and map paths at volume with
// a random mix of nil / present values and varying map/slice sizes.
func TestRandomPtrMapRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	in := make([]PtrRecord, 300)
	for i := range in {
		r := PtrRecord{ID: int64(i)}
		if rng.Intn(3) != 0 { // ~2/3 present
			r.OptInt = i32p(int32(rng.Intn(2000) - 1000))
		}
		if rng.Intn(2) == 0 {
			r.OptName = strp(randString(rng, 8))
		}
		if rng.Intn(2) == 0 {
			r.OptTag = tagp(Tag{Key: randString(rng, 5), Weight: int32(rng.Intn(100) - 50)})
		}
		for j, m := 0, rng.Intn(5); j < m; j++ {
			if rng.Intn(4) == 0 {
				r.List = append(r.List, nil)
			} else {
				r.List = append(r.List, i32p(int32(rng.Intn(500))))
			}
		}
		if rng.Intn(2) == 0 {
			r.Counts = map[string]int32{}
			for j, m := 0, 1+rng.Intn(4); j < m; j++ {
				r.Counts[randString(rng, 4)] = int32(rng.Intn(1000))
			}
		}
		if rng.Intn(2) == 0 {
			r.Refs = map[int32]*Tag{}
			for j, m := 0, 1+rng.Intn(3); j < m; j++ {
				key := int32(rng.Intn(1000))
				if rng.Intn(3) == 0 {
					r.Refs[key] = nil
				} else {
					r.Refs[key] = tagp(Tag{Key: randString(rng, 4), Weight: int32(rng.Intn(50))})
				}
			}
		}
		in[i] = r
	}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []PtrRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("random ptr/map round-trip mismatch")
	}
}

// ExplicitIDRecord uses explicit numeric field ids via the cb tag: bare int, and
// name+int; one field keeps a hashed id.
type ExplicitIDRecord struct {
	A int32  `cb:"5"`      // id = 5, no hash
	B int32  `cb:"beta,7"` // name "beta", id = 7
	C string `cb:"gamma"`  // hashed id from "gamma"
	D int32  `cb:"200"`    // id = 200
}

func TestExplicitFieldIDs(t *testing.T) {
	ti, err := getTypeInfo(reflect.TypeOf(ExplicitIDRecord{}))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]uint8{}
	for i := range ti.fields {
		got[ti.fields[i].name] = ti.fields[i].id
	}
	if got["A"] != 5 || got["beta"] != 7 || got["D"] != 200 {
		t.Fatalf("explicit ids wrong: %+v", got)
	}
	// Round-trip to confirm encode/decode agree on the ids.
	in := []ExplicitIDRecord{{A: 1, B: 2, C: "x", D: 3}, {A: 10, B: 20, C: "y", D: 30}}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []ExplicitIDRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("explicit-id round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// TestDuplicateExplicitID rejects two fields claiming the same id.
func TestDuplicateExplicitID(t *testing.T) {
	type Dup struct {
		A int32 `cb:"3"`
		B int32 `cb:"3"`
	}
	if _, err := Marshal([]Dup{{}}); err == nil {
		t.Fatal("expected duplicate-id error")
	}
}

// TestNilVsZeroPointer is the crux: a nil pointer and a pointer to zero must not
// collapse into each other.
func TestNilVsZeroPointer(t *testing.T) {
	in := []PtrRecord{{ID: 1, OptInt: nil}, {ID: 2, OptInt: i32p(0)}}
	data, _ := Marshal(in)
	var out []PtrRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out[0].OptInt != nil {
		t.Fatalf("nil pointer decoded as non-nil %v", out[0].OptInt)
	}
	if out[1].OptInt == nil || *out[1].OptInt != 0 {
		t.Fatalf("&0 pointer decoded wrong: %v", out[1].OptInt)
	}
}
