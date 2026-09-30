#!/usr/bin/env python3
"""Generate the golden vectors used by crypt_test.go.

Independent re-implementation of the Proxmox Backup Server client-side
encryption (proxmox-backup: pbs-tools/src/crypt_config.rs,
pbs-key-config/src/lib.rs, pbs-datastore/src/data_blob.rs and manifest.rs),
driving AES-256-GCM through OpenSSL's libcrypto -- the library PBS itself
uses -- so the Go implementation is checked against OpenSSL, not against
itself. Only the Python standard library and libcrypto are needed:

    python3 pbscommon/testdata/gen_crypt_vectors.py
"""
import base64
import ctypes
import ctypes.util
import hashlib
import hmac
import json
import zlib

lib = ctypes.CDLL(ctypes.util.find_library("crypto"))
lib.EVP_CIPHER_CTX_new.restype = ctypes.c_void_p
lib.EVP_aes_256_gcm.restype = ctypes.c_void_p
lib.EVP_EncryptInit_ex.argtypes = [ctypes.c_void_p] * 5
lib.EVP_EncryptUpdate.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.POINTER(ctypes.c_int), ctypes.c_char_p, ctypes.c_int]
lib.EVP_EncryptFinal_ex.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.POINTER(ctypes.c_int)]
lib.EVP_CIPHER_CTX_ctrl.argtypes = [ctypes.c_void_p, ctypes.c_int, ctypes.c_int, ctypes.c_void_p]
lib.EVP_CIPHER_CTX_free.argtypes = [ctypes.c_void_p]
EVP_CTRL_GCM_SET_IVLEN, EVP_CTRL_GCM_GET_TAG = 0x9, 0x10


def aes_256_gcm_encrypt(key, iv, data):
    ctx = lib.EVP_CIPHER_CTX_new()
    assert lib.EVP_EncryptInit_ex(ctx, lib.EVP_aes_256_gcm(), None, None, None) == 1
    assert lib.EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, len(iv), None) == 1
    assert lib.EVP_EncryptInit_ex(ctx, None, None, key, iv) == 1
    out = ctypes.create_string_buffer(len(data) + 16)
    n = ctypes.c_int(0)
    assert lib.EVP_EncryptUpdate(ctx, out, ctypes.byref(n), data, len(data)) == 1
    total = n.value
    fin = ctypes.create_string_buffer(16)
    assert lib.EVP_EncryptFinal_ex(ctx, fin, ctypes.byref(n)) == 1
    assert n.value == 0
    tag = ctypes.create_string_buffer(16)
    assert lib.EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag) == 1
    lib.EVP_CIPHER_CTX_free(ctx)
    return out.raw[:total], tag.raw


ENCRYPTED_BLOB_MAGIC = bytes([123, 103, 133, 190, 34, 45, 76, 240])

key = bytes(range(32))
id_key = hashlib.pbkdf2_hmac("sha256", key, b"_id_key", 10, 32)
fp_input = hashlib.sha256(b"Proxmox Backup Encryption Key Fingerprint").digest()
fingerprint = hashlib.sha256(fp_input + id_key).digest()

v = {}
v["key"] = key.hex()
v["id_key"] = id_key.hex()
v["fingerprint"] = ":".join("%02x" % b for b in fingerprint)

chunk = b"Proxmox Backup chunk digest test vector"
v["digest_input"] = chunk.decode()
v["digest"] = hashlib.sha256(chunk + id_key).hexdigest()

# Encrypted (uncompressed) blob: MAGIC || CRC32(ciphertext) || IV || TAG || ciphertext
blob_plain = b"Proxmox Backup encrypted blob test vector. " * 3
blob_iv = bytes(range(100, 116))
ct, tag = aes_256_gcm_encrypt(key, blob_iv, blob_plain)
crc = zlib.crc32(ct).to_bytes(4, "little")
v["blob_plain"] = blob_plain.decode()
v["blob_iv"] = blob_iv.hex()
v["blob"] = (ENCRYPTED_BLOB_MAGIC + crc + blob_iv + tag + ct).hex()


def keyfile(kdf, derived, salt, iv):
    ct, tag = aes_256_gcm_encrypt(derived, iv, key)
    return json.dumps({
        "kdf": kdf,
        "created": "2026-09-30T12:00:00+02:00",
        "modified": "2026-09-30T12:00:00+02:00",
        "data": base64.b64encode(iv + tag + ct).decode(),
        "fingerprint": v["fingerprint"],
        "hint": "test",
    })


passphrase = b"correct horse battery"
salt = bytes(range(200, 232))
kiv = bytes(range(50, 66))
scrypt_key = hashlib.scrypt(passphrase, salt=salt, n=65536, r=8, p=1, maxmem=1025 * 1024 * 1024, dklen=32)
v["passphrase"] = passphrase.decode()
v["keyfile_scrypt"] = keyfile({"Scrypt": {"n": 65536, "r": 8, "p": 1, "salt": base64.b64encode(salt).decode()}}, scrypt_key, salt, kiv)
pbkdf2_key = hashlib.pbkdf2_hmac("sha256", passphrase, salt, 65535, 32)
v["keyfile_pbkdf2"] = keyfile({"PBKDF2": {"iter": 65535, "salt": base64.b64encode(salt).decode()}}, pbkdf2_key, salt, kiv)

# Manifest signature: HMAC-SHA256(id_key, canonical JSON of the manifest
# without "unprotected"/"signature"). Only the PBS manifest schema is signed
# (PBS drops unknown top-level members when it rewrites the manifest).
manifest = {
    "backup-type": "host",
    "backup-id": "win-dc01",
    "backup-time": 1790000000,
    "files": [
        {"crypt-mode": "encrypt", "csum": "ab" * 32, "filename": "C.pxar.didx", "size": 123456789012},
        {"crypt-mode": "encrypt", "csum": "cd" * 32, "filename": "catalog.pcat1.didx", "size": 42},
    ],
    "signature": None,
    "unprotected": {"chunk_upload_stats": {"count": 3}},
}
signed = {k: val for k, val in manifest.items() if k not in ("unprotected", "signature")}


def canonical(obj):
    # Python's sorted compact dump with ensure_ascii=False escapes exactly like
    # serde_json: short forms for \b \f \n \r \t, lowercase \u00XX for other
    # control characters, everything else (<, >, &, /, U+2028) left raw.
    return json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


v["manifest_canonical"] = canonical(signed).decode()
v["manifest_signature"] = hmac.new(id_key, canonical(signed), hashlib.sha256).hexdigest()

tricky = "R&D <files> \u00e9t\u00e9 \u2028 tab\there \x01\x1f \"q\" \\ / \b\f\r\n"
v["canonical_string_input"] = tricky
v["canonical_string"] = canonical({"s": tricky, "a": [True, False, 1]}).decode()

print(json.dumps(v, indent=2, ensure_ascii=False))
