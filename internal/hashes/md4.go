// This file is a self-contained implementation of the MD4 message digest
// (RFC 1320), used only to derive NTLM hashes for Active Directory audits. It
// is vendored here, rather than pulling in golang.org/x/crypto/md4, so the
// project keeps its single-static-binary, minimal-dependency posture.
//
// MD4 is cryptographically broken and MUST NOT be used for anything but
// interoperability with legacy hash formats (here: NTLM) in authorized audits.
package hashes

import "encoding/binary"

// md4Sum returns the 16-byte MD4 digest of data.
func md4Sum(data []byte) [16]byte {
	var a uint32 = 0x67452301
	var b uint32 = 0xefcdab89
	var c uint32 = 0x98badcfe
	var d uint32 = 0x10325476

	// Pad: append 0x80, then zeros, then the 64-bit little-endian bit length.
	msgLenBits := uint64(len(data)) * 8
	padded := make([]byte, 0, len(data)+72)
	padded = append(padded, data...)
	padded = append(padded, 0x80)
	for len(padded)%64 != 56 {
		padded = append(padded, 0x00)
	}
	var lenBuf [8]byte
	binary.LittleEndian.PutUint64(lenBuf[:], msgLenBits)
	padded = append(padded, lenBuf[:]...)

	rol := func(x uint32, n uint) uint32 { return x<<n | x>>(32-n) }

	for off := 0; off < len(padded); off += 64 {
		var x [16]uint32
		for i := 0; i < 16; i++ {
			x[i] = binary.LittleEndian.Uint32(padded[off+i*4:])
		}
		aa, bb, cc, dd := a, b, c, d

		// Round 1: F(x,y,z) = (x&y)|(~x&z)
		f := func(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
		r1 := func(a, b, c, d, k uint32, s uint) uint32 { return rol(a+f(b, c, d)+x[k], s) }
		for i := 0; i < 16; i += 4 {
			a = r1(a, b, c, d, uint32(i), 3)
			d = r1(d, a, b, c, uint32(i+1), 7)
			c = r1(c, d, a, b, uint32(i+2), 11)
			b = r1(b, c, d, a, uint32(i+3), 19)
		}

		// Round 2: G(x,y,z) = (x&y)|(x&z)|(y&z), constant 0x5a827999
		g := func(x, y, z uint32) uint32 { return (x & y) | (x & z) | (y & z) }
		r2 := func(a, b, c, d, k uint32, s uint) uint32 { return rol(a+g(b, c, d)+x[k]+0x5a827999, s) }
		for _, k := range []uint32{0, 1, 2, 3} {
			a = r2(a, b, c, d, k, 3)
			d = r2(d, a, b, c, k+4, 5)
			c = r2(c, d, a, b, k+8, 9)
			b = r2(b, c, d, a, k+12, 13)
		}

		// Round 3: H(x,y,z) = x^y^z, constant 0x6ed9eba1
		h := func(x, y, z uint32) uint32 { return x ^ y ^ z }
		r3 := func(a, b, c, d, k uint32, s uint) uint32 { return rol(a+h(b, c, d)+x[k]+0x6ed9eba1, s) }
		for _, k := range []uint32{0, 2, 1, 3} {
			a = r3(a, b, c, d, k, 3)
			d = r3(d, a, b, c, k+8, 9)
			c = r3(c, d, a, b, k+4, 11)
			b = r3(b, c, d, a, k+12, 15)
		}

		a += aa
		b += bb
		c += cc
		d += dd
	}

	var out [16]byte
	binary.LittleEndian.PutUint32(out[0:], a)
	binary.LittleEndian.PutUint32(out[4:], b)
	binary.LittleEndian.PutUint32(out[8:], c)
	binary.LittleEndian.PutUint32(out[12:], d)
	return out
}
