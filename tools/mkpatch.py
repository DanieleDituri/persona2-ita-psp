"""Sector-level ISO patch generator (format P2IT v1) -- pure stdlib.

python mkpatch.py ORIGINAL.iso PATCHED.iso OUT.p2it
Format: 8 magic "P2IT\x01\x00\x00\x00", u64 out_size, u32 block, then raw-DEFLATE stream of ops:
  0x01 COPY  u64 src_block u32 n     (n blocks copied from the original)
  0x02 DATA  u32 n + n*block bytes   (new data)
  0x03 ZERO  u32 n                   (n zero blocks)
  0x00 END
Output is truncated to out_size. Prints sha256 of original/output (json line).
"""
import hashlib, json, mmap, os, struct, sys, zlib

BS = 2048

def main(orig, new, outp):
    fo, fn = open(orig, "rb"), open(new, "rb")
    mo = mmap.mmap(fo.fileno(), 0, access=mmap.ACCESS_READ)
    mn = mmap.mmap(fn.fileno(), 0, access=mmap.ACCESS_READ)
    no, nn = (len(mo) + BS - 1) // BS, (len(mn) + BS - 1) // BS
    zero = bytes(BS)
    idx = {}
    for i in range(no):
        b = mo[i * BS:(i + 1) * BS]
        if len(b) < BS:
            b = b + bytes(BS - len(b))
        if b == zero:
            continue
        h = hashlib.blake2b(b, digest_size=16).digest()
        idx.setdefault(h, i)
    comp = zlib.compressobj(9, zlib.DEFLATED, -15, 9)
    out = open(outp + ".tmp", "wb")
    out.write(b"P2IT\x01\x00\x00\x00" + struct.pack("<QI", len(mn), BS))
    stats = {"copy": 0, "data": 0, "zero": 0}
    op = None  # [type, start/src, n, databuf]
    def flush():
        if not op:
            return
        t = op[0]
        if t == 1:
            out.write(comp.compress(struct.pack("<BQI", 1, op[1], op[2])))
        elif t == 2:
            out.write(comp.compress(struct.pack("<BI", 2, op[2])))
            for chunk in op[3]:
                out.write(comp.compress(chunk))
        else:
            out.write(comp.compress(struct.pack("<BI", 3, op[2])))
    for i in range(nn):
        b = mn[i * BS:(i + 1) * BS]
        if len(b) < BS:
            b = b + bytes(BS - len(b))
        if b == zero:
            if op and op[0] == 3:
                op[2] += 1
            else:
                flush(); op = [3, 0, 1, None]
            stats["zero"] += 1
            continue
        # continue an existing copy run?
        if op and op[0] == 1:
            s = op[1] + op[2]
            if s < no and mo[s * BS:(s + 1) * BS].ljust(BS, b"\0") == b:
                op[2] += 1; stats["copy"] += 1
                continue
        src = idx.get(hashlib.blake2b(b, digest_size=16).digest())
        if src is not None and mo[src * BS:(src + 1) * BS].ljust(BS, b"\0") == b:
            flush(); op = [1, src, 1, None]; stats["copy"] += 1
            continue
        if op and op[0] == 2 and op[2] < 4096:
            op[2] += 1; op[3].append(b)
        else:
            flush(); op = [2, 0, 1, [b]]
        stats["data"] += 1
    flush()
    out.write(comp.compress(b"\x00"))
    out.write(comp.flush())
    out.close()
    os.replace(outp + ".tmp", outp)
    def sha(m):
        h = hashlib.sha256(); step = 1 << 24
        for o in range(0, len(m), step):
            h.update(m[o:o + step])
        return h.hexdigest()
    info = {"orig_size": len(mo), "orig_sha256": sha(mo), "out_size": len(mn), "out_sha256": sha(mn),
            "patch_size": os.path.getsize(outp), **stats}
    print(json.dumps(info))

if __name__ == "__main__":
    main(*sys.argv[1:4])
