package hashes

import "testing"

// Known-answer tests. NTLM and MD4 vectors are the canonical published values.
func TestKnownDigests(t *testing.T) {
	cases := []struct {
		algo, in, want string
	}{
		{"md5", "password", "5f4dcc3b5aa765d61d8327deb882cf99"},
		{"sha1", "password", "5baa61e4c9b93f3f0682250b6cf8331b7ee68fd8"},
		{"sha256", "password", "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"},
		{"sha512", "password", "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86"},
		// NTLM(password) = MD4(UTF-16LE("password")) — the canonical NT hash.
		{"ntlm", "password", "8846f7eaee8fb117ad06bdd830b7586c"},
		{"ntlm", "", "31d6cfe0d16ae931b73c59d7e0c089c0"}, // empty-string NT hash
		{"ntlm", "Password123!", "2b576acbe6bcfda7294d6bd18041b8fe"},
	}
	for _, c := range cases {
		a, ok := Lookup(c.algo)
		if !ok {
			t.Fatalf("Lookup(%q) failed", c.algo)
		}
		if got := a.Hash(c.in); got != c.want {
			t.Errorf("%s(%q) = %s, want %s", c.algo, c.in, got, c.want)
		}
	}
}

func TestMD4KnownAnswers(t *testing.T) {
	// RFC 1320 test vectors.
	cases := map[string]string{
		"":               "31d6cfe0d16ae931b73c59d7e0c089c0",
		"a":              "bde52cb31de33e46245e05fbdbd6fb24",
		"abc":            "a448017aaf21d8525fc10ae87aa6729d",
		"message digest": "d9130a8164549fe818874806e1c7014b",
	}
	for in, want := range cases {
		sum := md4Sum([]byte(in))
		got := hexHash(sum[:])
		if got != want {
			t.Errorf("md4(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestLookupAliases(t *testing.T) {
	for _, alias := range []string{"NTLM", "nt", "NT", " ntlm "} {
		if a, ok := Lookup(alias); !ok || a.Name != "ntlm" {
			t.Errorf("Lookup(%q) did not resolve to ntlm", alias)
		}
	}
	if a, ok := Lookup("sha-1"); !ok || a.Name != "sha1" {
		t.Errorf("Lookup(sha-1) = %v,%v", a.Name, ok)
	}
	if _, ok := Lookup("bcrypt"); ok {
		t.Error("bcrypt must not be supported (salted/slow)")
	}
}

func TestSetOrderingAndLine(t *testing.T) {
	// Request order should not matter; columns follow Supported() order.
	s, err := NewSet([]string{"ntlm,md5", "sha256"})
	if err != nil {
		t.Fatal(err)
	}
	got := s.Names()
	want := []string{"md5", "sha256", "ntlm"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names() = %v, want %v", got, want)
		}
	}
	line := s.Line("password", ":")
	wantLine := "password:5f4dcc3b5aa765d61d8327deb882cf99:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8:8846f7eaee8fb117ad06bdd830b7586c"
	if line != wantLine {
		t.Errorf("Line = %q, want %q", line, wantLine)
	}
}

func TestEmptySet(t *testing.T) {
	s, err := NewSet(nil)
	if err != nil || !s.Empty() {
		t.Fatalf("empty set: %v, empty=%v", err, s.Empty())
	}
	if got := s.Line("password", ":"); got != "password" {
		t.Errorf("empty set Line = %q, want password", got)
	}
}

func TestUnknownAlgoRejected(t *testing.T) {
	if _, err := NewSet([]string{"md5", "crc32"}); err == nil {
		t.Error("expected error for unknown algo")
	}
}
